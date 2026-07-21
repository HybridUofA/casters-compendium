package deckio

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

// TestReadDeckListLegacyFormat verifies legacy input remains readable and is
// exported in the canonical Speedrobo format without changing deck contents.
func TestReadDeckListLegacyFormat(t *testing.T) {
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
	wantExport := `Deck: Arthur Test Deck
Game: The Caster Chronicles

3x Arthur (DD02: Away Game)

Total: 3 cards

--- Side Deck ---

2x Arthur Lv2 (DD02: Away Game)

Side Total: 2 cards
`
	if exported.String() != wantExport {
		t.Fatalf("round-trip decklist:\n%s", exported.String())
	}

	roundTripped, err := ReadDeckList(strings.NewReader(exported.String()), repository)
	if err != nil {
		t.Fatalf("read exported decklist: %v", err)
	}
	if roundTripped.Name != deck.Name ||
		!slices.Equal(roundTripped.MainOrder, deck.MainOrder) ||
		!slices.Equal(roundTripped.SideOrder, deck.SideOrder) {
		t.Fatalf("round-tripped deck = %#v, want %#v", roundTripped, deck)
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

// TestReadDeckListSpeedroboFormat verifies the published Speedrobo format loads
// metadata, parenthesized expansions, totals, zones, and per-copy order.
func TestReadDeckListSpeedroboFormat(t *testing.T) {
	repository, err := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition", Expansion: "DD03: Forgotten in the Moonlight"},
		{ID: "pentachi", Name: "Pentachi", Expansion: "DD03: Forgotten in the Moonlight"},
		{ID: "call-forth", Name: "Call Forth", Expansion: "EX01: The Bell Tolls"},
	})
	if err != nil {
		t.Fatal(err)
	}

	input := `Deck: Luna/Aqua Control
Game: The Caster Chronicles

1x Abolition (DD03: Forgotten in the Moonlight)
2x Pentachi (DD03: Forgotten in the Moonlight)

Total: 3 cards

--- Side Deck ---

2x Call Forth (EX01: The Bell Tolls)

Side Total: 2 cards
`
	deck, err := ReadDeckList(strings.NewReader(input), repository)
	if err != nil {
		t.Fatal(err)
	}

	if deck.Name != "Luna/Aqua Control" {
		t.Fatalf("deck.Name = %q", deck.Name)
	}
	if deck.MainTotal() != 3 || deck.SideTotal() != 2 {
		t.Fatalf("deck totals = main %d, side %d", deck.MainTotal(), deck.SideTotal())
	}
	wantMainOrder := []string{"abolition", "pentachi", "pentachi"}
	if !slices.Equal(deck.MainOrder, wantMainOrder) {
		t.Fatalf("deck.MainOrder = %v, want %v", deck.MainOrder, wantMainOrder)
	}
	wantSideOrder := []string{"call-forth", "call-forth"}
	if !slices.Equal(deck.SideOrder, wantSideOrder) {
		t.Fatalf("deck.SideOrder = %v, want %v", deck.SideOrder, wantSideOrder)
	}
}

// TestReadDeckListSpeedroboRejectsUnsupportedGame verifies game metadata is
// validated before any card lines are processed.
func TestReadDeckListSpeedroboRejectsUnsupportedGame(t *testing.T) {
	repository, _ := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition", Expansion: "DD03: Forgotten in the Moonlight"},
	})
	input := `Deck: Wrong Game
Game: Chess
`

	_, err := ReadDeckList(strings.NewReader(input), repository)
	if err == nil || !strings.Contains(err.Error(), `unsupported game "Chess"`) {
		t.Fatalf("ReadDeckList() error = %v", err)
	}
}

// TestReadDeckListSpeedroboRejectsInvalidTotals verifies invalid numeric text
// and negative totals return useful errors rather than panicking.
func TestReadDeckListSpeedroboRejectsInvalidTotals(t *testing.T) {
	repository, _ := cards.NewRepository([]cards.Card{
		{ID: "abolition", Name: "Abolition", Expansion: "DD03: Forgotten in the Moonlight"},
	})

	tests := []struct {
		name      string
		totalLine string
		wantError string
	}{
		{name: "nonnumeric main", totalLine: "Total: fifty cards", wantError: `invalid main deck total "fifty"`},
		{name: "negative main", totalLine: "Total: -1 cards", wantError: `invalid main deck total "-1"`},
		{name: "nonnumeric side", totalLine: "Side Total: twelve cards", wantError: `invalid side deck total "twelve"`},
		{name: "negative side", totalLine: "Side Total: -1 cards", wantError: `invalid side deck total "-1"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := "Deck: Invalid Total\nGame: The Caster Chronicles\n" + test.totalLine + "\n"
			_, err := ReadDeckList(strings.NewReader(input), repository)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("ReadDeckList() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

// TestParseDecklistCardRejectsMissingExpansionDelimiters verifies malformed
// card lines fail safely after delimiter detection.
func TestParseDecklistCardRejectsMissingExpansionDelimiters(t *testing.T) {
	_, _, _, err := parseDecklistCard("1x Abolition DD03: Forgotten in the Moonlight")
	if err == nil || !strings.Contains(err.Error(), "brackets or parentheses") {
		t.Fatalf("parseDecklistCard() error = %v", err)
	}
}
