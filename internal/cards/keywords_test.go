package cards

import (
	"reflect"
	"testing"
)

// TestRepositoryKeywords verifies that keyword choices come from the ability
// data rather than a fixed application list.
func TestRepositoryKeywords(t *testing.T) {
	repository, err := NewRepository([]Card{
		{ID: "1", Name: "One", Ability: "• Break (Definition.)\n• Enter: Draw a card."},
		{ID: "2", Name: "Two", Ability: "[Unity](Definition.)\n• Last Words → Return me."},
		{ID: "3", Name: "Three", Ability: "• Double Corrupt\n• Quickcast (Definition.)"},
		{ID: "4", Name: "Four", Ability: "• [Universal](Definition.)\n• Rest, discard a caster: Draw a card."},
		{ID: "5", Name: "Five", Ability: "• Slow Start (Definition.)\n• Discard a card: Draw two cards."},
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	want := []string{
		"Break",
		"Double Corrupt",
		"Enter",
		"Last Words",
		"Quickcast",
		"Rest",
		"Slow Start",
		"Unity",
		"Universal",
	}
	if got := repository.Keywords(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Keywords() = %#v, want %#v", got, want)
	}
}

// TestBundledDatabaseKeywords guards the keyword forms currently used by the
// shipped card data while allowing the data-driven list to grow over time.
func TestBundledDatabaseKeywords(t *testing.T) {
	repository, err := LoadFile("../../data/cards.json")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	available := make(map[string]bool)
	for _, keyword := range repository.Keywords() {
		available[keyword] = true
	}

	for _, keyword := range []string{
		"Break",
		"Double Corrupt",
		"Enter",
		"Hubris",
		"Last Words",
		"Quickcast",
		"Rest",
		"Slow Start",
		"Unity",
		"Universal",
	} {
		if !available[keyword] {
			t.Errorf("Keywords() did not extract %q from bundled data", keyword)
		}
	}
}

// TestFilterByKeyword verifies that keyword filtering composes with existing
// filters and does not match a keyword embedded inside a larger word.
func TestFilterByKeyword(t *testing.T) {
	repository, err := NewRepository([]Card{
		{ID: "1", Name: "Matching", Type: "Servant", Ability: "Enter: Draw a card."},
		{ID: "2", Name: "Wrong type", Type: "Conjure", Ability: "Enter: Draw a card."},
		{ID: "3", Name: "Wrong word", Type: "Servant", Ability: "A servant entered the field."},
	})
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	matches := repository.Filter(Filter{
		Types:    []string{"Servant"},
		Keywords: []string{"Enter"},
	})
	if len(matches) != 1 || matches[0].ID != "1" {
		t.Fatalf("Filter() = %#v, want only card 1", matches)
	}
}
