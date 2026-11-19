package deckbuilder

import (
	//	"path/filepath"
	//	"strings"

	"image/color"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	assets "github.com/HybridUofA/casters-compendium/data"
	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/deckio"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type mainMenuActions struct {
	PlayGame          func()
	PlaySoloVsTsubasa func()
	OnlineLobby       func()
	HostOnlineGame    func()
	JoinOnlineGame    func()
	OpenDeckEditor    func()
	NewDeck           func()
	LoadDeck          func()
	GenerateImage     func()
	GenerateDecklist  func()
	UpdateDatabase    func()
	CheckForUpdates   func()
	Changelog         func()
	HowToUse          func()
	Diagnostics       func()
	Settings          func()
	DiagnosticsText   string
}

type percentSidePadLayout struct {
	percent float32 // 0.10 means 10% on each side
}

func (l percentSidePadLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pad := size.Width * l.percent
	inner := fyne.NewSize(size.Width-2*pad, size.Height)
	for _, obj := range objects {
		obj.Move(fyne.NewPos(pad, 0))
		obj.Resize(inner)
	}
}

func (l percentSidePadLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var w, h float32
	for _, obj := range objects {
		m := obj.MinSize()
		if m.Width > w {
			w = m.Width
		}
		if m.Height > h {
			h = m.Height
		}
	}
	return fyne.NewSize(w, h)
}

// newOnlineMenu lists the online actions shown when the Online button is pressed.
func newOnlineMenu(actions mainMenuActions) *fyne.Menu {
	return fyne.NewMenu("",
		fyne.NewMenuItem("Online Lobby", actions.OnlineLobby),
		fyne.NewMenuItem("Host Online Game", actions.HostOnlineGame),
		fyne.NewMenuItem("Join Online Game", actions.JoinOnlineGame),
	)
}

// buildMainMenu constructs the application's primary workflow chooser.
func buildMainMenu(window fyne.Window, actions mainMenuActions) fyne.CanvasObject {

	//This is the title of the application, it is bold and centered.
	/*Start Making the Main Menu Here*/
	description := widget.NewLabel("Build a deck, play a game, and summon to win!\nRemember to check for updates!")
	description.Alignment = fyne.TextAlignCenter

	playButton := widget.NewButton("Play a Game (Hotseat)", actions.PlayGame)
	if actions.PlayGame == nil {
		playButton.Disable()
	}
	soloButton := widget.NewButton("Vs Tsubasa", actions.PlaySoloVsTsubasa)
	if actions.PlaySoloVsTsubasa == nil {
		soloButton.Disable()
	}
	deckButton := widget.NewButton("Decks", actions.OpenDeckEditor)
	if actions.OpenDeckEditor == nil {
		deckButton.Disable()
	}

	onlineMenu := newOnlineMenu(actions)

	var onlineButton *widget.Button
	onlineButton = widget.NewButton("Online", func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(onlineButton)
		pos.X += onlineButton.Size().Width + 10 // open to the right of the button
		widget.ShowPopUpMenuAtPosition(onlineMenu, window.Canvas(), pos)
	})

	var sidebar *widget.PopUp

	changelog := buildChangelogPanel(fyne.NewSize(340, 120), window.Canvas())

	diagText := widget.NewLabel(actions.DiagnosticsText)
	diagText.Wrapping = fyne.TextWrapWord

	sidebar = widget.NewPopUp(container.NewBorder(
		nil,
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabelWithStyle("Diagnostic Information", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			diagText,
			widget.NewButton("Copy to Clipboard", func() {
				fyne.CurrentApp().Clipboard().SetContent(actions.DiagnosticsText)
			}),
		),
		nil,
		nil,
		changelog,
	), window.Canvas())

	var burger *widget.Button
	burger = widget.NewButtonWithIcon("", theme.MenuIcon(), func() {
		size := window.Canvas().Size()
		width := float32(380)
		sidebar.Resize(fyne.NewSize(width, size.Height))
		sidebar.ShowAtPosition(fyne.NewPos(size.Width, 0)) // start just off the right edge

		slide := fyne.NewAnimation(250*time.Millisecond, func(progress float32) {
			// progress runs from 0 to 1 over the duration
			sidebar.Move(fyne.NewPos(size.Width-width*progress, 0))
		})
		slide.Curve = fyne.AnimationEaseOut // fast start, gentle stop
		slide.Start()
	})

	menu := container.NewHBox(
		container.NewVBox(
			onlineButton,
			playButton,
			soloButton,
			deckButton,
			widget.NewButton("Check for Updates", actions.CheckForUpdates),
			widget.NewButton("How to Use", actions.HowToUse),
			widget.NewButton("Settings", actions.Settings),
			widget.NewButton("Quit", func() { window.Close() }),
		),
	)

	background := fyne.CurrentApp().Preferences().StringWithFallback(
		backgroundPreferenceKey,
		backgroundNone,
	)

	title := canvas.NewImageFromResource(fyne.NewStaticResource("title.png", assets.TitlePNG))
	title.FillMode = canvas.ImageFillContain
	title.SetMinSize(fyne.NewSize(360, 280))

	header := container.NewHBox(layout.NewSpacer(), container.NewVBox(burger))

	centeredMenu := container.NewVBox(
		container.NewHBox(
			layout.NewSpacer(),
			container.NewVBox(
				container.NewCenter(title),
				container.NewCenter(menu),
			),
			layout.NewSpacer(),
		),
	)

	row := container.NewStack(centeredMenu, header)

	if background == backgroundNone {
		return container.NewStack(canvas.NewRectangle(color.Black), row)
	}
	return row
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
	onOpened func(*decks.Deck, fyne.URI, bool),
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
			migrated, err := deck.CanonicalizeCardIDs(repository)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			onOpened(deck, uri, migrated)
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json", ".txt"}))
	fileDialog.Show()
}

