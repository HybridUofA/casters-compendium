package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestBuildMainMenuDisplaysApplicationVersion(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{})
	want := "v" + applicationVersion
	if !containsLabelText(menu, want) {
		t.Fatalf("main menu does not display application version %q", want)
	}
}

func containsLabelText(object fyne.CanvasObject, text string) bool {
	if label, ok := object.(*widget.Label); ok && label.Text == text {
		return true
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if containsLabelText(child, text) {
				return true
			}
		}
	}
	return false
}
