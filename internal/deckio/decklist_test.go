package deckio

import (
	"bytes"
	"strings"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

// TestReadDeckListArthurFormat verifies the reference text format loads both zones and order.
func TestReadDeckListArthurFormat(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "56", Name: "Arthur", Expansion: "DD02: Away Game"},
		{ID: "71", Name: "Arthur Lv2", Expansion: "DD02: Away Game"},
	})
	if err != nil {
		t.Fatal(err)
	}

	input := `Deck Name: Arthur Test Deck

Main Deck (3)
3x Arthur [DD02: Away Game]

Side Deck (2)
2x Arthur Lv2 [DD02: Away Game]
`
	deck, err := ReadDeckList(strings.NewReader(input), repository)
	if err != nil {
		t.Fatal(err)
	}
	if deck.Name != "Arthur Test Deck" || deck.MainTotal() != 3 || deck.SideTotal() != 2 {
		t.Fatalf("unexpected deck: %#v", deck)
	}
	if len(deck.MainOrder) != 3 || len(deck.SideOrder) != 2 {
		t.Fatalf("card order was not expanded: %#v", deck)
	}

	var exported bytes.Buffer
	if err := WriteDeckList(&exported, deck, repository); err != nil {
		t.Fatal(err)
	}
	if exported.String() != input {
		t.Fatalf("round-trip decklist:\n%s", exported.String())
	}
}

// TestReadDeckListRejectsIncorrectTotal verifies declared totals must match parsed quantities.
func TestReadDeckListRejectsIncorrectTotal(t *testing.T) {
	repository, _ := cards.NewRepository([]cards.Card{
		{ID: "56", Name: "Arthur", Expansion: "DD02: Away Game"},
	})
	input := `Deck Name: Bad Total
Main Deck (2)
1x Arthur [DD02: Away Game]
Side Deck (0)
`
	_, err := ReadDeckList(strings.NewReader(input), repository)
	if err == nil || !strings.Contains(err.Error(), "declares 2") {
		t.Fatalf("ReadDeckList() error = %v", err)
	}
}
