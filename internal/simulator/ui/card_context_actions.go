package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// handCallMenuItems builds Call / Level Up / Move items for a hand card in Call.
func handCallMenuItems(
	anchor fyne.CanvasObject,
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	definitions cardLookup,
	actions BoardActions,
	currentRevision func() model.Revision,
) []*fyne.MenuItem {
	if card.MatchID == "" {
		return nil
	}
	items := make([]*fyne.MenuItem, 0, 8)
	revision := model.Revision(0)
	if currentRevision != nil {
		revision = currentRevision()
	}
	if actions.CallFaceUpLevelOne != nil && isFaceUpLevelOneCallDefinition(definition) {
		items = append(items, fyne.NewMenuItem("Call Face Up", func() {
			actions.CallFaceUpLevelOne(card.MatchID, revision)
		}))
	}
	// Face-down Call may place any printed hand card as a Level 1 Caster.
	if actions.CallFaceDownLevelOne != nil {
		items = append(items, fyne.NewMenuItem("Call Face Down", func() {
			actions.CallFaceDownLevelOne(card.MatchID, revision)
		}))
	}
	if actions.LevelUpCaster != nil {
		for _, candidate := range eligibleLevelUpTargets(definition, player.CasterZone, definitions) {
			candidate := candidate
			items = append(items, fyne.NewMenuItem("Level Up onto "+candidate.label, func() {
				actions.LevelUpCaster(card.MatchID, candidate.matchID, revision)
			}))
		}
	}
	if actions.MoveCard != nil {
		items = append(items, fyne.NewMenuItem("Move…", func() {
			showManualMoveDialog(anchor, card, currentRevision)
		}))
	}
	return items
}

// showHandCallContextMenu offers Call / Level Up / Move for a hand card in Call.
func showHandCallContextMenu(
	anchor fyne.CanvasObject,
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	definitions cardLookup,
	actions BoardActions,
	currentRevision func() model.Revision,
	event *fyne.PointEvent,
) {
	showCardContextMenu(anchor, event, handCallMenuItems(
		anchor, card, definition, player, definitions, actions, currentRevision,
	))
}

// handCastMenuItems builds Cast / Move items for a hand spell in Main.
func handCastMenuItems(
	anchor fyne.CanvasObject,
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	definitions cardLookup,
	actions BoardActions,
	currentRevision func() model.Revision,
) []*fyne.MenuItem {
	if card.MatchID == "" {
		return nil
	}
	items := make([]*fyne.MenuItem, 0, 4)
	if canCastDefinition(definition, actions) {
		items = append(items, fyne.NewMenuItem("Cast…", func() {
			showCastDialog(anchor, card, definition, player, definitions, actions, currentRevision)
		}))
	}
	if actions.MoveCard != nil {
		items = append(items, fyne.NewMenuItem("Move…", func() {
			showManualMoveDialog(anchor, card, currentRevision)
		}))
	}
	return items
}

// showHandCastContextMenu offers Cast / Move for a hand spell in Main.
func showHandCastContextMenu(
	anchor fyne.CanvasObject,
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	definitions cardLookup,
	actions BoardActions,
	currentRevision func() model.Revision,
	event *fyne.PointEvent,
) {
	showCardContextMenu(anchor, event, handCastMenuItems(
		anchor, card, definition, player, definitions, actions, currentRevision,
	))
}

func menuItemByLabel(items []*fyne.MenuItem, label string) *fyne.MenuItem {
	for _, item := range items {
		if item != nil && item.Label == label {
			return item
		}
	}
	return nil
}

func canCastDefinition(definition cards.Card, actions BoardActions) bool {
	switch strings.ToLower(strings.TrimSpace(definition.Type)) {
	case "servant":
		return actions.CastServant != nil || actions.CastServantWithPlan != nil
	case "conjure":
		return actions.CastConjure != nil || actions.CastConjureWithPlan != nil
	case "barrier":
		return actions.CastBarrier != nil || actions.CastBarrierWithPlan != nil
	default:
		return false
	}
}

func strconvAtoiTrimmed(raw string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(raw))
}

func showCardContextMenu(anchor fyne.CanvasObject, event *fyne.PointEvent, items []*fyne.MenuItem) {
	if len(items) == 0 || event == nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(anchor)
	if canvas == nil {
		return
	}
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), canvas, event.AbsolutePosition)
}

func showCastDialog(
	anchor fyne.CanvasObject,
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	definitions cardLookup,
	actions BoardActions,
	currentRevision func() model.Revision,
) {
	window := windowForObject(anchor)
	if window == nil {
		return
	}
	panel := newSuggestedCastPanel(card, definition, player, definitions, actions, currentRevision)
	if panel == nil {
		panel = newManualCastPanel(card, definition, player, actions, currentRevision)
	}
	if panel == nil {
		return
	}
	title := fmt.Sprintf("Cast %s", strings.TrimSpace(definition.Name))
	if title == "Cast " {
		title = "Cast card"
	}
	showScaledCustom(title, "Close", container.NewVScroll(panel), window, 0.45, 0.55, 320, 280, 560, 520)
}

func showManualMoveDialog(anchor fyne.CanvasObject, card simulatorview.CardView, currentRevision func() model.Revision) {
	window := windowForObject(anchor)
	if window == nil {
		return
	}
	screen := boardScreenForObject(anchor)
	if screen == nil || screen.actions.MoveCard == nil {
		return
	}
	panel := screen.newManualMovePanelForCard(card)
	showScaledCustom("Move card", "Close", container.NewVScroll(panel), window, 0.4, 0.55, 300, 280, 480, 520)
}

