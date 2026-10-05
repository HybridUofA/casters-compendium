// Package ui presents simulator state without owning or enforcing game rules.
package ui

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const (
	previewPanelWidth float32 = 270
	boardMinHeight    float32 = 300
	sideZoneWidth     float32 = 150
	orbZoneWidth      float32 = 200
	handZoneHeight    float32 = 82
	casterZoneHeight  float32 = 82
	fieldCardWidth    float32 = 56
	fieldCardHeight   float32 = 78
	orbCardWidth      float32 = 98
	orbCardHeight     float32 = 68
	orbLayerStep      float32 = 27
	utilityCardWidth  float32 = 41
	utilityCardHeight float32 = 57
	utilityZoneHeight float32 = 61
)

var (
	boardBackground = color.NRGBA{R: 29, G: 50, B: 55, A: 255}
	zoneBackground  = color.NRGBA{R: 45, G: 73, B: 78, A: 255}
	zoneBorder      = color.NRGBA{R: 104, G: 151, B: 155, A: 255}
	boardForeground = color.NRGBA{R: 240, G: 250, B: 250, A: 255}
)

type previewState struct {
	title         *widget.Label
	description   *widget.Label
	actions       *fyne.Container
	manualActions *fyne.Container
	image         *fyne.Container
	imageSizer    *canvas.Rectangle
	shownCardID   *model.CardID
	fullArtwork   *previewArtworkCache
}

type previewArtworkCache struct {
	cardID  model.CardID
	artwork image.Image
}

type cardLookup map[model.CardID]cards.Card

// BoardActions translates presentation choices into session-owned commands.
type BoardActions struct {
	MoveCard                   func(model.MoveCardCommand, model.Revision)
	DrawCards                  func(count int, revision model.Revision)
	ShuffleDeck                func(revision model.Revision)
	PeekDeckTops               func(ownerID model.PlayerID, count int, done func([]simulatorview.CardView, error))
	MoveDeckTopToBottom        func(ownerID model.PlayerID, revision model.Revision)
	ResolveDeckDig             func(keep model.MatchCardID, bottomOrder []model.MatchCardID, revision model.Revision)
	SubmitOpeningHand          func([]model.MatchCardID, model.Revision)
	CallFaceDownLevelOne       func(model.MatchCardID, model.Revision)
	CallFaceUpLevelOne         func(model.MatchCardID, model.Revision)
	LevelUpCaster              func(model.MatchCardID, model.MatchCardID, model.Revision)
	GenerateNonElementalAether func(model.MatchCardID, model.Revision)
	GenerateCasterAether       func(model.MatchCardID, model.Revision)
	UseCasterToken             func(model.MatchCardID, model.Revision)
	CastServant                func(model.MatchCardID, model.AetherPayment, model.CardOrientation, model.Revision)
	CastConjure                func(model.MatchCardID, model.AetherPayment, model.Revision)
	CastBarrier                func(model.MatchCardID, model.AetherPayment, model.Revision)
	CastServantWithPlan        func(model.MatchCardID, model.CastPaymentPlan, model.CardOrientation, model.Revision)
	CastConjureWithPlan        func(model.MatchCardID, model.CastPaymentPlan, model.Revision)
	CastBarrierWithPlan        func(model.MatchCardID, model.CastPaymentPlan, model.Revision)
	PassPriority               func(model.Revision)
	DeclareAttack              func(model.MatchCardID, model.AttackTargetKind, model.MatchCardID, model.Revision)
	CorruptOrbs                func(orbIndexes []int, revision model.Revision)
	SetGrantedDoubleCorrupt    func(cardID model.MatchCardID, enabled bool, revision model.Revision)
	PlayBreak                  func(cardID model.MatchCardID, orientation model.CardOrientation, revision model.Revision)
	DeclineBreak               func(revision model.Revision)
	AcceptSageAdvice           func(revision model.Revision)
	DeclineDrawReplacement     func(revision model.Revision)
	PeekOrb                    func(ownerID model.PlayerID, orbIndex int, revision model.Revision, done func(simulatorview.CardView, error))
	RevealOrb                  func(orbIndex int, revision model.Revision)
	CompleteCurrentPhase       func(model.Revision)
	// ShouldAutoPassPriority opts into Arena-style auto-pass when true for the
	// current view. The board calls PassPriority at most once per revision.
	ShouldAutoPassPriority func(simulatorview.MatchView) bool
	BackLabel              string
}