// deckNameFromSaveURI derives the deck identity from a Save As JSON filename.
func deckNameFromSaveURI(uri fyne.URI) string {
	if uri == nil {
		return ""
	}
	base := filepath.Base(uri.Name())
	if base == "" || base == "." {
		base = filepath.Base(uri.Path())
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return strings.TrimSpace(base)
}

// showSaveDeckDialog selects a destination and writes the deck's editable JSON form.
// The chosen filename stem becomes deck.Name before write so TTS Install matches Save As.
func showSaveDeckDialog(
	window fyne.Window,
	deck *decks.Deck,
	initialDirectory string,
	onSaved func(uri fyne.URI, previousName string),
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
			previousName := deck.Name
			if derived := deckNameFromSaveURI(uri); derived != "" {
				deck.Name = derived
			}
			writeErr := deckio.WriteDeck(writer, deck)
			closeErr := writer.Close()
			if writeErr != nil {
				deck.Name = previousName
				dialog.ShowError(writeErr, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
				return
			}
			if onSaved != nil {
				onSaved(uri, previousName)
			}
		},
		window,
	)
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".json"}))
	fileDialog.SetFileName(safeDeckFileName(deck.Name) + ".json")
	setFileDialogLocation(fileDialog, initialDirectory)
	fileDialog.Show()
}

// setFileDialogLocation starts a file dialog in the deck library when the
// platform's storage provider can expose that directory.
func setFileDialogLocation(fileDialog *dialog.FileDialog, directory string) {
	if strings.TrimSpace(directory) == "" {
		return
	}
	location, err := storage.ListerForURI(storage.NewFileURI(directory))
	if err == nil {
		fileDialog.SetLocation(location)
	}
}

// saveDeckToURI overwrites a previously selected deck destination.
func saveDeckToURI(window fyne.Window, uri fyne.URI, deck *decks.Deck) bool {
	writer, err := storage.Writer(uri)
	if err != nil {
		dialog.ShowError(err, window)
		return false
	}
	writeErr := deckio.WriteDeck(writer, deck)
	closeErr := writer.Close()
	if writeErr != nil {
		dialog.ShowError(writeErr, window)
		return false
	}
	if closeErr != nil {
		dialog.ShowError(closeErr, window)
		return false
	}
	return true
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
			if _, err := deck.CanonicalizeCardIDs(repository); err != nil {
				dialog.ShowError(err, window)
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
			if _, err := deck.CanonicalizeCardIDs(repository); err != nil {
				dialog.ShowError(err, window)
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
