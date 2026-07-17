package main

import (
	"fmt"
	"image/color"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

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

func optionalValue(value string) []string {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	return []string{value}
}

func main() {
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
		fyne.NewSize(220, 308),
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
				"Card Number: %s\n"+
				"Card ID: %s\n\n"+
				"%s",
			card.Type,
			card.Element,
			card.CostLevel,
			card.Traits,
			card.Expansion,
			card.CardNumber,
			card.ID,
			card.Ability,
		))

		imageURL := strings.TrimSpace(card.ImageURL)
		if imageURL == "" {
			showPreviewMessage("No image available")
			return
		}

		uri, parseErr := storage.ParseURI(imageURL)
		if parseErr != nil {
			showPreviewMessage("Invalid image URL")

			fmt.Printf(
				"could not parse image URL for %s: %v\n",
				card.Name,
				parseErr,
			)

			return
		}

		cardImage := canvas.NewImageFromURI(uri)
		cardImage.FillMode = canvas.ImageFillContain
		cardImage.SetMinSize(
			fyne.NewSize(220, 308),
		)

		cardPreview.RemoveAll()
		cardPreview.Add(previewBackground)
		cardPreview.Add(cardImage)
		cardPreview.Refresh()
	}

	leftContent := container.NewVBox(
		cardPreview,
		widget.NewSeparator(),
		cardNameLabel,
		cardDetailsLabel,
	)

	leftPanel := container.NewBorder(
		widget.NewLabel("Card Information"),
		nil,
		nil,
		nil,
		container.NewVScroll(leftContent),
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
		fyne.NewSize(100, 140),
	)

	sideDeckGrid := container.NewGridWrap(
		fyne.NewSize(100, 140),
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

		for _, entry := range deck.MainDeck {
			card, found := repository.FindByID(entry.CardID)
			if !found {
				continue
			}

			currentCard := card

			tile := deckgui.NewCardTile(
				currentCard,

				func(selected cards.Card) {
					showCard(selected)
				},

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

		for _, entry := range deck.SideDeck {
			card, found := repository.FindByID(entry.CardID)
			if !found {
				continue
			}

			currentCard := card

			tile := deckgui.NewCardTile(
				currentCard,

				func(selected cards.Card) {
					showCard(selected)
				},

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

	var selectedType string
	var selectedTrait string
	var selectedExpansion string

	typeSelect := widget.NewSelect(
		repository.Types(),
		func(value string) {
			selectedType = value
		},
	)
	typeSelect.PlaceHolder = "Any type"

	traitSelect := widget.NewSelect(
		repository.Traits(),
		func(value string) {
			selectedTrait = value
		},
	)
	traitSelect.PlaceHolder = "Any trait"

	expansionSelect := widget.NewSelect(
		repository.Expansions(),
		func(value string) {
			selectedExpansion = value
		},
	)
	expansionSelect.PlaceHolder = "Any expansion"

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder(
		"Search card names...",
	)

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
		fyne.NewSize(110, 154),
	)

	resultCountLabel := widget.NewLabel(
		"No search performed",
	)

	searchButton := widget.NewButton(
		"Search",
		func() {
			filter := cards.Filter{
				Name: searchEntry.Text,

				Elements: checkedValues(
					elementOptions,
					elementChecks,
				),

				Types: optionalValue(
					selectedType,
				),

				Traits: optionalValue(
					selectedTrait,
				),

				CostLevels: optionalValue(
					costEntry.Text,
				),

				Expansions: optionalValue(
					selectedExpansion,
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

						addErr := deck.AddCard(
							zone,
							selected.ID,
							1,
						)
						if addErr != nil {
							dialog.ShowError(
								addErr,
								window,
							)
							return
						}

						refreshDeckDisplay()
					},
				)

				searchResultsGrid.Add(cardTile)
			}

			searchResultsGrid.Refresh()
		},
	)

	clearButton := widget.NewButton(
		"Clear",
		func() {
			searchEntry.SetText("")
			costEntry.SetText("")

			for _, check := range elementChecks {
				check.SetChecked(false)
			}

			typeSelect.ClearSelected()
			traitSelect.ClearSelected()
			expansionSelect.ClearSelected()

			selectedType = ""
			selectedTrait = ""
			selectedExpansion = ""

			includeTestingCheck.SetChecked(false)

			searchResultsGrid.RemoveAll()
			searchResultsGrid.Refresh()

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
	leftCenter.SetOffset(0.23)

	root := container.NewHSplit(
		leftCenter,
		rightPanel,
	)
	root.SetOffset(0.77)

	refreshDeckDisplay()

	window.SetContent(root)
	window.ShowAndRun()
}