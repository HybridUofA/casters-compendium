package deckbuilder

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	dataassets "github.com/HybridUofA/casters-compendium/data"
	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	deckexport "github.com/HybridUofA/casters-compendium/internal/deckbuilder/export"
	deckgui "github.com/HybridUofA/casters-compendium/internal/deckbuilder/ui"
	"github.com/HybridUofA/casters-compendium/internal/deckio"
	"github.com/HybridUofA/casters-compendium/internal/decklibrary"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorui "github.com/HybridUofA/casters-compendium/internal/simulator/ui"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// checkedValues returns option names whose corresponding checkboxes are selected.
func checkedValues(
	options []string,
	checks map[string]*widget.Check,
) []string {
	selected := make([]string, 0)

	for _, option := range options {
		check, exists := checks[option]
		if exists && check.Checked {
			selected = append(selected, option)
		}
	}

	return selected
}

const anyOption = "- Any -"
const ttsRootPreferenceKey = "tts.root"
const activeDeckPreferenceKey = "decklibrary.active-path"
const officialTemplatePreferencePrefix = "official:"

// defaultTTSCardBack returns a fresh reader for the bundled MTD card back.
// A new reader is required for each export because copying consumes it.
func defaultTTSCardBack() *bytes.Reader {
	return bytes.NewReader(dataassets.CardBackPNG)
}

// withAnyOption prepends the shared no-filter choice to a select-option list.
func withAnyOption(options []string) []string {
	result := make(
		[]string,
		0,
		len(options)+1,
	)
	result = append(result, anyOption)
	result = append(result, options...)

	return result
}

// optionalSelection converts one meaningful select value into a filter slice.
func optionalSelection(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == anyOption {
		return nil
	}
	return []string{value}
}

// optionalValue converts nonblank entry text into a filter slice.
func optionalValue(value string) []string {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	return []string{value}
}

// showDeckImageExportDialog saves either deck zone as a Tabletop Simulator PNG sheet.
func showDeckImageExportDialog(
	window fyne.Window,
	deck *decks.Deck,
	sideboard bool,
) {
	writeImage := deckexport.WriteDeckImage
	fileSuffix := ""
	if sideboard {
		writeImage = deckexport.WriteSideboardImage
		fileSuffix = " - Sideboard"
	}

	exportDialog := dialog.NewFileSave(
		func(writer fyne.URIWriteCloser, saveErr error) {
			if saveErr != nil {
				dialog.ShowError(saveErr, window)
				return
			}
			if writer == nil {
				return
			}

			exportErr := writeImage(
				writer,
				deck,
				cardimages.DefaultDirectory,
			)
			closeErr := writer.Close()
			if exportErr != nil {
				dialog.ShowError(exportErr, window)
				return
			}
			if closeErr != nil {
				dialog.ShowError(closeErr, window)
			}
		},
		window,
	)
	exportDialog.SetFilter(
		storage.NewExtensionFileFilter([]string{".png"}),
	)
	exportDialog.SetFileName(safeDeckFileName(deck.Name) + fileSuffix + ".png")
	exportDialog.Show()
}

// showTTSInstallDialog installs to the remembered or standard TTS directory,
// falling back to a folder picker when automatic location is unavailable.
func showTTSInstallDialog(
	window fyne.Window,
	deck *decks.Deck,
	repository *cards.Repository,
) {
	preferences := fyne.CurrentApp().Preferences()
	root, locateErr := deckexport.LocateTTSRoot(
		preferences.String(ttsRootPreferenceKey),
	)
	if locateErr == nil {
		installDeckToTTSRoot(window, deck, repository, root)
		return
	}

	folderDialog := dialog.NewFolderOpen(
		func(root fyne.ListableURI, openErr error) {
			if openErr != nil {
				dialog.ShowError(openErr, window)
				return
			}
			if root == nil {
				return
			}

			installDeckToTTSRoot(window, deck, repository, root.Path())
		},
		window,
	)
	folderDialog.Show()
}

// showTTSCardBackDialog stores an optional, sanitized custom-back URL on the
// deck. Uploads happen in the browser so Turnstile and cloud credentials never
// enter the desktop process.
func showTTSCardBackDialog(
	window fyne.Window,
	deck *decks.Deck,
	onChanged func(),
) {
	backURL := widget.NewEntry()
	backURL.SetText(deck.TTSCardBackURL)
	backURL.SetPlaceHolder("Leave blank to use the official card back")
	backURL.Validator = func(value string) error {
		return deckexport.ValidateCustomTTSCardBackURL(value)
	}
	uploadURL, err := url.Parse(deckexport.CustomAssetUploadURL)
	if err != nil {
		dialog.ShowError(err, window)
		return
	}
	uploadLink := widget.NewHyperlink("Upload and copy a custom-back URL", uploadURL)
	dialog.ShowForm(
		"Tabletop Simulator Card Back",
		"Save",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Upload", uploadLink),
			widget.NewFormItem("Custom URL", backURL),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			value := strings.TrimSpace(backURL.Text)
			if value == deck.TTSCardBackURL {
				return
			}
			deck.TTSCardBackURL = value
			if onChanged != nil {
				onChanged()
			}
		},
		window,
	)
}

