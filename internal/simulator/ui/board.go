// Package ui presents simulator state without owning or enforcing game rules.
package ui

import (
	"fmt"
	"image/color"
	"reflect"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const (
	previewPanelWidth float32 = 290
	boardMinHeight    float32 = 480
	sideZoneWidth     float32 = 200
	handZoneHeight    float32 = 145
	casterZoneHeight  float32 = 145
)

var (
	boardBackground = color.NRGBA{R: 29, G: 50, B: 55, A: 255}
	zoneBackground  = color.NRGBA{R: 45, G: 73, B: 78, A: 255}
	zoneBorder      = color.NRGBA{R: 104, G: 151, B: 155, A: 255}
	boardForeground = color.NRGBA{R: 240, G: 250, B: 250, A: 255}
)

type previewState struct {
	title       *widget.Label
	description *widget.Label
	image       *fyne.Container
	imageSizer  *canvas.Rectangle
}

type cardLookup map[model.CardID]cards.Card

type aetherActionKind uint8

const (
	aetherActionNone aetherActionKind = iota
	aetherActionFaceDownCaster
	aetherActionFaceUpCaster
	aetherActionToken
)

// BoardActions translates presentation choices into session-owned commands.
type BoardActions struct {
	SubmitOpeningHand          func([]model.MatchCardID, model.Revision)
	CallFaceDownLevelOne       func(model.MatchCardID, model.Revision)
	CallFaceUpLevelOne         func(model.MatchCardID, model.Revision)
	LevelUpCaster              func(model.MatchCardID, model.MatchCardID, model.Revision)
	GenerateNonElementalAether func(model.MatchCardID, model.Revision)
	GenerateCasterAether       func(model.MatchCardID, model.Revision)
	UseCasterToken             func(model.MatchCardID, model.Revision)
	CompleteCurrentPhase       func(model.Revision)
	BackLabel                  string
}

// BoardScreen owns a persistent simulator widget tree. Update changes public
// match metadata in place and rebuilds card zones only when projected player
// data changes.
type BoardScreen struct {
	content      fyne.CanvasObject
	status       *canvas.Text
	phaseButtons map[model.Phase]*widget.Button
	boards       *fyne.Container
	match        simulatorview.MatchView
	definitions  cardLookup
	preview      previewState
	actions      BoardActions
}

// NewBoardScreen renders one viewer-safe match projection. It deliberately
// accepts MatchView rather than MatchState so concealed identities cannot be
// recovered by presentation callbacks or future network clients.
func NewBoardScreen(
	match simulatorview.MatchView,
	cardDefinitions []cards.Card,
	actions BoardActions,
	onBack func(),
) fyne.CanvasObject {
	return NewBoardController(match, cardDefinitions, actions, onBack).Content()
}

// NewBoardController constructs a board that can receive later viewer-safe
// projections without replacing the containing window's complete content.
func NewBoardController(
	match simulatorview.MatchView,
	cardDefinitions []cards.Card,
	actions BoardActions,
	onBack func(),
) *BoardScreen {
	screen := &BoardScreen{
		match:        match,
		definitions:  newCardLookup(cardDefinitions),
		preview:      newPreviewPanel(),
		actions:      actions,
		phaseButtons: make(map[model.Phase]*widget.Button, 6),
	}
	screen.boards = container.NewGridWithRows(2)
	screen.rebuildBoards()
	boardScroll := container.NewVScroll(screen.boards)
	boardScroll.SetMinSize(fyne.NewSize(780, boardMinHeight*2))

	title := widget.NewLabelWithStyle(
		"Simulator Prototype",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	screen.status = canvas.NewText("", boardForeground)
	screen.status.TextStyle = fyne.TextStyle{Bold: true}
	phaseBar := screen.newPhaseBar()
	screen.updateMetadata()

	var back fyne.CanvasObject = layout.NewSpacer()
	if onBack != nil {
		backLabel := actions.BackLabel
		if strings.TrimSpace(backLabel) == "" {
			backLabel = "Back to Main Menu"
		}
		back = widget.NewButton(backLabel, onBack)
	}
	header := container.NewBorder(
		nil,
		nil,
		nil,
		back,
		container.NewVBox(title, screen.status, phaseBar),
	)

	screen.content = container.NewBorder(
		header,
		nil,
		newPreviewRegion(screen.preview),
		nil,
		boardScroll,
	)
	return screen
}

// Content returns the stable canvas object that should be installed in a
// window once.
func (screen *BoardScreen) Content() fyne.CanvasObject {
	if screen == nil {
		return layout.NewSpacer()
	}
	return screen.content
}

// Update applies a newer projection. Metadata-only transitions retain the
// existing card widgets; zone changes rebuild only the two player fields.
func (screen *BoardScreen) Update(match simulatorview.MatchView) {
	if screen == nil {
		return
	}
	previous := screen.match
	screen.match = match
	screen.updateMetadata()
	if previous.ViewerID != match.ViewerID {
		screen.rebuildBoards()
		return
	}

	viewerIndex := screen.viewerIndex()
	callAvailabilityChanged :=
		(screen.actions.CallFaceDownLevelOne != nil || screen.actions.CallFaceUpLevelOne != nil || screen.actions.LevelUpCaster != nil) &&
			canViewerCallFaceDownLevelOne(previous) != canViewerCallFaceDownLevelOne(match)
	aetherAvailabilityChanged :=
		screen.actions.GenerateNonElementalAether != nil &&
			canViewerGenerateNonElementalAether(previous) != canViewerGenerateNonElementalAether(match)
	casterAetherAvailabilityChanged :=
		screen.actions.GenerateCasterAether != nil &&
			canViewerGenerateCasterAether(previous) != canViewerGenerateCasterAether(match)
	tokenAvailabilityChanged :=
		screen.actions.UseCasterToken != nil &&
			canViewerUseCasterToken(previous) != canViewerUseCasterToken(match)
	fieldsChanged := false
	for playerIndex := range match.Players {
		position := 0
		isViewer := playerIndex == viewerIndex
		playerChanged := !reflect.DeepEqual(previous.Players[playerIndex], match.Players[playerIndex])
		if !playerChanged && !(isViewer && (callAvailabilityChanged || aetherAvailabilityChanged || casterAetherAvailabilityChanged || tokenAvailabilityChanged)) {
			continue
		}
		if isViewer {
			position = 1
		}
		screen.boards.Objects[position] = screen.newProjectedPlayerBoard(playerIndex, isViewer)
		fieldsChanged = true
	}
	if fieldsChanged {
		screen.boards.Refresh()
	}
}

func (screen *BoardScreen) rebuildBoards() {
	viewerIndex := screen.viewerIndex()
	opponentIndex := 1 - viewerIndex
	opponentBoard := screen.newProjectedPlayerBoard(opponentIndex, false)
	playerBoard := screen.newProjectedPlayerBoard(viewerIndex, true)
	screen.boards.Objects = []fyne.CanvasObject{opponentBoard, playerBoard}
	screen.boards.Refresh()
}

func (screen *BoardScreen) viewerIndex() int {
	if screen.match.Players[1].ID == screen.match.ViewerID {
		return 1
	}
	return 0
}

func (screen *BoardScreen) newProjectedPlayerBoard(
	playerIndex int,
	isViewer bool,
) fyne.CanvasObject {
	playerName := "Opponent"
	if isViewer {
		playerName = "Player"
	}
	return newPlayerBoard(
		playerName,
		screen.match.Players[playerIndex],
		isViewer,
		screen.definitions,
		screen.preview,
		screen.actions,
		func() model.Revision { return screen.match.Revision },
		canViewerCallFaceDownLevelOne(screen.match) &&
			(screen.actions.CallFaceDownLevelOne != nil || screen.actions.CallFaceUpLevelOne != nil || screen.actions.LevelUpCaster != nil),
		canViewerGenerateNonElementalAether(screen.match) && screen.actions.GenerateNonElementalAether != nil,
		canViewerGenerateCasterAether(screen.match) && screen.actions.GenerateCasterAether != nil,
		canViewerUseCasterToken(screen.match) && screen.actions.UseCasterToken != nil,
	)
}

func (screen *BoardScreen) updateMetadata() {
	match := screen.match
	screen.status.Text = fmt.Sprintf(
		"Turn %d • %s • Revision %d • Active player: %s",
		match.Turn.Number,
		match.Turn.Phase,
		match.Revision,
		match.Turn.ActivePlayer,
	)
	screen.status.Refresh()
	screen.updatePhaseButtons()
}

func canViewerCallFaceDownLevelOne(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress &&
		match.Turn.Phase == model.PhaseCall &&
		match.Turn.ActivePlayer == match.ViewerID &&
		!match.Turn.CallActionTaken
}

func canViewerGenerateNonElementalAether(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress
}

func canViewerGenerateCasterAether(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress
}

func canViewerUseCasterToken(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress
}

func (screen *BoardScreen) newPhaseBar() fyne.CanvasObject {
	phases := []model.Phase{
		model.PhaseRecovery,
		model.PhaseDraw,
		model.PhaseCall,
		model.PhaseMain,
		model.PhaseBattle,
		model.PhaseEnd,
	}
	buttons := make([]fyne.CanvasObject, 0, len(phases))
	for _, phase := range phases {
		button := widget.NewButton(string(phase), func() {
			if screen.actions.CompleteCurrentPhase != nil {
				screen.actions.CompleteCurrentPhase(screen.match.Revision)
			}
		})
		screen.phaseButtons[phase] = button
		buttons = append(buttons, button)
	}
	screen.updatePhaseButtons()
	return container.NewGridWithColumns(len(buttons), buttons...)
}

func (screen *BoardScreen) updatePhaseButtons() {
	match := screen.match
	completionTarget, hasCompletionTarget := prototypePhaseCompletionTarget(match)
	for phase, button := range screen.phaseButtons {
		label := string(phase)
		if match.Turn.Phase == model.PhaseEnd && phase == model.PhaseEnd {
			label = "End Turn"
		}
		if button.Text != label {
			button.SetText(label)
		}
		button.Importance = widget.MediumImportance
		if phase == match.Turn.Phase {
			button.Importance = widget.HighImportance
		}
		legalCompletion :=
			screen.actions.CompleteCurrentPhase != nil &&
				match.MatchStatus == model.StatusInProgress &&
				match.Turn.ActivePlayer == match.ViewerID &&
				hasCompletionTarget &&
				phase == completionTarget
		if !legalCompletion {
			button.Disable()
		} else {
			button.Enable()
		}
		button.Refresh()
	}
}

// prototypePhaseCompletionTarget mirrors only the transitions implemented by
// the current simulator skeleton. A future legal-actions projection should
// replace this presentation-only lookup as phase rules grow.
func prototypePhaseCompletionTarget(match simulatorview.MatchView) (model.Phase, bool) {
	switch {
	case match.Turn.Phase == model.PhaseRecovery && match.Turn.Number == 1:
		return model.PhaseCall, true
	case match.Turn.Phase == model.PhaseRecovery && match.Turn.Number > 1:
		return model.PhaseDraw, true
	case match.Turn.Phase == model.PhaseDraw && match.Turn.Number > 1:
		return model.PhaseCall, true
	case match.Turn.Phase == model.PhaseCall && match.Turn.Number > 0:
		return model.PhaseMain, true
	case match.Turn.Phase == model.PhaseMain && match.Turn.Number > 0:
		return model.PhaseBattle, true
	case match.Turn.Phase == model.PhaseBattle && match.Turn.Number > 0:
		return model.PhaseEnd, true
	case match.Turn.Phase == model.PhaseEnd && match.Turn.Number > 0:
		return model.PhaseEnd, true
	default:
		return "", false
	}
}

func newCardLookup(cardDefinitions []cards.Card) cardLookup {
	result := make(cardLookup, len(cardDefinitions)+1)
	for _, card := range cardDefinitions {
		result[model.CardID(card.ID)] = card
		if strings.EqualFold(strings.TrimSpace(card.Name), "Caster Token") {
			result[model.CasterTokenCardID] = card
		}
	}
	return result
}

func newPreviewPanel() previewState {
	title := widget.NewLabelWithStyle(
		"No card selected",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	title.Wrapping = fyne.TextWrapWord

	description := widget.NewLabel(
		"Hover over a visible card to inspect it. Concealed cards reveal no identity.",
	)
	description.Wrapping = fyne.TextWrapWord

	imageSizer := canvas.NewRectangle(color.NRGBA{R: 22, G: 25, B: 29, A: 255})
	imageSizer.StrokeColor = zoneBorder
	imageSizer.StrokeWidth = 2
	imageSizer.SetMinSize(fyne.NewSize(210, 294))

	placeholder := widget.NewLabel("Card Preview")
	placeholder.Alignment = fyne.TextAlignCenter
	placeholder.Importance = widget.LowImportance

	return previewState{
		title:       title,
		description: description,
		image:       container.NewStack(imageSizer, container.NewCenter(placeholder)),
		imageSizer:  imageSizer,
	}
}

func newPreviewRegion(preview previewState) fyne.CanvasObject {
	descriptionScroll := container.NewVScroll(preview.description)
	descriptionScroll.SetMinSize(fyne.NewSize(0, 145))

	heading := container.NewVBox(
		widget.NewLabelWithStyle(
			"Card Information",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		widget.NewSeparator(),
		preview.image,
		preview.title,
		widget.NewSeparator(),
	)
	content := container.NewBorder(heading, nil, nil, nil, descriptionScroll)

	sizer := canvas.NewRectangle(color.Transparent)
	sizer.SetMinSize(fyne.NewSize(previewPanelWidth, 0))
	return container.NewStack(sizer, container.NewPadded(content))
}

func newPlayerBoard(
	playerName string,
	player simulatorview.PlayerView,
	isViewer bool,
	definitions cardLookup,
	preview previewState,
	actions BoardActions,
	currentRevision func() model.Revision,
	canCallLevelOne bool,
	canGenerateNonElementalAether bool,
	canGenerateCasterAether bool,
	canUseCasterToken bool,
) fyne.CanvasObject {
	orbZone := newCardZone(
		playerName,
		"Orb Zone",
		"Face-down Orbs. Their identities are concealed from both players.",
		player.Orbs,
		fyne.NewSize(86, 60),
		true,
		true,
		definitions,
		preview,
	)
	deckZone := newDeckZone(playerName, player.DeckCount, preview)
	graveZone := newCardZone(
		playerName,
		"Graveyard",
		"Used and destroyed cards are displayed here.",
		player.Graveyard,
		fyne.NewSize(86, 120),
		true,
		false,
		definitions,
		preview,
	)
	exileZone := newCardZone(
		playerName,
		"Exile",
		"Cards removed from the game are displayed here.",
		player.Exile,
		fyne.NewSize(86, 120),
		true,
		false,
		definitions,
		preview,
	)
	servantZone := newCardZone(
		playerName,
		"Servant Zone",
		"Servants and barriers occupy this field.",
		player.ServantZone,
		fyne.NewSize(86, 120),
		false,
		false,
		definitions,
		preview,
	)
	casterZone := newAetherCasterZone(
		playerName,
		player,
		isViewer,
		definitions,
		preview,
		actions,
		currentRevision,
		canGenerateNonElementalAether,
		canGenerateCasterAether,
		canUseCasterToken,
	)
	handZone := newHandZone(
		playerName,
		player,
		isViewer,
		definitions,
		preview,
		actions,
		currentRevision,
		canCallLevelOne,
	)

	orbRegion := withMinimumSize(orbZone, fyne.NewSize(sideZoneWidth, 0))
	rightZones := withMinimumSize(
		container.NewGridWithRows(3, deckZone, graveZone, exileZone),
		fyne.NewSize(sideZoneWidth, 0),
	)
	centerZones := container.NewBorder(
		nil,
		handZone,
		nil,
		nil,
		container.NewBorder(nil, casterZone, nil, nil, servantZone),
	)

	field := container.NewBorder(nil, nil, orbRegion, rightZones, centerZones)
	background := canvas.NewRectangle(boardBackground)
	background.StrokeColor = zoneBorder
	background.StrokeWidth = 1
	background.SetMinSize(fyne.NewSize(0, boardMinHeight))

	label := canvas.NewText(fmt.Sprintf("%s Field — %s", playerName, player.ID), boardForeground)
	label.TextStyle = fyne.TextStyle{Bold: true}
	aether := newAetherPoolDisplay(player.Aether)
	return container.NewStack(
		background,
		container.NewBorder(
			container.NewVBox(label, aether),
			nil,
			nil,
			nil,
			container.NewPadded(field),
		),
	)
}

func newHandZone(
	playerName string,
	player simulatorview.PlayerView,
	isViewer bool,
	definitions cardLookup,
	preview previewState,
	actions BoardActions,
	currentRevision func() model.Revision,
	canCallLevelOne bool,
) fyne.CanvasObject {
	if !isViewer {
		zone := newCardZone(
			playerName,
			"Hand",
			"Cards currently held by this player.",
			player.Hand,
			fyne.NewSize(86, 120),
			false,
			false,
			definitions,
			preview,
		)
		return zone
	}
	if player.OpeningHandFinalized {
		if canCallLevelOne {
			return newLevelOneCallHandZone(
				playerName,
				player,
				definitions,
				preview,
				actions,
				currentRevision,
			)
		}
		return newCardZone(
			playerName,
			"Hand",
			"Cards currently held by this player.",
			player.Hand,
			fyne.NewSize(86, 120),
			false,
			false,
			definitions,
			preview,
		)
	}

	selected := make(map[model.MatchCardID]struct{})
	tiles := make([]*CardTile, 0, len(player.Hand))
	objects := make([]fyne.CanvasObject, 0, len(player.Hand))
	var replaceButton *widget.Button

	for _, projectedCard := range player.Hand {
		projectedCard := projectedCard
		definition := definitions[projectedCard.CardID]
		tile := NewCardTile(
			projectedCard,
			definition,
			fyne.NewSize(86, 120),
			preview.showCard,
			func() { preview.showHiddenCard(playerName, "Hand") },
		)
		tile.OnActivate = func() {
			if projectedCard.MatchID == "" {
				return
			}
			if _, exists := selected[projectedCard.MatchID]; exists {
				delete(selected, projectedCard.MatchID)
				tile.SetSelected(false)
			} else {
				selected[projectedCard.MatchID] = struct{}{}
				tile.SetSelected(true)
			}
			if len(selected) == 0 {
				replaceButton.Disable()
			} else {
				replaceButton.Enable()
			}
		}
		tiles = append(tiles, tile)
		objects = append(objects, tile)
	}

	submit := func(replace []model.MatchCardID) {
		if actions.SubmitOpeningHand != nil {
			revision := model.Revision(0)
			if currentRevision != nil {
				revision = currentRevision()
			}
			actions.SubmitOpeningHand(replace, revision)
		}
	}
	keepButton := widget.NewButton("Keep Hand", func() { submit(nil) })
	replaceButton = widget.NewButton("Replace Selected", func() {
		replacements := make([]model.MatchCardID, 0, len(selected))
		for _, tile := range tiles {
			if _, exists := selected[tile.View.MatchID]; exists {
				replacements = append(replacements, tile.View.MatchID)
			}
		}
		submit(replacements)
	})
	replaceButton.Disable()
	if actions.SubmitOpeningHand == nil {
		keepButton.Disable()
	}

	cardRow := container.NewHScroll(container.NewHBox(objects...))
	content := container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), keepButton, replaceButton),
		nil,
		nil,
		cardRow,
	)
	return newZone(
		playerName,
		"Hand",
		"Select cards to replace, or keep the complete opening hand.",
		content,
		preview,
	)
}

func newLevelOneCallHandZone(
	playerName string,
	player simulatorview.PlayerView,
	definitions cardLookup,
	preview previewState,
	actions BoardActions,
	currentRevision func() model.Revision,
) fyne.CanvasObject {
	selectedID := model.MatchCardID("")
	tiles := make([]*CardTile, 0, len(player.Hand))
	objects := make([]fyne.CanvasObject, 0, len(player.Hand))
	var faceDownButton *widget.Button
	var faceUpButton *widget.Button
	var levelUpTarget *widget.Select
	var levelUpButton *widget.Button
	levelUpTargetsByLabel := make(map[string]model.MatchCardID)
	selectedTargetID := model.MatchCardID("")
	updateLevelUpTargets := func(definition cards.Card) {
		if levelUpTarget == nil || levelUpButton == nil {
			return
		}
		clear(levelUpTargetsByLabel)
		selectedTargetID = ""
		options := make([]string, 0)
		for _, candidate := range eligibleLevelUpTargets(definition, player.CasterZone, definitions) {
			options = append(options, candidate.label)
			levelUpTargetsByLabel[candidate.label] = candidate.matchID
		}
		levelUpTarget.SetOptions(options)
		levelUpTarget.ClearSelected()
		levelUpButton.Disable()
		if len(options) == 0 {
			levelUpTarget.Hide()
			levelUpButton.Hide()
			return
		}
		levelUpTarget.Show()
		levelUpButton.Show()
	}

	for _, projectedCard := range player.Hand {
		projectedCard := projectedCard
		definition := definitions[projectedCard.CardID]
		tile := NewCardTile(
			projectedCard,
			definition,
			fyne.NewSize(86, 120),
			preview.showCard,
			func() { preview.showHiddenCard(playerName, "Hand") },
		)
		tile.OnActivate = func() {
			if projectedCard.MatchID == "" {
				return
			}
			wasSelected := selectedID == projectedCard.MatchID
			for _, candidate := range tiles {
				candidate.SetSelected(false)
			}
			if wasSelected {
				selectedID = ""
				if faceDownButton != nil {
					faceDownButton.Disable()
				}
				if faceUpButton != nil {
					faceUpButton.Disable()
				}
				updateLevelUpTargets(cards.Card{})
				return
			}
			selectedID = projectedCard.MatchID
			tile.SetSelected(true)
			if faceDownButton != nil {
				faceDownButton.Enable()
			}
			if faceUpButton != nil {
				if isFaceUpLevelOneCallDefinition(definition) {
					faceUpButton.Enable()
				} else {
					faceUpButton.Disable()
				}
			}
			updateLevelUpTargets(definition)
		}
		tiles = append(tiles, tile)
		objects = append(objects, tile)
	}

	actionButtons := []fyne.CanvasObject{layout.NewSpacer()}
	if actions.CallFaceUpLevelOne != nil {
		faceUpButton = widget.NewButton("Call Selected Face Up", func() {
			if selectedID == "" {
				return
			}
			revision := model.Revision(0)
			if currentRevision != nil {
				revision = currentRevision()
			}
			actions.CallFaceUpLevelOne(selectedID, revision)
		})
		faceUpButton.Disable()
		actionButtons = append(actionButtons, faceUpButton)
	}
	if actions.CallFaceDownLevelOne != nil {
		faceDownButton = widget.NewButton("Call Selected Face Down", func() {
			if selectedID == "" {
				return
			}
			revision := model.Revision(0)
			if currentRevision != nil {
				revision = currentRevision()
			}
			actions.CallFaceDownLevelOne(selectedID, revision)
		})
		faceDownButton.Disable()
		actionButtons = append(actionButtons, faceDownButton)
	}
	if actions.LevelUpCaster != nil {
		levelUpTarget = widget.NewSelect(nil, func(label string) {
			selectedTargetID = levelUpTargetsByLabel[label]
			if selectedID == "" || selectedTargetID == "" {
				levelUpButton.Disable()
				return
			}
			levelUpButton.Enable()
		})
		levelUpTarget.PlaceHolder = "Choose Caster to level up"
		levelUpTarget.Hide()
		levelUpButton = widget.NewButton("Level Up Selected", func() {
			if selectedID == "" || selectedTargetID == "" {
				return
			}
			revision := model.Revision(0)
			if currentRevision != nil {
				revision = currentRevision()
			}
			actions.LevelUpCaster(selectedID, selectedTargetID, revision)
		})
		levelUpButton.Disable()
		levelUpButton.Hide()
		actionButtons = append(actionButtons, levelUpTarget, levelUpButton)
	}

	cardRow := container.NewHScroll(container.NewHBox(objects...))
	content := container.NewBorder(
		nil,
		container.NewHBox(actionButtons...),
		nil,
		nil,
		cardRow,
	)
	return newZone(
		playerName,
		"Hand",
		"Select one card to Call as Level 1 or use it to level up a matching Caster.",
		content,
		preview,
	)
}

type levelUpTargetOption struct {
	label   string
	matchID model.MatchCardID
}

func eligibleLevelUpTargets(
	upper cards.Card,
	casterZone []simulatorview.CardView,
	definitions cardLookup,
) []levelUpTargetOption {
	upperLevel, upperLevelValid := definitionLevel(upper)
	upperName := strings.ToLower(strings.TrimSpace(upper.Name))
	if !strings.EqualFold(strings.TrimSpace(upper.Type), "caster") ||
		!upperLevelValid || upperLevel < 2 || upperName == "" {
		return nil
	}
	result := make([]levelUpTargetOption, 0)
	for _, target := range casterZone {
		if target.MatchID == "" || target.Face != model.CardFaceUp || target.CardID == model.CasterTokenCardID {
			continue
		}
		definition, found := definitions[target.CardID]
		if !found || !strings.EqualFold(strings.TrimSpace(definition.Type), "caster") ||
			strings.ToLower(strings.TrimSpace(definition.Name)) != upperName {
			continue
		}
		targetLevel, targetLevelValid := definitionLevel(definition)
		if !targetLevelValid || upperLevel != targetLevel+1 {
			continue
		}
		identity := strings.TrimSpace(definition.Name)
		if subname := strings.TrimSpace(definition.Subname); subname != "" {
			identity += " — " + subname
		}
		result = append(result, levelUpTargetOption{
			label:   fmt.Sprintf("%s (Level %d, %s)", identity, targetLevel, target.MatchID),
			matchID: target.MatchID,
		})
	}
	return result
}

func definitionLevel(definition cards.Card) (int, bool) {
	level, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	return level, err == nil
}

func isFaceUpLevelOneCallDefinition(definition cards.Card) bool {
	if !strings.EqualFold(strings.TrimSpace(definition.Type), "caster") {
		return false
	}
	level, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	return err == nil && level == 1
}

func newCardZone(
	playerName string,
	zoneName string,
	description string,
	cardViews []simulatorview.CardView,
	tileSize fyne.Size,
	vertical bool,
	sideways bool,
	definitions cardLookup,
	preview previewState,
) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(cardViews))
	for _, projectedCard := range cardViews {
		definition := definitions[projectedCard.CardID]
		tile := NewCardTile(
			projectedCard,
			definition,
			tileSize,
			preview.showCard,
			func() { preview.showHiddenCard(playerName, zoneName) },
		)
		if sideways || projectedCard.Orientation == model.OrientationRested {
			tile.SetSideways(true)
		}
		objects = append(objects, tile)
	}

	content := fyne.CanvasObject(layout.NewSpacer())
	if len(objects) > 0 {
		if vertical {
			content = container.NewVScroll(container.NewVBox(objects...))
		} else {
			content = container.NewHScroll(container.NewHBox(objects...))
		}
	}
	return newZone(playerName, zoneName, description, content, preview)
}

