package catalog

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
	repository, err := LoadFile("../../../data/cards.json")
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
// filters and only matches labels, not references within effect prose.
func TestFilterByKeyword(t *testing.T) {
	repository, err := NewRepository([]Card{
		{ID: "1", Name: "Matching", Type: "Servant", Ability: "Enter: Draw a card."},
		{ID: "2", Name: "Wrong type", Type: "Conjure", Ability: "Enter: Draw a card."},
		{ID: "3", Name: "Wrong word", Type: "Servant", Ability: "A servant entered the field."},
		{ID: "4", Name: "Keyword reference", Type: "Servant", Ability: "Rest: This card gains Enter."},
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

// TestBreakFilterExcludesEffectReferences covers cards that discuss Break but
// do not have Break as one of their own ability labels.
func TestBreakFilterExcludesEffectReferences(t *testing.T) {
	repository, err := LoadFile("../../../data/cards.json")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	for _, card := range repository.Filter(Filter{
		Keywords:       []string{"Break"},
		IncludeTesting: true,
	}) {
		if card.ID == "181" || card.ID == "186" {
			t.Errorf(
				"Break filter included %s (%s), whose ability only references Break",
				card.Name,
				card.CardNumber,
			)
		}
	}
}

// TestBundledSearchKeepsDistinctPrintingsAndCollapsesSharedArtwork covers
// upstream records for alternate and promotional printings. Distinct artwork
// remains selectable while records sharing one image use the lowest card
// number in ordinary search results.
func TestBundledSearchKeepsDistinctPrintingsAndCollapsesSharedArtwork(t *testing.T) {
	repository, err := LoadFile("../../../data/cards.json")
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	passionIDs := make(map[string]bool)
	for _, card := range repository.Filter(Filter{
		Name:           "Passion Wing",
		IncludeTesting: true,
	}) {
		passionIDs[card.ID] = true
	}
	for _, wantedID := range []string{"1", "1101", "1117", "1222", "1246"} {
		if !passionIDs[wantedID] {
			t.Errorf("Passion Wing search IDs = %#v, missing current printing %s", passionIDs, wantedID)
		}
	}
	for _, duplicateID := range []string{"181", "186", "1247"} {
		if passionIDs[duplicateID] {
			t.Errorf("Passion Wing search IDs = %#v, included removed or shared-art record %s", passionIDs, duplicateID)
		}
	}

	pentachiIDs := make(map[string]bool)
	for _, card := range repository.Filter(Filter{
		Name:           "Pentachi",
		IncludeTesting: true,
	}) {
		pentachiIDs[card.ID] = true
	}
	if !pentachiIDs["78"] || !pentachiIDs["1218"] || pentachiIDs["182"] || pentachiIDs["1272"] {
		t.Errorf("Pentachi search IDs = %#v, want base card 78 and alternate-art card 1218", pentachiIDs)
	}

	// Duplicate IDs remain addressable for compatibility with saved decks.
	if _, found := repository.FindByID("182"); !found {
		t.Error("duplicate printing ID 182 no longer resolves")
	}
	if _, found := repository.FindByID("1272"); !found {
		t.Error("duplicate printing ID 1272 no longer resolves")
	}
}
