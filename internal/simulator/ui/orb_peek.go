package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func (board *playerBoardController) beginPeekOrbPrompt() {
	if board == nil || board.actions.PeekOrb == nil || board.host == nil {
		return
	}
	if len(board.player.Orbs) == 0 {
		return
	}
	board.host.beginOrbPick(orbPickPeek)
}

func (board *playerBoardController) beginRevealOrbPrompt() {
	if board == nil || board.actions.RevealOrb == nil || board.host == nil {
		return
	}
	if len(board.player.Orbs) == 0 {
		return
	}
	board.host.beginOrbPick(orbPickReveal)
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
