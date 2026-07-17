package decks

import (
	"reflect"
	"testing"

	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
)

// TestDeckSortOrdersBothZonesAndEntries verifies visual and serialized zone order stay aligned.
func TestDeckSortOrdersBothZonesAndEntries(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "conjure", Name: "Burst", Type: "Conjure", CostLevel: "1"},
		{ID: "servant-high", Name: "Wolf", Type: "Servant", CostLevel: "4"},
		{ID: "caster", Name: "Alice", Type: "Caster", CostLevel: "2"},
		{ID: "servant-low", Name: "Bee", Type: "Servant", CostLevel: "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	deck := &Deck{
		SchemaVersion: 1,
		Name:          "Sort Test",
		MainDeck: []DeckEntry{
			{CardID: "conjure", Quantity: 1},
			{CardID: "servant-high", Quantity: 2},
			{CardID: "caster", Quantity: 1},
			{CardID: "servant-low", Quantity: 1},
		},
		SideDeck: []DeckEntry{
			{CardID: "conjure", Quantity: 1},
			{CardID: "caster", Quantity: 1},
		},
		MainOrder: []string{
			"servant-high",
			"conjure",
			"caster",
			"servant-low",
			"servant-high",
		},
		SideOrder: []string{"conjure", "caster"},
	}

	if err := deck.Sort(repository); err != nil {
		t.Fatal(err)
	}
	wantMainOrder := []string{
		"caster",
		"servant-low",
		"servant-high",
		"servant-high",
		"conjure",
	}
	if !reflect.DeepEqual(deck.MainOrder, wantMainOrder) {
		t.Fatalf("main order = %#v, want %#v", deck.MainOrder, wantMainOrder)
	}
	wantSideOrder := []string{"caster", "conjure"}
	if !reflect.DeepEqual(deck.SideOrder, wantSideOrder) {
		t.Fatalf("side order = %#v, want %#v", deck.SideOrder, wantSideOrder)
	}
	wantEntryIDs := []string{"caster", "servant-low", "servant-high", "conjure"}
	gotEntryIDs := make([]string, len(deck.MainDeck))
	for index, entry := range deck.MainDeck {
		gotEntryIDs[index] = entry.CardID
	}
	if !reflect.DeepEqual(gotEntryIDs, wantEntryIDs) {
		t.Fatalf("main entries = %#v, want %#v", gotEntryIDs, wantEntryIDs)
	}
}
