package ui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestSelectionFollowsContainedArtworkAfterRotation(t *testing.T) {
	tile := NewCardTile(simulatorview.CardView{ShowFace: true}, cards.Card{}, fyne.NewSize(60, 90), nil, nil)
	tile.uprightImage = newSimulatorCanvasImage(image.NewNRGBA(image.Rect(0, 0, 60, 90)))
	tile.image = tile.uprightImage
	tile.Resize(fyne.NewSize(120, 120))
	tile.SetSelected(true)
	renderer := tile.CreateRenderer()
	for _, sideways := range []bool{false, true} {
		tile.SetSideways(sideways)
		renderer.Refresh()
		border := tile.selectionBorder
		wantPosition, wantSize := fyne.NewPos(22, 2), fyne.NewSize(76, 116)
		if sideways {
			wantPosition, wantSize = fyne.NewPos(2, 22), fyne.NewSize(116, 76)
		}
		if math.Abs(float64(border.Position().X-wantPosition.X)) > 0.01 ||
			math.Abs(float64(border.Position().Y-wantPosition.Y)) > 0.01 || border.Size() != wantSize {
			t.Fatalf("sideways=%v: outline at %v size %v; want %v size %v", sideways, border.Position(), border.Size(), wantPosition, wantSize)
		}
		if border.CornerRadius <= 0 || !border.Visible() {
			t.Fatal("selected card needs a visible rounded outline")
		}
		if renderer.Objects()[0] != tile.image {
			t.Fatal("renderer retained artwork from the previous orientation")
		}
	}
}

func TestReversedCardDisplaysUpsideDownWithoutChangingSize(t *testing.T) {
	tile := NewCardTile(simulatorview.CardView{Orientation: model.OrientationReversed}, cards.Card{}, fyne.NewSize(60, 90), nil, nil)
	if tile.image == tile.uprightImage {
		t.Fatal("Reversed card was constructed with upright artwork")
	}
	source := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	marker := color.NRGBA{R: 255, A: 255}
	source.SetNRGBA(0, 0, marker)
	tile.uprightImage = newSimulatorCanvasImage(source)
	tile.reversedImage = nil
	tile.setSidewaysState(false)
	bitmap := tile.image.Image
	if bitmap.Bounds() != source.Bounds() || color.NRGBAModel.Convert(bitmap.At(1, 2)) != marker {
		t.Fatal("Reversed artwork must rotate 180 degrees")
	}
	if tile.MinSize() != fyne.NewSize(60, 90) {
		t.Fatalf("Reversed card size = %v; want portrait 60x90", tile.MinSize())
	}
}

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