func newAetherCasterZone(
	playerName string,
	player simulatorview.PlayerView,
	isViewer bool,
	definitions cardLookup,
	preview previewState,
	actions BoardActions,
	currentRevision func() model.Revision,
	canGenerate bool,
	canGenerateCaster bool,
	canUseToken bool,
) fyne.CanvasObject {
	eligibleFaceDownCaster := func(card simulatorview.CardView) bool {
		return isViewer &&
			canGenerate &&
			card.MatchID != "" &&
			card.Face == model.CardFaceDown &&
			card.Orientation == model.OrientationRecovered
	}
	eligibleToken := func(card simulatorview.CardView) bool {
		return isViewer &&
			canUseToken &&
			card.MatchID != "" &&
			card.CardID == model.CasterTokenCardID &&
			card.Face == model.CardFaceUp &&
			card.Orientation == model.OrientationRecovered
	}
	eligibleFaceUpCaster := func(card simulatorview.CardView) bool {
		definition, found := definitions[card.CardID]
		return isViewer &&
			canGenerateCaster &&
			found &&
			card.MatchID != "" &&
			card.CardID != model.CasterTokenCardID &&
			card.Face == model.CardFaceUp &&
			card.Orientation == model.OrientationRecovered &&
			strings.EqualFold(strings.TrimSpace(definition.Type), "Caster")
	}
	eligible := func(card simulatorview.CardView) bool {
		return eligibleFaceDownCaster(card) || eligibleFaceUpCaster(card) || eligibleToken(card)
	}
	hasEligibleCard := false
	for _, card := range player.CasterZone {
		if eligible(card) {
			hasEligibleCard = true
			break
		}
	}
	if !hasEligibleCard {
		return newCardZone(
			playerName,
			"Caster Zone",
			"Casters and the starting Caster Token occupy this zone.",
			player.CasterZone,
			fyne.NewSize(86, 120),
			false,
			false,
			definitions,
			preview,
		)
	}

	selectedID := model.MatchCardID("")
	selectedAction := aetherActionNone
	tiles := make([]*CardTile, 0, len(player.CasterZone))
	objects := make([]fyne.CanvasObject, 0, len(player.CasterZone))
	var actionButton *widget.Button
	for _, projectedCard := range player.CasterZone {
		projectedCard := projectedCard
		definition := definitions[projectedCard.CardID]
		tile := NewCardTile(
			projectedCard,
			definition,
			fyne.NewSize(86, 120),
			preview.showCard,
			func() { preview.showHiddenCard(playerName, "Caster Zone") },
		)
		if projectedCard.Orientation == model.OrientationRested {
			tile.SetSideways(true)
		}
		if eligible(projectedCard) {
			tile.OnActivate = func() {
				wasSelected := selectedID == projectedCard.MatchID
				for _, candidate := range tiles {
					candidate.SetSelected(false)
				}
				if wasSelected {
					selectedID = ""
					selectedAction = aetherActionNone
					actionButton.Disable()
					actionButton.Hide()
					return
				}
				selectedID = projectedCard.MatchID
				tile.SetSelected(true)
				switch {
				case eligibleToken(projectedCard):
					selectedAction = aetherActionToken
					actionButton.SetText("Remove Token for 1 Aether")
				case eligibleFaceUpCaster(projectedCard):
					selectedAction = aetherActionFaceUpCaster
					actionButton.SetText("Rest Selected for Elemental Aether")
				default:
					selectedAction = aetherActionFaceDownCaster
					actionButton.SetText("Rest Selected for 1 Aether")
				}
				actionButton.Enable()
				actionButton.Show()
			}
		}
		tiles = append(tiles, tile)
		objects = append(objects, tile)
	}

	actionButton = widget.NewButton("Produce Aether", func() {
		if selectedID == "" {
			return
		}
		revision := model.Revision(0)
		if currentRevision != nil {
			revision = currentRevision()
		}
		switch selectedAction {
		case aetherActionToken:
			if actions.UseCasterToken != nil {
				actions.UseCasterToken(selectedID, revision)
			}
		case aetherActionFaceUpCaster:
			if actions.GenerateCasterAether != nil {
				actions.GenerateCasterAether(selectedID, revision)
			}
		case aetherActionFaceDownCaster:
			if actions.GenerateNonElementalAether != nil {
				actions.GenerateNonElementalAether(selectedID, revision)
			}
		}
	})
	actionButton.Disable()
	actionButton.Hide()

	cardRow := container.NewHScroll(container.NewHBox(objects...))
	content := container.NewBorder(
		nil,
		container.NewHBox(
			layout.NewSpacer(),
			widget.NewLabel("Choose a usable Caster or Caster Token."),
			actionButton,
		),
		nil,
		nil,
		cardRow,
	)
	return newZone(
		playerName,
		"Caster Zone",
		"Rest a Caster to produce its Aether, or remove the Caster Token to produce one non-elemental Aether.",
		content,
		preview,
	)
}

