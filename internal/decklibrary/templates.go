package decklibrary

import (
	"fmt"
	"sort"
	"strings"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

const officialDeckSourceURL = "https://speedrobogames.com/public-decks/"

// Template describes one publisher-authored, read-only starting deck.
type Template struct {
	ID            string
	Name          string
	Product       string
	SourceURL     string
	SourceUpdated string
	expansion     string
	element       string
	twoCopyIDs    []string
}

var officialTemplates = []Template{
	{
		ID: "dd01-ignus", Name: "DD01 Ignus", Product: "DD01: Magical Girl Duel",
		SourceUpdated: "2026-07-13", expansion: "DD01: Magical Girl Duel", element: "Ignus",
		twoCopyIDs: []string{"1", "2", "3", "4", "5", "7", "9", "11", "13", "15", "16", "18", "37"},
	},
	{
		ID: "dd01-terra", Name: "DD01 Terra", Product: "DD01: Magical Girl Duel",
		SourceUpdated: "2026-07-13", expansion: "DD01: Magical Girl Duel", element: "Terra",
		twoCopyIDs: []string{"19", "20", "21", "22", "23", "25", "27", "28", "29", "33", "35", "36", "38"},
	},
	{
		ID: "dd02-aqua", Name: "DD02 Aqua", Product: "DD02: Away Game",
		SourceUpdated: "2026-06-01", expansion: "DD02: Away Game", element: "Aqua",
		twoCopyIDs: []string{"39", "40", "41", "43", "44", "45", "48", "49", "50", "54", "70"},
	},
	{
		ID: "dd02-void", Name: "DD02 Void", Product: "DD02: Away Game",
		SourceUpdated: "2026-06-01", expansion: "DD02: Away Game", element: "Void",
		twoCopyIDs: []string{"56", "60", "61", "66", "71"},
	},
	{
		ID: "dd03-luna", Name: "DD03 Luna", Product: "DD03: Forgotten in the Moonlight",
		SourceUpdated: "2026-06-01", expansion: "DD03: Forgotten in the Moonlight", element: "Luna",
		twoCopyIDs: []string{"72", "73", "74", "75", "76", "77", "80", "84", "87"},
	},
	{
		ID: "dd03-silva", Name: "DD03 Silva", Product: "DD03: Forgotten in the Moonlight",
		SourceUpdated: "2026-06-01", expansion: "DD03: Forgotten in the Moonlight", element: "Silva",
		twoCopyIDs: []string{"89", "90", "91", "92", "93", "94", "101"},
	},
	{
		ID: "dd04-solis", Name: "DD04 Solis", Product: "DD04: Popularity Contest",
		SourceUpdated: "2026-06-01", expansion: "DD04: Popularity Contest", element: "Solis",
		twoCopyIDs: []string{"121", "122", "123", "124", "125", "126", "128", "130", "131", "134", "135", "137", "138"},
	},
	{
		ID: "dd04-aes", Name: "DD04 Aes", Product: "DD04: Popularity Contest",
		SourceUpdated: "2026-06-01", expansion: "DD04: Popularity Contest", element: "Aes",
		twoCopyIDs: []string{"105", "106", "107", "108", "109", "110", "112", "117", "120"},
	},
}

// OfficialTemplates returns detached metadata for the bundled official decks.
func OfficialTemplates() []Template {
	result := make([]Template, len(officialTemplates))
	copy(result, officialTemplates)
	for index := range result {
		result[index].SourceURL = officialDeckSourceURL
		result[index].twoCopyIDs = nil
	}
	return result
}

// BuildOfficialTemplate resolves a bundled definition against the active card
// catalog. This makes templates small while ensuring stale or incomplete card
// databases fail explicitly instead of producing a damaged deck.
func BuildOfficialTemplate(id string, repository *cards.Repository) (*decks.Deck, error) {
	if repository == nil {
		return nil, fmt.Errorf("card repository cannot be nil")
	}

	var definition *Template
	for index := range officialTemplates {
		if officialTemplates[index].ID == id {
			definition = &officialTemplates[index]
			break
		}
	}
	if definition == nil {
		return nil, fmt.Errorf("unknown official deck template %q", id)
	}

	twoCopies := make(map[string]bool, len(definition.twoCopyIDs))
	for _, cardID := range definition.twoCopyIDs {
		twoCopies[cardID] = true
	}

	matchingCards := make([]cards.Card, 0)
	for _, card := range repository.All() {
		if card.Expansion == definition.expansion &&
			strings.EqualFold(card.Element, definition.element) {
			matchingCards = append(matchingCards, card)
		}
	}
	sort.Slice(matchingCards, func(left, right int) bool {
		return matchingCards[left].CardNumber < matchingCards[right].CardNumber
	})

	deck, err := decks.NewDeck(definition.Name)
	if err != nil {
		return nil, err
	}
	for _, card := range matchingCards {
		quantity := 4
		if twoCopies[card.ID] {
			quantity = 2
			delete(twoCopies, card.ID)
		}
		deck.MainDeck = append(deck.MainDeck, decks.DeckEntry{
			CardID: card.ID, Quantity: quantity,
		})
	}
	if len(twoCopies) != 0 {
		return nil, fmt.Errorf("%s requires cards missing from the active catalog", definition.Name)
	}
	if deck.MainTotal() != decks.MaxMainDeckCards {
		return nil, fmt.Errorf(
			"%s resolved to %d cards; expected %d",
			definition.Name,
			deck.MainTotal(),
			decks.MaxMainDeckCards,
		)
	}
	deck.EnsureOrder()
	return deck, nil
}
