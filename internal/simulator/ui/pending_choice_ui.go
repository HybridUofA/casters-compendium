package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// installPendingChoiceTap prepends pending-choice targeting to a card's primary
// tap so players can click hand/field cards instead of using raw Match IDs.
func (screen *BoardScreen) installPendingChoiceTap(tile *CardTile) {
	if screen == nil || tile == nil || tile.View.MatchID == "" {
		return
	}
	id := tile.View.MatchID
	previous := tile.OnActivate
	tile.OnActivate = func() {
		if screen.tryPendingChoiceCardTap(id) {
			return
		}
		if previous != nil {
			previous()
		}
	}
}

func (screen *BoardScreen) installPendingChoiceTaps(root fyne.CanvasObject) {
	if screen == nil || root == nil {
		return
	}
	for _, tile := range collectCardTiles(root) {
		screen.installPendingChoiceTap(tile)
	}
}

func (screen *BoardScreen) clearPendingChoiceSelection() {
	if screen == nil {
		return
	}
	screen.pendingChoiceSelected = nil
	screen.pendingChoiceSelectKey = ""
}

func (screen *BoardScreen) syncPendingChoiceSelectionKey() {
	if screen == nil {
		return
	}
	choice := screen.match.PendingChoice
	key := fmt.Sprintf("%s|%s|%d|%d",
		choice.PlayerID, choice.Kind, screen.match.Revision, len(choice.SelectableCardIDs))
	if screen.pendingChoiceSelectKey != key {
		screen.pendingChoiceSelected = nil
		screen.pendingChoiceSelectKey = key
	}
}

func (screen *BoardScreen) tryPendingChoiceCardTap(cardID model.MatchCardID) bool {
	if screen == nil || cardID == "" || !canViewerDecidePendingChoice(screen.match) {
		return false
	}
	choice := screen.match.PendingChoice
	if !matchIDInList(choice.SelectableCardIDs, cardID) {
		return false
	}

	screen.syncPendingChoiceSelectionKey()

	// Option-based prompts: click selects the target, then an option button submits.
	if len(choice.Options) > 0 {
		screen.pendingChoiceSelected = []model.MatchCardID{cardID}
		screen.refreshPendingChoiceHighlights()
		screen.renderPendingChoicePanel()
		return true
	}

	want := pendingChoiceWantCount(choice)
	if want <= 1 {
		if screen.actions.ResolvePendingChoice != nil {
			screen.pendingChoiceSelected = nil
			screen.actions.ResolvePendingChoice("", []model.MatchCardID{cardID}, screen.match.Revision)
		}
		return true
	}

	screen.pendingChoiceSelected = toggleMatchID(screen.pendingChoiceSelected, cardID)
	screen.refreshPendingChoiceHighlights()
	if len(screen.pendingChoiceSelected) >= want && screen.actions.ResolvePendingChoice != nil {
		selected := append([]model.MatchCardID(nil), screen.pendingChoiceSelected...)
		screen.pendingChoiceSelected = nil
		screen.actions.ResolvePendingChoice("", selected, screen.match.Revision)
		return true
	}
	// Rebuild the strip so Confirm reflects the current multi-select.
	screen.renderPendingChoicePanel()
	return true
}

func pendingChoiceWantCount(choice model.PendingChoice) int {
	switch choice.Kind {
	case "fx:discard_two_from_hand":
		return 2
	default:
		if len(choice.SelectableCardIDs) == 0 {
			return 0
		}
		return 1
	}
}