func newDeckZone(playerName string, count int, preview previewState) fyne.CanvasObject {
	content := fyne.CanvasObject(layout.NewSpacer())
	if count > 0 {
		cardBack := NewCardTile(
			simulatorview.CardView{Face: model.CardFaceDown},
			cards.Card{},
			fyne.NewSize(86, 120),
			preview.showCard,
			func() { preview.showHiddenCard(playerName, "Deck Zone") },
		)
		countLabel := widget.NewLabel(fmt.Sprintf("%d cards", count))
		countLabel.Alignment = fyne.TextAlignCenter
		content = container.NewCenter(container.NewVBox(cardBack, countLabel))
	}
	return newZone(
		playerName,
		"Deck Zone",
		fmt.Sprintf("%d cards remain in this deck.", count),
		content,
		preview,
	)
}

func newZone(
	playerName string,
	zoneName string,
	description string,
	content fyne.CanvasObject,
	preview previewState,
) fyne.CanvasObject {
	button := widget.NewButton(zoneName, func() {
		preview.title.SetText(playerName + " — " + zoneName)
		preview.description.SetText(description)
	})
	button.Importance = widget.LowImportance

	background := canvas.NewRectangle(zoneBackground)
	background.StrokeColor = zoneBorder
	background.StrokeWidth = 1

	minimum := fyne.NewSize(0, casterZoneHeight)
	switch zoneName {
	case "Hand":
		minimum.Height = handZoneHeight
	case "Servant Zone":
		minimum.Height = 170
	}

	return withMinimumSize(
		container.NewStack(
			background,
			container.NewBorder(button, nil, nil, nil, container.NewPadded(content)),
		),
		minimum,
	)
}

