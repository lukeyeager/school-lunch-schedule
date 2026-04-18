package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/lukeyeager/school-lunch-schedule/internal/config"
	"github.com/lukeyeager/school-lunch-schedule/internal/healthepro"
	"github.com/lukeyeager/school-lunch-schedule/internal/metrics"
	"github.com/lukeyeager/school-lunch-schedule/internal/scheduler"
	"github.com/lukeyeager/school-lunch-schedule/internal/slack"
	"github.com/lukeyeager/school-lunch-schedule/internal/store"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, relying on environment variables")
	}

	cfg, err := config.Load("config.yaml")
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		slog.Error("SLACK_WEBHOOK_URL environment variable is not set")
		os.Exit(1)
	}

	m := metrics.New()

	db, err := store.New(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			slog.Error("failed to close database", "err", closeErr)
		}
	}()

	hepClient := healthepro.NewClient(cfg.OrgID, cfg.MenuID, m)
	slackClient := slack.NewClient(webhookURL, m)

	sched, err := scheduler.New(cfg, hepClient, slackClient, db)
	if err != nil {
		slog.Error("failed to create scheduler", "err", err)
		os.Exit(1)
	}
	if err := sched.Start(); err != nil {
		slog.Error("failed to start scheduler", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: ":9090", Handler: mux}
	go func() {
		slog.Info("metrics server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("metrics server error", "err", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	slog.Info("shutting down")
	sched.Stop()
	if err := srv.Shutdown(context.Background()); err != nil {
		slog.Error("metrics server shutdown error", "err", err)
	}
}