func boardScreenForObject(object fyne.CanvasObject) *BoardScreen {
	// Walk is unavailable; tests and live play use the window's board content.
	// Controllers stash themselves via content identity — use AllWindows scan of
	// known BoardScreen Content roots when needed. For now, prefer the active
	// app window's last board through a package-level weak map set on Update.
	if object == nil {
		return nil
	}
	if screen, ok := boardByContent[object]; ok {
		return screen
	}
	// Tile may not be the content root; search registered screens.
	for root, screen := range boardByContent {
		if root == nil || screen == nil {
			continue
		}
		if containsCanvasObject(root, object) {
			return screen
		}
	}
	return nil
}

var boardByContent = map[fyne.CanvasObject]*BoardScreen{}

func registerBoardScreen(screen *BoardScreen) {
	if screen == nil || screen.content == nil {
		return
	}
	boardByContent[screen.content] = screen
}

func containsCanvasObject(root, needle fyne.CanvasObject) bool {
	if root == nil || needle == nil {
		return false
	}
	if root == needle {
		return true
	}
	switch typed := root.(type) {
	case *fyne.Container:
		for _, child := range typed.Objects {
			if containsCanvasObject(child, needle) {
				return true
			}
		}
	case fyne.Widget:
		if renderer := typed.CreateRenderer(); renderer != nil {
			for _, child := range renderer.Objects() {
				if containsCanvasObject(child, needle) {
					return true
				}
			}
		}
	}
	return false
}

// newManualCastPanel is the pool-payment fallback when suggested sources are unavailable.
func newManualCastPanel(
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	actions BoardActions,
	currentRevision func() model.Revision,
) fyne.CanvasObject {
	kind := strings.ToLower(strings.TrimSpace(definition.Type))
	payment := model.AetherPayment{}
	orientation := widget.NewSelect(
		[]string{string(model.OrientationRecovered), string(model.OrientationReversed)},
		nil,
	)
	orientation.SetSelected(string(model.OrientationRecovered))
	if kind != "servant" {
		orientation.Hide()
	}
	castButton := widget.NewButton("Cast", nil)
	castButton.Disable()
	var updateAction func()
	type paymentSource struct {
		name      string
		available int
		set       func(*model.AetherPayment, int)
	}
	sources := []paymentSource{
		{name: "Aes", available: player.Aether.Aes, set: func(value *model.AetherPayment, amount int) { value.Aes = amount }},
		{name: "Aqua", available: player.Aether.Aqua, set: func(value *model.AetherPayment, amount int) { value.Aqua = amount }},
		{name: "Ignus", available: player.Aether.Ignus, set: func(value *model.AetherPayment, amount int) { value.Ignus = amount }},
		{name: "Luna", available: player.Aether.Luna, set: func(value *model.AetherPayment, amount int) { value.Luna = amount }},
		{name: "Silva", available: player.Aether.Silva, set: func(value *model.AetherPayment, amount int) { value.Silva = amount }},
		{name: "Solis", available: player.Aether.Solis, set: func(value *model.AetherPayment, amount int) { value.Solis = amount }},
		{name: "Terra", available: player.Aether.Terra, set: func(value *model.AetherPayment, amount int) { value.Terra = amount }},
		{name: "Void", available: player.Aether.Void, set: func(value *model.AetherPayment, amount int) { value.Void = amount }},
		{name: "Non-elemental", available: player.Aether.NonElemental, set: func(value *model.AetherPayment, amount int) { value.NonElemental = amount }},
	}
	paymentControls := make([]fyne.CanvasObject, 0, len(sources))
	for _, source := range sources {
		if source.available <= 0 {
			continue
		}
		source := source
		options := make([]string, source.available+1)
		for amount := range options {
			options[amount] = fmt.Sprintf("%d", amount)
		}
		selector := widget.NewSelect(options, func(selected string) {
			amount, err := strconvAtoiTrimmed(selected)
			if err != nil {
				return
			}
			source.set(&payment, amount)
			if updateAction != nil {
				updateAction()
			}
		})
		selector.SetSelected("0")
		paymentControls = append(paymentControls, container.NewVBox(widget.NewLabel(source.name), selector))
	}
	updateAction = func() {
		cost, err := strconvAtoiTrimmed(definition.CostLevel)
		if err != nil || cost < 0 || !paymentMatchesCastDefinition(payment, cost, definition.Element) {
			castButton.Disable()
			return
		}
		castButton.Enable()
	}
	orientation.OnChanged = func(string) { updateAction() }
	castButton.OnTapped = func() {
		if castButton.Disabled() {
			return
		}
		revision := model.Revision(0)
		if currentRevision != nil {
			revision = currentRevision()
		}
		switch kind {
		case "servant":
			if actions.CastServant != nil {
				actions.CastServant(card.MatchID, payment, model.CardOrientation(orientation.Selected), revision)
			}
		case "conjure":
			if actions.CastConjure != nil {
				actions.CastConjure(card.MatchID, payment, revision)
			}
		case "barrier":
			if actions.CastBarrier != nil {
				actions.CastBarrier(card.MatchID, payment, revision)
			}
		}
	}
	updateAction()
	objects := []fyne.CanvasObject{
		widget.NewLabel("Allocate Aether payment, then Cast."),
	}
	objects = append(objects, paymentControls...)
	if len(paymentControls) == 0 {
		objects = append(objects, widget.NewLabel("No Aether in pool (0-cost only)."))
	}
	objects = append(objects, orientation, castButton)
	return container.NewVBox(objects...)
}