// BoardScreen owns a persistent simulator widget tree. Update changes public
// match metadata in place and rebuilds card zones only when projected player
// data changes.
type BoardScreen struct {
	content              fyne.CanvasObject
	status               *canvas.Text
	phaseHint            *canvas.Text
	phaseButtons         map[model.Phase]*widget.Button
	passPriority         *widget.Button
	attackPanel          *fyne.Container
	boards               *fyne.Container
	playerBoards         [2]*playerBoardController
	boardArea            fyne.CanvasObject
	aetherPools          *fyne.Container
	match                simulatorview.MatchView
	definitions          cardLookup
	preview              previewState
	actions              BoardActions
	dropTargets          []zoneDropTarget
	drag                 cardDragSession
	lastAutoPassRevision model.Revision
	lastSageDigRevision  model.Revision
	// pendingAttackerID is a local UI selection for left-click attack targeting.
	pendingAttackerID model.MatchCardID
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
	screen.aetherPools = container.NewVBox()
	screen.updateAetherPools()
	screen.updateManualMoves()

	title := widget.NewLabelWithStyle(
		"Simulator Prototype",
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	screen.status = canvas.NewText("", boardForeground)
	screen.status.TextStyle = fyne.TextStyle{Bold: true}
	phaseBar := screen.newPhaseBar()
	phaseBackground := canvas.NewRectangle(color.NRGBA{R: 20, G: 35, B: 39, A: 245})
	phaseBackground.StrokeColor = zoneBorder
	phaseBackground.StrokeWidth = 1
	phaseBand := container.NewStack(
		phaseBackground,
		container.NewPadded(container.NewCenter(phaseBar)),
	)
	screen.boardArea = container.NewStack(
		screen.boards,
		container.NewCenter(phaseBand),
	)
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
		container.NewVBox(title, screen.status),
	)

	screen.content = container.NewBorder(
		header,
		nil,
		newPreviewRegion(screen.preview, screen.aetherPools),
		nil,
		screen.boardArea,
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
	if screen.pendingAttackerID != "" && !canViewerDeclareAttack(match) {
		screen.pendingAttackerID = ""
	}
	if previous.Revision != match.Revision || previous.ViewerID != match.ViewerID || previous.MatchStatus != match.MatchStatus {
		screen.updateManualMoves()
	}
	screen.updateMetadata()
	// Preview action panels (Cast payment / Level Up) are cleared when the
	// viewer's Hand is rebuilt, not on every revision bump — otherwise a peer
	// view push or auto-pass can wipe the Level Up prompt mid-selection.
	if previous.ViewerID != match.ViewerID ||
		previous.Players[0].Aether != match.Players[0].Aether ||
		previous.Players[1].Aether != match.Players[1].Aether {
		screen.updateAetherPools()
	}
	if previous.ViewerID != match.ViewerID {
		screen.rebuildBoards()
		return
	}

	viewerIndex := screen.viewerIndex()
	canCall := canViewerCallFaceDownLevelOne(match) &&
		(screen.actions.CallFaceDownLevelOne != nil || screen.actions.CallFaceUpLevelOne != nil || screen.actions.LevelUpCaster != nil)
	canCast := canViewerCast(match) &&
		(screen.actions.CastServant != nil || screen.actions.CastConjure != nil || screen.actions.CastBarrier != nil ||
			screen.actions.CastServantWithPlan != nil || screen.actions.CastConjureWithPlan != nil || screen.actions.CastBarrierWithPlan != nil)
	canNonElemental := canViewerGenerateNonElementalAether(match) && screen.actions.GenerateNonElementalAether != nil
	canCasterAether := canViewerGenerateCasterAether(match) && screen.actions.GenerateCasterAether != nil
	canUseToken := canViewerUseCasterToken(match) && screen.actions.UseCasterToken != nil
	for playerIndex := range match.Players {
		position := 0
		isViewer := playerIndex == viewerIndex
		if isViewer {
			position = 1
		}
		board := screen.playerBoards[position]
		if board == nil {
			continue
		}
		board.update(
			match.Players[playerIndex],
			isViewer && canCall,
			isViewer && canCast,
			isViewer && canNonElemental,
			isViewer && canCasterAether,
			isViewer && canUseToken,
		)
	}
}

func (screen *BoardScreen) rebuildBoards() {
	screen.clearDropTargets()
	viewerIndex := screen.viewerIndex()
	opponentIndex := 1 - viewerIndex
	opponentBoard := screen.newProjectedPlayerBoardController(opponentIndex, false)
	playerBoard := screen.newProjectedPlayerBoardController(viewerIndex, true)
	screen.playerBoards = [2]*playerBoardController{opponentBoard, playerBoard}
	screen.boards.Objects = []fyne.CanvasObject{opponentBoard.root, playerBoard.root}
	refreshContainerStructure(screen.boards)
	screen.refreshDropTargets()
}

func (screen *BoardScreen) viewerIndex() int {
	if screen.match.Players[1].ID == screen.match.ViewerID {
		return 1
	}
	return 0
}

func (screen *BoardScreen) newProjectedPlayerBoardController(
	playerIndex int,
	isViewer bool,
) *playerBoardController {
	playerName := "Opponent"
	if isViewer {
		playerName = "Player"
	}
	return newPlayerBoardController(
		screen,
		playerName,
		screen.match.Players[playerIndex],
		isViewer,
		screen.definitions,
		screen.preview,
		screen.actions,
		func() model.Revision { return screen.match.Revision },
		canViewerCallFaceDownLevelOne(screen.match) &&
			(screen.actions.CallFaceDownLevelOne != nil || screen.actions.CallFaceUpLevelOne != nil || screen.actions.LevelUpCaster != nil),
		canViewerCast(screen.match) &&
			(screen.actions.CastServant != nil || screen.actions.CastConjure != nil || screen.actions.CastBarrier != nil ||
				screen.actions.CastServantWithPlan != nil || screen.actions.CastConjureWithPlan != nil || screen.actions.CastBarrierWithPlan != nil),
		canViewerGenerateNonElementalAether(screen.match) && screen.actions.GenerateNonElementalAether != nil,
		canViewerGenerateCasterAether(screen.match) && screen.actions.GenerateCasterAether != nil,
		canViewerUseCasterToken(screen.match) && screen.actions.UseCasterToken != nil,
	)
}

func (screen *BoardScreen) updateMetadata() {
	match := screen.match
	activeLabel := displayNameFor(match, match.Turn.ActivePlayer)
	priorityLabel := displayNameFor(match, match.PriorityHolder)
	status := fmt.Sprintf(
		"Turn %d • %s • Revision %d • Active player: %s • Priority: %s • Chase: %d",
		match.Turn.Number,
		match.Turn.Phase,
		match.Revision,
		activeLabel,
		priorityLabel,
		match.ChaseLinkCount,
	)
	if match.Spectator {
		status = "Spectating • " + status
	}
	if match.MatchStatus == model.StatusFinished {
		if match.Result.IsDraw {
			status = fmt.Sprintf("Match finished — draw (%s) • Revision %d", match.Result.Reason, match.Revision)
		} else {
			status = fmt.Sprintf(
				"Match finished — %s defeated %s (%s) • Revision %d",
				displayNameFor(match, match.Result.Winner),
				displayNameFor(match, match.Result.Loser),
				match.Result.Reason,
				match.Revision,
			)
		}
	} else if match.Attack.Step != model.BattleStepIdle {
		status += fmt.Sprintf(" • Attack: %s", match.Attack.Step)
	}
	if screen.status.Text != status {
		screen.status.Text = status
		screen.status.Refresh()
	}
	screen.updatePhaseButtons()
	screen.updatePriorityButton()
	screen.updateAttackPanel()
	screen.maybePromptSageDig()
	screen.maybeAutoPassPriority()
}

func displayNameFor(match simulatorview.MatchView, id model.PlayerID) string {
	if id == "" {
		return ""
	}
	if name, ok := match.DisplayNames[id]; ok && strings.TrimSpace(name) != "" {
		return name
	}
	return string(id)
}

func (screen *BoardScreen) maybeAutoPassPriority() {
	if screen == nil || screen.actions.PassPriority == nil || screen.actions.ShouldAutoPassPriority == nil {
		return
	}
	match := screen.match
	if match.Revision == 0 || match.Revision == screen.lastAutoPassRevision {
		return
	}
	if !screen.actions.ShouldAutoPassPriority(match) {
		return
	}
	screen.lastAutoPassRevision = match.Revision
	revision := match.Revision
	// Run after the current view apply finishes so Call/Level Up hand rebuilds
	// are not interleaved with a follow-up priority command on the same tick.
	fyne.Do(func() {
		if screen.actions.PassPriority != nil {
			screen.actions.PassPriority(revision)
		}
	})
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

func canViewerCast(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress &&
		match.Turn.Phase == model.PhaseMain &&
		match.Turn.ActivePlayer == match.ViewerID &&
		match.PriorityHolder == match.ViewerID &&
		match.ChaseLinkCount == 0
}

func canViewerDeclareAttack(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress &&
		match.Turn.Phase == model.PhaseBattle &&
		match.Turn.ActivePlayer == match.ViewerID &&
		match.PrioritySequenceOpen &&
		match.PriorityHolder == match.ViewerID &&
		match.PassCount == 0 &&
		match.ChaseLinkCount == 0 &&
		match.Attack.Step == model.BattleStepIdle &&
		match.Attack.AttackerID == ""
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
	screen.phaseHint = canvas.NewText("", boardForeground)
	screen.phaseHint.Alignment = fyne.TextAlignCenter
	screen.phaseHint.TextStyle = fyne.TextStyle{Bold: true}
	screen.updatePhaseButtons()
	screen.passPriority = widget.NewButton("Pass Priority", func() {
		if screen.actions.PassPriority != nil {
			screen.actions.PassPriority(screen.match.Revision)
		}
	})
	screen.attackPanel = container.NewVBox()
	screen.updatePriorityButton()
	screen.updateAttackPanel()
	return container.NewVBox(
		container.NewCenter(screen.phaseHint),
		container.NewGridWithColumns(len(buttons), buttons...),
		container.NewCenter(screen.passPriority),
		screen.attackPanel,
	)
}

func (screen *BoardScreen) updatePriorityButton() {
	if screen == nil || screen.passPriority == nil {
		return
	}
	if screen.actions.PassPriority != nil &&
		screen.match.MatchStatus == model.StatusInProgress &&
		screen.match.PrioritySequenceOpen &&
		screen.match.PriorityHolder == screen.match.ViewerID {
		screen.passPriority.Enable()
		return
	}
	screen.passPriority.Disable()
}

func (screen *BoardScreen) updateAttackPanel() {
	if screen == nil || screen.attackPanel == nil {
		return
	}
	screen.attackPanel.Objects = nil
	match := screen.match
	if canViewerDecideSageAdvice(match) &&
		(screen.actions.AcceptSageAdvice != nil || screen.actions.DeclineDrawReplacement != nil) {
		screen.renderSageAdvicePanel()
		return
	}
	if match.PendingDraw.Step != model.PendingDrawIdle &&
		match.PendingDraw.PlayerID != "" &&
		match.PendingDraw.PlayerID != match.ViewerID {
		hint := widget.NewLabel("Waiting for opponent's Sage Advice draw decision…")
		hint.Wrapping = fyne.TextWrapWord
		screen.attackPanel.Objects = []fyne.CanvasObject{hint}
		refreshContainerStructure(screen.attackPanel)
		return
	}
	if canViewerDecideBreak(match) &&
		(screen.actions.PlayBreak != nil || screen.actions.DeclineBreak != nil) {
		screen.renderBreakPanel()
		return
	}
	if screen.actions.CorruptOrbs != nil && canViewerCorruptOrb(match) {
		screen.renderOrbChoicePanel()
		return
	}
	if match.PendingBreak.PlayerID != "" && match.PendingBreak.PlayerID != match.ViewerID {
		hint := widget.NewLabel("Waiting for opponent to decide whether to use Break…")
		hint.Wrapping = fyne.TextWrapWord
		screen.attackPanel.Objects = []fyne.CanvasObject{hint}
		refreshContainerStructure(screen.attackPanel)
		return
	}
	if screen.actions.DeclareAttack == nil || !canViewerDeclareAttack(match) {
		refreshContainerStructure(screen.attackPanel)
		return
	}

	attackerLabels := make([]string, 0)
	attackerIDs := make(map[string]model.MatchCardID)
	targetLabels := []string{"Enemy player"}
	targetIDs := make(map[string]model.MatchCardID)
	viewerHasReversedEnemy := false
	for _, player := range match.Players {
		for _, card := range player.ServantZone {
			if card.MatchID == "" {
				continue
			}
			definition := screen.definitions[card.CardID]
			name := strings.TrimSpace(definition.Name)
			if name == "" {
				name = string(card.MatchID)
			}
			if player.ID == match.ViewerID {
				if card.Orientation != model.OrientationRecovered {
					continue
				}
				label := fmt.Sprintf("%s (%s)", name, card.MatchID)
				if cardShowsDoubleCorrupt(card, definition) {
					label += " [Double Corrupt]"
				}
				attackerLabels = append(attackerLabels, label)
				attackerIDs[label] = card.MatchID
				continue
			}
			if card.Orientation == model.OrientationReversed {
				viewerHasReversedEnemy = true
			}
			label := fmt.Sprintf("%s (%s)", name, card.MatchID)
			targetLabels = append(targetLabels, label)
			targetIDs[label] = card.MatchID
		}
	}
	if viewerHasReversedEnemy {
		targetLabels = targetLabels[1:] // remove "Enemy player"
	}

	attackerSelect := widget.NewSelect(attackerLabels, nil)
	attackerSelect.PlaceHolder = "Choose attacker"
	targetSelect := widget.NewSelect(targetLabels, nil)
	targetSelect.PlaceHolder = "Choose target"
	confirm := widget.NewButton("Declare Attack", func() {
		attackerID, ok := attackerIDs[attackerSelect.Selected]
		if !ok {
			return
		}
		if targetSelect.Selected == "Enemy player" {
			screen.actions.DeclareAttack(
				attackerID,
				model.AttackTargetPlayer,
				"",
				match.Revision,
			)
			screen.clearPendingAttacker()
			return
		}
		targetID, ok := targetIDs[targetSelect.Selected]
		if !ok {
			return
		}
		screen.actions.DeclareAttack(
			attackerID,
			model.AttackTargetServant,
			targetID,
			match.Revision,
		)
		screen.clearPendingAttacker()
	})
	confirm.Disable()
	updateConfirm := func(string) {
		if attackerSelect.Selected == "" || targetSelect.Selected == "" {
			confirm.Disable()
			return
		}
		confirm.Enable()
	}
	attackerSelect.OnChanged = updateConfirm
	targetSelect.OnChanged = updateConfirm
	if pending := screen.pendingAttackerID; pending != "" {
		for label, id := range attackerIDs {
			if id == pending {
				attackerSelect.SetSelected(label)
				break
			}
		}
	}
	attackPlayer := widget.NewButton("Attack Enemy Player", func() {
		attackerID := screen.pendingAttackerID
		if attackerID == "" {
			attackerID = attackerIDs[attackerSelect.Selected]
		}
		if attackerID == "" || screen.actions.DeclareAttack == nil || viewerHasReversedEnemy {
			return
		}
		screen.actions.DeclareAttack(
			attackerID,
			model.AttackTargetPlayer,
			"",
			match.Revision,
		)
		screen.clearPendingAttacker()
	})
	updateAttackPlayer := func() {
		attackerReady := screen.pendingAttackerID != "" || attackerSelect.Selected != ""
		if viewerHasReversedEnemy || !attackerReady {
			attackPlayer.Disable()
			return
		}
		attackPlayer.Enable()
	}
	updateAttackPlayer()
	previousAttackerChanged := attackerSelect.OnChanged
	attackerSelect.OnChanged = func(value string) {
		if previousAttackerChanged != nil {
			previousAttackerChanged(value)
		}
		updateAttackPlayer()
	}
	hintText := "Battle: left-click your Servant, then left-click a target (or Attack Enemy Player). Right-click a Servant to grant/clear Double Corrupt."
	if screen.pendingAttackerID != "" {
		hintText = "Attacker selected. Left-click an enemy Servant, or Attack Enemy Player. Right-click grants Double Corrupt."
	}
	hint := widget.NewLabel(hintText)
	hint.Wrapping = fyne.TextWrapWord
	screen.attackPanel.Objects = []fyne.CanvasObject{
		hint,
		attackerSelect,
		targetSelect,
		confirm,
		attackPlayer,
	}
	refreshContainerStructure(screen.attackPanel)
	screen.refreshAttackSelectionHighlights()
}

func (screen *BoardScreen) clearPendingAttacker() {
	if screen == nil {
		return
	}
	screen.pendingAttackerID = ""
	screen.refreshAttackSelectionHighlights()
}

func (screen *BoardScreen) refreshAttackSelectionHighlights() {
	if screen == nil {
		return
	}
	for _, board := range screen.playerBoards {
		if board == nil || board.servants == nil {
			continue
		}
		for _, tile := range collectCardTiles(board.servants) {
			tile.SetSelected(tile.View.MatchID != "" && tile.View.MatchID == screen.pendingAttackerID)
		}
	}
}

func (screen *BoardScreen) handleServantPrimaryTap(
	board *playerBoardController,
	card simulatorview.CardView,
) {
	if screen == nil || board == nil || card.MatchID == "" {
		return
	}
	if screen.actions.DeclareAttack == nil || !canViewerDeclareAttack(screen.match) {
		return
	}
	if board.isViewer {
		if card.Orientation != model.OrientationRecovered {
			return
		}
		if screen.pendingAttackerID == card.MatchID {
			screen.clearPendingAttacker()
			screen.updateAttackPanel()
			return
		}
		screen.pendingAttackerID = card.MatchID
		screen.updateAttackPanel()
		return
	}
	if screen.pendingAttackerID == "" {
		return
	}
	screen.actions.DeclareAttack(
		screen.pendingAttackerID,
		model.AttackTargetServant,
		card.MatchID,
		screen.match.Revision,
	)
	screen.clearPendingAttacker()
}

func canViewerCorruptOrb(match simulatorview.MatchView) bool {
	if match.MatchStatus != model.StatusInProgress ||
		match.Turn.Phase != model.PhaseBattle ||
		match.Turn.ActivePlayer != match.ViewerID ||
		match.Attack.Step != model.BattleStepAwaitingJudgment ||
		match.Attack.TargetKind != model.AttackTargetPlayer ||
		match.PendingBreak.PlayerID != "" {
		return false
	}
	for _, player := range match.Players {
		if player.ID == match.ViewerID {
			continue
		}
		return len(player.Orbs) > 0
	}
	return false
}

func viewerHasMandatoryAttacker(match simulatorview.MatchView) bool {
	if match.MatchStatus != model.StatusInProgress ||
		match.Turn.Phase != model.PhaseBattle ||
		match.Turn.ActivePlayer != match.ViewerID ||
		match.Attack.Step != model.BattleStepIdle ||
		match.PendingBreak.PlayerID != "" {
		return false
	}
	var viewer, opponent *simulatorview.PlayerView
	for index := range match.Players {
		player := &match.Players[index]
		if player.ID == match.ViewerID {
			viewer = player
		} else {
			opponent = player
		}
	}
	if viewer == nil || opponent == nil {
		return false
	}
	opponentHasReversed := false
	for _, card := range opponent.ServantZone {
		if card.Orientation == model.OrientationReversed {
			opponentHasReversed = true
			break
		}
	}
	for _, card := range viewer.ServantZone {
		if card.MatchID == "" || card.Orientation != model.OrientationRecovered {
			continue
		}
		if !opponentHasReversed {
			return true
		}
		if len(opponent.ServantZone) > 0 {
			return true
		}
	}
	return false
}

func canViewerDecideBreak(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress &&
		match.PendingBreak.PlayerID == match.ViewerID &&
		len(match.PendingBreak.CardIDs) > 0
}

func canViewerDecideSageAdvice(match simulatorview.MatchView) bool {
	return match.MatchStatus == model.StatusInProgress &&
		match.PendingDraw.PlayerID == match.ViewerID &&
		match.PendingDraw.Step == model.PendingDrawOffer
}

func (screen *BoardScreen) renderSageAdvicePanel() {
	match := screen.match
	remaining := match.PendingDraw.Remaining
	hint := widget.NewLabel(fmt.Sprintf(
		"Sage Advice: replace the next draw with a dig (look at top cards, keep 1)? Remaining draws in this sequence: %d.",
		remaining,
	))
	hint.Wrapping = fyne.TextWrapWord
	useButton := widget.NewButton("Use Sage Advice", func() {
		if screen.actions.AcceptSageAdvice != nil {
			screen.actions.AcceptSageAdvice(match.Revision)
		}
	})
	declineButton := widget.NewButton("Draw Normally", func() {
		if screen.actions.DeclineDrawReplacement != nil {
			screen.actions.DeclineDrawReplacement(match.Revision)
		}
	})
	screen.attackPanel.Objects = []fyne.CanvasObject{
		hint,
		container.NewHBox(useButton, declineButton),
	}
	refreshContainerStructure(screen.attackPanel)
}

func (screen *BoardScreen) maybePromptSageDig() {
	if screen == nil || screen.actions.PeekDeckTops == nil || screen.actions.ResolveDeckDig == nil {
		return
	}
	match := screen.match
	if match.PendingDraw.Step != model.PendingDrawDig ||
		match.PendingDraw.PlayerID != match.ViewerID {
		return
	}
	if screen.lastSageDigRevision == match.Revision {
		return
	}
	screen.lastSageDigRevision = match.Revision
	count := 3
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			continue
		}
		if player.DeckCount > 0 && player.DeckCount < count {
			count = player.DeckCount
		}
		break
	}
	if count < 1 {
		return
	}
	screen.actions.PeekDeckTops(match.ViewerID, count, func(cards []simulatorview.CardView, err error) {
		if err != nil {
			window := windowForObject(screen.content)
			if window != nil {
				dialog.ShowError(err, window)
			}
			return
		}
		screen.showSageDigDialog(cards)
	})
}

