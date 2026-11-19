package deckbuilder

import (
	"fmt"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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

// buildChangelogPanel returns the release notes in a fixed-size, scrollable,
// outlined panel with a version selector, for embedding in the main menu.
func buildChangelogPanel(size fyne.Size, c fyne.Canvas) fyne.CanvasObject {
	notes, err := releasenotes.All()
	if err != nil || len(notes) == 0 {
		return widget.NewLabel("Release notes are unavailable.")
	}

	markdownByVersion := make(map[string]string, len(notes))
	versions := make([]string, 0, len(notes))
	for _, note := range notes {
		versions = append(versions, note.Version)
		markdownByVersion[note.Version] = note.Markdown
	}

	text := widget.NewRichTextFromMarkdown(notes[0].Markdown)
	text.Wrapping = fyne.TextWrapWord

	scroll := container.NewVScroll(text)
	scroll.SetMinSize(size)

	var picker *widget.Button
	var popup *widget.PopUp

	list := widget.NewList(
		func() int { return len(versions) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(versions[i])
		},
	)
	list.OnSelected = func(i widget.ListItemID) {
		picker.SetText(versions[i])
		text.ParseMarkdown(markdownByVersion[versions[i]])
		scroll.ScrollToTop()
		popup.Hide()
		list.UnselectAll()
	}
	popup = widget.NewPopUp(list, c)

	picker = widget.NewButtonWithIcon(versions[0], theme.MenuDropDownIcon(), func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(picker)
		pos.Y += picker.Size().Height // open just below the button
		popup.Resize(fyne.NewSize(picker.Size().Width, c.Size().Height/3))
		popup.ShowAtPosition(pos)
	})
	picker.IconPlacement = widget.ButtonIconTrailingText

	outline := canvas.NewRectangle(color.Transparent)
	outline.StrokeColor = theme.Color(theme.ColorNameForeground)
	outline.StrokeWidth = 2
	outline.CornerRadius = 6

	return container.NewStack(
		outline,
		container.NewPadded(container.NewBorder(picker, nil, nil, nil, scroll)),
	)
}
