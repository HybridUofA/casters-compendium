package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestResizeDialogFractionScalesWithWindow(t *testing.T) {
	window := test.NewWindow(container.NewVBox(widget.NewLabel("board")))
	defer window.Close()
	window.Resize(fyne.NewSize(1200, 800))

	popup := dialog.NewCustom("Scaled", "Close", widget.NewLabel("content"), window)
	resizeDialogFraction(popup, window, 0.5, 0.5, 200, 200, 900, 700)
	popup.Show()
	popup.Hide()
}