func (screen *BoardScreen) renderBreakPanel() {
	match := screen.match
	cardID := match.PendingBreak.CardIDs[0]
	cardName := string(cardID)
	var definitionType string
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			continue
		}
		for _, card := range player.Hand {
			if card.MatchID != cardID {
				continue
			}
			definition := screen.definitions[card.CardID]
			if name := strings.TrimSpace(definition.Name); name != "" {
				cardName = name
			}
			definitionType = strings.ToLower(strings.TrimSpace(definition.Type))
			break
		}
	}
	hint := widget.NewLabel(fmt.Sprintf(
		"%s entered your hand from an Orb. Use Break to play it immediately without paying its cost? Printed effects stay manual.",
		cardName,
	))
	hint.Wrapping = fyne.TextWrapWord

	orientationSelect := widget.NewSelect([]string{"Recovered", "Reversed"}, nil)
	orientationSelect.SetSelected("Recovered")
	if definitionType != "servant" {
		orientationSelect.Hide()
	}

	useButton := widget.NewButton("Use Break", func() {
		if screen.actions.PlayBreak == nil {
			return
		}
		orientation := model.CardOrientation("")
		if definitionType == "servant" {
			orientation = model.CardOrientation(orientationSelect.Selected)
		}
		screen.actions.PlayBreak(cardID, orientation, match.Revision)
	})
	declineButton := widget.NewButton("Decline Break", func() {
		if screen.actions.DeclineBreak != nil {
			screen.actions.DeclineBreak(match.Revision)
		}
	})
	objects := []fyne.CanvasObject{hint}
	if definitionType == "servant" {
		objects = append(objects, orientationSelect)
	}
	objects = append(objects, container.NewHBox(useButton, declineButton))
	screen.attackPanel.Objects = objects
	refreshContainerStructure(screen.attackPanel)
}

func cardShowsDoubleCorrupt(card simulatorview.CardView, definition cards.Card) bool {
	if card.GrantedDoubleCorrupt {
		return true
	}
	return cardDeclaresDoubleCorruptAbility(definition.Ability)
}

func cardDeclaresDoubleCorruptAbility(ability string) bool {
	for _, line := range strings.Split(ability, "\n") {
		trimmed := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "•*-"))
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			if closing := strings.IndexRune(trimmed, ']'); closing > 1 {
				trimmed = trimmed[1:closing]
			}
		} else if separator := strings.IndexAny(trimmed, "(:,→"); separator >= 0 {
			trimmed = trimmed[:separator]
		} else if strings.ContainsAny(trimmed, ".!?;") {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(trimmed), "Double Corrupt") {
			return true
		}
	}
	return false
}

func (screen *BoardScreen) renderOrbChoicePanel() {
	match := screen.match
	var opponentOrbs []simulatorview.CardView
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			opponentOrbs = player.Orbs
			break
		}
	}
	wantCount := match.Attack.CorruptCount
	if wantCount < 1 {
		wantCount = 1
	}
	if wantCount > len(opponentOrbs) {
		wantCount = len(opponentOrbs)
	}
	labels := make([]string, 0, len(opponentOrbs))
	indexes := make(map[string]int, len(opponentOrbs))
	for index := range opponentOrbs {
		label := fmt.Sprintf("Orb %d", index+1)
		labels = append(labels, label)
		indexes[label] = index
	}

	selected := make(map[string]bool, len(labels))
	confirm := widget.NewButton("Corrupt Orb", nil)
	confirm.Disable()
	updateConfirm := func() {
		count := 0
		for _, on := range selected {
			if on {
				count++
			}
		}
		if count == wantCount {
			confirm.Enable()
			return
		}
		confirm.Disable()
	}
	checks := make([]fyne.CanvasObject, 0, len(labels))
	for _, label := range labels {
		captured := label
		check := widget.NewCheck(captured, func(on bool) {
			selected[captured] = on
			updateConfirm()
		})
		checks = append(checks, check)
	}
	confirm.OnTapped = func() {
		if screen.actions.CorruptOrbs == nil {
			return
		}
		chosen := make([]int, 0, wantCount)
		for _, label := range labels {
			if selected[label] {
				chosen = append(chosen, indexes[label])
			}
		}
		if len(chosen) != wantCount {
			return
		}
		screen.actions.CorruptOrbs(chosen, match.Revision)
	}
	hintText := "Player attack connected. Choose which enemy Orb to corrupt (faces stay hidden)."
	if wantCount > 1 {
		hintText = fmt.Sprintf(
			"Double Corrupt: choose %d enemy Orbs to corrupt simultaneously (faces stay hidden).",
			wantCount,
		)
		confirm.SetText(fmt.Sprintf("Corrupt %d Orbs", wantCount))
	}
	hint := widget.NewLabel(hintText)
	hint.Wrapping = fyne.TextWrapWord
	objects := []fyne.CanvasObject{hint}
	objects = append(objects, checks...)
	objects = append(objects, confirm)
	screen.attackPanel.Objects = objects
	refreshContainerStructure(screen.attackPanel)
}

