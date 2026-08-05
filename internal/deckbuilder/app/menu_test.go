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

func TestBuildMainMenuDisplaysEnabledSimulatorPrototype(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{PlayGame: func() {}})
	button := findButtonText(menu, "Play a Game (Prototype)")
	if button == nil {
		t.Fatal("main menu does not display the simulator prototype")
	}
	if button.Disabled() {
		t.Fatal("simulator prototype is disabled despite having an action")
	}
}

func TestBuildMainMenuDisplaysDirectDeckEditorAction(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{OpenDeckEditor: func() {}})
	button := findButtonText(menu, "Open Deck Editor")
	if button == nil {
		t.Fatal("main menu does not display the direct deck-editor action")
	}
	if button.Disabled() {
		t.Fatal("direct deck-editor action is disabled")
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

func findButtonText(object fyne.CanvasObject, text string) *widget.Button {
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return button
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if button := findButtonText(child, text); button != nil {
				return button
			}
		}
	}
	return nil
}
