package scheduler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/lukeyeager/school-lunch-schedule/internal/config"
	"github.com/lukeyeager/school-lunch-schedule/internal/healthepro"
	"github.com/lukeyeager/school-lunch-schedule/internal/slack"
	"github.com/lukeyeager/school-lunch-schedule/internal/store"
)

// Scheduler runs the two-phase cron jobs.
type Scheduler struct {
	cfg   *config.Config
	hep   *healthepro.Client
	slack *slack.Client
	store *store.Store
	cron  *cron.Cron
	loc   *time.Location
}

// New creates a Scheduler. The timezone in cfg is used for cron scheduling.
func New(cfg *config.Config, hep *healthepro.Client, slackClient *slack.Client, db *store.Store) (*Scheduler, error) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", cfg.Timezone, err)
	}
	return &Scheduler{
		cfg:   cfg,
		hep:   hep,
		slack: slackClient,
		store: db,
		cron:  cron.New(cron.WithLocation(loc)),
		loc:   loc,
	}, nil
}

// Start registers cron jobs and starts the scheduler.
func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc(s.cfg.EveningCron, s.runEvening); err != nil {
		return fmt.Errorf("adding evening cron %q: %w", s.cfg.EveningCron, err)
	}
	if _, err := s.cron.AddFunc(s.cfg.MorningCron, s.runMorning); err != nil {
		return fmt.Errorf("adding morning cron %q: %w", s.cfg.MorningCron, err)
	}
	s.cron.Start()
	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// runEvening fetches tomorrow's menu and always posts it as a preview.
func (s *Scheduler) runEvening() {
	tomorrow := time.Now().In(s.loc).AddDate(0, 0, 1)
	slog.Info("evening run", "date", tomorrow.Format("2006-01-02"))

	entry, err := s.hep.FetchMenu(tomorrow)
	if err != nil {
		slog.Error("evening: failed to fetch menu", "err", err)
		return
	}
	if entry == nil {
		slog.Info("evening: no menu data for date", "date", tomorrow.Format("2006-01-02"))
		return
	}

	if err := s.slack.PostEveningPreview(tomorrow, entry); err != nil {
		slog.Error("evening: failed to post to slack", "err", err)
		return
	}

	if err := s.store.Upsert(entry.Day, entry, false); err != nil {
		slog.Error("evening: failed to store menu", "err", err)
	}
}

// runMorning fetches today's menu and posts only if it changed since the evening preview.
func (s *Scheduler) runMorning() {
	today := time.Now().In(s.loc)
	dateStr := today.Format("2006-01-02")
	slog.Info("morning run", "date", dateStr)

	entry, err := s.hep.FetchMenu(today)
	if err != nil {
		slog.Error("morning: failed to fetch menu", "err", err)
		return
	}
	if entry == nil {
		slog.Info("morning: no menu data for date", "date", dateStr)
		return
	}

	stored, err := s.store.Get(dateStr)
	if err != nil {
		slog.Error("morning: failed to get stored menu", "err", err)
		return
	}

	if stored != nil && itemsEqual(stored.Items, entry.Items) {
		slog.Info("morning: menu unchanged, skipping slack post")
		return
	}

	changed := stored != nil
	if err := s.slack.PostMorningUpdate(today, entry); err != nil {
		slog.Error("morning: failed to post to slack", "err", err)
		return
	}

	if err := s.store.Upsert(entry.Day, entry, changed); err != nil {
		slog.Error("morning: failed to store updated menu", "err", err)
	}
}

// itemsEqual compares two DisplayItem slices by their JSON representation.
func itemsEqual(a, b []healthepro.DisplayItem) bool {
	aj, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bj, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aj) == string(bj)
}