func (screen *BoardScreen) updatePhaseButtons() {
	match := screen.match
	completionTarget, hasCompletionTarget := prototypePhaseCompletionTarget(match)
	if screen.phaseHint != nil {
		hint := fmt.Sprintf("Current phase: %s", match.Turn.Phase)
		if hasCompletionTarget {
			if match.Turn.Phase == model.PhaseEnd {
				hint += "  •  Select End Turn to continue"
			} else {
				hint += fmt.Sprintf("  •  Select %s to continue", completionTarget)
			}
		}
		if match.Turn.Phase == model.PhaseBattle &&
			match.Turn.ActivePlayer == match.ViewerID &&
			viewerHasMandatoryAttacker(match) {
			hint += "  •  Attack with every able Servant before ending Battle"
		}
		if match.ChaseLinkCount > 0 {
			hint += "  •  Pass priority to resolve; printed effects are manual in this alpha"
		}
		if screen.phaseHint.Text != hint {
			screen.phaseHint.Text = hint
			screen.phaseHint.Refresh()
		}
	}
	for phase, button := range screen.phaseButtons {
		label := string(phase)
		if match.Turn.Phase == model.PhaseEnd && phase == model.PhaseEnd {
			label = "End Turn"
		}
		if button.Text != label {
			button.SetText(label)
		}
		importance := widget.MediumImportance
		if phase == match.Turn.Phase {
			importance = widget.HighImportance
		}
		if button.Importance != importance {
			button.Importance = importance
			button.Refresh()
		}
		legalCompletion :=
			screen.actions.CompleteCurrentPhase != nil &&
				match.MatchStatus == model.StatusInProgress &&
				match.Turn.ActivePlayer == match.ViewerID &&
				!match.PrioritySequenceOpen &&
				hasCompletionTarget &&
				phase == completionTarget
		if legalCompletion &&
			match.Turn.Phase == model.PhaseBattle &&
			viewerHasMandatoryAttacker(match) {
			legalCompletion = false
		}
		if !legalCompletion {
			button.Disable()
		} else {
			button.Enable()
		}
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
	case match.Turn.Phase == model.PhaseMain &&
		match.Turn.Number == 1 &&
		match.Turn.ActivePlayer == match.FirstPlayer:
		return model.PhaseEnd, true
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
		title:         title,
		description:   description,
		actions:       container.NewVBox(),
		manualActions: container.NewVBox(),
		image:         container.NewStack(imageSizer, container.NewCenter(placeholder)),
		imageSizer:    imageSizer,
		shownCardID:   new(model.CardID),
		fullArtwork:   &previewArtworkCache{},
	}
}

func newPreviewRegion(preview previewState, aetherPools fyne.CanvasObject) fyne.CanvasObject {
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
	aetherSection := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabelWithStyle(
			"Aether Pools",
			fyne.TextAlignCenter,
			fyne.TextStyle{Bold: true},
		),
		aetherPools,
	)
	context := container.NewVScroll(container.NewVBox(
		preview.description,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Actions", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		preview.actions,
		preview.manualActions,
	))
	context.SetMinSize(fyne.NewSize(0, 180))
	content := container.NewBorder(heading, aetherSection, nil, nil, context)

	sizer := canvas.NewRectangle(color.Transparent)
	sizer.SetMinSize(fyne.NewSize(previewPanelWidth, 0))
	return container.NewStack(sizer, container.NewPadded(content))
}

func (screen *BoardScreen) updateAetherPools() {
	if screen == nil || screen.aetherPools == nil {
		return
	}
	viewerIndex := screen.viewerIndex()
	opponentIndex := 1 - viewerIndex
	newPool := func(label string, player simulatorview.PlayerView) fyne.CanvasObject {
		heading := widget.NewLabelWithStyle(
			fmt.Sprintf("%s — %s", label, player.ID),
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		)
		display := container.NewHScroll(newAetherPoolDisplay(player.Aether))
		display.SetMinSize(fyne.NewSize(0, aetherIconSize+8))
		return container.NewVBox(heading, display)
	}
	screen.aetherPools.Objects = []fyne.CanvasObject{
		newPool("Opponent", screen.match.Players[opponentIndex]),
		newPool("You", screen.match.Players[viewerIndex]),
	}
	refreshContainerStructure(screen.aetherPools)
}

// refreshContainerStructure recalculates only the changed container's layout
// and schedules one canvas update. Container.Refresh recursively refreshes
// every descendant, which is disproportionately expensive for card-heavy
// simulator fields and needlessly touches the unchanged player field.
func refreshContainerStructure(target *fyne.Container) {
	if target == nil {
		return
	}
	if target.Layout != nil {
		target.Layout.Layout(target.Objects, target.Size())
	}
	canvas.Refresh(target)
}

type playerBoardController struct {
	root            fyne.CanvasObject
	host            *BoardScreen
	playerName      string
	isViewer        bool
	definitions     cardLookup
	preview         previewState
	actions         BoardActions
	currentRevision func() model.Revision
	player          simulatorview.PlayerView
	canCall         bool
	canCast         bool
	canNonElemental bool
	canCasterAether bool
	canUseToken     bool
	orbs            *fyne.Container
	deck            *fyne.Container
	graveyard       *fyne.Container
	exile           *fyne.Container
	servants        *fyne.Container
	barriers        *fyne.Container
	caster          *fyne.Container
	hand            *fyne.Container
}

func newZoneHolder(object fyne.CanvasObject) *fyne.Container {
	return container.NewStack(object)
}

func replaceZone(holder *fyne.Container, object fyne.CanvasObject) {
	if holder == nil || object == nil {
		return
	}
	holder.Objects = []fyne.CanvasObject{object}
	refreshContainerStructure(holder)
}

func newPlayerBoardController(
	host *BoardScreen,
	playerName string,
	player simulatorview.PlayerView,
	isViewer bool,
	definitions cardLookup,
	preview previewState,
	actions BoardActions,
	currentRevision func() model.Revision,
	canCallLevelOne bool,
	canCastCards bool,
	canGenerateNonElementalAether bool,
	canGenerateCasterAether bool,
	canUseCasterToken bool,
) *playerBoardController {
	board := &playerBoardController{
		host:            host,
		playerName:      playerName,
		isViewer:        isViewer,
		definitions:     definitions,
		preview:         preview,
		actions:         actions,
		currentRevision: currentRevision,
		player:          player,
		canCall:         canCallLevelOne,
		canCast:         canCastCards,
		canNonElemental: canGenerateNonElementalAether,
		canCasterAether: canGenerateCasterAether,
		canUseToken:     canUseCasterToken,
	}
	servants, barriers := splitPersistentFieldCards(player.ServantZone, definitions)
	board.orbs = newZoneHolder(board.newOrbZone(player.Orbs))
	board.deck = newZoneHolder(board.newDeckZone(player.DeckCount))
	board.graveyard = newZoneHolder(board.newPileZone(
		"Graveyard",
		"Used and destroyed cards are displayed here.",
		model.ZoneGraveyard,
		player.Graveyard,
	))
	board.exile = newZoneHolder(board.newPileZone(
		"Exile",
		"Cards removed from the game are displayed here.",
		model.ZoneExile,
		player.Exile,
	))
	board.servants = newZoneHolder(board.newInteractiveCardZone(
		"Servant Zone",
		"Servants in play occupy this row.",
		model.ZoneServant,
		servants,
		fyne.NewSize(fieldCardWidth, fieldCardHeight),
		false,
	))
	board.barriers = newZoneHolder(board.newInteractiveCardZone(
		"Barrier Zone",
		"Barriers in play occupy this row.",
		model.ZoneServant,
		barriers,
		fyne.NewSize(fieldCardWidth, fieldCardHeight),
		false,
	))
	board.caster = newZoneHolder(newAetherCasterZone(
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
	))
	if board.host != nil {
		for _, tile := range collectCardTiles(board.caster) {
			board.host.bindCardDrag(tile, model.ZoneCaster, board.player.ID)
		}
	}
	board.hand = newZoneHolder(board.newHandZone(
		player,
		canCallLevelOne,
		canCastCards,
	))

	centerRows := []fyne.CanvasObject{board.hand, board.caster, board.barriers, board.servants}
	pileZones := []fyne.CanvasObject{board.exile, board.graveyard, board.deck}
	if isViewer {
		centerRows = []fyne.CanvasObject{board.servants, board.barriers, board.caster, board.hand}
		pileZones = []fyne.CanvasObject{board.deck, board.graveyard, board.exile}
	}
	pileColumn := withMinimumSize(
		container.NewGridWithRows(len(pileZones), pileZones...),
		fyne.NewSize(sideZoneWidth, 0),
	)
	orbColumn := withMinimumSize(board.orbs, fyne.NewSize(orbZoneWidth, 0))
	leftColumn, rightColumn := pileColumn, orbColumn
	if isViewer {
		leftColumn, rightColumn = orbColumn, pileColumn
	}
	centerZones := container.NewGridWithRows(len(centerRows), centerRows...)
	field := container.NewBorder(nil, nil, leftColumn, rightColumn, centerZones)
	background := canvas.NewRectangle(boardBackground)
	background.StrokeColor = zoneBorder
	background.StrokeWidth = 1
	background.SetMinSize(fyne.NewSize(0, boardMinHeight))

	label := canvas.NewText(fmt.Sprintf("%s Field — %s", playerName, player.ID), boardForeground)
	label.TextStyle = fyne.TextStyle{Bold: true}
	board.root = container.NewStack(
		background,
		container.NewBorder(
			label,
			nil,
			nil,
			nil,
			container.NewPadded(field),
		),
	)
	return board
}

