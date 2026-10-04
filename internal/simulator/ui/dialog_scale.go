package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// resizeDialogFraction sizes a dialog relative to its parent window so popups
// stay readable on large boards instead of staying at tiny fixed pixels.
func resizeDialogFraction(
	popup dialog.Dialog,
	window fyne.Window,
	widthFraction, heightFraction float32,
	minWidth, minHeight, maxWidth, maxHeight float32,
) {
	if popup == nil || window == nil {
		return
	}
	size := window.Canvas().Size()
	if size.Width < 1 || size.Height < 1 {
		size = fyne.NewSize(minWidth, minHeight)
	}
	width := size.Width * widthFraction
	height := size.Height * heightFraction
	if width < minWidth {
		width = minWidth
	}
	if height < minHeight {
		height = minHeight
	}
	if maxWidth > 0 && width > maxWidth {
		width = maxWidth
	}
	if maxHeight > 0 && height > maxHeight {
		height = maxHeight
	}
	popup.Resize(fyne.NewSize(width, height))
}

func showScaledCustom(
	title, dismiss string,
	content fyne.CanvasObject,
	window fyne.Window,
	widthFraction, heightFraction float32,
	minWidth, minHeight, maxWidth, maxHeight float32,
) dialog.Dialog {
	popup := dialog.NewCustom(title, dismiss, content, window)
	resizeDialogFraction(popup, window, widthFraction, heightFraction, minWidth, minHeight, maxWidth, maxHeight)
	popup.Show()
	return popup
}

func showScaledForm(
	title, confirm, dismiss string,
	items []*widget.FormItem,
	callback func(bool),
	window fyne.Window,
	widthFraction, heightFraction float32,
	minWidth, minHeight, maxWidth, maxHeight float32,
) {
	popup := dialog.NewForm(title, confirm, dismiss, items, callback, window)
	resizeDialogFraction(popup, window, widthFraction, heightFraction, minWidth, minHeight, maxWidth, maxHeight)
	popup.Show()
}
