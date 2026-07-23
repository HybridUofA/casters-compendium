package deckexport

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteTTSFaceSheetRendersCompactRow verifies small unique sheets use only
// the required columns and preserve face order without requiring a back image.
func TestWriteTTSFaceSheetRendersCompactRow(t *testing.T) {
	imageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(imageDirectory, "red.png"), color.RGBA{R: 255, A: 255})
	writeSolidPNG(t, filepath.Join(imageDirectory, "green.png"), color.RGBA{G: 255, A: 255})
	writeSolidPNG(t, filepath.Join(imageDirectory, "blue.png"), color.RGBA{B: 255, A: 255})

	var encoded bytes.Buffer
	err := writeTTSFaceSheet(
		&encoded,
		[]string{"red", "green", "blue"},
		imageDirectory,
	)
	if err != nil {
		t.Fatal(err)
	}

	exported, err := png.Decode(&encoded)
	if err != nil {
		t.Fatalf("decode TTS face sheet: %v", err)
	}
	wantBounds := image.Rect(
		0,
		0,
		3*deckImageCardWidth,
		deckImageCardHeight,
	)
	if exported.Bounds() != wantBounds {
		t.Fatalf("bounds = %v, want %v", exported.Bounds(), wantBounds)
	}
	assertPixel(
		t,
		exported,
		deckImageCardWidth/2,
		deckImageCardHeight/2,
		color.RGBA{R: 255, A: 255},
	)
	assertPixel(
		t,
		exported,
		deckImageCardWidth+deckImageCardWidth/2,
		deckImageCardHeight/2,
		color.RGBA{G: 255, A: 255},
	)
	assertPixel(
		t,
		exported,
		2*deckImageCardWidth+deckImageCardWidth/2,
		deckImageCardHeight/2,
		color.RGBA{B: 255, A: 255},
	)
}

// TestWriteTTSFaceSheetWrapsRows verifies the eleventh face begins row two and
// unused cells retain the configured opaque background.
func TestWriteTTSFaceSheetWrapsRows(t *testing.T) {
	imageDirectory := t.TempDir()
	sheetIDs := make([]string, 11)
	for index := range sheetIDs {
		sheetIDs[index] = "face-" + string(rune('a'+index))
		fill := color.RGBA{R: uint8(index + 1), A: 255}
		if index == 10 {
			fill = color.RGBA{B: 255, A: 255}
		}
		writeSolidPNG(
			t,
			filepath.Join(imageDirectory, sheetIDs[index]+".png"),
			fill,
		)
	}

	var encoded bytes.Buffer
	if err := writeTTSFaceSheet(&encoded, sheetIDs, imageDirectory); err != nil {
		t.Fatal(err)
	}
	exported, err := png.Decode(&encoded)
	if err != nil {
		t.Fatalf("decode TTS face sheet: %v", err)
	}
	wantBounds := image.Rect(
		0,
		0,
		ttsSheetColumns*deckImageCardWidth,
		2*deckImageCardHeight,
	)
	if exported.Bounds() != wantBounds {
		t.Fatalf("bounds = %v, want %v", exported.Bounds(), wantBounds)
	}
	assertPixel(
		t,
		exported,
		deckImageCardWidth/2,
		deckImageCardHeight+deckImageCardHeight/2,
		color.RGBA{B: 255, A: 255},
	)
	assertPixel(
		t,
		exported,
		deckImageCardWidth+deckImageCardWidth/2,
		deckImageCardHeight+deckImageCardHeight/2,
		deckImageBackground,
	)
}

// TestWriteTTSFaceSheetRejectsInvalidInput verifies a destination and at least
// one face are required before image processing begins.
func TestWriteTTSFaceSheetRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		writer    io.Writer
		sheetIDs  []string
		wantError string
	}{
		{name: "nil writer", writer: nil, sheetIDs: []string{"face"}, wantError: "writer cannot be nil"},
		{name: "empty sheet", writer: &bytes.Buffer{}, sheetIDs: nil, wantError: "cannot be empty"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := writeTTSFaceSheet(test.writer, test.sheetIDs, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("writeTTSFaceSheet() error = %v", err)
			}
		})
	}
}

// TestWriteTTSFaceSheetRejectsExcessFaces verifies grid-capacity errors are
// returned before attempting to load card artwork.
func TestWriteTTSFaceSheetRejectsExcessFaces(t *testing.T) {
	err := writeTTSFaceSheet(
		&bytes.Buffer{},
		uniqueTestCardIDs(ttsSheetMaxCards+1),
		t.TempDir(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "calculate sheet dimensions") ||
		!strings.Contains(err.Error(), "at most") {
		t.Fatalf("writeTTSFaceSheet() error = %v", err)
	}
}

// TestWriteTTSFaceSheetReportsMissingArtwork verifies failures identify both
// the one-based sheet position and internal card ID.
func TestWriteTTSFaceSheetReportsMissingArtwork(t *testing.T) {
	imageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(imageDirectory, "present.png"), color.Black)

	err := writeTTSFaceSheet(
		&bytes.Buffer{},
		[]string{"present", "missing"},
		imageDirectory,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "face 2") ||
		!strings.Contains(err.Error(), `card "missing"`) {
		t.Fatalf("writeTTSFaceSheet() error = %v", err)
	}
}

// TestWriteTTSFaceSheetReportsWriterFailure verifies PNG encoding failures
// retain useful export context and the underlying writer error.
func TestWriteTTSFaceSheetReportsWriterFailure(t *testing.T) {
	imageDirectory := t.TempDir()
	writeSolidPNG(t, filepath.Join(imageDirectory, "face.png"), color.Black)

	err := writeTTSFaceSheet(
		failingImageWriter{},
		[]string{"face"},
		imageDirectory,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "encode TTS face sheet") ||
		!strings.Contains(err.Error(), "forced writer failure") {
		t.Fatalf("writeTTSFaceSheet() error = %v", err)
	}
}

type failingImageWriter struct{}

func (failingImageWriter) Write([]byte) (int, error) {
	return 0, errors.New("forced writer failure")
}
