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
