package deckbuilder

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

// diagnosticInformation returns a deliberately small support snapshot. It
// excludes deck contents, credentials, usernames, and exact filesystem paths.
func diagnosticInformation(paths applicationPaths, repository *cards.Repository) string {
	revision, modified := buildRevision()
	cardCount := 0
	if repository != nil {
		cardCount = len(repository.All())
	}

	return fmt.Sprintf(
		`Caster's Compendium diagnostic information
Application version: %s
Source revision: %s
Source modified: %t
Go version: %s
Operating system: %s
Architecture: %s
Card records: %d
Setup complete: %t
Card database present: %t
Hosted catalog: %s
`,
		applicationVersion,
		revision,
		modified,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
		cardCount,
		fileExists(paths.SetupComplete),
		fileExists(paths.CardDatabase),
		hostedCatalogPointerURL,
	)
}

func buildRevision() (string, bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown", false
	}
	revision := "unknown"
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if strings.TrimSpace(setting.Value) != "" {
				revision = setting.Value
				if len(revision) > 12 {
					revision = revision[:12]
				}
			}
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	return revision, modified
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// showDiagnosticInformationDialog lets users review the exact text before
// copying it into a bug report. Nothing is transmitted automatically.
func showDiagnosticInformationDialog(
	window fyne.Window,
	paths applicationPaths,
	repository *cards.Repository,
) {
	information := diagnosticInformation(paths, repository)
	text := widget.NewMultiLineEntry()
	text.SetText(information)
	text.Disable()
	text.Wrapping = fyne.TextWrapWord
	text.SetMinRowsVisible(12)

	copyButton := widget.NewButton("Copy to Clipboard", func() {
		fyne.CurrentApp().Clipboard().SetContent(information)
	})
	content := container.NewBorder(
		widget.NewLabel("Review this non-sensitive support information before sharing it."),
		copyButton,
		nil,
		nil,
		text,
	)
	content.Resize(fyne.NewSize(680, 420))
	dialog.ShowCustom("Diagnostic Information", "Close", content, window)
}
