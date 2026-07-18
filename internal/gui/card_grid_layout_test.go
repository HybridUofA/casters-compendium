package deckgui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// TestCardGridLayoutFitsAllRowsWithinAvailableHeight verifies a full main deck is never clipped.
func TestCardGridLayoutFitsAllRowsWithinAvailableHeight(t *testing.T) {
	const heightToWidth float32 = 1.4
	layout := &CardGridLayout{
		Columns:       10,
		HeightToWidth: heightToWidth,
		Padding:       6,
	}
	objects := make([]fyne.CanvasObject, 50)
	for index := range objects {
		objects[index] = canvas.NewRectangle(nil)
	}
	available := fyne.NewSize(600, 300)
	layout.Layout(objects, available)

	last := objects[len(objects)-1]
	if bottom := last.Position().Y + last.Size().Height; bottom > available.Height+0.01 {
		t.Fatalf("last row bottom = %f, available height = %f", bottom, available.Height)
	}
	if ratio := objects[0].Size().Height / objects[0].Size().Width; ratio != heightToWidth {
		t.Fatalf("card ratio = %f, want %f", ratio, heightToWidth)
	}
}

// TestCardGridLayoutLeftAlignsCards verifies spare horizontal space stays after the cards.
func TestCardGridLayoutLeftAlignsCards(t *testing.T) {
	layout := &CardGridLayout{
		Columns:       10,
		HeightToWidth: 1.4,
		Padding:       6,
	}
	objects := make([]fyne.CanvasObject, 3)
	for index := range objects {
		objects[index] = canvas.NewRectangle(nil)
	}

	layout.Layout(objects, fyne.NewSize(600, 100))

	if position := objects[0].Position(); position.X != 0 || position.Y != 0 {
		t.Fatalf("first card position = %v, want (0, 0)", position)
	}
	wantSecondX := objects[0].Size().Width + layout.Padding
	if secondX := objects[1].Position().X; secondX != wantSecondX {
		t.Fatalf("second card x = %f, want %f", secondX, wantSecondX)
	}
}
