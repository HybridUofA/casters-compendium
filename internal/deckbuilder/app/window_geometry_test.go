package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestClampSizeToBounds(t *testing.T) {
	desired := fyne.NewSize(1400, 850)
	if got := clampSizeToBounds(desired, fyne.Size{}); got != desired {
		t.Fatalf("empty max should keep desired, got %#v", got)
	}
	got := clampSizeToBounds(desired, fyne.NewSize(1200, 700))
	if got != (fyne.NewSize(1200, 700)) {
		t.Fatalf("clampSizeToBounds = %#v, want 1200x700", got)
	}
	got = clampSizeToBounds(desired, fyne.NewSize(1600, 900))
	if got != desired {
		t.Fatalf("larger max should keep desired, got %#v", got)
	}
}

func TestPreferredWindowSizeNeverExceedsDefaults(t *testing.T) {
	got := preferredWindowSize()
	if got.Width > defaultWindowWidth || got.Height > defaultWindowHeight {
		t.Fatalf("preferredWindowSize = %#v, exceeds defaults", got)
	}
	if got.Width < 1 || got.Height < 1 {
		t.Fatalf("preferredWindowSize = %#v, want positive size", got)
	}
}