func installDeckToTTSRoot(
	window fyne.Window,
	deck *decks.Deck,
	repository *cards.Repository,
	root string,
) {
	progress := dialog.NewCustomWithoutButtons(
		"Installing Tabletop Simulator Deck",
		container.NewVBox(
			widget.NewLabel("Preparing shared card assets and saved object…"),
			widget.NewProgressBarInfinite(),
		),
		window,
	)
	progress.Show()

	go func() {
		paths, hosted, fallbackReason, installErr := installPreferredTTSDeck(
			context.Background(),
			root,
			deck,
			repository,
		)
		fyne.Do(func() {
			progress.Hide()
			if installErr != nil {
				dialog.ShowError(installErr, window)
				return
			}
			fyne.CurrentApp().Preferences().SetString(ttsRootPreferenceKey, paths.Root)
			if hosted {
				dialog.ShowInformation(
					"Tabletop Simulator Export Complete",
					fmt.Sprintf(
						"%q is ready in Tabletop Simulator with shared online card assets for multiplayer.\n\nSaved object:\n%s",
						deck.Name,
						paths.JSONPath,
					),
					window,
				)
				return
			}
			dialog.ShowInformation(
				"Local Tabletop Simulator Export Complete",
				fmt.Sprintf(
					"%q was installed using local image files because the shared catalog was unavailable.\n\nOther players may need those image files manually.\n\nHosted catalog error: %v\n\nSaved object:\n%s",
					deck.Name,
					fallbackReason,
					paths.JSONPath,
				),
				window,
			)
		})
	}()
}

// applyCardDrop applies a completed drag operation to the active deck.
func applyCardDrop(
	deck *decks.Deck,
	repository decks.CardCatalog,
	source deckgui.CardDragSource,
	target *deckgui.CardDropTarget,
) error {
	if target == nil {
		return nil
	}

	if target.Remove {
		if source.Kind != deckgui.DragFromDeck {
			return nil
		}
		return deck.RemoveCardAt(source.Zone, source.Index)
	}

	switch source.Kind {
	case deckgui.DragFromSearch:
		_, err := deck.AddCardCheckedAt(
			target.Zone,
			source.Card,
			1,
			repository,
			target.Index,
		)
		return err
	case deckgui.DragFromDeck:
		moveIndices := source.Indices
		if len(moveIndices) == 0 {
			moveIndices = []int{source.Index}
		}
		_, err := deck.MoveOrderedCards(
			source.Zone,
			moveIndices,
			target.Zone,
			target.Index,
		)
		return err
	default:
		return fmt.Errorf("unknown card drag source %d", source.Kind)
	}
}

