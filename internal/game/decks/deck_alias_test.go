package decks

import (
	"reflect"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

type aliasTestCatalog map[string]gamecards.Card

func (catalog aliasTestCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, found := catalog[id]
	return card, found
}

func TestCanonicalizeCardIDsReplacesAliasesAndMergesEntries(t *testing.T) {
	canonical := gamecards.Card{ID: "1225", Name: "Carella"}
	catalog := aliasTestCatalog{"235": canonical, "1225": canonical}
	deck := &Deck{
		MainDeck:  []DeckEntry{{CardID: "235", Quantity: 1}, {CardID: "1225", Quantity: 2}},
		MainOrder: []string{"1225", "235", "1225"},
		SideDeck:  []DeckEntry{{CardID: "235", Quantity: 1}},
		SideOrder: []string{"235"},
	}

	changed, err := deck.CanonicalizeCardIDs(catalog)
	if err != nil {
		t.Fatalf("CanonicalizeCardIDs() error = %v", err)
	}
	if !changed {
		t.Fatal("CanonicalizeCardIDs() did not report migration")
	}
	wantMain := []DeckEntry{{CardID: "1225", Quantity: 3}}
	if !reflect.DeepEqual(deck.MainDeck, wantMain) {
		t.Fatalf("main entries = %#v, want %#v", deck.MainDeck, wantMain)
	}
	if !reflect.DeepEqual(deck.MainOrder, []string{"1225", "1225", "1225"}) ||
		!reflect.DeepEqual(deck.SideOrder, []string{"1225"}) {
		t.Fatalf("orders were not canonicalized: main %#v, side %#v", deck.MainOrder, deck.SideOrder)
	}
}

func TestCanonicalizeCardIDsLeavesDeckUntouchedOnUnknownID(t *testing.T) {
	deck := &Deck{
		MainDeck:  []DeckEntry{{CardID: "missing", Quantity: 1}},
		MainOrder: []string{"missing"},
	}
	before := *deck
	before.MainDeck = append([]DeckEntry(nil), deck.MainDeck...)
	before.MainOrder = append([]string(nil), deck.MainOrder...)

	if _, err := deck.CanonicalizeCardIDs(aliasTestCatalog{}); err == nil {
		t.Fatal("CanonicalizeCardIDs() accepted an unknown ID")
	}
	if !reflect.DeepEqual(*deck, before) {
		t.Fatalf("failed migration mutated deck: %#v", deck)
	}
}
