package main

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
	deckgui "github.com/HybridUofA/caster-deckbuilder/internal/gui"
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
	writeImage := decks.WriteDeckImage
	fileSuffix := ""
	if sideboard {
		writeImage = decks.WriteSideboardImage
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

// showApplication builds the main menu and deck editor around the active card repository.
func showApplication(
	window fyne.Window,
	paths applicationPaths,
	repository *cards.Repository,
) {

	const previewWidth float32 = 160
	const previewHeight float32 = 224
	mainDeckTileMinSize := fyne.NewSize(48, 67)
	sideDeckTileMinSize := fyne.NewSize(32, 45)

	dragLayer := container.NewWithoutLayout()

	deck, err := decks.NewDeck("New Deck")
	if err != nil {
		dialog.ShowError(err, window)
		return
	}
	var currentDeckURI fyne.URI
	var showMainMenu func()
	var makeNewDeck func()
	var loadDeck func()
	var saveDeck func()
	var saveDeckAs func()
	var refreshDeckDisplay func()

	/*
		Left panel: selected card preview and information
	*/

	cardNameLabel := widget.NewLabel("No card selected")
	cardNameLabel.TextStyle = fyne.TextStyle{
		Bold: true,
	}

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

	detailsScroll := container.NewVScroll(cardDetailsLabel)

	detailsScroll.SetMinSize(fyne.NewSize(0, 180))

	detailsPanel := container.NewBorder(
		cardNameLabel,
		nil,
		nil,
		nil,
		detailsScroll,
	)

	leftBody := container.NewVSplit(
		cardPreview,
		detailsPanel,
	)

	leftBody.SetOffset(0.58)

	leftPanel := container.NewBorder(
		widget.NewLabel("Card Information"),
		nil,
		nil,
		nil,
		leftBody,
	)
	/*
		Center panel: deck controls and deck zones
	*/

	deckControls := container.NewGridWithColumns(
		5,
		widget.NewButton("New", func() {
			makeNewDeck()
		}),
		widget.NewButton("Open", func() {
			loadDeck()
		}),
		widget.NewButton("Save", func() {
			saveDeck()
		}),
		widget.NewButton("Save As", func() {
			saveDeckAs()
		}),
		widget.NewButton("Export Main", func() {
			showDeckImageExportDialog(window, deck, false)
		}),
		widget.NewButton("Export Sideboard", func() {
			showDeckImageExportDialog(window, deck, true)
		}),
		widget.NewButton("Export Decklist", func() {
			showDecklistSaveDialog(window, deck, repository)
		}),
		widget.NewButton("Rename", func() {
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
				},
				window,
			)
		}),
		widget.NewButton("Sort Deck", func() {
			if err := deck.Sort(repository); err != nil {
				dialog.ShowError(err, window)
				return
			}
			refreshDeckDisplay()
		}),
		widget.NewButton("Main Menu", func() {
			showMainMenu()
		}),
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
		if target == nil {
			return
		}
		switch source.Kind {
		case deckgui.DragFromSearch:
			_, err := deck.AddCardCheckedAt(target.Zone, source.Card, 1, repository, target.Index)
			if err != nil {
				dialog.ShowError(err, window)
			}
		case deckgui.DragFromDeck:
			_, err := deck.MoveOrderedCard(source.Zone, source.Index, target.Zone, target.Index)
			if err != nil {
				dialog.ShowError(err, window)
			}
		}
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

					refreshDeckDisplay()
				},
			)

			tile.EnableDrag(
				deckgui.CardDragSource{
					Kind:  deckgui.DragFromDeck,
					Card:  currentCard,
					Zone:  decks.MainZone,
					Index: currentIndex,
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

					refreshDeckDisplay()
				},
			)

			tile.EnableDrag(
				deckgui.CardDragSource{
					Kind:  deckgui.DragFromDeck,
					Card:  currentCard,
					Zone:  decks.SideZone,
					Index: currentIndex,
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

	deckSplit := container.NewVSplit(
		mainDeckPanel,
		sideDeckPanel,
	)
	deckSplit.SetOffset(0.72)

	centerPanel := container.NewBorder(
		deckControls,
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
		2,
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

			CostLevels: optionalValue(
				costEntry.Text,
			),

			Expansions: optionalSelection(
				expansionSelect.Selected,
			),

			IncludeTesting: includeTestingCheck.Checked,
		}

		matches := repository.Filter(filter)

		cards.SortForSearch(matches)

		resultCountLabel.SetText(fmt.Sprintf(
			"%d matching card(s)",
			len(matches),
		))

		searchResultsGrid.RemoveAll()

		for _, match := range matches {
			matchedCard := match

			cardTile := deckgui.NewCardTile(
				matchedCard,

				/*
					Left-click:
					Show the card in the preview panel.
				*/
				func(selected cards.Card) {
					showCard(selected)
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
			expansionSelect.SetSelected(anyOption)

			includeTestingCheck.SetChecked(false)

			updatingFilters = false

			runSearch()

			resultCountLabel.SetText(
				"Filters cleared",
			)
		},
	)

	searchControls := container.NewVBox(
		widget.NewLabel("Card Search"),

		searchEntry,

		widget.NewLabel("Elements"),
		elementGrid,

		widget.NewLabel("Cost / Level"),
		costEntry,

		widget.NewLabel("Type"),
		typeSelect,

		widget.NewLabel("Trait"),
		traitSelect,

		widget.NewLabel("Expansion"),
		expansionSelect,

		includeTestingCheck,

		container.NewGridWithColumns(
			2,
			searchButton,
			clearButton,
		),

		resultCountLabel,
	)

	rightPanel := container.NewVSplit(
		container.NewVScroll(
			searchControls,
		),
		container.NewVScroll(
			searchResultsGrid,
		),
	)
	rightPanel.SetOffset(0.58)

	/*
		Complete application layout
	*/

	leftCenter := container.NewHSplit(
		leftPanel,
		centerPanel,
	)
	leftCenter.SetOffset(0.28)

	root := container.NewHSplit(
		leftCenter,
		rightPanel,
	)
	root.SetOffset(0.77)

	refreshDeckDisplay()

	editorContent := container.NewStack(root, dragLayer)
	showEditor := func() {
		refreshDeckDisplay()
		window.SetTitle(deck.Name + " — " + applicationName)
		window.SetContent(editorContent)
	}

	makeNewDeck = func() {
		showNewDeckDialog(window, func(created *decks.Deck) {
			*deck = *created
			currentDeckURI = nil
			showEditor()
		})
	}
	loadDeck = func() {
		showOpenDeckDialog(window, repository, func(opened *decks.Deck, uri fyne.URI) {
			*deck = *opened
			if strings.EqualFold(uri.Extension(), ".json") {
				currentDeckURI = uri
			} else {
				currentDeckURI = nil
			}
			showEditor()
		})
	}
	saveDeckAs = func() {
		showSaveDeckDialog(window, deck, func(uri fyne.URI) {
			currentDeckURI = uri
		})
	}
	saveDeck = func() {
		if currentDeckURI == nil {
			saveDeckAs()
			return
		}
		saveDeckToURI(window, currentDeckURI, deck)
	}
	showMainMenu = func() {
		window.SetTitle(applicationName)
		window.SetContent(buildMainMenu(window, mainMenuActions{
			NewDeck:          makeNewDeck,
			LoadDeck:         loadDeck,
			GenerateImage:    func() { showGenerateImageFromDecklistDialog(window, repository) },
			GenerateDecklist: func() { showGenerateDecklistDialog(window, repository) },
			UpdateDatabase:   func() { confirmManualCardDatabaseUpdate(window, paths, repository) },
			Settings:         func() { showSettingsDialog(window, fyne.CurrentApp()) },
		}))
	}

	runSearch()
	showMainMenu()
}
