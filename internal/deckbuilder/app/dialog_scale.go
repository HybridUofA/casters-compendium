package deckbuilder

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

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

func showScaledCustomConfirm(
	title, confirm, dismiss string,
	content fyne.CanvasObject,
	callback func(bool),
	window fyne.Window,
	widthFraction, heightFraction float32,
	minWidth, minHeight, maxWidth, maxHeight float32,
) {
	popup := dialog.NewCustomConfirm(title, confirm, dismiss, content, callback, window)
	resizeDialogFraction(popup, window, widthFraction, heightFraction, minWidth, minHeight, maxWidth, maxHeight)
	popup.Show()
}
