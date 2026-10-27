package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestIsApplicationOutdated(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		{name: "same", current: "1.2.3", latest: "v1.2.3", want: false},
		{name: "newer available", current: "1.2.2", latest: "v1.2.3", want: true},
		{name: "running ahead", current: "1.2.4", latest: "v1.2.3", want: false},
		{name: "hotfix newer than stable", current: "1.2.3", latest: "v1.2.3-hotfix.1", want: true},
		{name: "empty latest", current: "1.2.3", latest: "", want: false},
	}
	for _, testCase := range tests {
		if got := isApplicationOutdated(testCase.current, testCase.latest); got != testCase.want {
			t.Errorf(
				"%s: isApplicationOutdated(%q, %q) = %v, want %v",
				testCase.name,
				testCase.current,
				testCase.latest,
				got,
				testCase.want,
			)
		}
	}
}

func TestShouldPromptForApplicationUpdateHonorsSkip(t *testing.T) {
	if !shouldPromptForApplicationUpdate("1.2.2", "v1.2.3", "", false) {
		t.Fatal("automatic check should prompt when a newer release is available")
	}
	if shouldPromptForApplicationUpdate("1.2.2", "v1.2.3", "1.2.3", false) {
		t.Fatal("automatic check should not prompt for a skipped release")
	}
	if !shouldPromptForApplicationUpdate("1.2.2", "v1.2.3", "1.2.3", true) {
		t.Fatal("manual check should still prompt even when the release was skipped")
	}
	if shouldPromptForApplicationUpdate("1.2.3", "v1.2.3", "", false) {
		t.Fatal("should not prompt when already on the latest release")
	}
	if !shouldPromptForApplicationUpdate("1.2.2", "v1.2.3", "1.2.2", false) {
		t.Fatal("skipping an older version should not suppress a newer release prompt")
	}
}

func TestBuildMainMenuDisplaysCheckForUpdates(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()

	menu := buildMainMenu(window, mainMenuActions{CheckForUpdates: func() {}})
	button := findButtonText(menu, "Check for Updates")
	if button == nil {
		t.Fatal("main menu does not display Check for Updates")
	}
	if button.Disabled() {
		t.Fatal("Check for Updates is disabled despite having an action")
	}
}

func TestNormalizeApplicationVersion(t *testing.T) {
	if got := normalizeApplicationVersion(" v1.2.3 "); got != "1.2.3" {
		t.Fatalf("normalizeApplicationVersion = %q, want 1.2.3", got)
	}
}
