package store

import (
	"testing"

	"github.com/lukeyeager/school-lunch-schedule/internal/healthepro"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

var sampleItems = []healthepro.DisplayItem{
	{Type: "category", Name: "Lunch Entree"},
	{Type: "recipe", Name: "Pizza"},
}

func TestUpsertAndGet(t *testing.T) {
	s := newTestStore(t)
	entry := &healthepro.DayEntry{Day: "2026-04-20", Source: "current", Items: sampleItems}

	if err := s.Upsert("2026-04-20", entry, false); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	rec, err := s.Get("2026-04-20")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if rec == nil {
		t.Fatal("expected record, got nil")
	}
	if rec.Date != "2026-04-20" {
		t.Errorf("expected date 2026-04-20, got %q", rec.Date)
	}
	if rec.Source != "current" {
		t.Errorf("expected source=current, got %q", rec.Source)
	}
	if rec.Changed {
		t.Error("expected changed=false")
	}
	if len(rec.Items) != 2 || rec.Items[1].Name != "Pizza" {
		t.Errorf("unexpected items: %+v", rec.Items)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	rec, err := s.Get("2026-04-20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec != nil {
		t.Errorf("expected nil, got %+v", rec)
	}
}

func TestUpsert_OverwritesAndSetsChanged(t *testing.T) {
	s := newTestStore(t)
	entry := &healthepro.DayEntry{Day: "2026-04-20", Source: "current", Items: sampleItems}

	if err := s.Upsert("2026-04-20", entry, false); err != nil {
		t.Fatalf("initial upsert failed: %v", err)
	}

	updated := &healthepro.DayEntry{
		Day:    "2026-04-20",
		Source: "original",
		Items:  []healthepro.DisplayItem{{Type: "recipe", Name: "Tacos"}},
	}
	if err := s.Upsert("2026-04-20", updated, true); err != nil {
		t.Fatalf("second upsert failed: %v", err)
	}

	rec, err := s.Get("2026-04-20")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !rec.Changed {
		t.Error("expected changed=true")
	}
	if rec.Source != "original" {
		t.Errorf("expected source=original, got %q", rec.Source)
	}
	if len(rec.Items) != 1 || rec.Items[0].Name != "Tacos" {
		t.Errorf("expected Tacos, got %+v", rec.Items)
	}
}