func (board *playerBoardController) update(
	player simulatorview.PlayerView,
	canCall bool,
	canCast bool,
	canNonElemental bool,
	canCasterAether bool,
	canUseToken bool,
) {
	previous := board.player
	if previous.DeckCount != player.DeckCount || !reflect.DeepEqual(previous.Deck, player.Deck) {
		replaceZone(board.deck, board.newDeckZone(player.DeckCount))
	}
	if !reflect.DeepEqual(previous.Orbs, player.Orbs) {
		replaceZone(board.orbs, board.newOrbZone(player.Orbs))
	}
	if !reflect.DeepEqual(previous.Graveyard, player.Graveyard) {
		replaceZone(board.graveyard, board.newPileZone(
			"Graveyard",
			"Used and destroyed cards are displayed here.",
			model.ZoneGraveyard,
			player.Graveyard,
		))
	}
	if !reflect.DeepEqual(previous.Exile, player.Exile) {
		replaceZone(board.exile, board.newPileZone(
			"Exile",
			"Cards removed from the game are displayed here.",
			model.ZoneExile,
			player.Exile,
		))
	}
	if !reflect.DeepEqual(previous.ServantZone, player.ServantZone) {
		servants, barriers := splitPersistentFieldCards(player.ServantZone, board.definitions)
		replaceZone(board.servants, board.newInteractiveCardZone(
			"Servant Zone",
			"Servants in play occupy this row.",
			model.ZoneServant,
			servants,
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
			false,
		))
		replaceZone(board.barriers, board.newInteractiveCardZone(
			"Barrier Zone",
			"Barriers in play occupy this row.",
			model.ZoneServant,
			barriers,
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
			false,
		))
	}
	aetherChangedForCasting := (canCast || board.canCast) && previous.Aether != player.Aether
	// Caster Zone changes must rebuild Call hand so Level Up target options
	// stay current (eligibleLevelUpTargets captures the zone at build time).
	casterChangedForCall := (canCall || board.canCall) && !reflect.DeepEqual(previous.CasterZone, player.CasterZone)
	if !reflect.DeepEqual(previous.Hand, player.Hand) ||
		previous.OpeningHandFinalized != player.OpeningHandFinalized ||
		aetherChangedForCasting ||
		casterChangedForCall ||
		board.canCall != canCall || board.canCast != canCast {
		if board.isViewer && board.preview.actions != nil && len(board.preview.actions.Objects) > 0 {
			board.preview.actions.Objects = nil
			refreshContainerStructure(board.preview.actions)
		}
		replaceZone(board.hand, board.newHandZone(player, canCall, canCast))
	}
	if !reflect.DeepEqual(previous.CasterZone, player.CasterZone) ||
		board.canNonElemental != canNonElemental || board.canCasterAether != canCasterAether || board.canUseToken != canUseToken {
		replaceZone(board.caster, newAetherCasterZone(board.playerName, player, board.isViewer, board.definitions, board.preview, board.actions, board.currentRevision, canNonElemental, canCasterAether, canUseToken))
		if board.host != nil {
			for _, tile := range collectCardTiles(board.caster) {
				board.host.bindCardDrag(tile, model.ZoneCaster, player.ID)
			}
		}
	}
	board.player = player
	board.canCall = canCall
	board.canCast = canCast
	board.canNonElemental = canNonElemental
	board.canCasterAether = canCasterAether
	board.canUseToken = canUseToken
	if board.host != nil {
		board.host.refreshDropTargets()
	}
}

