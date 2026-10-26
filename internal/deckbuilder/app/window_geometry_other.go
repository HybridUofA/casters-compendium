//go:build !windows

package deckbuilder

import "fyne.io/fyne/v2"

func desktopWorkAreaDIPPlatform() fyne.Size {
	return fyne.Size{}
}

func clampWindowToWorkAreaPlatform(window fyne.Window) {
}
