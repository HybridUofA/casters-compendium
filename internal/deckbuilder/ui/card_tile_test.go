package deckui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	fynetest "fyne.io/fyne/v2/test"

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

// TestCardTileMouseUpTogglesSelection verifies the platform shortcut modifier triggers selection.
func TestCardTileMouseUpTogglesSelection(t *testing.T) {
	toggleCount := 0
	tile := &CardTile{
		OnToggleSelection: func() {
			toggleCount++
		},
	}

	tile.MouseUp(&desktop.MouseEvent{
		Button:   desktop.MouseButtonPrimary,
		Modifier: fyne.KeyModifierShortcutDefault,
	})

	if toggleCount != 1 {
		t.Fatalf("toggle count = %d, want 1", toggleCount)
	}
}

// TestCardTileMouseUpSelectionModifierHandling verifies unrelated click paths do not toggle selection.
func TestCardTileMouseUpSelectionModifierHandling(t *testing.T) {
	tests := []struct {
		name     string
		button   desktop.MouseButton
		modifier fyne.KeyModifier
		want     int
	}{
		{
			name:   "ordinary primary click",
			button: desktop.MouseButtonPrimary,
		},
		{
			name:     "shift primary click",
			button:   desktop.MouseButtonPrimary,
			modifier: fyne.KeyModifierShift,
		},
		{
			name:     "shortcut secondary click",
			button:   desktop.MouseButtonSecondary,
			modifier: fyne.KeyModifierShortcutDefault,
		},
		{
			name:     "shortcut with additional modifier",
			button:   desktop.MouseButtonPrimary,
			modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
			want:     1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			toggleCount := 0
			tile := &CardTile{
				OnToggleSelection: func() {
					toggleCount++
				},
			}

			tile.MouseUp(&desktop.MouseEvent{
				Button:   test.button,
				Modifier: test.modifier,
			})

			if toggleCount != test.want {
				t.Fatalf("toggle count = %d, want %d", toggleCount, test.want)
			}
		})
	}
}

// TestCardTileMouseUpToggleSelectionNilCallback verifies search tiles remain safe.
func TestCardTileMouseUpToggleSelectionNilCallback(t *testing.T) {
	tile := &CardTile{}

	tile.MouseUp(&desktop.MouseEvent{
		Button:   desktop.MouseButtonPrimary,
		Modifier: fyne.KeyModifierShortcutDefault,
	})
}

// TestCardTileMouseUpSelectionDoesNotDispatchRightClick verifies click actions remain isolated.
func TestCardTileMouseUpSelectionDoesNotDispatchRightClick(t *testing.T) {
	toggleCount := 0
	rightClickCount := 0
	tile := &CardTile{
		OnToggleSelection: func() {
			toggleCount++
		},
		OnRightClick: func(cards.Card, bool) {
			rightClickCount++
		},
	}

	tile.MouseUp(&desktop.MouseEvent{
		Button:   desktop.MouseButtonPrimary,
		Modifier: fyne.KeyModifierShortcutDefault,
	})

	if toggleCount != 1 {
		t.Fatalf("toggle count = %d, want 1", toggleCount)
	}
	if rightClickCount != 0 {
		t.Fatalf("right-click count = %d, want 0", rightClickCount)
	}
}

// TestCardTileSelectionVisualStartsHidden verifies ordinary tiles have no selection border.
func TestCardTileSelectionVisualStartsHidden(t *testing.T) {
	tile := newVisualTestCardTile(t, "MISSING-SELECTION-TEST")

	if tile.selectionBorder == nil {
		t.Fatal("selection border was not created")
	}
	if tile.selectionBorder.Visible() {
		t.Fatal("selection border is visible before the tile is selected")
	}
	if tile.selectionBorder.StrokeWidth != 4 {
		t.Fatalf("selection border width = %v, want 4", tile.selectionBorder.StrokeWidth)
	}
	_, _, _, alpha := tile.selectionBorder.FillColor.RGBA()
	if alpha != 0 {
		t.Fatalf("selection border fill alpha = %d, want 0", alpha)
	}
}

// TestCardTileSetSelectedVisual verifies selection state controls border visibility.
func TestCardTileSetSelectedVisual(t *testing.T) {
	tile := newVisualTestCardTile(t, "MISSING-VISIBILITY-TEST")

	tile.SetSelectedVisual(true)
	if !tile.selectionBorder.Visible() {
		t.Fatal("selection border remains hidden after selecting the tile")
	}

	tile.SetSelectedVisual(false)
	if tile.selectionBorder.Visible() {
		t.Fatal("selection border remains visible after deselecting the tile")
	}
}

// TestCardTileSetSelectedVisualNilBorder verifies partially constructed test tiles remain safe.
func TestCardTileSetSelectedVisualNilBorder(t *testing.T) {
	tile := &CardTile{}
	tile.SetSelectedVisual(true)
}

// TestCardTileRendererStacksSelectionBorderAboveImage verifies the border is the top layer.
func TestCardTileRendererStacksSelectionBorderAboveImage(t *testing.T) {
	tile := newVisualTestCardTile(t, "MISSING-RENDERER-TEST")

	renderer := tile.CreateRenderer()
	objects := renderer.Objects()
	if len(objects) != 1 {
		t.Fatalf("renderer object count = %d, want 1 stacked container", len(objects))
	}

	stack, ok := objects[0].(*fyne.Container)
	if !ok {
		t.Fatalf("renderer object type = %T, want *fyne.Container", objects[0])
	}
	if len(stack.Objects) != 2 {
		t.Fatalf("stack layer count = %d, want 2", len(stack.Objects))
	}
	if stack.Objects[0] != tile.image {
		t.Fatal("card image is not the bottom renderer layer")
	}
	if stack.Objects[1] != tile.selectionBorder {
		t.Fatal("selection border is not the top renderer layer")
	}
}

func newVisualTestCardTile(t *testing.T, cardID string) *CardTile {
	t.Helper()

	guiApp := fynetest.NewApp()
	t.Cleanup(guiApp.Quit)

	return NewCardTileSized(
		cards.Card{ID: cardID, Name: "Selection Visual Test"},
		fyne.NewSize(48, 67),
		nil,
		nil,
	)
}