// showApplication builds the main menu and deck editor around the active card repository.
func showApplication(
	window fyne.Window,
	paths applicationPaths,
	repository *cards.Repository,
) {

	const previewWidth float32 = 160
	const previewHeight float32 = 224
	const informationPanelWidth float32 = 260
	const cardPreviewHeight float32 = 300
	const sideDeckPanelHeight float32 = 155
	const searchPanelWidth float32 = 370
	const searchControlsHeight float32 = 370
	mainDeckTileMinSize := fyne.NewSize(48, 67)
	sideDeckTileMinSize := fyne.NewSize(32, 45)

	dragLayer := container.NewWithoutLayout()

	deck, err := decks.NewDeck("New Deck")
	if err != nil {
		dialog.ShowError(err, window)
		return
	}

	var selection SelectedState

	var currentDeckURI fyne.URI
	var currentDeckPath string
	var currentTemplateID string
	deckDirty := false
	var showMainMenu func()
	var openDeckEditor func()
	var makeNewDeck func()
	var loadDeck func()
	var saveDeck func()
	var saveDeckAs func()
	var refreshDeckDisplay func()
	var refreshDeckLibrary func()
	var loadLibraryDeck func(string)
	var loadOfficialTemplate func(string)

	deckLibraryDirectory, libraryPathErr := decklibrary.DefaultDirectory()
	if libraryPathErr == nil {
		libraryPathErr = decklibrary.Ensure(deckLibraryDirectory)
	}

	/*
		Left panel: selected card preview and information
	*/

	cardNameLabel := widget.NewLabel("No card selected")
	cardNameLabel.TextStyle = fyne.TextStyle{
		Bold: true,
	}

	printingLabel := widget.NewLabel("Artwork / Printing")
	printingSelect := widget.NewSelect(nil, nil)
	printingLabel.Hide()
	printingSelect.Hide()
	printingCards := make(map[string]cards.Card)
	updatingPrinting := false
	var onPrintingChanged func(cards.Card)

	cardDetailsLabel := widget.NewLabel(
		"Select a card to view its details.",
	)
	cardDetailsLabel.Wrapping = fyne.TextWrapWord

	previewBackground := canvas.NewRectangle(
		color.Transparent,
	)
	previewBackground.SetMinSize(
		fyne.NewSize(previewWidth, previewHeight),
	)

	previewMessage := widget.NewLabel(
		"Select a card",
	)
	previewMessage.Alignment = fyne.TextAlignCenter
	previewMessage.Wrapping = fyne.TextWrapWord

	cardPreview := container.NewStack(
		previewBackground,
		previewMessage,
	)
	cardPreviewSizer := canvas.NewRectangle(color.Transparent)
	cardPreviewSizer.SetMinSize(fyne.NewSize(0, cardPreviewHeight))
	cardPreviewRegion := container.NewStack(cardPreviewSizer, cardPreview)

	showPreviewMessage := func(message string) {
		label := widget.NewLabel(message)
		label.Alignment = fyne.TextAlignCenter
		label.Wrapping = fyne.TextWrapWord

		cardPreview.RemoveAll()
		cardPreview.Add(previewBackground)
		cardPreview.Add(label)
		cardPreview.Refresh()
	}

	showCard := func(card cards.Card) {
		cardNameLabel.SetText(card.Name)

		cardDetailsLabel.SetText(fmt.Sprintf(
			"Type: %s\n"+
				"Element: %s\n"+
				"Cost/Lv: %s\n"+
				"Traits: %s\n"+
				"Expansion: %s\n"+
				"Card Number: %s\n\n"+
				"%s",
			card.Type,
			card.Element,
			card.CostLevel,
			card.Traits,
			card.Expansion,
			card.CardNumber,
			card.Ability,
		))

		localImagePath, found := cardimages.Find(
			card.ID,
		)
		if !found {
			showPreviewMessage(
				"Image has not been downloaded",
			)
			return
		}

		cardImage := canvas.NewImageFromFile(
			localImagePath,
		)
		cardImage.FillMode = canvas.ImageFillContain
		cardImage.ScaleMode = canvas.ImageScaleSmooth

		cardPreview.RemoveAll()
		cardPreview.Add(previewBackground)
		cardPreview.Add(cardImage)
		cardPreview.Refresh()
	}

	printingSelect.OnChanged = func(selected string) {
		if updatingPrinting {
			return
		}

		card, found := printingCards[selected]
		if !found {
			return
		}

		showCard(card)
		if onPrintingChanged != nil {
			onPrintingChanged(card)
		}
	}

	showPrintingGroup := func(
		group cards.PrintingGroup,
		selectedID string,
		onChanged func(cards.Card),
	) {
		selected := group.Preferred
		options := make([]string, 0, len(group.Printings))
		printingCards = make(map[string]cards.Card, len(group.Printings))
		selectedOption := ""

		for _, printing := range group.Printings {
			option := printingOptionLabel(printing)
			if _, duplicate := printingCards[option]; duplicate {
				option = fmt.Sprintf("%s [%s]", option, printing.ID)
			}
			options = append(options, option)
			printingCards[option] = printing
			if printing.ID == selectedID {
				selected = printing
				selectedOption = option
			}
		}

		if selectedOption == "" && len(options) > 0 {
			selectedOption = options[0]
		}

		onPrintingChanged = onChanged
		updatingPrinting = true
		printingSelect.Options = options
		printingSelect.SetSelected(selectedOption)
		updatingPrinting = false

		if len(options) > 1 {
			printingLabel.Show()
			printingSelect.Show()
		} else {
			printingLabel.Hide()
			printingSelect.Hide()
		}

		showCard(selected)
	}

	detailsScroll := container.NewVScroll(cardDetailsLabel)

	detailsScroll.SetMinSize(fyne.NewSize(0, 180))

	detailsPanel := container.NewBorder(
		container.NewVBox(
			cardNameLabel,
			printingLabel,
			printingSelect,
		),
		nil,
		nil,
		nil,
		detailsScroll,
	)

	leftBody := container.NewBorder(
		cardPreviewRegion,
		nil,
		nil,
		nil,
		detailsPanel,
	)

	leftPanel := container.NewBorder(
		widget.NewLabel("Card Information"),
		nil,
		nil,
		nil,
		leftBody,
	)
	leftPanelSizer := canvas.NewRectangle(color.Transparent)
	leftPanelSizer.SetMinSize(fyne.NewSize(informationPanelWidth, 0))
	leftPanelRegion := container.NewStack(leftPanelSizer, leftPanel)
	/*
		Center panel: deck controls and deck zones
	*/

	newButton := widget.NewButton("New", func() {
		makeNewDeck()
	})
	openButton := widget.NewButton("Open", func() {
		loadDeck()
	})
	saveButton := widget.NewButton("Save", func() {
		saveDeck()
	})
	saveAsButton := widget.NewButton("Save As", func() {
		saveDeckAs()
	})
	renameButton := widget.NewButton("Rename", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetText(deck.Name)
		dialog.ShowForm(
			"Rename Deck",
			"Rename",
			"Cancel",
			[]*widget.FormItem{widget.NewFormItem("Deck Name", nameEntry)},
			func(confirmed bool) {
				if !confirmed || strings.TrimSpace(nameEntry.Text) == "" {
					return
				}
				deck.Name = strings.TrimSpace(nameEntry.Text)
				deckDirty = true
			},
			window,
		)
	})
	mainMenuButton := widget.NewButton("Main Menu", func() {
		showMainMenu()
	})

	primaryDeckControls := container.NewBorder(
		nil,
		nil,
		nil,
		mainMenuButton,
		container.NewHBox(
			newButton,
			openButton,
			saveButton,
			saveAsButton,
			renameButton,
		),
	)

	sortButton := widget.NewButton("Sort", func() {
		if err := deck.Sort(repository); err != nil {
			dialog.ShowError(err, window)
			return
		}
		deckDirty = true
		refreshDeckDisplay()
	})
	cardBackButton := widget.NewButton("TTS Card Back", func() {
		showTTSCardBackDialog(window, deck, func() {
			deckDirty = true
		})
	})
	installTTSButton := widget.NewButton("Install to TTS", func() {
		showTTSInstallDialog(window, deck, repository)
	})

	var exportSelect *widget.Select
	exportSelect = widget.NewSelect(
		[]string{"Decklist", "Main Image", "Sideboard Image"},
		func(selected string) {
			switch selected {
			case "Decklist":
				showDecklistSaveDialog(window, deck, repository)
			case "Main Image":
				showDeckImageExportDialog(window, deck, false)
			case "Sideboard Image":
				showDeckImageExportDialog(window, deck, true)
			default:
				return
			}
			exportSelect.ClearSelected()
		},
	)
	exportSelect.PlaceHolder = "Export…"

	exportControls := container.NewHBox(
		widget.NewLabel("Actions"),
		sortButton,
		exportSelect,
		cardBackButton,
		installTTSButton,
	)

	const cardHeightToWidth float32 = 182.0 / 130.0

	mainDeckGrid := container.New(
		&deckgui.CardGridLayout{
			Columns:          10,
			HeightToWidth:    cardHeightToWidth,
			Padding:          6,
			MinimumCellWidth: 44,
		},
	)

	sideDeckGrid := container.New(
		&deckgui.CardGridLayout{
			Columns:          decks.MaxSideDeckCards,
			HeightToWidth:    cardHeightToWidth,
			Padding:          4,
			MinimumCellWidth: 32,
		},
	)

	mainDeckLabel := widget.NewLabel(
		"Main Deck (0)",
	)

	sideDeckLabel := widget.NewLabel(
		"Side Deck (0)",
	)

	var mainDeckPanel *fyne.Container
	var sideDeckPanel *fyne.Container

	mainDeckPanel = container.NewBorder(
		mainDeckLabel,
		nil,
		nil,
		nil,
		mainDeckGrid,
	)

	sideDeckPanel = container.NewBorder(
		sideDeckLabel,
		nil,
		nil,
		nil,
		sideDeckGrid,
	)

	var dragController *deckgui.CardDragController

	dragController = deckgui.NewCardDragController(dragLayer, mainDeckPanel, sideDeckPanel, mainDeckGrid, sideDeckGrid, func(source deckgui.CardDragSource, target *deckgui.CardDropTarget) {
		defer refreshDeckDisplay()
		if err := applyCardDrop(deck, repository, source, target); err != nil {
			dialog.ShowError(err, window)
			return
		}
		deckDirty = true
		selection.Clear()
	},
	)

	/*
		refreshDeckDisplay is declared first because its card-tile
		callbacks call refreshDeckDisplay again after removing a card.
	*/

	refreshDeckDisplay = func() {
		// The display is rebuilt each time, so remove the old tiles first.
		mainDeckGrid.RemoveAll()
		sideDeckGrid.RemoveAll()

		deck.EnsureOrder()

		/*
			Main deck
		*/
		for index, cardID := range deck.MainOrder {
			currentIndex := index

			card, found := repository.FindByID(cardID)
			if !found {
				continue
			}

			currentCard := card

			tile := deckgui.NewCardTileSized(
				currentCard,
				mainDeckTileMinSize,

				func(selected cards.Card) {
					showCard(selected)
				},

				func(_ cards.Card, _ bool) {
					err := deck.RemoveCardAt(
						decks.MainZone,
						currentIndex,
					)
					if err != nil {
						dialog.ShowError(err, window)
						return
					}
					selection.Clear()
					deckDirty = true

					refreshDeckDisplay()
				},
			)
			isSelected := selection.Contains(decks.MainZone, currentIndex)
			tile.SetSelectedVisual(isSelected)

			tile.OnToggleSelection = func() {
				selection.Toggle(decks.MainZone, currentIndex)
				refreshDeckDisplay()
			}

			dragIndices := []int{currentIndex}

			if isSelected {
				dragIndices = selection.SortedIndices()
			}

			tile.EnableDrag(
				deckgui.CardDragSource{
					Kind:    deckgui.DragFromDeck,
					Card:    currentCard,
					Zone:    decks.MainZone,
					Index:   currentIndex,
					Indices: dragIndices,
				},
				dragController.Start,
				dragController.Move,
				dragController.End,
			)

			mainDeckGrid.Add(tile)
		}

		/*
			Side deck
		*/
		for index, cardID := range deck.SideOrder {
			currentIndex := index

			card, found := repository.FindByID(cardID)
			if !found {
				continue
			}

			currentCard := card

			tile := deckgui.NewCardTileSized(
				currentCard,
				sideDeckTileMinSize,

				func(selected cards.Card) {
					showCard(selected)
				},

				func(_ cards.Card, _ bool) {
					err := deck.RemoveCardAt(
						decks.SideZone,
						currentIndex,
					)
					if err != nil {
						dialog.ShowError(err, window)
						return
					}
					selection.Clear()
					deckDirty = true

					refreshDeckDisplay()
				},
			)

			isSelected := selection.Contains(decks.SideZone, currentIndex)
			tile.SetSelectedVisual(isSelected)

			tile.OnToggleSelection = func() {
				selection.Toggle(decks.SideZone, currentIndex)
				refreshDeckDisplay()
			}

			dragIndices := []int{currentIndex}
			if isSelected {
				dragIndices = selection.SortedIndices()
			}

			tile.EnableDrag(
				deckgui.CardDragSource{
					Kind:    deckgui.DragFromDeck,
					Card:    currentCard,
					Zone:    decks.SideZone,
					Index:   currentIndex,
					Indices: dragIndices,
				},
				dragController.Start,
				dragController.Move,
				dragController.End,
			)

			sideDeckGrid.Add(tile)
		}

		mainDeckGrid.Refresh()
		sideDeckGrid.Refresh()

		mainDeckLabel.SetText(fmt.Sprintf(
			"Main Deck (%d/%d)",
			deck.MainTotal(),
			decks.MaxMainDeckCards,
		))

		sideDeckLabel.SetText(fmt.Sprintf(
			"Side Deck (%d/%d)",
			deck.SideTotal(),
			decks.MaxSideDeckCards,
		))
	}

	sideDeckSizer := canvas.NewRectangle(color.Transparent)
	sideDeckSizer.SetMinSize(fyne.NewSize(0, sideDeckPanelHeight))
	sideDeckRegion := container.NewStack(sideDeckSizer, sideDeckPanel)
	deckSplit := container.NewBorder(
		nil,
		sideDeckRegion,
		nil,
		nil,
		mainDeckPanel,
	)

	deckSelector := widget.NewSelect(nil, nil)
	deckSelector.PlaceHolder = "No saved decks"
	refreshLibraryButton := widget.NewButton("Refresh", func() {
		refreshDeckLibrary()
	})
	openLibraryButton := widget.NewButton("Open Folder", func() {
		if libraryPathErr != nil {
			dialog.ShowError(libraryPathErr, window)
			return
		}
		folderURL, err := url.Parse(storage.NewFileURI(deckLibraryDirectory).String())
		if err != nil {
			dialog.ShowError(err, window)
			return
		}
		if err := fyne.CurrentApp().OpenURL(folderURL); err != nil {
			dialog.ShowError(err, window)
		}
	})

	libraryPaths := make(map[string]string)
	officialTemplateIDs := make(map[string]string)
	updatingDeckSelector := false
	refreshDeckLibrary = func() {
		templates := decklibrary.OfficialTemplates()
		options := make([]string, 0, len(templates))
		officialTemplateIDs = make(map[string]string, len(templates))
		for _, template := range templates {
			label := "Official — " + template.Name
			options = append(options, label)
			officialTemplateIDs[label] = template.ID
		}

		var entries []decklibrary.Entry
		if libraryPathErr == nil {
			var discoverErr error
			entries, discoverErr = decklibrary.Discover(deckLibraryDirectory)
			if discoverErr != nil {
				dialog.ShowError(discoverErr, window)
				return
			}
		} else {
			deckSelector.PlaceHolder = "Personal deck library unavailable"
		}
		libraryPaths = make(map[string]string, len(entries))
		for _, entry := range entries {
			label := entry.Name + " (" + strings.TrimPrefix(
				strings.ToLower(filepath.Ext(entry.Path)),
				".",
			) + ")"
			options = append(options, label)
			libraryPaths[label] = entry.Path
		}

		updatingDeckSelector = true
		deckSelector.Options = options
		deckSelector.ClearSelected()
		if currentTemplateID != "" {
			for label, templateID := range officialTemplateIDs {
				if templateID == currentTemplateID {
					deckSelector.SetSelected(label)
					break
				}
			}
		} else if currentDeckPath != "" {
			currentPath := filepath.Clean(currentDeckPath)
			for label, path := range libraryPaths {
				if filepath.Clean(path) == currentPath {
					deckSelector.SetSelected(label)
					break
				}
			}
		}
		updatingDeckSelector = false
		deckSelector.Refresh()
	}
	deckSelector.OnChanged = func(selected string) {
		if updatingDeckSelector {
			return
		}
		path := libraryPaths[selected]
		templateID := officialTemplateIDs[selected]
		if path != "" || templateID != "" {
			switchDeck := func() {
				if templateID != "" {
					loadOfficialTemplate(templateID)
					return
				}
				loadLibraryDeck(path)
			}
			if deckDirty {
				dialog.ShowConfirm(
					"Discard Unsaved Changes?",
					"Switching decks will discard changes that have not been saved.",
					func(discard bool) {
						if discard {
							switchDeck()
							return
						}
						refreshDeckLibrary()
					},
					window,
				)
				return
			}
			switchDeck()
		}
	}

	deckLibraryControls := container.NewBorder(
		widget.NewLabel("Deck"),
		nil,
		nil,
		container.NewHBox(refreshLibraryButton, openLibraryButton),
		deckSelector,
	)

	centerPanel := container.NewBorder(
		container.NewVBox(
			primaryDeckControls,
			deckLibraryControls,
			exportControls,
		),
		nil,
		nil,
		nil,
		deckSplit,
	)

	/*
		Right panel: card search filters and results
	*/

	typeSelect := widget.NewSelect(
		withAnyOption(repository.Types()),
		nil,
	)
	typeSelect.SetSelected(anyOption)

	traitSelect := widget.NewSelect(
		withAnyOption(repository.Traits()),
		nil,
	)
	traitSelect.SetSelected(anyOption)

	keywordSelect := widget.NewSelect(
		withAnyOption(repository.Keywords()),
		nil,
	)
	keywordSelect.SetSelected(anyOption)

	expansionSelect := widget.NewSelect(
		withAnyOption(repository.Expansions()),
		nil,
	)
	expansionSelect.SetSelected(anyOption)

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search card names...")

	costEntry := widget.NewEntry()
	costEntry.SetPlaceHolder(
		"Any cost/level",
	)

	includeTestingCheck := widget.NewCheck(
		"Include playtesting cards",
		nil,
	)

	showAllPrintingsCheck := widget.NewCheck(
		"Show each artwork separately",
		nil,
	)

	elementOptions := repository.Elements()

	elementChecks := make(
		map[string]*widget.Check,
	)

	elementObjects := make(
		[]fyne.CanvasObject,
		0,
		len(elementOptions),
	)

	for _, element := range elementOptions {
		check := widget.NewCheck(
			element,
			nil,
		)

		elementChecks[element] = check
		elementObjects = append(
			elementObjects,
			check,
		)
	}

	elementGrid := container.NewGridWithColumns(
		4,
		elementObjects...,
	)

	searchResultsGrid := container.NewGridWrap(
		fyne.NewSize(140, 196),
	)

	resultCountLabel := widget.NewLabel(
		"No search performed",
	)

	runSearch := func() {
		filter := cards.Filter{
			Name: searchEntry.Text,

			Elements: checkedValues(
				elementOptions,
				elementChecks,
			),

			Types: optionalSelection(
				typeSelect.Selected,
			),

			Traits: optionalSelection(
				traitSelect.Selected,
			),

			Keywords: optionalSelection(
				keywordSelect.Selected,
			),

			CostLevels: optionalValue(
				costEntry.Text,
			),

			Expansions: optionalSelection(
				expansionSelect.Selected,
			),

			IncludeTesting: includeTestingCheck.Checked,
		}

		groups := repository.FilterPrintingGroups(filter)
		printingCount := 0
		for _, group := range groups {
			printingCount += len(group.Printings)
		}

		if showAllPrintingsCheck.Checked {
			separateGroups := make([]cards.PrintingGroup, 0, printingCount)
			for _, group := range groups {
				for _, printing := range group.Printings {
					separateGroups = append(separateGroups, cards.PrintingGroup{
						Preferred: printing,
						Printings: []cards.Card{printing},
					})
				}
			}
			groups = separateGroups
		}

		if printingCount == len(groups) {
			resultCountLabel.SetText(fmt.Sprintf(
				"%d matching card(s)",
				len(groups),
			))
		} else {
			resultCountLabel.SetText(fmt.Sprintf(
				"%d matching card(s), %d printings",
				len(groups),
				printingCount,
			))
		}

		searchResultsGrid.RemoveAll()

		for _, resultGroup := range groups {
			matchedGroup := resultGroup
			matchedCard := matchedGroup.Preferred

			var cardTile *deckgui.CardTile
			cardTile = deckgui.NewCardTile(
				matchedCard,

				/*
					Hover or click:
					Show the card in the preview panel.
				*/
				func(selected cards.Card) {
					showPrintingGroup(
						matchedGroup,
						selected.ID,
						func(printing cards.Card) {
							cardTile.SetCard(printing)
						},
					)
				},

				/*
					Right-click:
					Add one copy to the main deck.

					Shift + right-click:
					Add one copy to the side deck.
				*/
				func(
					selected cards.Card,
					shiftHeld bool,
				) {
					zone := decks.MainZone

					if shiftHeld {
						zone = decks.SideZone
					}

					added, addErr := deck.AddCardChecked(
						zone,
						selected,
						1,
						repository,
					)
					if addErr != nil {
						dialog.ShowError(addErr, window)
						return
					}

					if !added {
						return
					}

					deckDirty = true
					refreshDeckDisplay()
				},
			)
			cardTile.EnableDrag(deckgui.CardDragSource{
				Kind: deckgui.DragFromSearch,
				Card: matchedCard,
			},
				dragController.Start,
				dragController.Move,
				dragController.End,
			)
			searchResultsGrid.Add(cardTile)
		}

		searchResultsGrid.Refresh()
	}

	var searchTimer *time.Timer

	scheduleSearch := func() {
		if searchTimer != nil {
			searchTimer.Stop()
		}
		searchTimer = time.AfterFunc(250*time.Millisecond, func() { fyne.Do(runSearch) })
	}

	updatingFilters := false

	typeSelect.OnChanged = func(_ string) {
		if !updatingFilters {
			runSearch()
		}
	}

	traitSelect.OnChanged = func(_ string) {
		if !updatingFilters {
			runSearch()
		}
	}

	keywordSelect.OnChanged = func(_ string) {
		if !updatingFilters {
			runSearch()
		}
	}

	expansionSelect.OnChanged = func(_ string) {
		if !updatingFilters {
			runSearch()
		}
	}

	for _, check := range elementChecks {
		check.OnChanged = func(_ bool) {
			if !updatingFilters {
				runSearch()
			}
		}
	}

	includeTestingCheck.OnChanged = func(_ bool) {
		if updatingFilters {
			return
		}

		runSearch()
	}

	showAllPrintingsCheck.OnChanged = func(_ bool) {
		if updatingFilters {
			return
		}

		runSearch()
	}

	searchEntry.OnChanged = func(_ string) {
		if updatingFilters {
			return
		}

		scheduleSearch()
	}

	searchEntry.OnSubmitted = func(_ string) {
		if searchTimer != nil {
			searchTimer.Stop()
			searchTimer = nil
		}
		runSearch()
	}

	searchButton := widget.NewButton(
		"Search",
		runSearch,
	)

	clearButton := widget.NewButton(
		"Clear",
		func() {
			if searchTimer != nil {
				searchTimer.Stop()
				searchTimer = nil
			}

			updatingFilters = true

			searchEntry.SetText("")
			costEntry.SetText("")

			for _, check := range elementChecks {
				check.SetChecked(false)
			}

			typeSelect.SetSelected(anyOption)
			traitSelect.SetSelected(anyOption)
			keywordSelect.SetSelected(anyOption)
			expansionSelect.SetSelected(anyOption)

			includeTestingCheck.SetChecked(false)
			showAllPrintingsCheck.SetChecked(false)

			updatingFilters = false

			runSearch()

			resultCountLabel.SetText(
				"Filters cleared",
			)
		},
	)

	filterGrid := container.NewGridWithColumns(
		2,
		container.NewVBox(widget.NewLabel("Cost / Level"), costEntry),
		container.NewVBox(widget.NewLabel("Type"), typeSelect),
		container.NewVBox(widget.NewLabel("Trait"), traitSelect),
		container.NewVBox(widget.NewLabel("Keyword"), keywordSelect),
		container.NewVBox(widget.NewLabel("Expansion"), expansionSelect),
		container.NewVBox(
			widget.NewLabel("Card Pool"),
			includeTestingCheck,
			showAllPrintingsCheck,
		),
	)

	searchControls := container.NewVBox(
		widget.NewLabel("Card Search"),
		container.NewBorder(
			nil,
			nil,
			nil,
			container.NewHBox(
				searchButton,
				clearButton,
			),
			searchEntry,
		),
		widget.NewLabel("Elements"),
		elementGrid,
		filterGrid,
		resultCountLabel,
		widget.NewLabel("Drag deck cards here to remove them."),
	)

	searchControlsScroll := container.NewVScroll(searchControls)
	searchControlsScroll.SetMinSize(fyne.NewSize(0, searchControlsHeight))
	searchResultsScroll := container.NewVScroll(searchResultsGrid)
	rightPanel := container.NewBorder(
		searchControlsScroll,
		nil,
		nil,
		nil,
		searchResultsScroll,
	)
	rightPanelSizer := canvas.NewRectangle(color.Transparent)
	rightPanelSizer.SetMinSize(fyne.NewSize(searchPanelWidth, 0))
	rightPanelRegion := container.NewStack(rightPanelSizer, rightPanel)
	dragController.SetRemovalTarget(rightPanel)

	/*
		Complete application layout
	*/

	root := container.NewBorder(
		nil,
		nil,
		leftPanelRegion,
		rightPanelRegion,
		centerPanel,
	)

	refreshDeckDisplay()

	editorContent := container.NewStack(root, dragLayer)
	showEditor := func() {
		refreshDeckDisplay()
		refreshDeckLibrary()
		window.SetTitle(deck.Name + " — " + applicationName)
		setWindowContent(window, editorContent)
	}

	makeNewDeck = func() {
		showNewDeckDialog(window, func(created *decks.Deck) {
			*deck = *created
			currentDeckURI = nil
			currentDeckPath = ""
			currentTemplateID = ""
			deckDirty = false
			fyne.CurrentApp().Preferences().SetString(activeDeckPreferenceKey, "")
			showEditor()
		})
	}
	loadLibraryDeck = func(path string) {
		var opened *decks.Deck
		var openErr error
		if strings.EqualFold(filepath.Ext(path), ".txt") {
			reader, err := os.Open(path)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			opened, openErr = deckio.ReadDeckList(reader, repository)
			closeErr := reader.Close()
			if openErr == nil && closeErr != nil {
				openErr = closeErr
			}
		} else {
			opened, openErr = deckio.LoadFile(path)
		}
		if openErr != nil {
			dialog.ShowError(openErr, window)
			return
		}
		migrated, migrateErr := opened.CanonicalizeCardIDs(repository)
		if migrateErr != nil {
			dialog.ShowError(migrateErr, window)
			return
		}
		*deck = *opened
		currentDeckPath = path
		currentTemplateID = ""
		currentDeckURI = storage.NewFileURI(path)
		if strings.EqualFold(filepath.Ext(path), ".txt") {
			currentDeckURI = nil
		}
		fyne.CurrentApp().Preferences().SetString(activeDeckPreferenceKey, path)
		selection.Clear()
		deckDirty = migrated
		showEditor()
	}
	loadOfficialTemplate = func(templateID string) {
		opened, buildErr := decklibrary.BuildOfficialTemplate(templateID, repository)
		if buildErr != nil {
			dialog.ShowError(buildErr, window)
			return
		}
		*deck = *opened
		currentDeckURI = nil
		currentDeckPath = ""
		currentTemplateID = templateID
		fyne.CurrentApp().Preferences().SetString(
			activeDeckPreferenceKey,
			officialTemplatePreferencePrefix+templateID,
		)
		selection.Clear()
		deckDirty = false
		showEditor()
	}
	loadDeck = func() {
		showOpenDeckDialog(window, repository, func(opened *decks.Deck, uri fyne.URI, migrated bool) {
			*deck = *opened
			if strings.EqualFold(uri.Extension(), ".json") {
				currentDeckURI = uri
			} else {
				currentDeckURI = nil
			}
			currentDeckPath = uri.Path()
			currentTemplateID = ""
			fyne.CurrentApp().Preferences().SetString(activeDeckPreferenceKey, uri.Path())
			deckDirty = migrated
			showEditor()
		})
	}
	saveDeckAs = func() {
		showSaveDeckDialog(window, deck, deckLibraryDirectory, func(uri fyne.URI) {
			currentDeckURI = uri
			currentDeckPath = uri.Path()
			currentTemplateID = ""
			fyne.CurrentApp().Preferences().SetString(activeDeckPreferenceKey, uri.Path())
			deckDirty = false
			refreshDeckLibrary()
		})
	}
	saveDeck = func() {
		if currentDeckURI == nil {
			saveDeckAs()
			return
		}
		if saveDeckToURI(window, currentDeckURI, deck) {
			deckDirty = false
		}
	}
	openDeckEditor = func() {
		lastSelection := fyne.CurrentApp().Preferences().String(activeDeckPreferenceKey)
		if strings.HasPrefix(lastSelection, officialTemplatePreferencePrefix) {
			loadOfficialTemplate(strings.TrimPrefix(lastSelection, officialTemplatePreferencePrefix))
			return
		}
		if lastSelection != "" {
			if _, statErr := os.Stat(lastSelection); statErr == nil {
				loadLibraryDeck(lastSelection)
				return
			}
		}
		loadOfficialTemplate("dd01-ignus")
	}
	showMainMenu = func() {
		window.SetTitle(applicationName)
		setWindowContent(window, buildMainMenu(window, mainMenuActions{
			PlayGame: func() {
				showSimulatorDeckSelection(window, deckLibraryDirectory, repository, func(playerDecks [2]decks.Deck) {
					playerSessions, err := buildSimulatorSessions(repository, playerDecks, newSimulatorMatchSeed())
					if err != nil {
						dialog.ShowError(err, window)
						return
					}
					playerWindows := [2]fyne.Window{
						window,
						fyne.CurrentApp().NewWindow(applicationName + " — Player Two"),
					}
					playerWindows[0].SetTitle(applicationName + " — Player One")
					playerWindows[1].Resize(fyne.NewSize(1400, 850))
					cardDefinitions := repository.All()
					var playerScreens [2]*simulatorui.BoardScreen

					var renderPlayers func()
					var renderPlayersWithView func(int, simulatorview.MatchView)
					renderPlayersWithView = func(updatedIndex int, updatedView simulatorview.MatchView) {
						for index, playerSession := range playerSessions {
							matchView := updatedView
							if index != updatedIndex {
								var viewErr error
								matchView, viewErr = playerSession.View()
								if viewErr != nil {
									dialog.ShowError(viewErr, playerWindows[index])
									continue
								}
							}
							if playerScreens[index] == nil {
								playerIndex := index
								back := func() {
									if playerIndex == 0 {
										playerWindows[1].Close()
										showMainMenu()
										return
									}
									playerWindows[1].Close()
								}
								backLabel := "Back to Main Menu"
								if index == 1 {
									backLabel = "Close Player Two Window"
								}
								playerScreens[index] = simulatorui.NewBoardController(
									matchView,
									cardDefinitions,
									simulatorui.BoardActions{
										BackLabel: backLabel,
										UseCasterToken: func(
											tokenID model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, tokenErr := playerSessions[playerIndex].
												UseCasterToken(tokenID, expectedRevision)
											if tokenErr != nil {
												dialog.ShowError(tokenErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										GenerateCasterAether: func(
											cardID model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, aetherErr := playerSessions[playerIndex].
												GenerateCasterAether(cardID, expectedRevision)
											if aetherErr != nil {
												dialog.ShowError(aetherErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										GenerateNonElementalAether: func(
											cardID model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, aetherErr := playerSessions[playerIndex].
												GenerateNonElementalAether(cardID, expectedRevision)
											if aetherErr != nil {
												dialog.ShowError(aetherErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										CallFaceDownLevelOne: func(
											cardID model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, callErr := playerSessions[playerIndex].
												CallFaceDownLevelOne(cardID, expectedRevision)
											if callErr != nil {
												dialog.ShowError(callErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										CallFaceUpLevelOne: func(
											cardID model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, callErr := playerSessions[playerIndex].
												CallFaceUpLevelOne(cardID, expectedRevision)
											if callErr != nil {
												dialog.ShowError(callErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										LevelUpCaster: func(
											upperCardID model.MatchCardID,
											targetCasterID model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, levelErr := playerSessions[playerIndex].
												LevelUpCaster(upperCardID, targetCasterID, expectedRevision)
											if levelErr != nil {
												dialog.ShowError(levelErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										CastServant: func(
											cardID model.MatchCardID,
											payment model.AetherPayment,
											orientation model.CardOrientation,
											expectedRevision model.Revision,
										) {
											updatedView, castErr := playerSessions[playerIndex].
												CastServant(cardID, payment, orientation, expectedRevision)
											if castErr != nil {
												dialog.ShowError(castErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										CastConjure: func(
											cardID model.MatchCardID,
											payment model.AetherPayment,
											expectedRevision model.Revision,
										) {
											updatedView, castErr := playerSessions[playerIndex].
												CastConjure(cardID, payment, expectedRevision)
											if castErr != nil {
												dialog.ShowError(castErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										CastBarrier: func(
											cardID model.MatchCardID,
											payment model.AetherPayment,
											expectedRevision model.Revision,
										) {
											updatedView, castErr := playerSessions[playerIndex].
												CastBarrier(cardID, payment, expectedRevision)
											if castErr != nil {
												dialog.ShowError(castErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										PassPriority: func(expectedRevision model.Revision) {
											updatedView, priorityErr := playerSessions[playerIndex].
												PassPriority(expectedRevision)
											if priorityErr != nil {
												dialog.ShowError(priorityErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										CompleteCurrentPhase: func(expectedRevision model.Revision) {
											updatedView, phaseErr := playerSessions[playerIndex].
												CompleteCurrentPhase(expectedRevision)
											if phaseErr != nil {
												dialog.ShowError(phaseErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
										SubmitOpeningHand: func(
											replace []model.MatchCardID,
											expectedRevision model.Revision,
										) {
											updatedView, submitErr := playerSessions[playerIndex].
												SubmitOpeningHandDecision(replace, expectedRevision)
											if submitErr != nil {
												dialog.ShowError(submitErr, playerWindows[playerIndex])
												return
											}
											renderPlayersWithView(playerIndex, updatedView)
										},
									},
									back,
								)
								setWindowContent(playerWindows[index], playerScreens[index].Content())
								continue
							}
							playerScreens[index].Update(matchView)
						}
					}
					renderPlayers = func() {
						renderPlayersWithView(-1, simulatorview.MatchView{})
					}
					renderPlayers()
					playerWindows[1].Show()
				})
			},
			OpenDeckEditor:   openDeckEditor,
			NewDeck:          makeNewDeck,
			LoadDeck:         loadDeck,
			GenerateImage:    func() { showGenerateImageFromDecklistDialog(window, repository) },
			GenerateDecklist: func() { showGenerateDecklistDialog(window, repository) },
			UpdateDatabase:   func() { confirmManualCardDatabaseUpdate(window, paths, repository) },
			Changelog:        func() { showChangelogDialog(window) },
			HowToUse:         func() { showHowToUseDialog(window) },
			Diagnostics:      func() { showDiagnosticInformationDialog(window, paths, repository) },
			Settings: func() {
				showSettingsDialog(window, fyne.CurrentApp(), showMainMenu)
			},
		}))
	}

	runSearch()
	showMainMenu()
}
