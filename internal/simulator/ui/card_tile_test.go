package ui

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestSetSidewaysRotatesConcealedCardBack(t *testing.T) {
	tile := NewCardTile(
		simulatorview.CardView{},
		cards.Card{},
		fyne.NewSize(86, 60),
		nil,
		nil,
	)

	tile.SetSideways(true)

	if tile.image.Image == nil {
		t.Fatal("sideways card back did not load its embedded image")
	}
	bounds := tile.image.Image.Bounds()
	if bounds.Dx() <= bounds.Dy() {
		t.Fatalf(
			"sideways card-back dimensions = %dx%d; want landscape image",
			bounds.Dx(),
			bounds.Dy(),
		)
	}
	if tile.MinSize().Width <= tile.MinSize().Height {
		t.Fatalf("sideways tile size = %v; want landscape dimensions", tile.MinSize())
	}
	if tile.MinSize() != fyne.NewSize(86, 60) {
		t.Fatalf("already-landscape tile size = %v; want original 86x60", tile.MinSize())
	}
}

func TestSetSidewaysPreservesPortraitCardScale(t *testing.T) {
	tile := NewCardTile(
		simulatorview.CardView{},
		cards.Card{},
		fyne.NewSize(86, 120),
		nil,
		nil,
	)

	tile.SetSideways(true)

	if tile.MinSize() != fyne.NewSize(120, 86) {
		t.Fatalf("sideways portrait tile size = %v; want 120x86", tile.MinSize())
	}
	tile.SetSideways(false)
	if tile.MinSize() != fyne.NewSize(86, 120) {
		t.Fatalf("restored tile size = %v; want original 86x120", tile.MinSize())
	}
}