func splitPersistentFieldCards(
	field []simulatorview.CardView,
	definitions cardLookup,
) (servants []simulatorview.CardView, barriers []simulatorview.CardView) {
	for _, card := range field {
		definition, found := definitions[card.CardID]
		if found && strings.EqualFold(strings.TrimSpace(definition.Type), "Barrier") {
			barriers = append(barriers, card)
			continue
		}
		servants = append(servants, card)
	}
	return servants, barriers
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
	canCastCards bool,
) fyne.CanvasObject {
	if !isViewer {
		zone := newCardZone(
			playerName,
			"Hand",
			"Cards currently held by this player.",
			player.Hand,
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
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
		if canCastCards {
			return newCastHandZone(
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
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
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
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
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
		nil,
		nil,
		container.NewVBox(keepButton, replaceButton),
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

	clearLevelUpPanel := func() {
		if preview.actions == nil {
			return
		}
		if len(preview.actions.Objects) == 0 {
			return
		}
		preview.actions.Objects = nil
		refreshContainerStructure(preview.actions)
	}
	// Level Up lives in the preview action strip (same place as Cast payment).
	// The 82px Hand row cannot fit a Select + button without clipping them.
	showLevelUpPanel := func(definition cards.Card) {
		if preview.actions == nil || actions.LevelUpCaster == nil {
			clearLevelUpPanel()
			return
		}
		candidates := eligibleLevelUpTargets(definition, player.CasterZone, definitions)
		if len(candidates) == 0 {
			clearLevelUpPanel()
			return
		}
		targetsByLabel := make(map[string]model.MatchCardID, len(candidates))
		options := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			options = append(options, candidate.label)
			targetsByLabel[candidate.label] = candidate.matchID
		}
		selectedTargetID := model.MatchCardID("")
		var levelUpButton *widget.Button
		levelUpTarget := widget.NewSelect(options, func(label string) {
			selectedTargetID = targetsByLabel[label]
			if levelUpButton == nil {
				return
			}
			if selectedID == "" || selectedTargetID == "" {
				levelUpButton.Disable()
				return
			}
			levelUpButton.Enable()
		})
		levelUpTarget.PlaceHolder = "Choose Caster to level up"
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
		heading := widget.NewLabel("Level Up")
		heading.TextStyle = fyne.TextStyle{Bold: true}
		preview.actions.Objects = []fyne.CanvasObject{
			container.NewVBox(
				heading,
				widget.NewLabel("Place the selected hand Caster onto a matching lower-level Caster."),
				levelUpTarget,
				levelUpButton,
			),
		}
		refreshContainerStructure(preview.actions)
	}

	for _, projectedCard := range player.Hand {
		projectedCard := projectedCard
		definition := definitions[projectedCard.CardID]
		tile := NewCardTile(
			projectedCard,
			definition,
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
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
				clearLevelUpPanel()
				return
			}
			selectedID = projectedCard.MatchID
			tile.SetSelected(true)
			preview.showCard(definition)
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
			showLevelUpPanel(definition)
		}
		tiles = append(tiles, tile)
		objects = append(objects, tile)
	}

	actionButtons := make([]fyne.CanvasObject, 0, 2)
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

	cardRow := container.NewHScroll(container.NewHBox(objects...))
	var content fyne.CanvasObject = cardRow
	if len(actionButtons) > 0 {
		// Horizontal Call buttons under the cards — keeps the 82px strip usable
		// without stacking Level Up controls that need the preview panel.
		content = container.NewBorder(
			nil,
			container.NewHBox(actionButtons...),
			nil,
			nil,
			cardRow,
		)
	}
	return newZone(
		playerName,
		"Hand",
		"Select one card to Call as Level 1, or select a higher Caster to Level Up from the preview panel.",
		content,
		preview,
	)
}

func newCastHandZone(
	playerName string,
	player simulatorview.PlayerView,
	definitions cardLookup,
	preview previewState,
	actions BoardActions,
	currentRevision func() model.Revision,
) fyne.CanvasObject {
	selectedID := model.MatchCardID("")
	selectedDefinition := cards.Card{}
	payment := model.AetherPayment{}
	tiles := make([]*CardTile, 0, len(player.Hand))
	objects := make([]fyne.CanvasObject, 0, len(player.Hand))

	orientation := widget.NewSelect(
		[]string{string(model.OrientationRecovered), string(model.OrientationReversed)},
		nil,
	)
	orientation.PlaceHolder = "Servant orientation"
	orientation.SetSelected(string(model.OrientationRecovered))
	orientation.Hide()

	castButton := widget.NewButton("Cast Selected", func() {
		if selectedID == "" {
			return
		}
		revision := model.Revision(0)
		if currentRevision != nil {
			revision = currentRevision()
		}
		switch strings.ToLower(strings.TrimSpace(selectedDefinition.Type)) {
		case "servant":
			if actions.CastServant != nil {
				actions.CastServant(selectedID, payment, model.CardOrientation(orientation.Selected), revision)
			}
		case "conjure":
			if actions.CastConjure != nil {
				actions.CastConjure(selectedID, payment, revision)
			}
		case "barrier":
			if actions.CastBarrier != nil {
				actions.CastBarrier(selectedID, payment, revision)
			}
		}
	})
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
	paymentSelectors := make([]*widget.Select, 0, len(sources))
	paymentControls := make([]fyne.CanvasObject, 0, len(sources))
	for _, source := range sources {
		if source.available <= 0 {
			continue
		}
		source := source
		options := make([]string, source.available+1)
		for amount := range options {
			options[amount] = strconv.Itoa(amount)
		}
		selector := widget.NewSelect(options, func(selected string) {
			amount, err := strconv.Atoi(selected)
			if err != nil {
				return
			}
			source.set(&payment, amount)
			if updateAction != nil {
				updateAction()
			}
		})
		selector.SetSelected("0")
		paymentSelectors = append(paymentSelectors, selector)
		paymentControls = append(paymentControls, container.NewVBox(widget.NewLabel(source.name), selector))
	}
	var emptyPaymentNotice fyne.CanvasObject
	if len(paymentControls) == 0 {
		emptyPaymentNotice = widget.NewLabel("No Aether (0-cost only)")
	}

	actionAvailable := func(definition cards.Card) bool {
		switch strings.ToLower(strings.TrimSpace(definition.Type)) {
		case "servant":
			return actions.CastServant != nil
		case "conjure":
			return actions.CastConjure != nil
		case "barrier":
			return actions.CastBarrier != nil
		default:
			return false
		}
	}
	updateAction = func() {
		if selectedID == "" || !actionAvailable(selectedDefinition) {
			castButton.Disable()
			return
		}
		cost, err := strconv.Atoi(strings.TrimSpace(selectedDefinition.CostLevel))
		if err != nil || cost < 0 || !paymentMatchesCastDefinition(payment, cost, selectedDefinition.Element) {
			castButton.Disable()
			return
		}
		castButton.Enable()
	}
	orientation.OnChanged = func(string) { updateAction() }

	resetPayment := func() {
		payment = model.AetherPayment{}
		for _, selector := range paymentSelectors {
			selector.SetSelected("0")
		}
	}
	var actionsPanel *container.Scroll
	for _, projectedCard := range player.Hand {
		projectedCard := projectedCard
		definition := definitions[projectedCard.CardID]
		tile := NewCardTile(
			projectedCard,
			definition,
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
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
			selectedID = ""
			selectedDefinition = cards.Card{}
			resetPayment()
			orientation.Hide()
			if wasSelected {
				castButton.SetText("Cast Selected")
				updateAction()
				preview.actions.Objects = nil
				refreshContainerStructure(preview.actions)
				return
			}
			selectedID = projectedCard.MatchID
			selectedDefinition = definition
			tile.SetSelected(true)
			kind := strings.TrimSpace(definition.Type)
			cost := strings.TrimSpace(definition.CostLevel)
			selectionText := fmt.Sprintf("Cast %s (%s %s)", kind, cost, strings.TrimSpace(definition.Element))
			if strings.EqualFold(kind, "Conjure") || strings.EqualFold(kind, "Barrier") {
				selectionText += " — manual effect"
			}
			castButton.SetText(selectionText)
			if strings.EqualFold(kind, "Servant") {
				orientation.SetSelected(string(model.OrientationRecovered))
				orientation.Show()
			}
			updateAction()
			preview.showCard(definition)
			plannedPanel := newSuggestedCastPanel(projectedCard, definition, player, definitions, actions, currentRevision)
			if plannedPanel != nil {
				preview.actions.Objects = []fyne.CanvasObject{plannedPanel}
				refreshContainerStructure(preview.actions)
			} else if actionsPanel != nil {
				preview.actions.Objects = []fyne.CanvasObject{actionsPanel}
				refreshContainerStructure(preview.actions)
			}
		}
		tiles = append(tiles, tile)
		objects = append(objects, tile)
	}

	cardRow := container.NewHScroll(container.NewHBox(objects...))
	actionObjects := append([]fyne.CanvasObject(nil), paymentControls...)
	if emptyPaymentNotice != nil {
		actionObjects = append(actionObjects, emptyPaymentNotice)
	}
	actionObjects = append(actionObjects, orientation, castButton)
	actionsPanel = container.NewHScroll(container.NewHBox(actionObjects...))
	actionsPanel.SetMinSize(fyne.NewSize(280, fieldCardHeight))
	return newZone(
		playerName,
		"Hand",
		"Select a spell, allocate its Aether payment, and cast it.",
		cardRow,
		preview,
	)
}

type suggestedAetherSource struct {
	id         model.MatchCardID
	label      string
	production model.AetherPool
}

func newSuggestedCastPanel(
	card simulatorview.CardView,
	definition cards.Card,
	player simulatorview.PlayerView,
	definitions cardLookup,
	actions BoardActions,
	currentRevision func() model.Revision,
) fyne.CanvasObject {
	kind := strings.ToLower(strings.TrimSpace(definition.Type))
	switch kind {
	case "servant":
		if actions.CastServantWithPlan == nil {
			return nil
		}
	case "conjure":
		if actions.CastConjureWithPlan == nil {
			return nil
		}
	case "barrier":
		if actions.CastBarrierWithPlan == nil {
			return nil
		}
	default:
		return nil
	}
	cost, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	if err != nil || cost < 0 {
		return widget.NewLabel("This card has an invalid casting cost.")
	}
	sources := suggestedSources(player.CasterZone, definitions)
	selected := suggestedSourceSelection(player.Aether, sources, cost, definition.Element)
	checks := make([]*widget.Check, len(sources))
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	orientation := widget.NewSelect([]string{string(model.OrientationRecovered), string(model.OrientationReversed)}, nil)
	orientation.SetSelected(string(model.OrientationRecovered))
	if kind != "servant" {
		orientation.Hide()
	}
	confirm := widget.NewButton("Confirm Suggested Cast", nil)
	var plan model.CastPaymentPlan
	updating := false
	refresh := func() {
		pool := player.Aether
		plan.SourceCardIDs = plan.SourceCardIDs[:0]
		for index, source := range sources {
			if checks[index] == nil || !checks[index].Checked {
				continue
			}
			plan.SourceCardIDs = append(plan.SourceCardIDs, source.id)
			pool = sumAetherPools(pool, source.production)
		}
		payment, valid := suggestedPayment(pool, cost, definition.Element)
		plan.Payment = payment
		if valid {
			status.SetText(fmt.Sprintf("Suggested payment: %s", formatAetherPayment(payment)))
			confirm.Enable()
		} else {
			status.SetText(fmt.Sprintf("Selected sources cannot pay %d %s Aether.", cost, strings.TrimSpace(definition.Element)))
			confirm.Disable()
		}
	}
	for index, source := range sources {
		index := index
		check := widget.NewCheck(source.label, func(bool) {
			if !updating {
				refresh()
			}
		})
		checks[index] = check
	}
	updating = true
	for index, check := range checks {
		check.SetChecked(selected[index])
	}
	updating = false
	refresh()
	confirm.OnTapped = func() {
		if confirm.Disabled() {
			return
		}
		revision := model.Revision(0)
		if currentRevision != nil {
			revision = currentRevision()
		}
		switch kind {
		case "servant":
			actions.CastServantWithPlan(card.MatchID, plan, model.CardOrientation(orientation.Selected), revision)
		case "conjure":
			actions.CastConjureWithPlan(card.MatchID, plan, revision)
		case "barrier":
			actions.CastBarrierWithPlan(card.MatchID, plan, revision)
		}
	}
	sourceObjects := make([]fyne.CanvasObject, 0, len(checks)+1)
	sourceObjects = append(sourceObjects, widget.NewLabel("Suggested sources — adjust as desired:"))
	for _, check := range checks {
		sourceObjects = append(sourceObjects, check)
	}
	if len(checks) == 0 {
		sourceObjects = append(sourceObjects, widget.NewLabel("No recovered Caster sources available."))
	}
	sourceObjects = append(sourceObjects, status, orientation, confirm)
	return container.NewVBox(sourceObjects...)
}

func suggestedSources(casterZone []simulatorview.CardView, definitions cardLookup) []suggestedAetherSource {
	result := make([]suggestedAetherSource, 0, len(casterZone))
	for _, card := range casterZone {
		if card.MatchID == "" || card.Orientation != model.OrientationRecovered {
			continue
		}
		source := suggestedAetherSource{id: card.MatchID}
		switch {
		case card.CardID == model.CasterTokenCardID:
			source.label = "Caster Token — 1 non-elemental"
			source.production.NonElemental = 1
		case card.Face == model.CardFaceDown:
			source.label = fmt.Sprintf("Face-down Caster %s — 1 non-elemental", card.MatchID)
			source.production.NonElemental = 1
		case card.Face == model.CardFaceUp:
			definition, found := definitions[card.CardID]
			amount, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
			if !found || err != nil || amount < 1 || !strings.EqualFold(strings.TrimSpace(definition.Type), "Caster") {
				continue
			}
			if !setPoolElement(&source.production, definition.Element, amount) {
				continue
			}
			source.label = fmt.Sprintf("%s — %d %s", strings.TrimSpace(definition.Name), amount, strings.TrimSpace(definition.Element))
		default:
			continue
		}
		result = append(result, source)
	}
	return result
}

func suggestedSourceSelection(pool model.AetherPool, sources []suggestedAetherSource, cost int, element string) []bool {
	selected := make([]bool, len(sources))
	if cost <= 0 {
		return selected
	}
	if poolElement(pool, element) < 1 {
		for index, source := range sources {
			if poolElement(source.production, element) > 0 {
				selected[index] = true
				pool = sumAetherPools(pool, source.production)
				break
			}
		}
	}
	for aetherPoolTotal(pool) < cost {
		best := -1
		bestAmount := 0
		for index, source := range sources {
			amount := aetherPoolTotal(source.production)
			if !selected[index] && amount > bestAmount {
				best, bestAmount = index, amount
			}
		}
		if best == -1 {
			break
		}
		selected[best] = true
		pool = sumAetherPools(pool, sources[best].production)
	}
	return selected
}

func suggestedPayment(pool model.AetherPool, cost int, element string) (model.AetherPayment, bool) {
	if cost < 0 || aetherPoolTotal(pool) < cost || (cost > 0 && poolElement(pool, element) < 1) {
		return model.AetherPayment{}, false
	}
	payment := model.AetherPayment{}
	remaining := cost
	if cost > 0 {
		if !setPaymentElement(&payment, element, 1) {
			return model.AetherPayment{}, false
		}
		setPoolElement(&pool, element, poolElement(pool, element)-1)
		remaining--
	}
	spend := func(available int, destination *int) {
		amount := min(available, remaining)
		*destination += amount
		remaining -= amount
	}
	spend(pool.NonElemental, &payment.NonElemental)
	spend(pool.Aes, &payment.Aes)
	spend(pool.Aqua, &payment.Aqua)
	spend(pool.Ignus, &payment.Ignus)
	spend(pool.Luna, &payment.Luna)
	spend(pool.Silva, &payment.Silva)
	spend(pool.Solis, &payment.Solis)
	spend(pool.Terra, &payment.Terra)
	spend(pool.Void, &payment.Void)
	return payment, remaining == 0
}

func aetherPoolTotal(pool model.AetherPool) int {
	return pool.Aes + pool.Aqua + pool.Ignus + pool.Luna + pool.Silva + pool.Solis + pool.Terra + pool.Void + pool.NonElemental
}

func sumAetherPools(left, right model.AetherPool) model.AetherPool {
	left.Aes += right.Aes
	left.Aqua += right.Aqua
	left.Ignus += right.Ignus
	left.Luna += right.Luna
	left.Silva += right.Silva
	left.Solis += right.Solis
	left.Terra += right.Terra
	left.Void += right.Void
	left.NonElemental += right.NonElemental
	return left
}

func poolElement(pool model.AetherPool, element string) int {
	switch strings.ToLower(strings.TrimSpace(element)) {
	case "aes":
		return pool.Aes
	case "aqua":
		return pool.Aqua
	case "ignus":
		return pool.Ignus
	case "luna":
		return pool.Luna
	case "silva":
		return pool.Silva
	case "solis":
		return pool.Solis
	case "terra":
		return pool.Terra
	case "void":
		return pool.Void
	default:
		return 0
	}
}

func setPoolElement(pool *model.AetherPool, element string, amount int) bool {
	switch strings.ToLower(strings.TrimSpace(element)) {
	case "aes":
		pool.Aes = amount
	case "aqua":
		pool.Aqua = amount
	case "ignus":
		pool.Ignus = amount
	case "luna":
		pool.Luna = amount
	case "silva":
		pool.Silva = amount
	case "solis":
		pool.Solis = amount
	case "terra":
		pool.Terra = amount
	case "void":
		pool.Void = amount
	default:
		return false
	}
	return true
}

func setPaymentElement(payment *model.AetherPayment, element string, amount int) bool {
	switch strings.ToLower(strings.TrimSpace(element)) {
	case "aes":
		payment.Aes = amount
	case "aqua":
		payment.Aqua = amount
	case "ignus":
		payment.Ignus = amount
	case "luna":
		payment.Luna = amount
	case "silva":
		payment.Silva = amount
	case "solis":
		payment.Solis = amount
	case "terra":
		payment.Terra = amount
	case "void":
		payment.Void = amount
	default:
		return false
	}
	return true
}

func formatAetherPayment(payment model.AetherPayment) string {
	parts := make([]string, 0, 9)
	for _, entry := range []struct {
		name   string
		amount int
	}{
		{"Aes", payment.Aes}, {"Aqua", payment.Aqua}, {"Ignus", payment.Ignus}, {"Luna", payment.Luna},
		{"Silva", payment.Silva}, {"Solis", payment.Solis}, {"Terra", payment.Terra}, {"Void", payment.Void},
		{"non-elemental", payment.NonElemental},
	} {
		if entry.amount > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", entry.amount, entry.name))
		}
	}
	if len(parts) == 0 {
		return "0 Aether"
	}
	return strings.Join(parts, ", ")
}

func paymentMatchesCastDefinition(payment model.AetherPayment, cost int, element string) bool {
	total := payment.Aes + payment.Aqua + payment.Ignus + payment.Luna + payment.Silva +
		payment.Solis + payment.Terra + payment.Void + payment.NonElemental
	if total != cost {
		return false
	}
	if cost == 0 {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(element)) {
	case "aes":
		return payment.Aes > 0
	case "aqua":
		return payment.Aqua > 0
	case "ignus":
		return payment.Ignus > 0
	case "luna":
		return payment.Luna > 0
	case "silva":
		return payment.Silva > 0
	case "solis":
		return payment.Solis > 0
	case "terra":
		return payment.Terra > 0
	case "void":
		return payment.Void > 0
	default:
		return false
	}
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
	labelsTaken := make(map[string]struct{})
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
		label := fmt.Sprintf("%s (Level %d)", identity, targetLevel)
		if _, taken := labelsTaken[label]; taken {
			label = fmt.Sprintf("%s · %s", label, target.MatchID)
		}
		labelsTaken[label] = struct{}{}
		result = append(result, levelUpTargetOption{
			label:   label,
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
	return newCardZoneWithMinimum(
		playerName,
		zoneName,
		description,
		cardViews,
		tileSize,
		vertical,
		sideways,
		definitions,
		preview,
		fyne.Size{},
	)
}

func newCompactCardZone(
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
	return newCardZoneWithMinimum(
		playerName,
		zoneName,
		description,
		cardViews,
		tileSize,
		vertical,
		sideways,
		definitions,
		preview,
		fyne.NewSize(0, utilityZoneHeight),
	)
}

// verticalCardStackLayout keeps the Orb zone readable without introducing a
// nested scrollbar. Each landscape card exposes part of the card below it,
// while the step contracts when the available height is unusually small.
type verticalCardStackLayout struct {
	step        float32
	alignBottom bool
}

func (stack *verticalCardStackLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}
	width := float32(0)
	height := float32(0)
	for _, object := range objects {
		minimum := object.MinSize()
		if minimum.Width > width {
			width = minimum.Width
		}
		if minimum.Height > height {
			height = minimum.Height
		}
	}
	return fyne.NewSize(width, height+stack.step*float32(len(objects)-1))
}

func (stack *verticalCardStackLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	cardSize := objects[0].MinSize()
	for _, object := range objects[1:] {
		minimum := object.MinSize()
		if minimum.Width > cardSize.Width {
			cardSize.Width = minimum.Width
		}
		if minimum.Height > cardSize.Height {
			cardSize.Height = minimum.Height
		}
	}
	step := stack.step
	if len(objects) > 1 {
		availableStep := (size.Height - cardSize.Height) / float32(len(objects)-1)
		if availableStep < step {
			step = availableStep
		}
	}
	if step < 0 {
		step = 0
	}
	stackHeight := cardSize.Height + step*float32(len(objects)-1)
	startY := float32(0)
	if stack.alignBottom && size.Height > stackHeight {
		startY = size.Height - stackHeight
	}
	x := (size.Width - cardSize.Width) / 2
	if x < 0 {
		x = 0
	}
	for index, object := range objects {
		object.Resize(cardSize)
		object.Move(fyne.NewPos(x, startY+float32(index)*step))
	}
}

func (board *playerBoardController) newOrbZone(cards []simulatorview.CardView) fyne.CanvasObject {
	var onSecondary func(*fyne.PointEvent)
	canPeek := !board.isViewer && board.actions.PeekOrb != nil && len(cards) > 0
	canReveal := board.isViewer && board.actions.RevealOrb != nil && len(cards) > 0
	if canPeek || canReveal {
		onSecondary = func(event *fyne.PointEvent) {
			board.showOrbContextMenu(event)
		}
	}
	return newLayeredOrbZone(
		board.playerName,
		cards,
		board.definitions,
		board.preview,
		!board.isViewer,
		onSecondary,
	)
}

func (board *playerBoardController) showOrbContextMenu(event *fyne.PointEvent) {
	if board == nil || board.orbs == nil || event == nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(board.orbs)
	if canvas == nil {
		return
	}
	items := make([]*fyne.MenuItem, 0, 2)
	if board.isViewer && board.actions.RevealOrb != nil && len(board.player.Orbs) > 0 {
		items = append(items, fyne.NewMenuItem("Reveal Orb to opponent...", board.beginRevealOrbPrompt))
	}
	if !board.isViewer && board.actions.PeekOrb != nil && len(board.player.Orbs) > 0 {
		items = append(items, fyne.NewMenuItem("Peek Orb (remember)...", board.beginPeekOrbPrompt))
	}
	if len(items) == 0 {
		return
	}
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), canvas, event.AbsolutePosition)
}

