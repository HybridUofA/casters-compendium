package main

import (
	"fmt"
	"image/color"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/caster-deckbuilder/internal/cardimages"
	"github.com/HybridUofA/caster-deckbuilder/internal/cards"
	"github.com/HybridUofA/caster-deckbuilder/internal/decks"
	deckgui "github.com/HybridUofA/caster-deckbuilder/internal/gui"
)

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

func withAnyOption(options []string) []string {
	result := make(
		[]string,
		0,
		len(options)+1,
	)
	result = append(result,anyOption)
	result = append(result, options...)

	return result
}

func optionalSelection(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || value == anyOption {
		return nil
	}
	return []string{value}
}

func optionalValue(value string) []string {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	return []string{value}
}

func main() {

	const previewWidth float32 = 160
	const previewHeight float32 = 224
	repository, err := cards.LoadFile("data/cards.json")
	if err != nil {
		log.Fatal(err)
	}

	deck, err := decks.NewDeck("New Deck")
	if err != nil {
		log.Fatal(err)
	}

	guiApp := app.NewWithID(
		"io.github.hybriduofa.casterdeckbuilder",
	)

	window := guiApp.NewWindow(
		"Caster Chronicles Deckbuilder",
	)
	window.Resize(fyne.NewSize(1400, 850))

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

	deckControls := container.NewHBox(
		widget.NewButton("New", func() {
			fmt.Println("New deck is not implemented yet.")
		}),
		widget.NewButton("Open", func() {
			fmt.Println("Open deck is not implemented yet.")
		}),
		widget.NewButton("Save", func() {
			fmt.Println("Save deck is not implemented yet.")
		}),
		widget.NewButton("Save As", func() {
			fmt.Println("Save As is not implemented yet.")
		}),
		widget.NewButton("Rename", func() {
			fmt.Println("Rename deck is not implemented yet.")
		}),
	)

	mainDeckGrid := container.NewGridWrap(
		fyne.NewSize(130, 182),
	)

	sideDeckGrid := container.NewGridWrap(
		fyne.NewSize(130, 182),
	)

	mainDeckLabel := widget.NewLabel(
		"Main Deck (0)",
	)

	sideDeckLabel := widget.NewLabel(
		"Side Deck (0)",
	)

	/*
		refreshDeckDisplay is declared first because its card-tile
		callbacks call refreshDeckDisplay again after removing a card.
	*/

	var refreshDeckDisplay func()

	refreshDeckDisplay = func() {
		mainDeckGrid.RemoveAll()
		sideDeckGrid.RemoveAll()

		/*
			Main deck
		*/
		for _, entry := range deck.MainDeck {
			card, found := repository.FindByID(entry.CardID)
			if !found {
				continue
			}

			// Create one thumbnail for every physical copy.
			for copyNumber := 0; copyNumber < entry.Quantity; copyNumber++ {
				currentCard := card

				tile := deckgui.NewCardTile(
					currentCard,

					// Left-click: show the card preview.
					func(selected cards.Card) {
						showCard(selected)
					},

					// Right-click: remove one copy.
					func(selected cards.Card, _ bool) {
						removeErr := deck.RemoveCard(
							decks.MainZone,
							selected.ID,
							1,
						)
						if removeErr != nil {
							dialog.ShowError(
								removeErr,
								window,
							)
							return
						}

						refreshDeckDisplay()
					},
				)

				mainDeckGrid.Add(tile)
			}
		}

		/*
			Side deck
		*/
		for _, entry := range deck.SideDeck {
			card, found := repository.FindByID(entry.CardID)
			if !found {
				continue
			}

			// Create one thumbnail for every physical copy.
			for copyNumber := 0; copyNumber < entry.Quantity; copyNumber++ {
				currentCard := card

				tile := deckgui.NewCardTile(
					currentCard,

					// Left-click: show the card preview.
					func(selected cards.Card) {
						showCard(selected)
					},

					// Right-click: remove one copy.
					func(selected cards.Card, _ bool) {
						removeErr := deck.RemoveCard(
							decks.SideZone,
							selected.ID,
							1,
						)
						if removeErr != nil {
							dialog.ShowError(
								removeErr,
								window,
							)
							return
						}

						refreshDeckDisplay()
					},
				)

				sideDeckGrid.Add(tile)
			}
		}

		mainDeckGrid.Refresh()
		sideDeckGrid.Refresh()

		mainDeckLabel.SetText(fmt.Sprintf(
			"Main Deck (%d)",
			deck.MainTotal(),
		))

		sideDeckLabel.SetText(fmt.Sprintf(
			"Side Deck (%d)",
			deck.SideTotal(),
		))
	}

	mainDeckPanel := container.NewBorder(
		mainDeckLabel,
		nil,
		nil,
		nil,
		container.NewVScroll(mainDeckGrid),
	)

	sideDeckPanel := container.NewBorder(
		sideDeckLabel,
		nil,
		nil,
		nil,
		container.NewVScroll(sideDeckGrid),
	)

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

			searchResultsGrid.Add(cardTile)
		}

		searchResultsGrid.Refresh()
	}

	var searchTimer *time.Timer

	scheduleSearch := func() {
		if searchTimer != nil {
			searchTimer.Stop()
		}
		searchTimer = time.AfterFunc(250 * time.Millisecond, func() {fyne.Do(runSearch)})
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

	window.SetContent(root)
	runSearch()
	window.ShowAndRun()
}