package deckbuilder

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	releasenotes "github.com/HybridUofA/casters-compendium/docs/releases"
)

// showChangelogDialog presents the canonical release notes bundled into the
// executable. It deliberately performs no network access.
func showChangelogDialog(window fyne.Window) {
	notes, err := releasenotes.All()
	if err != nil {
		dialog.ShowError(err, window)
		return
	}
	if len(notes) == 0 {
		dialog.ShowError(fmt.Errorf("no bundled release notes are available"), window)
		return
	}

	markdownByVersion := make(map[string]string, len(notes))
	versions := make([]string, 0, len(notes))
	for _, note := range notes {
		versions = append(versions, note.Version)
		markdownByVersion[note.Version] = note.Markdown
	}

	releaseText := widget.NewRichTextFromMarkdown(notes[0].Markdown)
	releaseText.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(releaseText)
	scroll.SetMinSize(fyne.NewSize(760, 540))
	selector := widget.NewSelect(versions, func(version string) {
		releaseText.ParseMarkdown(markdownByVersion[version])
		scroll.ScrollToTop()
	})
	selector.PlaceHolder = "Choose a release"
	selector.SetSelected(notes[0].Version)

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Release notes are included with the application and available offline."),
			selector,
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		scroll,
	)
	dialog.ShowCustom("Changelog", "Close", content, window)
}
