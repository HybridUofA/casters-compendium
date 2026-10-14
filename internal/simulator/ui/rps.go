package ui

import (
	_ "image/jpeg"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	sourceRPS "github.com/HybridUofA/casters-compendium/internal/sources/rps"
	"github.com/HybridUofA/casters-compendium/internal/simulator/effects"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

var rpsOptionResources = map[string]fyne.Resource{
	"rock":     fyne.NewStaticResource("rps-rock.jpg", sourceRPS.RockJPG),
	"paper":    fyne.NewStaticResource("rps-paper.jpg", sourceRPS.PaperJPG),
	"scissors": fyne.NewStaticResource("rps-scissors.jpg", sourceRPS.ScissorsJPG),
}

func pendingChoiceIsRPS(choice model.PendingChoice) bool {
	return choice.Kind == effects.PendingRPSChoice
}

func (screen *BoardScreen) renderRPSPendingChoicePanel() {
	if screen == nil || screen.attackPanel == nil {
		return
	}
	match := screen.match
	choice := match.PendingChoice
	prompt := strings.TrimSpace(choice.Prompt)
	if prompt == "" {
		prompt = "Choose Rock, Paper, or Scissors"
	}
	hint := widget.NewLabel(prompt)
	hint.Wrapping = fyne.TextWrapWord

	buttons := make([]fyne.CanvasObject, 0, len(choice.Options))
	for _, option := range choice.Options {
		option := option
		resource := rpsOptionResources[option.ID]
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = option.ID
		}
		var face fyne.CanvasObject
		if resource != nil {
			img := canvas.NewImageFromResource(resource)
			img.FillMode = canvas.ImageFillContain
			img.SetMinSize(fyne.NewSize(96, 144))
			face = img
		} else {
			face = widget.NewLabel(label)
		}
		tap := widget.NewButton(label, func() {
			if screen.actions.ResolvePendingChoice == nil {
				return
			}
			screen.actions.ResolvePendingChoice(option.ID, nil, match.Revision)
		})
		buttons = append(buttons, container.NewVBox(face, tap))
	}
	screen.attackPanel.Objects = []fyne.CanvasObject{
		hint,
		container.NewHBox(buttons...),
	}
	refreshContainerStructure(screen.attackPanel)
}
