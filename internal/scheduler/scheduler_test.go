package scheduler

import (
	"testing"

	"github.com/lukeyeager/school-lunch-schedule/internal/healthepro"
)

func TestItemsEqual(t *testing.T) {
	a := []healthepro.DisplayItem{
		{Type: "category", Name: "Lunch Entree"},
		{Type: "recipe", Name: "Pizza"},
	}
	b := []healthepro.DisplayItem{
		{Type: "category", Name: "Lunch Entree"},
		{Type: "recipe", Name: "Pizza"},
	}
	if !itemsEqual(a, b) {
		t.Error("expected equal items to be equal")
	}
}

func TestItemsEqual_Different(t *testing.T) {
	a := []healthepro.DisplayItem{{Type: "recipe", Name: "Pizza"}}
	b := []healthepro.DisplayItem{{Type: "recipe", Name: "Tacos"}}
	if itemsEqual(a, b) {
		t.Error("expected different items to not be equal")
	}
}

func TestItemsEqual_DifferentLength(t *testing.T) {
	a := []healthepro.DisplayItem{{Type: "recipe", Name: "Pizza"}}
	b := []healthepro.DisplayItem{}
	if itemsEqual(a, b) {
		t.Error("expected different-length slices to not be equal")
	}
}

func TestItemsEqual_BothEmpty(t *testing.T) {
	if !itemsEqual(nil, nil) {
		t.Error("expected nil slices to be equal")
	}
	if !itemsEqual([]healthepro.DisplayItem{}, []healthepro.DisplayItem{}) {
		t.Error("expected empty slices to be equal")
	}
}
