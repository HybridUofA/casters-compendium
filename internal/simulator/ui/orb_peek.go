package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func (board *playerBoardController) beginPeekOrbPrompt() {
	if board == nil || board.actions.PeekOrb == nil {
		return
	}
	count := len(board.player.Orbs)
	if count == 0 {
		return
	}
	promptOrbIndex(
		windowForObject(board.orbs),
		"Peek which enemy Orb?",
		count,
		func(orbIndex int) {
			revision := board.currentRevision()
			board.actions.PeekOrb(board.player.ID, orbIndex, revision, func(card simulatorview.CardView, err error) {
				if err != nil {
					window := windowForObject(board.orbs)
					if window != nil {
						dialog.ShowError(err, window)
					}
					return
				}
				if board.host != nil {
					board.host.showOrbPeekDialog(card, "Orb peek")
				}
			})
		},
	)
}

func (board *playerBoardController) beginRevealOrbPrompt() {
	if board == nil || board.actions.RevealOrb == nil {
		return
	}
	count := len(board.player.Orbs)
	if count == 0 {
		return
	}
	promptOrbIndex(
		windowForObject(board.orbs),
		"Reveal which of your Orbs to the opponent?",
		count,
		func(orbIndex int) {
			board.actions.RevealOrb(orbIndex, board.currentRevision())
		},
	)
}

func (screen *BoardScreen) showOrbPeekDialog(card simulatorview.CardView, title string) {
	if screen == nil {
		return
	}
	window := windowForObject(screen.content)
	if window == nil {
		return
	}
	definition := screen.definitions[card.CardID]
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		name = string(card.CardID)
	}
	screen.preview.showCard(definition)
	label := widget.NewLabel(fmt.Sprintf("Orb: %s", name))
	label.Wrapping = fyne.TextWrapWord
	content := container.NewVBox(
		widget.NewLabel("You will keep seeing this Orb's identity on the board."),
		label,
	)
	showScaledCustom(title, "OK", content, window, 0.45, 0.35, 480, 240, 900, 520)
}

var promptOrbIndex = promptOrbIndexDialog

func promptOrbIndexDialog(window fyne.Window, prompt string, orbCount int, onConfirm func(int)) {
	if window == nil || onConfirm == nil || orbCount < 1 {
		return
	}
	labels := make([]string, 0, orbCount)
	for index := 0; index < orbCount; index++ {
		labels = append(labels, fmt.Sprintf("Orb %d", index+1))
	}
	selectWidget := widget.NewSelect(labels, nil)
	selectWidget.PlaceHolder = "Choose Orb"
	status := widget.NewLabel(prompt)
	status.Wrapping = fyne.TextWrapWord
	var popup dialog.Dialog
	confirm := widget.NewButton("Confirm", func() {
		if selectWidget.Selected == "" {
			dialog.ShowError(fmt.Errorf("choose an Orb"), window)
			return
		}
		parts := strings.Fields(selectWidget.Selected)
		if len(parts) != 2 {
			dialog.ShowError(fmt.Errorf("invalid Orb selection"), window)
			return
		}
		number, err := strconv.Atoi(parts[1])
		if err != nil || number < 1 || number > orbCount {
			dialog.ShowError(fmt.Errorf("invalid Orb selection"), window)
			return
		}
		onConfirm(number - 1)
		if popup != nil {
			popup.Hide()
		}
	})
	confirm.Disable()
	selectWidget.OnChanged = func(string) {
		if selectWidget.Selected == "" {
			confirm.Disable()
			return
		}
		confirm.Enable()
	}
	content := container.NewVBox(status, selectWidget, confirm)
	popup = dialog.NewCustom("Orb", "Cancel", content, window)
	resizeDialogFraction(popup, window, 0.4, 0.35, 420, 240, 800, 520)
	popup.Show()
}
