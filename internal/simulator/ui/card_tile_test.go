package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
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

func TestOrientedCardTileStartsSidewaysWithoutFollowUpMutation(t *testing.T) {
	tile := newOrientedCardTile(
		simulatorview.CardView{},
		cards.Card{},
		fyne.NewSize(86, 120),
		nil,
		nil,
		true,
	)

	if tile.MinSize() != fyne.NewSize(120, 86) {
		t.Fatalf("initial sideways tile size = %v; want 120x86", tile.MinSize())
	}
	if tile.image == tile.uprightImage {
		t.Fatal("initial sideways tile retained upright artwork")
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

func TestSetSidewaysRotatesVisibleCardArtwork(t *testing.T) {
	tile := NewCardTile(
		simulatorview.CardView{CardID: "visible", ShowFace: true, Face: model.CardFaceUp},
		cards.Card{ID: "visible"},
		fyne.NewSize(86, 120),
		nil,
		nil,
	)
	source := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	source.Set(0, 0, color.NRGBA{R: 255, A: 255})
	tile.image = canvas.NewImageFromImage(source)
	tile.image.FillMode = canvas.ImageFillContain
	tile.uprightImage = tile.image

	tile.SetSideways(true)

	if tile.image == tile.uprightImage {
		t.Fatal("visible Rested card retained its upright artwork")
	}
	if tile.image.Image == nil {
		t.Fatal("visible Rested card has no rotated bitmap")
	}
	bounds := tile.image.Image.Bounds()
	if bounds.Dx() != 3 || bounds.Dy() != 2 {
		t.Fatalf("rotated visible artwork dimensions = %dx%d; want 3x2", bounds.Dx(), bounds.Dy())
	}
	if tile.MinSize() != fyne.NewSize(120, 86) {
		t.Fatalf("Rested visible tile size = %v; want 120x86", tile.MinSize())
	}
}