func newLayeredOrbZone(
	playerName string,
	cardViews []simulatorview.CardView,
	definitions cardLookup,
	preview previewState,
	alignBottom bool,
	onSecondary func(*fyne.PointEvent),
) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(cardViews))
	for _, projectedCard := range cardViews {
		definition := definitions[projectedCard.CardID]
		onHidden := func() { preview.showHiddenCard(playerName, "Orb Zone") }
		onShow := preview.showCard
		if projectedCard.CardID != "" {
			captured := definition
			onHidden = func() { preview.showCard(captured) }
		}
		tile := newOrientedCardTile(
			projectedCard,
			definition,
			fyne.NewSize(orbCardWidth, orbCardHeight),
			onShow,
			onHidden,
			true,
		)
		objects = append(objects, tile)
	}

	content := fyne.CanvasObject(layout.NewSpacer())
	if len(objects) > 0 {
		content = container.New(&verticalCardStackLayout{
			step:        orbLayerStep,
			alignBottom: alignBottom,
		}, objects...)
	}
	zone := newZoneWithMinimum(
		playerName,
		"Orb Zone",
		"Face-down Orbs. You only see identities you already knew (hand-placed, peeked, or revealed).",
		content,
		preview,
		fyne.Size{},
	)
	if onSecondary != nil {
		return newZoneInteractLayer(zone, nil, onSecondary)
	}
	return zone
}