func withMinimumSize(object fyne.CanvasObject, size fyne.Size) fyne.CanvasObject {
	sizer := canvas.NewRectangle(color.Transparent)
	sizer.SetMinSize(size)
	return container.NewStack(sizer, object)
}

func (preview previewState) showCard(card cards.Card) {
	preview.title.SetText(card.Name)
	preview.description.SetText(fmt.Sprintf(
		"Type: %s\nElement: %s\nCost/Lv: %s\nTraits: %s\n\n%s",
		card.Type,
		card.Element,
		card.CostLevel,
		card.Traits,
		card.Ability,
	))

	preview.image.RemoveAll()
	preview.image.Add(preview.imageSizer)
	if imagePath, found := cardimages.Find(card.ID); found {
		cardImage := canvas.NewImageFromFile(imagePath)
		cardImage.FillMode = canvas.ImageFillContain
		cardImage.ScaleMode = canvas.ImageScaleSmooth
		preview.image.Add(cardImage)
	} else {
		message := widget.NewLabel("Image unavailable")
		message.Alignment = fyne.TextAlignCenter
		preview.image.Add(container.NewCenter(message))
	}
	preview.image.Refresh()
}

func (preview previewState) showHiddenCard(playerName, zoneName string) {
	preview.title.SetText("Concealed Card")
	preview.description.SetText(
		fmt.Sprintf("%s's %s card is hidden from this viewer.", playerName, zoneName),
	)
	preview.image.RemoveAll()
	preview.image.Add(preview.imageSizer)
	preview.image.Add(cardBackImage())
	preview.image.Refresh()
}
