package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

type zoneDropTarget struct {
	object   fyne.CanvasObject
	playerID model.PlayerID
	zone     model.Zone
}

type cardDragSession struct {
	card           simulatorview.CardView
	sourceZone     model.Zone
	sourcePlayerID model.PlayerID
	lastAbs        fyne.Position
	active         bool
}

func (screen *BoardScreen) registerDropTarget(object fyne.CanvasObject, playerID model.PlayerID, zone model.Zone) {
	if screen == nil || object == nil {
		return
	}
	screen.dropTargets = append(screen.dropTargets, zoneDropTarget{
		object:   object,
		playerID: playerID,
		zone:     zone,
	})
}

func (screen *BoardScreen) clearDropTargets() {
	if screen == nil {
		return
	}
	screen.dropTargets = screen.dropTargets[:0]
}

func (screen *BoardScreen) beginCardDrag(
	card simulatorview.CardView,
	sourceZone model.Zone,
	sourcePlayerID model.PlayerID,
) {
	if screen == nil || !cardMovable(card) || screen.actions.MoveCard == nil {
		return
	}
	if screen.match.MatchStatus != model.StatusInProgress {
		return
	}
	screen.drag = cardDragSession{
		card:           card,
		sourceZone:     sourceZone,
		sourcePlayerID: sourcePlayerID,
		active:         true,
	}
}

func (screen *BoardScreen) trackCardDrag(absolute fyne.Position) {
	if screen == nil || !screen.drag.active {
		return
	}
	screen.drag.lastAbs = absolute
}

func (screen *BoardScreen) finishCardDrag() {
	if screen == nil || !screen.drag.active {
		return
	}
	drag := screen.drag
	screen.drag = cardDragSession{}
	target, ok := screen.dropTargetAt(drag.lastAbs)
	if !ok {
		return
	}
	screen.applyDragDrop(drag.card, drag.sourceZone, drag.sourcePlayerID, target, drag.lastAbs)
}

func (screen *BoardScreen) dropTargetAt(absolute fyne.Position) (zoneDropTarget, bool) {
	if screen == nil {
		return zoneDropTarget{}, false
	}
	driver := fyne.CurrentApp().Driver()
	if driver == nil {
		return zoneDropTarget{}, false
	}
	for index := len(screen.dropTargets) - 1; index >= 0; index-- {
		target := screen.dropTargets[index]
		if target.object == nil || !target.object.Visible() {
			continue
		}
		origin := driver.AbsolutePositionForObject(target.object)
		size := target.object.Size()
		if absolute.X >= origin.X && absolute.X < origin.X+size.Width &&
			absolute.Y >= origin.Y && absolute.Y < origin.Y+size.Height {
			return target, true
		}
	}
	return zoneDropTarget{}, false
}

func (screen *BoardScreen) applyDragDrop(
	card simulatorview.CardView,
	sourceZone model.Zone,
	sourcePlayerID model.PlayerID,
	target zoneDropTarget,
	dropAt fyne.Position,
) {
	if screen.actions.MoveCard == nil {
		return
	}
	window := windowForObject(screen.content)
	if sourceZone == target.zone && sourcePlayerID == target.playerID {
		return
	}
	if target.zone == model.ZoneDeck {
		screen.promptDeckDropPlacement(card, sourceZone, target, window, dropAt)
		return
	}
	command, err := buildDefaultMoveCommand(
		card,
		sourceZone,
		target.playerID,
		target.zone,
		screen.definitions[card.CardID],
		"",
	)
	if err != nil {
		if window != nil {
			dialog.ShowError(err, window)
		}
		return
	}
	screen.actions.MoveCard(command, screen.match.Revision)
}

func (screen *BoardScreen) promptDeckDropPlacement(
	card simulatorview.CardView,
	sourceZone model.Zone,
	target zoneDropTarget,
	window fyne.Window,
	dropAt fyne.Position,
) {
	submit := func(placement model.DeckPlacement) {
		command, err := buildDefaultMoveCommand(
			card,
			sourceZone,
			target.playerID,
			target.zone,
			screen.definitions[card.CardID],
			placement,
		)
		if err != nil {
			if window != nil {
				dialog.ShowError(err, window)
			}
			return
		}
		screen.actions.MoveCard(command, screen.match.Revision)
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(target.object)
	if canvas == nil {
		submit(model.DeckPlacementTop)
		return
	}
	top := fyne.NewMenuItem("Put on top of deck", func() { submit(model.DeckPlacementTop) })
	bottom := fyne.NewMenuItem("Put on bottom of deck", func() { submit(model.DeckPlacementBottom) })
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", top, bottom), canvas, dropAt)
}

func (screen *BoardScreen) bindCardDrag(
	tile *CardTile,
	sourceZone model.Zone,
	sourcePlayerID model.PlayerID,
) {
	if screen == nil || tile == nil || screen.actions.MoveCard == nil {
		return
	}
	if !cardMovable(tile.View) || sourceZone == model.ZoneOrbs {
		return
	}
	card := tile.View
	tile.OnDragBegin = func() {
		screen.beginCardDrag(card, sourceZone, sourcePlayerID)
	}
	tile.OnDragUpdate = func(absolute fyne.Position) {
		screen.trackCardDrag(absolute)
	}
	tile.OnDragEnd = func() {
		screen.finishCardDrag()
	}
}