func newCardZoneWithMinimum(
	playerName string,
	zoneName string,
	description string,
	cardViews []simulatorview.CardView,
	tileSize fyne.Size,
	vertical bool,
	sideways bool,
	definitions cardLookup,
	preview previewState,
	minimum fyne.Size,
) fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 0, len(cardViews))
	for _, projectedCard := range cardViews {
		definition := definitions[projectedCard.CardID]
		tile := newOrientedCardTile(
			projectedCard,
			definition,
			tileSize,
			preview.showCard,
			func() { preview.showHiddenCard(playerName, zoneName) },
			sideways || projectedCard.Orientation == model.OrientationRested,
		)
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
	if minimum.IsZero() {
		return newZone(playerName, zoneName, description, content, preview)
	}
	return newZoneWithMinimum(playerName, zoneName, description, content, preview, minimum)
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
	objects := make([]fyne.CanvasObject, 0, len(player.CasterZone))
	hasEligibleCard := false
	for _, projectedCard := range player.CasterZone {
		projectedCard := projectedCard
		definition := definitions[projectedCard.CardID]
		tile := newOrientedCardTile(
			projectedCard,
			definition,
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
			preview.showCard,
			func() { preview.showHiddenCard(playerName, "Caster Zone") },
			projectedCard.Orientation == model.OrientationRested,
		)
		if eligibleToken(projectedCard) || eligibleFaceUpCaster(projectedCard) || eligibleFaceDownCaster(projectedCard) {
			hasEligibleCard = true
			tile.OnActivate = func() {
				revision := model.Revision(0)
				if currentRevision != nil {
					revision = currentRevision()
				}
				switch {
				case eligibleToken(projectedCard):
					if actions.UseCasterToken != nil {
						actions.UseCasterToken(projectedCard.MatchID, revision)
					}
				case eligibleFaceUpCaster(projectedCard):
					if actions.GenerateCasterAether != nil {
						actions.GenerateCasterAether(projectedCard.MatchID, revision)
					}
				default:
					if actions.GenerateNonElementalAether != nil {
						actions.GenerateNonElementalAether(projectedCard.MatchID, revision)
					}
				}
			}
		}
		objects = append(objects, tile)
	}

	cardRow := fyne.CanvasObject(container.NewHBox(objects...))
	content := cardRow
	if hasEligibleCard {
		instruction := widget.NewLabel("Tap a recovered Caster to produce Aether")
		instruction.Wrapping = fyne.TextWrapWord
		content = container.NewBorder(nil, nil, nil, instruction, cardRow)
	}
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
			fyne.NewSize(fieldCardWidth, fieldCardHeight),
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

func (board *playerBoardController) newDeckZone(count int) fyne.CanvasObject {
	var onSecondary func(*fyne.PointEvent)
	canOwnMenu := board.isViewer && (board.actions.DrawCards != nil || board.host != nil || board.actions.PeekDeckTops != nil)
	canEnemyPeek := !board.isViewer && board.actions.PeekDeckTops != nil
	if canOwnMenu || canEnemyPeek {
		onSecondary = func(event *fyne.PointEvent) {
			board.showDeckContextMenu(event)
		}
	}
	return newCompactDeckZone(board.playerName, count, board.preview, onSecondary)
}

func (board *playerBoardController) newHandZone(
	player simulatorview.PlayerView,
	canCallLevelOne bool,
	canCastCards bool,
) fyne.CanvasObject {
	zone := newHandZone(
		board.playerName,
		player,
		board.isViewer,
		board.definitions,
		board.preview,
		board.actions,
		board.currentRevision,
		canCallLevelOne,
		canCastCards,
	)
	if board.host != nil && player.OpeningHandFinalized {
		for _, tile := range collectCardTiles(zone) {
			board.host.bindCardDrag(tile, model.ZoneHand, player.ID)
		}
	}
	return zone
}

func (board *playerBoardController) newPileZone(
	zoneName string,
	description string,
	zone model.Zone,
	cards []simulatorview.CardView,
) fyne.CanvasObject {
	openBrowser := func() {
		if board.host == nil {
			return
		}
		board.host.showPileViewer(pileViewerRequest{
			title:    fmt.Sprintf("%s %s", board.playerName, zoneName),
			zone:     zone,
			playerID: board.player.ID,
			cards:    cards,
			canMove:  board.isViewer && board.actions.MoveCard != nil,
		})
	}
	content := newCompactCardZone(
		board.playerName,
		zoneName,
		description,
		cards,
		fyne.NewSize(utilityCardWidth, utilityCardHeight),
		false,
		false,
		board.definitions,
		board.preview,
	)
	for _, tile := range collectCardTiles(content) {
		tile.OnActivate = openBrowser
		if board.host != nil {
			board.host.bindCardDrag(tile, zone, board.player.ID)
		}
	}
	return newZoneInteractLayer(content, openBrowser, nil)
}

func (board *playerBoardController) newInteractiveCardZone(
	zoneName string,
	description string,
	zone model.Zone,
	cards []simulatorview.CardView,
	tileSize fyne.Size,
	vertical bool,
) fyne.CanvasObject {
	content := newCardZone(
		board.playerName,
		zoneName,
		description,
		cards,
		tileSize,
		vertical,
		false,
		board.definitions,
		board.preview,
	)
	for _, tile := range collectCardTiles(content) {
		if board.host != nil {
			board.host.bindCardDrag(tile, zone, board.player.ID)
		}
		if zoneName != "Servant Zone" {
			continue
		}
		captured := tile.View
		tile.OnActivate = func() {
			if board.host != nil {
				board.host.handleServantPrimaryTap(board, captured)
			}
		}
		if board.actions.SetGrantedDoubleCorrupt != nil {
			tile.OnSecondaryActivate = func(event *fyne.PointEvent) {
				board.showServantContextMenu(captured, event)
			}
		}
	}
	return content
}

func (board *playerBoardController) showServantContextMenu(
	card simulatorview.CardView,
	event *fyne.PointEvent,
) {
	if board == nil || event == nil || card.MatchID == "" || board.actions.SetGrantedDoubleCorrupt == nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(board.servants)
	if canvas == nil {
		return
	}
	label := "Grant Double Corrupt"
	enabled := true
	if card.GrantedDoubleCorrupt {
		label = "Clear granted Double Corrupt"
		enabled = false
	}
	item := fyne.NewMenuItem(label, func() {
		board.actions.SetGrantedDoubleCorrupt(card.MatchID, enabled, board.currentRevision())
	})
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", item), canvas, event.AbsolutePosition)
}

func (screen *BoardScreen) refreshDropTargets() {
	if screen == nil {
		return
	}
	screen.clearDropTargets()
	for _, board := range screen.playerBoards {
		if board == nil {
			continue
		}
		screen.registerDropTarget(board.hand, board.player.ID, model.ZoneHand)
		screen.registerDropTarget(board.deck, board.player.ID, model.ZoneDeck)
		screen.registerDropTarget(board.graveyard, board.player.ID, model.ZoneGraveyard)
		screen.registerDropTarget(board.exile, board.player.ID, model.ZoneExile)
		screen.registerDropTarget(board.caster, board.player.ID, model.ZoneCaster)
		screen.registerDropTarget(board.servants, board.player.ID, model.ZoneServant)
		screen.registerDropTarget(board.barriers, board.player.ID, model.ZoneServant)
	}
}

func (board *playerBoardController) showDeckContextMenu(event *fyne.PointEvent) {
	if board == nil || board.deck == nil || event == nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(board.deck)
	if canvas == nil {
		return
	}
	items := make([]*fyne.MenuItem, 0, 4)
	if board.isViewer {
		if board.actions.DrawCards != nil {
			items = append(items, fyne.NewMenuItem("Draw X cards...", board.beginDrawCardsPrompt))
		}
		if board.host != nil {
			items = append(items, fyne.NewMenuItem("Browse deck...", board.beginBrowseDeck))
		}
		if board.actions.PeekDeckTops != nil {
			items = append(items, fyne.NewMenuItem("Dig top 3 (keep 1)...", board.beginSageDig))
		}
	} else if board.actions.PeekDeckTops != nil {
		items = append(items, fyne.NewMenuItem("Look at top card...", board.beginLookAtEnemyTop))
	}
	if len(items) == 0 {
		return
	}
	widget.ShowPopUpMenuAtPosition(fyne.NewMenu("", items...), canvas, event.AbsolutePosition)
}

func (board *playerBoardController) beginBrowseDeck() {
	if board == nil || board.host == nil {
		return
	}
	board.host.showPileViewer(pileViewerRequest{
		title:    fmt.Sprintf("%s Deck", board.playerName),
		zone:     model.ZoneDeck,
		playerID: board.player.ID,
		cards:    board.player.Deck,
		canMove:  board.isViewer && board.actions.MoveCard != nil,
	})
}

func (board *playerBoardController) beginDrawCardsPrompt() {
	if board == nil || board.actions.DrawCards == nil {
		return
	}
	revision := board.currentRevision()
	promptDrawCardCount(windowForObject(board.deck), func(count int) {
		board.actions.DrawCards(count, revision)
	})
}

var promptDrawCardCount = showDrawCardCountDialog

func showDrawCardCountDialog(window fyne.Window, onConfirm func(int)) {
	if window == nil || onConfirm == nil {
		return
	}
	entry := widget.NewEntry()
	entry.SetText("1")
	entry.Validator = func(text string) error {
		if _, err := parseDrawCount(text); err != nil {
			return err
		}
		return nil
	}
	showScaledForm(
		"Draw cards",
		"Draw",
		"Cancel",
		[]*widget.FormItem{widget.NewFormItem("How many?", entry)},
		func(ok bool) {
			if !ok {
				return
			}
			count, err := parseDrawCount(entry.Text)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			onConfirm(count)
		},
		window,
		0.4, 0.3, 420, 220, 720, 420,
	)
}

func parseDrawCount(text string) (int, error) {
	count, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, fmt.Errorf("enter a whole number of cards")
	}
	if count < 1 {
		return 0, fmt.Errorf("draw at least 1 card")
	}
	return count, nil
}

func windowForObject(object fyne.CanvasObject) fyne.Window {
	if object == nil || fyne.CurrentApp() == nil || fyne.CurrentApp().Driver() == nil {
		return nil
	}
	objectCanvas := fyne.CurrentApp().Driver().CanvasForObject(object)
	if objectCanvas == nil {
		return nil
	}
	for _, window := range fyne.CurrentApp().Driver().AllWindows() {
		if window != nil && window.Canvas() == objectCanvas {
			return window
		}
	}
	return nil
}

func newCompactDeckZone(
	playerName string,
	count int,
	preview previewState,
	onSecondary func(*fyne.PointEvent),
) fyne.CanvasObject {
	content := fyne.CanvasObject(layout.NewSpacer())
	if count > 0 {
		cardBack := NewCardTile(
			simulatorview.CardView{Face: model.CardFaceDown},
			cards.Card{},
			fyne.NewSize(utilityCardWidth, utilityCardHeight),
			preview.showCard,
			func() { preview.showHiddenCard(playerName, "Deck Zone") },
		)
		cardBack.OnSecondaryActivate = onSecondary
		countLabel := widget.NewLabel(fmt.Sprintf("%d", count))
		countLabel.Alignment = fyne.TextAlignCenter
		content = container.NewCenter(container.NewHBox(cardBack, countLabel))
	}
	zone := newZoneWithMinimum(
		playerName,
		"Deck Zone",
		fmt.Sprintf("%d cards remain in this deck.", count),
		content,
		preview,
		fyne.NewSize(0, utilityZoneHeight),
	)
	if onSecondary == nil {
		return zone
	}
	return newZoneInteractLayer(zone, nil, onSecondary)
}

func newZone(
	playerName string,
	zoneName string,
	description string,
	content fyne.CanvasObject,
	preview previewState,
) fyne.CanvasObject {
	minimum := fyne.NewSize(0, casterZoneHeight)
	if zoneName == "Hand" {
		minimum.Height = handZoneHeight
	}
	return newZoneWithMinimum(playerName, zoneName, description, content, preview, minimum)
}

func newZoneWithMinimum(
	_ string,
	_ string,
	_ string,
	content fyne.CanvasObject,
	_ previewState,
	minimum fyne.Size,
) fyne.CanvasObject {
	background := canvas.NewRectangle(zoneBackground)
	background.StrokeColor = zoneBorder
	background.StrokeWidth = 1

	return withMinimumSize(
		container.NewStack(
			background,
			container.New(layout.NewCustomPaddedLayout(3, 3, 4, 4), content),
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
	cardID := model.CardID(card.ID)
	if preview.shownCardID != nil && *preview.shownCardID == cardID {
		return
	}
	if preview.shownCardID != nil {
		*preview.shownCardID = cardID
	}
	preview.title.SetText(card.Name)
	preview.description.SetText(fmt.Sprintf(
		"Type: %s\nElement: %s\nCost/Lv: %s\nTraits: %s\n\n%s",
		card.Type,
		card.Element,
		card.CostLevel,
		card.Traits,
		card.Ability,
	))

	immediatePath, immediateFound := previewImagePath(card.ID)
	if immediateFound {
		if cached, loaded := cachedCardImage(immediatePath); loaded {
			preview.setArtwork(newSimulatorCanvasImage(cached))
		} else {
			preview.setArtwork(centeredPreviewMessage("Loading artwork…"))
		}
	} else {
		preview.setArtwork(centeredPreviewMessage("Loading artwork…"))
	}

	fullPath, fullFound := cardimages.Find(card.ID)
	if !fullFound {
		if !immediateFound {
			preview.setArtwork(centeredPreviewMessage("Image unavailable"))
		}
		return
	}
	if preview.fullArtwork != nil && preview.fullArtwork.cardID == cardID && preview.fullArtwork.artwork != nil {
		preview.setArtwork(newSimulatorCanvasImage(preview.fullArtwork.artwork))
		return
	}
	go func(requestedID model.CardID, imagePath string) {
		artwork, loaded := decodeCardImage(imagePath)
		if !loaded {
			return
		}
		fyne.Do(func() {
			if preview.shownCardID == nil || *preview.shownCardID != requestedID {
				return
			}
			if preview.fullArtwork != nil {
				preview.fullArtwork.cardID = requestedID
				preview.fullArtwork.artwork = artwork
			}
			preview.setArtwork(newSimulatorCanvasImage(artwork))
		})
	}(cardID, fullPath)
}

func (preview previewState) setArtwork(artwork fyne.CanvasObject) {
	preview.image.Objects = []fyne.CanvasObject{preview.imageSizer, artwork}
	refreshContainerStructure(preview.image)
}

func centeredPreviewMessage(text string) fyne.CanvasObject {
	message := widget.NewLabel(text)
	message.Alignment = fyne.TextAlignCenter
	return container.NewCenter(message)
}

func previewImagePath(cardID string) (string, bool) {
	if imagePath, found := cardimages.FindThumbnail(cardID); found {
		return imagePath, true
	}
	return cardimages.Find(cardID)
}

func (preview previewState) showHiddenCard(playerName, zoneName string) {
	if preview.shownCardID != nil {
		*preview.shownCardID = ""
	}
	preview.title.SetText("Concealed Card")
	preview.description.SetText(
		fmt.Sprintf("%s's %s card is hidden from this viewer.", playerName, zoneName),
	)
	preview.setArtwork(cardBackImage())
}
