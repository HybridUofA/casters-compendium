package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// Orb zones always reserve this many prize-style slots so remaining Orbs keep
// their positions when others are corrupted or banished.
const orbSlotCount = 7

// reconcileOrbSlotIDs keeps MatchIDs in stable prize-card slots across updates.
// Removed Orbs leave empty holes; newly added Orbs fill the first free slots.
func reconcileOrbSlotIDs(
	previous []model.MatchCardID,
	current []simulatorview.CardView,
) []model.MatchCardID {
	slots := make([]model.MatchCardID, orbSlotCount)
	if len(previous) == orbSlotCount {
		copy(slots, previous)
	}

	present := make(map[model.MatchCardID]struct{}, len(current))
	for _, card := range current {
		if card.MatchID == "" {
			continue
		}
		present[card.MatchID] = struct{}{}
	}

	for index, id := range slots {
		if id == "" {
			continue
		}
		if _, ok := present[id]; !ok {
			slots[index] = ""
			continue
		}
		delete(present, id)
	}

	for _, card := range current {
		if card.MatchID == "" {
			continue
		}
		if _, needsSlot := present[card.MatchID]; !needsSlot {
			continue
		}
		placed := false
		for index := range slots {
			if slots[index] != "" {
				continue
			}
			slots[index] = card.MatchID
			delete(present, card.MatchID)
			placed = true
			break
		}
		if !placed {
			break
		}
	}
	return slots
}

// orbViewsForSlots expands dense Orb projections into fixed prize-style slots.
//
// Orbs with empty MatchID (pre-1.2.1 hosts omitted identifiers the viewer did
// not already know) still occupy free slots as anonymous face-down cards so
// the client shows card backs instead of empty prize outlines.
func orbViewsForSlots(
	slots []model.MatchCardID,
	current []simulatorview.CardView,
) []simulatorview.CardView {
	byID := make(map[model.MatchCardID]simulatorview.CardView, len(current))
	anonymous := make([]simulatorview.CardView, 0)
	for _, card := range current {
		if card.MatchID == "" {
			anonymous = append(anonymous, card)
			continue
		}
		byID[card.MatchID] = card
	}
	out := make([]simulatorview.CardView, orbSlotCount)
	for index, id := range slots {
		if id == "" {
			continue
		}
		out[index] = byID[id]
	}
	anonIndex := 0
	for index := range out {
		if orbSlotOccupied(out[index]) {
			continue
		}
		if anonIndex >= len(anonymous) {
			break
		}
		card := anonymous[anonIndex]
		anonIndex++
		if card.Face == "" {
			card.Face = model.CardFaceDown
		}
		out[index] = card
	}
	return out
}

func orbSlotOccupied(card simulatorview.CardView) bool {
	return card.MatchID != "" || card.Face == model.CardFaceDown
}

func denseOrbIndex(orbs []simulatorview.CardView, matchID model.MatchCardID) int {
	if matchID == "" {
		return -1
	}
	for index, card := range orbs {
		if card.MatchID == matchID {
			return index
		}
	}
	return -1
}

// orbSlotPlaceholder is an empty prize-card outline for a vacant Orb slot.
type orbSlotPlaceholder struct {
	widget.BaseWidget
	size  fyne.Size
	frame *canvas.Rectangle
}

func newOrbSlotPlaceholder(size fyne.Size) *orbSlotPlaceholder {
	placeholder := &orbSlotPlaceholder{
		size:  size,
		frame: canvas.NewRectangle(color.NRGBA{R: 45, G: 73, B: 78, A: 55}),
	}
	placeholder.frame.StrokeColor = color.NRGBA{R: 104, G: 151, B: 155, A: 120}
	placeholder.frame.StrokeWidth = 2
	placeholder.frame.CornerRadius = 6
	placeholder.ExtendBaseWidget(placeholder)
	return placeholder
}

func (placeholder *orbSlotPlaceholder) CreateRenderer() fyne.WidgetRenderer {
	return &orbSlotPlaceholderRenderer{placeholder: placeholder}
}

func (placeholder *orbSlotPlaceholder) MinSize() fyne.Size {
	return placeholder.size
}

type orbSlotPlaceholderRenderer struct {
	placeholder *orbSlotPlaceholder
}

func (renderer *orbSlotPlaceholderRenderer) Layout(size fyne.Size) {
	renderer.placeholder.frame.Resize(size)
	renderer.placeholder.frame.Move(fyne.NewPos(0, 0))
}

func (renderer *orbSlotPlaceholderRenderer) MinSize() fyne.Size {
	return renderer.placeholder.MinSize()
}

func (renderer *orbSlotPlaceholderRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{renderer.placeholder.frame}
}

func (renderer *orbSlotPlaceholderRenderer) Refresh() {
	renderer.Layout(renderer.placeholder.Size())
	renderer.placeholder.frame.Refresh()
}

func (renderer *orbSlotPlaceholderRenderer) Destroy() {}
