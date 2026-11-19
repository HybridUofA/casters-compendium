package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	releasenotes "github.com/HybridUofA/casters-compendium/docs/releases"
)

func TestBuildMainMenuDisplaysEnabledSimulatorPrototype(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{PlayGame: func() {}})
	button := findButtonText(menu, "Play a Game (Hotseat)")
	if button == nil {
		t.Fatal("main menu does not display the simulator hotseat action")
	}
	if button.Disabled() {
		t.Fatal("simulator prototype is disabled despite having an action")
	}
}

func TestBuildMainMenuDisplaysSoloVsTsubasa(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{PlaySoloVsTsubasa: func() {}})
	button := findButtonText(menu, "Vs Tsubasa")
	if button == nil {
		t.Fatal("main menu does not display Vs Tsubasa")
	}
	if button.Disabled() {
		t.Fatal("Vs Tsubasa is disabled despite having an action")
	}
}

func TestBuildMainMenuDisplaysOnlineAndDeckButtons(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{})
	for _, label := range []string{"Online", "Decks"} {
		if findButtonText(menu, label) == nil {
			t.Fatalf("main menu does not display the %q button", label)
		}
	}
}

func TestBuildMainMenuDisplaysUtilityButtons(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{})
	for _, label := range []string{"Check for Updates", "How to Use", "Settings", "Quit"} {
		if findButtonText(menu, label) == nil {
			t.Fatalf("main menu does not display the %q button", label)
		}
	}
}

func TestOnlineMenuListsOnlinePlayActions(t *testing.T) {
	var lobby, host, join bool
	menu := newOnlineMenu(mainMenuActions{
		OnlineLobby:    func() { lobby = true },
		HostOnlineGame: func() { host = true },
		JoinOnlineGame: func() { join = true },
	})

	for _, label := range []string{"Online Lobby", "Host Online Game", "Join Online Game"} {
		item := findMenuItem(menu, label)
		if item == nil {
			t.Fatalf("online menu does not list %q", label)
		}
		item.Action()
	}
	if !lobby || !host || !join {
		t.Fatal("online menu items did not run their actions")
	}
}

func TestDecksButtonOpensDeckEditor(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	opened := false
	menu := buildMainMenu(window, mainMenuActions{OpenDeckEditor: func() { opened = true }})
	button := findButtonText(menu, "Decks")
	if button == nil {
		t.Fatal("main menu does not display the Decks button")
	}
	if button.Disabled() {
		t.Fatal("Decks button is disabled despite having an action")
	}

	test.Tap(button)
	if !opened {
		t.Fatal("Decks button did not open the deck editor")
	}
}

func TestSidebarShowsChangelogAndDiagnostics(t *testing.T) {
	notes, err := releasenotes.All()
	if err != nil || len(notes) == 0 {
		t.Fatalf("bundled release notes unavailable: %v", err)
	}

	window := test.NewWindow(nil)
	defer window.Close()
	window.Resize(fyne.NewSize(900, 700))

	menu := buildMainMenu(window, mainMenuActions{DiagnosticsText: "diagnostic sample"})

	// The sidebar only exists once the burger (the only button without a label) is tapped.
	burger := findButtonText(menu, "")
	if burger == nil {
		t.Fatal("main menu does not display the sidebar button")
	}

	burger.OnTapped()

	top := window.Canvas().Overlays().Top()
	if top == nil {
		t.Fatal("tapping the sidebar button did not open the sidebar")
	}

	// The changelog's version picker starts on the newest release.
	if findButtonDeep(top, notes[0].Version) == nil {
		t.Fatalf("sidebar does not display the changelog for %q", notes[0].Version)
	}
	if findButtonDeep(top, "Copy to Clipboard") == nil {
		t.Fatal("sidebar does not offer Copy to Clipboard")
	}
}

func findMenuItem(menu *fyne.Menu, label string) *fyne.MenuItem {
	for _, item := range menu.Items {
		if item.Label == label {
			return item
		}
	}
	return nil
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

// findButtonDeep searches containers and widget internals for a button with the given text.
func findButtonDeep(object fyne.CanvasObject, text string) *widget.Button {
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return button
	}
	var children []fyne.CanvasObject
	switch typed := object.(type) {
	case *fyne.Container:
		children = typed.Objects
	case fyne.Widget:
		children = test.WidgetRenderer(typed).Objects()
	}
	for _, child := range children {
		if button := findButtonDeep(child, text); button != nil {
			return button
		}
	}
	return nil
}