func (screen *BoardScreen) refreshPendingChoiceHighlights() {
	if screen == nil {
		return
	}
	active := canViewerDecidePendingChoice(screen.match) &&
		len(screen.match.PendingChoice.SelectableCardIDs) > 0
	selectable := matchIDSet(screen.match.PendingChoice.SelectableCardIDs)
	selected := matchIDSet(screen.pendingChoiceSelected)
	for _, board := range screen.playerBoards {
		if board == nil {
			continue
		}
		for _, holder := range []*fyne.Container{
			board.hand, board.servants, board.barriers, board.caster,
			board.graveyard, board.exile, board.orbs,
		} {
			for _, tile := range collectCardTiles(holder) {
				id := tile.View.MatchID
				_, isSelectable := selectable[id]
				if !active || id == "" || !isSelectable {
					if active {
						tile.SetTargetHoverEnabled(false)
					}
					continue
				}
				_, isSelected := selected[id]
				tile.SetTargetHoverEnabled(true)
				tile.SetSelected(isSelected)
			}
		}
	}
}

func (screen *BoardScreen) pendingChoiceCardName(id model.MatchCardID) string {
	if screen == nil || id == "" {
		return "Unknown card"
	}
	view, definition := screen.lookupProjectedCard(id)
	if name := strings.TrimSpace(definition.Name); name != "" {
		return name
	}
	if view.CardID != "" {
		return string(view.CardID)
	}
	return "Unknown card"
}

func (screen *BoardScreen) lookupProjectedCard(
	id model.MatchCardID,
) (simulatorview.CardView, cards.Card) {
	if screen == nil || id == "" {
		return simulatorview.CardView{}, cards.Card{}
	}
	for _, player := range screen.match.Players {
		for _, zone := range [][]simulatorview.CardView{
			player.Hand,
			player.ServantZone,
			player.CasterZone,
			player.Graveyard,
			player.Exile,
			player.Orbs,
			player.Deck,
		} {
			for _, card := range zone {
				if card.MatchID == id {
					return card, screen.definitions[card.CardID]
				}
				for _, stock := range card.Stock {
					if stock.MatchID == id {
						return stock, screen.definitions[stock.CardID]
					}
				}
			}
		}
	}
	return simulatorview.CardView{}, cards.Card{}
}

// pendingChoiceSelectEntries builds unique dropdown labels mapped to Match IDs.
func (screen *BoardScreen) pendingChoiceSelectEntries(
	ids []model.MatchCardID,
) (labels []string, byLabel map[string]model.MatchCardID) {
	byLabel = make(map[string]model.MatchCardID, len(ids))
	labels = make([]string, 0, len(ids))
	usedNames := make(map[string]int, len(ids))
	for _, id := range ids {
		base := screen.pendingChoiceCardName(id)
		count := usedNames[base]
		usedNames[base] = count + 1
		label := base
		if count > 0 {
			label = fmt.Sprintf("%s (%d)", base, count+1)
		}
		// Keep labels unique even across identical numbered names.
		for {
			if _, exists := byLabel[label]; !exists {
				break
			}
			count++
			label = fmt.Sprintf("%s (%d)", base, count+1)
		}
		byLabel[label] = id
		labels = append(labels, label)
	}
	return labels, byLabel
}

func matchIDInList(ids []model.MatchCardID, want model.MatchCardID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func matchIDSet(ids []model.MatchCardID) map[model.MatchCardID]struct{} {
	out := make(map[model.MatchCardID]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

func toggleMatchID(ids []model.MatchCardID, id model.MatchCardID) []model.MatchCardID {
	for index, existing := range ids {
		if existing != id {
			continue
		}
		return append(ids[:index], ids[index+1:]...)
	}
	return append(ids, id)
}

// selectedPendingChoiceID returns the first pending selection, if any.
func (screen *BoardScreen) selectedPendingChoiceID() model.MatchCardID {
	if screen == nil || len(screen.pendingChoiceSelected) == 0 {
		return ""
	}
	return screen.pendingChoiceSelected[0]
}

func syncSelectToMatchID(selectWidget *widget.Select, byLabel map[string]model.MatchCardID, want model.MatchCardID) {
	if selectWidget == nil || want == "" {
		return
	}
	for label, id := range byLabel {
		if id == want {
			selectWidget.SetSelected(label)
			return
		}
	}
}
