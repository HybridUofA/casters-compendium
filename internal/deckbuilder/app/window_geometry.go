package deckbuilder

import (
	"time"

	"fyne.io/fyne/v2"
)

const (
	defaultWindowWidth  float32 = 1400
	defaultWindowHeight float32 = 850
	// Leave room for the title bar and borders when mapping work-area pixels.
	windowChromeWidthPx  float32 = 16
	windowChromeHeightPx float32 = 48
)

// preferredWindowSize returns the startup window size, capped to the visible
// desktop work area when the platform can report one.
func preferredWindowSize() fyne.Size {
	desired := fyne.NewSize(defaultWindowWidth, defaultWindowHeight)
	return clampSizeToBounds(desired, desktopWorkAreaDIP())
}

// clampSizeToBounds limits desired to max when max is usable.
func clampSizeToBounds(desired, max fyne.Size) fyne.Size {
	if max.Width <= 0 || max.Height <= 0 {
		return desired
	}
	width := desired.Width
	height := desired.Height
	if width > max.Width {
		width = max.Width
	}
	if height > max.Height {
		height = max.Height
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return fyne.NewSize(width, height)
}

// scheduleWorkAreaClamp pulls the window back above the taskbar/dock after
// Fyne applies content minimum sizes that can overshoot the visible desktop.
func scheduleWorkAreaClamp(window fyne.Window) {
	if window == nil {
		return
	}
	go func() {
		time.Sleep(75 * time.Millisecond)
		fyne.Do(func() {
			clampWindowToWorkArea(window)
		})
	}()
}

// clampWindowToWorkArea is implemented per-OS. Non-Windows builds are a no-op.
func clampWindowToWorkArea(window fyne.Window) {
	clampWindowToWorkAreaPlatform(window)
}

// desktopWorkAreaDIP is implemented per-OS. Non-Windows builds return a zero size.
func desktopWorkAreaDIP() fyne.Size {
	return desktopWorkAreaDIPPlatform()
}
