package deckui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
)

// TestCardTileMouseInSelectsCard verifies entering a tile updates the card preview once.
func TestCardTileMouseInSelectsCard(t *testing.T) {
	card := cards.Card{ID: "TEST-001", Name: "Test Card"}
	selectionCount := 0
	var selected cards.Card
	tile := &CardTile{
		Card: card,
		OnSelected: func(card cards.Card) {
			selectionCount++
			selected = card
		},
	}

	tile.MouseIn(&desktop.MouseEvent{})
	tile.MouseMoved(&desktop.MouseEvent{})

	if selectionCount != 1 {
		t.Fatalf("selection count = %d, want 1", selectionCount)
	}
	if selected.ID != card.ID {
		t.Fatalf("selected card ID = %q, want %q", selected.ID, card.ID)
	}
}

// TestCardTileTappedSelectsCard verifies tapping remains available without pointer hover.
func TestCardTileTappedSelectsCard(t *testing.T) {
	card := cards.Card{ID: "TEST-002", Name: "Touch Card"}
	selectedID := ""
	tile := &CardTile{
		Card: card,
		OnSelected: func(card cards.Card) {
			selectedID = card.ID
		},
	}

	tile.Tapped(&fyne.PointEvent{})

	if selectedID != card.ID {
		t.Fatalf("selected card ID = %q, want %q", selectedID, card.ID)
	}
}
