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

func (screen *BoardScreen) showHildePeekDialog(
	ownerID model.PlayerID,
	cards []simulatorview.CardView,
) {
	if screen == nil || len(cards) == 0 {
		return
	}
	window := windowForObject(screen.content)
	if window == nil {
		return
	}
	card := cards[0]
	definition := screen.definitions[card.CardID]
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		name = string(card.CardID)
	}
	screen.preview.showCard(definition)
	label := widget.NewLabel(fmt.Sprintf("Top card: %s", name))
	label.Wrapping = fyne.TextWrapWord
	content := container.NewVBox(
		widget.NewLabel("Look at the top card of the enemy deck."),
		label,
	)
	popup := dialog.NewCustom("Deck peek", "Leave on top", content, window)
	if screen.actions.MoveDeckTopToBottom != nil {
		popup.SetButtons([]fyne.CanvasObject{
			widget.NewButton("Leave on top", func() { popup.Hide() }),
			widget.NewButton("Put on bottom", func() {
				screen.actions.MoveDeckTopToBottom(ownerID, screen.match.Revision)
				popup.Hide()
			}),
		})
	}
	resizeDialogFraction(popup, window, 0.45, 0.35, 480, 260, 900, 520)
	popup.Show()
}

func (screen *BoardScreen) showSageDigDialog(cards []simulatorview.CardView) {
	if screen == nil || len(cards) == 0 || screen.actions.ResolveDeckDig == nil {
		return
	}
	window := windowForObject(screen.content)
	if window == nil {
		return
	}

	type digEntry struct {
		label string
		card  simulatorview.CardView
	}
	entries := make([]digEntry, 0, len(cards))
	labels := make([]string, 0, len(cards))
	for index, card := range cards {
		definition := screen.definitions[card.CardID]
		name := strings.TrimSpace(definition.Name)
		if name == "" {
			name = string(card.CardID)
		}
		label := fmt.Sprintf("%d. %s", index+1, name)
		entries = append(entries, digEntry{label: label, card: card})
		labels = append(labels, label)
	}

	keepSelect := widget.NewSelect(labels, nil)
	keepSelect.PlaceHolder = "Card to keep in hand"
	status := widget.NewLabel("Choose one card for your hand. The rest go to the bottom in list order.")
	status.Wrapping = fyne.TextWrapWord
	bottomHint := widget.NewLabel("Bottom order (excluding keep): original peek order, keep removed.")
	bottomHint.Wrapping = fyne.TextWrapWord

	var popup dialog.Dialog
	confirm := widget.NewButton("Resolve dig", func() {
		var keep model.MatchCardID
		for _, entry := range entries {
			if entry.label == keepSelect.Selected {
				keep = entry.card.MatchID
				break
			}
		}
		if keep == "" {
			dialog.ShowError(fmt.Errorf("choose a card to keep"), window)
			return
		}
		bottom := make([]model.MatchCardID, 0, len(cards)-1)
		for _, card := range cards {
			if card.MatchID == keep {
				continue
			}
			bottom = append(bottom, card.MatchID)
		}
		screen.actions.ResolveDeckDig(keep, bottom, screen.match.Revision)
		if popup != nil {
			popup.Hide()
		}
	})
	confirm.Disable()
	keepSelect.OnChanged = func(label string) {
		confirm.Enable()
		for _, entry := range entries {
			if entry.label == label {
				screen.preview.showCard(screen.definitions[entry.card.CardID])
				return
			}
		}
	}

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Look at the top %d cards.", len(cards))),
		status,
		keepSelect,
		bottomHint,
		confirm,
	)
	popup = dialog.NewCustom("Deck dig", "Cancel", content, window)
	resizeDialogFraction(popup, window, 0.55, 0.45, 560, 320, 1100, 700)
	popup.Show()
}

func (board *playerBoardController) beginLookAtEnemyTop() {
	if board == nil || board.host == nil || board.actions.PeekDeckTops == nil {
		return
	}
	ownerID := board.player.ID
	board.actions.PeekDeckTops(ownerID, 1, func(cards []simulatorview.CardView, err error) {
		if err != nil {
			window := windowForObject(board.deck)
			if window != nil {
				dialog.ShowError(err, window)
			}
			return
		}
		board.host.showHildePeekDialog(ownerID, cards)
	})
}

func (board *playerBoardController) beginSageDig() {
	if board == nil || board.host == nil || board.actions.PeekDeckTops == nil {
		return
	}
	ownerID := board.player.ID
	board.actions.PeekDeckTops(ownerID, 3, func(cards []simulatorview.CardView, err error) {
		if err != nil {
			window := windowForObject(board.deck)
			if window != nil {
				dialog.ShowError(err, window)
			}
			return
		}
		board.host.showSageDigDialog(cards)
	})
}
