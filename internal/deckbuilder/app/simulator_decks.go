package deckbuilder

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/decklibrary"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
)

type simulatorDeckChoice struct {
	label      string
	path       string
	templateID string
}

func simulatorDeckChoices(libraryDirectory string) ([]simulatorDeckChoice, error) {
	choices := make([]simulatorDeckChoice, 0)
	entries, err := decklibrary.Discover(libraryDirectory)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(entry.Path)), ".")
		choices = append(choices, simulatorDeckChoice{
			label: fmt.Sprintf("My Decks — %s (%s)", entry.Name, extension),
			path:  entry.Path,
		})
	}
	for _, template := range decklibrary.OfficialTemplates() {
		choices = append(choices, simulatorDeckChoice{
			label:      "Official — " + template.Name,
			templateID: template.ID,
		})
	}
	return choices, nil
}

func resolveSimulatorDeckChoice(
	choice simulatorDeckChoice,
	repository *cards.Repository,
) (*decks.Deck, error) {
	switch {
	case choice.templateID != "":
		return decklibrary.BuildOfficialTemplate(choice.templateID, repository)
	case choice.path != "":
		return loadSimulatorDeck(choice.path, repository)
	default:
		return nil, fmt.Errorf("simulator deck choice is empty")
	}
}

func showSimulatorDeckSelection(
	window fyne.Window,
	libraryDirectory string,
	repository *cards.Repository,
	onStart func([2]decks.Deck),
) {
	choices, err := simulatorDeckChoices(libraryDirectory)
	if err != nil {
		dialog.ShowError(fmt.Errorf("discover simulator decks: %w", err), window)
		return
	}
	if len(choices) == 0 {
		dialog.ShowInformation("No Decks Available", "Save a deck before starting the simulator.", window)
		return
	}
	byLabel := make(map[string]simulatorDeckChoice, len(choices))
	labels := make([]string, 0, len(choices))
	for _, choice := range choices {
		labels = append(labels, choice.label)
		byLabel[choice.label] = choice
	}
	playerOne := widget.NewSelect(labels, nil)
	playerTwo := widget.NewSelect(labels, nil)
	playerOne.PlaceHolder = "Choose Player One's deck"
	playerTwo.PlaceHolder = "Choose Player Two's deck"
	playerOne.SetSelected(labels[0])
	if len(labels) > 1 {
		playerTwo.SetSelected(labels[1])
	} else {
		playerTwo.SetSelected(labels[0])
	}

	dialog.ShowForm(
		"Choose Simulator Decks",
		"Start Match",
		"Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Player One", playerOne),
			widget.NewFormItem("Player Two", playerTwo),
		},
		func(confirmed bool) {
			if !confirmed {
				return
			}
			selected := [2]simulatorDeckChoice{byLabel[playerOne.Selected], byLabel[playerTwo.Selected]}
			var playerDecks [2]decks.Deck
			for index, choice := range selected {
				resolved, resolveErr := resolveSimulatorDeckChoice(choice, repository)
				if resolveErr != nil {
					dialog.ShowError(fmt.Errorf("Player %d deck: %w", index+1, resolveErr), window)
					return
				}
				playerDecks[index] = *resolved
			}
			if onStart != nil {
				onStart(playerDecks)
			}
		},
		window,
	)
}
