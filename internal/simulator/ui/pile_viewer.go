package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

type pileViewerRequest struct {
	title    string
	zone     model.Zone
	playerID model.PlayerID
	cards    []simulatorview.CardView
	canMove  bool
}

func (screen *BoardScreen) showPileViewer(request pileViewerRequest) dialog.Dialog {
	if screen == nil {
		return nil
	}
	window := windowForObject(screen.content)
	if window == nil {
		return nil
	}
	if len(request.cards) == 0 {
		dialog.ShowInformation(request.title, "This pile is empty.", window)
		return nil
	}

	type pileEntry struct {
		label string
		card  simulatorview.CardView
	}
	entries := make([]pileEntry, 0, len(request.cards))
	labels := make([]string, 0, len(request.cards))
	for index, card := range request.cards {
		definition := screen.definitions[card.CardID]
		name := strings.TrimSpace(definition.Name)
		if name == "" {
			if card.CardID != "" {
				name = string(card.CardID)
			} else {
				name = "Unknown card"
			}
		}
		label := fmt.Sprintf("%d. %s", index+1, name)
		entries = append(entries, pileEntry{label: label, card: card})
		labels = append(labels, label)
	}

	search := widget.NewEntry()
	search.SetPlaceHolder("Search by name…")
	list := widget.NewList(
		func() int { return len(labels) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, object fyne.CanvasObject) {
			object.(*widget.Label).SetText(labels[id])
		},
	)

	filter := func(query string) {
		query = strings.ToLower(strings.TrimSpace(query))
		labels = labels[:0]
		entries = entries[:0]
		for index, card := range request.cards {
			definition := screen.definitions[card.CardID]
			name := strings.TrimSpace(definition.Name)
			if name == "" {
				name = string(card.CardID)
			}
			if query != "" && !strings.Contains(strings.ToLower(name), query) {
				continue
			}
			label := fmt.Sprintf("%d. %s", index+1, name)
			labels = append(labels, label)
			entries = append(entries, pileEntry{label: label, card: card})
		}
		list.Refresh()
	}
	search.OnChanged = filter

	status := widget.NewLabel("Select a card to preview.")
	status.Wrapping = fyne.TextWrapWord
	var selected simulatorview.CardView
	var selectedOK bool

	moveButtons := container.NewHBox()
	refreshMoves := func() {
		moveButtons.Objects = nil
		if !request.canMove || !selectedOK || screen.actions.MoveCard == nil || !cardMovable(selected) {
			refreshContainerStructure(moveButtons)
			return
		}
		destinations := []struct {
			label string
			zone  model.Zone
		}{
			{"To Hand", model.ZoneHand},
			{"To Graveyard", model.ZoneGraveyard},
			{"To Exile", model.ZoneExile},
			{"To Deck (top)", model.ZoneDeck},
			{"To Servant", model.ZoneServant},
			{"To Caster", model.ZoneCaster},
		}
		for _, destination := range destinations {
			if destination.zone == request.zone {
				continue
			}
			destination := destination
			button := widget.NewButton(destination.label, func() {
				command, err := buildDefaultMoveCommand(
					selected,
					request.zone,
					request.playerID,
					destination.zone,
					screen.definitions[selected.CardID],
					model.DeckPlacementTop,
				)
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				screen.actions.MoveCard(command, screen.match.Revision)
			})
			moveButtons.Objects = append(moveButtons.Objects, button)
		}
		refreshContainerStructure(moveButtons)
	}

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || int(id) >= len(entries) {
			selectedOK = false
			refreshMoves()
			return
		}
		selected = entries[id].card
		selectedOK = true
		definition := screen.definitions[selected.CardID]
		if definition.Name != "" {
			screen.preview.showCard(definition)
			status.SetText(definition.Name)
		} else {
			status.SetText(entries[id].label)
		}
		refreshMoves()
	}

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("%s (%d)", request.title, len(request.cards))),
			search,
			status,
			moveButtons,
		),
		nil,
		nil,
		nil,
		list,
	)
	popup := dialog.NewCustom(request.title, "Close", content, window)
	resizeDialogFraction(popup, window, 0.55, 0.7, 520, 480, 1200, 900)
	if request.zone == model.ZoneDeck && screen.actions.ShuffleDeck != nil {
		// Cockatrice-style: after looking at the library, shuffle so order is
		// not retained from the browse.
		popup.SetOnClosed(func() {
			screen.actions.ShuffleDeck(screen.match.Revision)
		})
	}
	popup.Show()
	return popup
}
