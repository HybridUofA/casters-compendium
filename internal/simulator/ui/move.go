package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

type manualMoveSource struct {
	card simulatorview.CardView
	zone model.Zone
}

func (screen *BoardScreen) updateManualMoves() {
	host := screen.preview.manualActions
	host.Objects = nil
	if screen.actions.MoveCard != nil && screen.match.MatchStatus == model.StatusInProgress {
		button := widget.NewButton("Move a card…", func() {
			host.Objects = []fyne.CanvasObject{screen.newManualMovePanel()}
			refreshContainerStructure(host)
		})
		host.Objects = []fyne.CanvasObject{button}
	}
	refreshContainerStructure(host)
}

func (screen *BoardScreen) newManualMovePanel() fyne.CanvasObject {
	// Capture the revision represented by these choices, never substitute a
	// newer revision when submitting an older selection.
	match := screen.match
	sources := make(map[string]manualMoveSource)
	options := []string{}
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			continue
		}
		for _, group := range []struct {
			zone  model.Zone
			cards []simulatorview.CardView
		}{
			{model.ZoneHand, player.Hand}, {model.ZoneCaster, player.CasterZone},
			{model.ZoneServant, player.ServantZone}, {model.ZoneGraveyard, player.Graveyard},
			{model.ZoneExile, player.Exile},
		} {
			for _, card := range group.cards {
				if card.MatchID == "" || card.CardID == "" || card.CardID == model.CasterTokenCardID {
					continue
				}
				if card.HasStock {
					continue
				}
				if group.zone == model.ZoneExile && card.Face == model.CardFaceDown {
					continue
				}
				definition, found := screen.definitions[card.CardID]
				if !found {
					continue
				}
				name := []rune(definition.Name)
				if len(name) > 20 {
					name = append(name[:19], '…')
				}
				label := fmt.Sprintf("%d. %s — %s", len(options)+1, string(name), group.zone)
				options = append(options, label)
				sources[label] = manualMoveSource{card: card, zone: group.zone}
			}
		}
	}
	cardChoice := widget.NewSelect(options, nil)
	cardChoice.PlaceHolder = "Choose card"
	zoneChoice := widget.NewSelect([]string{string(model.ZoneHand), string(model.ZoneDeck), string(model.ZoneGraveyard), string(model.ZoneExile), string(model.ZoneCaster), string(model.ZoneServant)}, nil)
	zoneChoice.PlaceHolder = "Destination zone"
	playerChoice := widget.NewSelect([]string{"Your side", "Opponent's side"}, nil)
	playerChoice.SetSelected("Your side")
	faceChoice := widget.NewSelect([]string{string(model.CardFaceUp), string(model.CardFaceDown)}, nil)
	orientationChoice := widget.NewSelect(nil, nil)
	placementChoice := widget.NewSelect([]string{string(model.DeckPlacementTop), string(model.DeckPlacementBottom)}, nil)
	placementChoice.SetSelected(string(model.DeckPlacementTop))
	status := widget.NewLabel("Select a card and destination.")
	status.Wrapping = fyne.TextWrapWord
	var command model.MoveCardCommand
	confirm := widget.NewButton("Confirm move", func() {
		screen.actions.MoveCard(command, match.Revision)
	})
	confirm.Disable()
	updating := false
	update := func(reset bool) {
		if updating {
			return
		}
		updating = true
		defer func() { updating = false }()
		confirm.Disable()
		zone := model.Zone(zoneChoice.Selected)
		source, selected := sources[cardChoice.Selected]
		field := zone == model.ZoneCaster || zone == model.ZoneServant
		faceChoice.Hide()
		orientationChoice.Hide()
		placementChoice.Hide()
		if zone == model.ZoneDeck {
			placementChoice.Show()
		}
		face := model.CardFaceDown
		if zone == model.ZoneGraveyard || zone == model.ZoneExile || field {
			face = model.CardFaceUp
		}
		if field {
			orientationChoice.Show()
			orientationChoice.Options = []string{string(model.OrientationRecovered), string(model.OrientationRested)}
			definition := screen.definitions[source.card.CardID]
			if zone == model.ZoneServant && strings.EqualFold(strings.TrimSpace(definition.Type), "Servant") {
				orientationChoice.Options = append(orientationChoice.Options, string(model.OrientationReversed))
			}
			orientationChoice.Enable()
			faceChoice.Enable()
			if reset || orientationChoice.Selected == "" {
				orientationChoice.SetSelected(string(model.OrientationRecovered))
			}
			if zone == model.ZoneCaster {
				faceChoice.Show()
				if reset || faceChoice.Selected == "" {
					if strings.EqualFold(strings.TrimSpace(definition.Type), "Caster") {
						faceChoice.SetSelected(string(model.CardFaceUp))
					} else {
						faceChoice.SetSelected(string(model.CardFaceDown))
					}
				}
				face = model.CardFace(faceChoice.Selected)
			}
			if selected && source.zone == zone && playerChoice.Selected == "Opponent's side" {
				face = source.card.Face
				faceChoice.SetSelected(string(face))
				orientationChoice.SetSelected(string(source.card.Orientation))
				faceChoice.Disable()
				orientationChoice.Disable()
			}
			orientationChoice.Refresh()
		}
		if !selected || zone == "" {
			return
		}
		destination := match.ViewerID
		if playerChoice.Selected == "Opponent's side" {
			for _, player := range match.Players {
				if player.ID != match.ViewerID {
					destination = player.ID
				}
			}
		}
		// Non-field destinations must stay on the card owner's side.
		if !field && source.card.Owner != "" {
			destination = source.card.Owner
		}
		command = model.MoveCardCommand{CardID: source.card.MatchID, DestinationPlayerID: destination, DestinationZone: zone, DestinationFace: face}
		if field {
			command.EntryOrientation = model.CardOrientation(orientationChoice.Selected)
		}
		if zone == model.ZoneDeck {
			command.Placement = model.DeckPlacement(placementChoice.Selected)
		}
		if source.zone == zone && destination == match.ViewerID {
			status.SetText("Choose a different zone or player.")
			return
		}
		definition := screen.definitions[source.card.CardID]
		kind := strings.ToLower(strings.TrimSpace(definition.Type))
		if (zone == model.ZoneCaster && face == model.CardFaceUp && kind != "caster") || (zone == model.ZoneServant && kind != "servant" && kind != "barrier") {
			status.SetText("That card type cannot enter this destination face up.")
			return
		}
		status.SetText(fmt.Sprintf("%s → %s on %s. Non-field destinations stay on the owner's side.", source.zone, zone, playerChoice.Selected))
		confirm.Enable()
	}
	cardChoice.OnChanged = func(label string) {
		if source, found := sources[label]; found {
			screen.preview.showCard(screen.definitions[source.card.CardID])
		}
		update(true)
	}
	zoneChoice.OnChanged = func(string) { update(true) }
	playerChoice.OnChanged = func(string) { update(true) }
	faceChoice.OnChanged = func(string) { update(false) }
	orientationChoice.OnChanged = func(string) { update(false) }
	placementChoice.OnChanged = func(string) { update(false) }
	update(true)
	note := widget.NewLabel("Move a printed card you control between supported zones.")
	note.Wrapping = fyne.TextWrapWord
	limits := widget.NewLabel("Prototype limits: Tokens, cards with Stock, Orbs, face-down Exile, and hidden Deck or Orb cards are not selectable.")
	limits.Wrapping = fyne.TextWrapWord
	limits.Importance = widget.LowImportance
	heading := widget.NewLabelWithStyle("Manual move", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(heading, note, limits, widget.NewSeparator(), cardChoice, zoneChoice, playerChoice, faceChoice, orientationChoice, placementChoice, status, confirm, widget.NewButton("Cancel move", screen.updateManualMoves))
}
