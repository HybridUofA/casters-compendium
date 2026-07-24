package deckbuilder

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/deckio"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type mainMenuActions struct {
	NewDeck          func()
	LoadDeck         func()
	GenerateImage    func()
	GenerateDecklist func()
	UpdateDatabase   func()
	HowToUse         func()
	Diagnostics      func()
	Settings         func()
}

// buildMainMenu constructs the application's primary workflow chooser.
func buildMainMenu(window fyne.Window, actions mainMenuActions) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		applicationName,
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	description := widget.NewLabel("Build a deck or convert an existing deck file.")
	description.Alignment = fyne.TextAlignCenter

	menu := container.NewVBox(
		title,
		description,
		widget.NewSeparator(),
		widget.NewButton("Make a New Deck", actions.NewDeck),
		widget.NewButton("Load a Deck", actions.LoadDeck),
		widget.NewButton("Generate Deck Image from Decklist", actions.GenerateImage),
		widget.NewButton("Generate Decklist File", actions.GenerateDecklist),
		widget.NewButton("Update Card Database", actions.UpdateDatabase),
		widget.NewButton("How to Use", actions.HowToUse),
		widget.NewButton("Diagnostic Information", actions.Diagnostics),
		widget.NewButton("Settings", actions.Settings),
		widget.NewSeparator(),
		widget.NewButton("Quit", func() { window.Close() }),
	)
	return container.NewCenter(menu)
}

// showNewDeckDialog collects a deck name and returns a newly validated deck.
func showNewDeckDialog(
	window fyne.Window,
	onCreated func(*decks.Deck),
) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText("New Deck")
	dialog.ShowForm(
		"Make a New Deck",
		"Create",
		"Cancel",
		[]*widget.FormItem{widget.NewFormItem("Deck Name", nameEntry)},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			deck, err := decks.NewDeck(nameEntry.Text)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			onCreated(deck)
		},
		window,
	)
}

// showOpenDeckDialog loads either JSON or text deck formats through a file picker.
func showOpenDeckDialog(
	window fyne.Window,
	repository *cards.Repository,
	onOpened func(*decks.Deck, fyne.URI),
) {
	fileDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, openErr error) {
			if openErr != nil {
				dialog.ShowError(openErr, window)
				return
			}
			if reader == nil {
				return
			}
			uri := reader.URI()
			var deck *decks.Deck
			var err error
			if strings.EqualFold(uri.Extension(), ".txt") {
				deck, err = deckio.ReadDeckList(reader, repository)
			} else {
				deck, err = deckio.ReadDeck(reader)
			}
			closeErr := reader.Close()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
				return
			}
			deck.EnsureOrder()
			onOpened(deck, uri)
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json", ".txt"}))
	fileDialog.Show()
}

// showSaveDeckDialog selects a destination and writes the deck's editable JSON form.
func showSaveDeckDialog(
	window fyne.Window,
	deck *decks.Deck,
	onSaved func(fyne.URI),
) {
	fileDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, saveErr error) {
			if saveErr != nil {
				dialog.ShowError(saveErr, window)
				return
			}
			if writer == nil {
				return
			}
			uri := writer.URI()
			writeErr := deckio.WriteDeck(writer, deck)
			closeErr := writer.Close()
			if writeErr != nil {
				dialog.ShowError(writeErr, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
				return
			}
			onSaved(uri)
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileDialog.SetFileName(safeDeckFileName(deck.Name) + ".json")
	fileDialog.Show()
}

// saveDeckToURI overwrites a previously selected deck destination.
func saveDeckToURI(window fyne.Window, uri fyne.URI, deck *decks.Deck) {
	writer, err := storage.Writer(uri)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}
	writeErr := deckio.WriteDeck(writer, deck)
	closeErr := writer.Close()
	if writeErr != nil {
		dialog.ShowError(writeErr, window)
		return
	}
	if closeErr != nil {
		dialog.ShowError(closeErr, window)
	}
}

// showGenerateImageFromDecklistDialog loads a text decklist and starts a zone-image export.
func showGenerateImageFromDecklistDialog(
	window fyne.Window,
	repository *cards.Repository,
) {
	fileDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, openErr error) {
			if openErr != nil {
				dialog.ShowError(openErr, window)
				return
			}
			if reader == nil {
				return
			}
			deck, err := deckio.ReadDeckList(reader, repository)
			closeErr := reader.Close()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
				return
			}

			choice := widget.NewRadioGroup([]string{"Main Deck", "Sideboard"}, nil)
			choice.SetSelected("Main Deck")
			dialog.NewCustomConfirm(
				"Choose Export",
				"Export",
				"Cancel",
				choice,
				func(confirmed bool) {
					if confirmed {
						showDeckImageExportDialog(
							window,
							deck,
							choice.Selected == "Sideboard",
						)
					}
				},
				window,
			).Show()
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".txt"}))
	fileDialog.Show()
}

// showGenerateDecklistDialog loads an editable JSON deck and offers a text export destination.
func showGenerateDecklistDialog(
	window fyne.Window,
	repository *cards.Repository,
) {
	fileDialog := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, openErr error) {
			if openErr != nil {
				dialog.ShowError(openErr, window)
				return
			}
			if reader == nil {
				return
			}
			deck, err := deckio.ReadDeck(reader)
			closeErr := reader.Close()
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
				return
			}
			showDecklistSaveDialog(window, deck, repository)
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileDialog.Show()
}

// showDecklistSaveDialog writes a deck in the human-readable interchange format.
func showDecklistSaveDialog(
	window fyne.Window,
	deck *decks.Deck,
	repository *cards.Repository,
) {
	fileDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, saveErr error) {
			if saveErr != nil {
				dialog.ShowError(saveErr, window)
				return
			}
			if writer == nil {
				return
			}
			writeErr := deckio.WriteDeckList(writer, deck, repository)
			closeErr := writer.Close()
			if writeErr != nil {
				dialog.ShowError(writeErr, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
			}
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".txt"}))
	fileDialog.SetFileName(safeDeckFileName(deck.Name) + ".txt")
	fileDialog.Show()
}

// safeDeckFileName replaces common path separators while preserving a readable deck name.
func safeDeckFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Deck"
	}
	return strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
	).Replace(name)
}
