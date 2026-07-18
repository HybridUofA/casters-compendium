package deckexport

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

// TestWriteDeckImageMatchesReferenceLayout verifies card order and back placement in the sheet.
func TestWriteDeckImageMatchesReferenceLayout(t *testing.T) {
	imageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(imageDirectory, "first.png"), color.RGBA{R: 255, A: 255})
	writeSolidPNG(t, filepath.Join(imageDirectory, "second.png"), color.RGBA{G: 255, A: 255})
	writeSolidPNG(t, filepath.Join(imageDirectory, deckImageBackFile), color.RGBA{B: 255, A: 255})

	deck := &decks.Deck{
		Name:      "Reference Layout",
		MainDeck:  []decks.DeckEntry{{CardID: "first", Quantity: 1}, {CardID: "second", Quantity: 1}},
		MainOrder: []string{"second", "first"},
	}

	var encoded bytes.Buffer
	if err := WriteDeckImage(&encoded, deck, imageDirectory); err != nil {
		t.Fatal(err)
	}

	exported, err := png.Decode(&encoded)
	if err != nil {
		t.Fatalf("decode exported PNG: %v", err)
	}
	if exported.Bounds() != image.Rect(0, 0, deckImageWidth, deckImageHeight) {
		t.Fatalf("exported bounds = %v", exported.Bounds())
	}

	assertPixel(t, exported, deckImageCardWidth/2, deckImageCardHeight/2, color.RGBA{G: 255, A: 255})
	assertPixel(t, exported, deckImageCardWidth+deckImageCardWidth/2, deckImageCardHeight/2, color.RGBA{R: 255, A: 255})
	assertPixel(
		t,
		exported,
		9*deckImageCardWidth+deckImageCardWidth/2,
		deckImageHeight-deckImageCardHeight/2,
		color.RGBA{B: 255, A: 255},
	)
	assertPixel(t, exported, deckImageWidth-1, deckImageHeight/2, deckImageBackground)
}

// TestWriteSideboardImageUsesSideboardFrontsAndCardBack verifies sideboard-specific export content.
func TestWriteSideboardImageUsesSideboardFrontsAndCardBack(t *testing.T) {
	imageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(imageDirectory, "side.png"), color.RGBA{R: 255, G: 255, A: 255})
	writeSolidPNG(t, filepath.Join(imageDirectory, deckImageBackFile), color.RGBA{B: 255, A: 255})

	deck := &decks.Deck{
		SideDeck:  []decks.DeckEntry{{CardID: "side", Quantity: 1}},
		SideOrder: []string{"side"},
	}

	var encoded bytes.Buffer
	if err := WriteSideboardImage(&encoded, deck, imageDirectory); err != nil {
		t.Fatal(err)
	}

	exported, err := png.Decode(&encoded)
	if err != nil {
		t.Fatalf("decode exported sideboard PNG: %v", err)
	}
	assertPixel(
		t,
		exported,
		deckImageCardWidth/2,
		deckImageCardHeight/2,
		color.RGBA{R: 255, G: 255, A: 255},
	)
	assertPixel(
		t,
		exported,
		9*deckImageCardWidth+deckImageCardWidth/2,
		deckImageHeight-deckImageCardHeight/2,
		color.RGBA{B: 255, A: 255},
	)
}

// TestWriteDeckImageReportsMissingArtwork verifies missing fronts produce actionable errors.
func TestWriteDeckImageReportsMissingArtwork(t *testing.T) {
	imageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(imageDirectory, deckImageBackFile), color.Black)
	deck := &decks.Deck{
		MainDeck: []decks.DeckEntry{{CardID: "missing", Quantity: 1}},
	}

	err := WriteDeckImage(&bytes.Buffer{}, deck, imageDirectory)
	if err == nil || !strings.Contains(err.Error(), `card "missing"`) {
		t.Fatalf("WriteDeckImage() error = %v", err)
	}
}

// TestWriteDeckImageRequiresCardBack verifies every Tabletop Simulator sheet includes the back asset.
func TestWriteDeckImageRequiresCardBack(t *testing.T) {
	err := WriteDeckImage(&bytes.Buffer{}, &decks.Deck{}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), deckImageBackFile) {
		t.Fatalf("WriteDeckImage() error = %v", err)
	}
}

// writeSolidPNG creates deterministic test artwork at path.
func writeSolidPNG(t *testing.T, path string, fill color.Color) {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(fill), image.Point{}, draw.Src)

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, canvas); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// assertPixel compares one rendered pixel with an expected RGBA color.
func assertPixel(t *testing.T, source image.Image, x int, y int, want color.RGBA) {
	t.Helper()
	got := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
	if got != want {
		t.Fatalf("pixel (%d, %d) = %#v, want %#v", x, y, got, want)
	}
}
