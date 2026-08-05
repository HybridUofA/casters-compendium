package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestNewBoardScreenContainsBothPlayerFieldsAndRequiredZones(t *testing.T) {
	match := testMatchView()
	match.Revision = 7
	screen := NewBoardScreen(match, testDefinitions(), BoardActions{}, nil)

	for _, text := range []string{
		"Opponent Field — player-two",
		"Player Field — player-one",
		"Orb Zone",
		"Deck Zone",
		"Graveyard",
		"Exile",
		"Servant Zone",
		"Caster Zone",
		"Hand",
		"Card Information",
		"Card Preview",
		"Turn 0 • Call • Revision 7 • Active player: player-one",
	} {
		if !containsText(screen, text) {
			t.Errorf("board screen does not contain %q", text)
		}
	}
}

func TestSelectingZoneUpdatesCardInformationPanel(t *testing.T) {
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	servantButton := findButton(screen, "Servant Zone")
	if servantButton == nil {
		t.Fatal("board screen does not contain a Servant Zone button")
	}

	test.Tap(servantButton)

	if !containsText(screen, "Opponent — Servant Zone") {
		t.Fatal("selecting opponent Servant Zone did not update preview title")
	}
	if !containsText(screen, "Servants and barriers occupy this field.") {
		t.Fatal("selecting Servant Zone did not update preview description")
	}
}

func TestProjectedServantGraveyardAndExileCardsRender(t *testing.T) {
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	if findCardTile(screen, "visible-servant-card") == nil {
		t.Fatal("projected Servant Zone card was not rendered")
	}
	if findCardTile(screen, "visible-grave-card") == nil {
		t.Fatal("projected Graveyard card was not rendered")
	}
	if findCardTile(screen, "visible-exile-card") == nil {
		t.Fatal("projected Exile card was not rendered")
	}
}

func TestAetherPoolRenderingOnlyIncludesPositiveTypes(t *testing.T) {
	pool := model.AetherPool{
		Aes:          2,
		Void:         1,
		NonElemental: 3,
	}
	entries := visibleAetherEntries(pool)
	want := []struct {
		name   string
		amount int
	}{
		{name: "Aes", amount: 2},
		{name: "Void", amount: 1},
		{name: "Non-elemental", amount: 3},
	}
	if len(entries) != len(want) {
		t.Fatalf("visible Aether entry count = %d; want %d", len(entries), len(want))
	}
	for index, expected := range want {
		if entries[index].name != expected.name || entries[index].amount != expected.amount {
			t.Fatalf("visible Aether entry %d = %s/%d; want %s/%d", index, entries[index].name, entries[index].amount, expected.name, expected.amount)
		}
		if len(entries[index].resource.Content()) == 0 {
			t.Fatalf("visible Aether entry %q has no embedded icon", entries[index].name)
		}
	}
}

func TestEmptyAetherPoolUsesCompactZeroState(t *testing.T) {
	display := newAetherPoolDisplay(model.AetherPool{})
	if !containsText(display, "Aether: 0") {
		t.Fatal("empty Aether pool did not render its compact zero state")
	}
}

func TestGenerateNonElementalAetherSubmitsOwnCasterAndCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.ViewerID = "player-two"
	match.MatchStatus = model.StatusInProgress
	match.Revision = 15
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseBattle
	match.Turn.ActivePlayer = "player-one"
	match.Players[0].CasterZone[1].MatchID = ""
	match.Players[0].CasterZone[1].CardID = ""
	match.Players[1].CasterZone[1].MatchID = "player-two-facedown-caster"
	match.Players[1].CasterZone[1].CardID = "face-down-caster-card"
	generatedBy := model.MatchCardID("")
	generatedRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			GenerateNonElementalAether: func(cardID model.MatchCardID, revision model.Revision) {
				generatedBy = cardID
				generatedRevision = revision
			},
		},
		nil,
	)
	generateButton := findButton(screen, "Produce 1 Aether")
	if generateButton == nil {
		t.Fatal("Aether control was not rendered for the non-active player's eligible Caster")
	}
	if generateButton.Visible() {
		t.Fatal("Aether control is visible before selecting a Caster")
	}
	caster := findCardTile(screen, "face-down-caster-card")
	if caster == nil {
		t.Fatal("viewer's known face-down Caster was not rendered")
	}

	test.Tap(caster)
	generateButton = findButton(screen, "Rest Selected for 1 Aether")
	if !generateButton.Visible() || generateButton.Disabled() {
		t.Fatal("Aether control did not appear and enable after selecting an eligible Caster")
	}
	test.Tap(caster)
	if generateButton.Visible() {
		t.Fatal("Aether control remained visible after deselecting the Caster")
	}
	test.Tap(caster)
	test.Tap(generateButton)

	if generatedBy != "player-two-facedown-caster" || generatedRevision != 15 {
		t.Fatalf("Aether action submitted Caster/revision %q/%d; want player-two-facedown-caster/15", generatedBy, generatedRevision)
	}
}

func TestGenerateNonElementalAetherControlRequiresEligibleCaster(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*simulatorview.MatchView)
	}{
		{
			name: "match is not in progress",
			mutate: func(match *simulatorview.MatchView) {
				match.MatchStatus = model.StatusSetup
			},
		},
		{
			name: "Caster is Rested",
			mutate: func(match *simulatorview.MatchView) {
				match.Players[0].CasterZone[1].Orientation = model.OrientationRested
			},
		},
		{
			name: "Caster is face up",
			mutate: func(match *simulatorview.MatchView) {
				match.Players[0].CasterZone[1].Face = model.CardFaceUp
				match.Players[0].CasterZone[1].ShowFace = true
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			testCase.mutate(&match)
			screen := NewBoardScreen(
				match,
				testDefinitions(),
				BoardActions{GenerateNonElementalAether: func(model.MatchCardID, model.Revision) {}},
				nil,
			)
			if findButton(screen, "Rest Selected for 1 Aether") != nil {
				t.Fatal("Aether control was rendered without an eligible Caster")
			}
		})
	}
}

func TestUseCasterTokenControlAppearsAfterSelectionAndSubmitsCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.ViewerID = "player-two"
	match.MatchStatus = model.StatusInProgress
	match.Revision = 16
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseBattle
	match.Turn.ActivePlayer = "player-one"
	usedToken := model.MatchCardID("")
	usedRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			UseCasterToken: func(tokenID model.MatchCardID, revision model.Revision) {
				usedToken = tokenID
				usedRevision = revision
			},
		},
		nil,
	)
	actionButton := findButton(screen, "Produce 1 Aether")
	if actionButton == nil {
		t.Fatal("hidden Caster Token action was not constructed")
	}
	if actionButton.Visible() {
		t.Fatal("Caster Token action is visible before selecting the token")
	}
	token := findCardTileByMatchID(screen, "player-two-token")
	if token == nil {
		t.Fatal("viewer's Caster Token was not rendered")
	}

	test.Tap(token)
	actionButton = findButton(screen, "Remove Token for 1 Aether")
	if actionButton == nil || !actionButton.Visible() || actionButton.Disabled() {
		t.Fatal("Caster Token action did not appear and enable after selection")
	}
	test.Tap(actionButton)

	if usedToken != "player-two-token" || usedRevision != 16 {
		t.Fatalf("token action submitted token/revision %q/%d; want player-two-token/16", usedToken, usedRevision)
	}
}

func TestHoveringCasterTokenUpdatesCardInformationPanel(t *testing.T) {
	token := testDefinitions()[0]
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	tile := findCardTile(screen, token.ID)
	if tile == nil {
		t.Fatal("board screen does not contain the starting Caster Token")
	}

	tile.MouseIn(&desktop.MouseEvent{})

	if !containsText(screen, token.Name) {
		t.Fatal("hovering Caster Token did not update preview title")
	}
	if !containsTextPart(screen, token.Ability) {
		t.Fatal("hovering Caster Token did not update preview description")
	}
}

func TestConcealedCardsDoNotExposeDefinitions(t *testing.T) {
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	tile := findConcealedCardTile(screen)
	if tile == nil {
		t.Fatal("board screen does not render concealed cards")
	}
	if tile.Card.ID != "" || tile.View.CardID != "" || tile.View.MatchID != "" {
		t.Fatalf("concealed tile leaked identifiers: Card=%#v View=%#v", tile.Card, tile.View)
	}

	tile.MouseIn(&desktop.MouseEvent{})

	if !containsText(screen, "Concealed Card") {
		t.Fatal("hovering concealed card did not show generic preview")
	}
	if containsTextPart(screen, "Produce one Aether.") {
		t.Fatal("hovering concealed card exposed a card ability")
	}
}

func TestOpeningHandSelectionSubmitsSelectedMatchIDs(t *testing.T) {
	var submitted []model.MatchCardID
	screen := NewBoardScreen(
		testMatchView(),
		testDefinitions(),
		BoardActions{
			SubmitOpeningHand: func(replace []model.MatchCardID, _ model.Revision) {
				submitted = append([]model.MatchCardID(nil), replace...)
			},
		},
		nil,
	)
	tile := findCardTile(screen, "visible-hand-card")
	if tile == nil {
		t.Fatal("viewer hand card was not rendered")
	}
	replaceButton := findButton(screen, "Replace Selected")
	if replaceButton == nil {
		t.Fatal("opening-hand replacement control was not rendered")
	}
	if !replaceButton.Disabled() {
		t.Fatal("replacement control is enabled before a card is selected")
	}

	test.Tap(tile)

	if replaceButton.Disabled() {
		t.Fatal("replacement control did not enable after selection")
	}
	test.Tap(replaceButton)

	want := model.MatchCardID("player-one-hand")
	if len(submitted) != 1 || submitted[0] != want {
		t.Fatalf("submitted replacement IDs = %v; want [%q]", submitted, want)
	}
}

func TestKeepHandSubmitsEmptyReplacement(t *testing.T) {
	submitted := []model.MatchCardID{"not-submitted"}
	screen := NewBoardScreen(
		testMatchView(),
		testDefinitions(),
		BoardActions{
			SubmitOpeningHand: func(replace []model.MatchCardID, _ model.Revision) {
				submitted = append([]model.MatchCardID(nil), replace...)
			},
		},
		nil,
	)
	keepButton := findButton(screen, "Keep Hand")
	if keepButton == nil {
		t.Fatal("Keep Hand control was not rendered")
	}

	test.Tap(keepButton)

	if len(submitted) != 0 {
		t.Fatalf("Keep Hand submitted replacements %v; want none", submitted)
	}
}

func TestFaceDownLevelOneCallSubmitsSelectedCardAndCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 12
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	match.Players[0].OpeningHandFinalized = true
	calledID := model.MatchCardID("")
	calledRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			CallFaceDownLevelOne: func(cardID model.MatchCardID, revision model.Revision) {
				calledID = cardID
				calledRevision = revision
			},
		},
		nil,
	)
	callButton := findButton(screen, "Call Selected Face Down")
	if callButton == nil {
		t.Fatal("face-down Call control was not rendered for the active player")
	}
	if !callButton.Disabled() {
		t.Fatal("face-down Call control is enabled before selecting a card")
	}
	handCard := findCardTile(screen, "visible-hand-card")
	if handCard == nil {
		t.Fatal("active player's visible hand card was not rendered")
	}

	test.Tap(handCard)
	if callButton.Disabled() {
		t.Fatal("face-down Call control did not enable after selecting a card")
	}
	test.Tap(callButton)

	if calledID != "player-one-hand" || calledRevision != 12 {
		t.Fatalf("face-down Call submitted card/revision %q/%d; want player-one-hand/12", calledID, calledRevision)
	}
}

func TestFaceDownLevelOneCallControlOnlyAppearsWhenAvailable(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*simulatorview.MatchView)
	}{
		{
			name: "non-active viewer",
			mutate: func(match *simulatorview.MatchView) {
				match.ViewerID = "player-two"
			},
		},
		{
			name: "wrong phase",
			mutate: func(match *simulatorview.MatchView) {
				match.Turn.Phase = model.PhaseMain
			},
		},
		{
			name: "Call already taken",
			mutate: func(match *simulatorview.MatchView) {
				match.Turn.CallActionTaken = true
			},
		},
		{
			name: "match not in progress",
			mutate: func(match *simulatorview.MatchView) {
				match.MatchStatus = model.StatusSetup
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			match.Turn.Number = 1
			match.Turn.Phase = model.PhaseCall
			match.Players[0].OpeningHandFinalized = true
			testCase.mutate(&match)
			screen := NewBoardScreen(
				match,
				testDefinitions(),
				BoardActions{CallFaceDownLevelOne: func(model.MatchCardID, model.Revision) {}},
				nil,
			)
			if findButton(screen, "Call Selected Face Down") != nil {
				t.Fatal("face-down Call control was rendered when the action was unavailable")
			}
		})
	}
}

func TestBoardControllerRebuildsViewerFieldWhenCallBecomesAvailable(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseRecovery
	match.Players[0].OpeningHandFinalized = true
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{CallFaceDownLevelOne: func(model.MatchCardID, model.Revision) {}},
		nil,
	)
	firstOpponentField := controller.boards.Objects[0]
	firstPlayerField := controller.boards.Objects[1]
	if findButton(controller.Content(), "Call Selected Face Down") != nil {
		t.Fatal("face-down Call control appeared during Recovery")
	}

	updated := match
	updated.Turn.Phase = model.PhaseCall
	updated.Revision++
	controller.Update(updated)

	if controller.boards.Objects[0] != firstOpponentField {
		t.Fatal("Call availability update rebuilt the opponent field")
	}
	if controller.boards.Objects[1] == firstPlayerField {
		t.Fatal("Call availability update did not rebuild the viewer field")
	}
	if findButton(controller.Content(), "Call Selected Face Down") == nil {
		t.Fatal("face-down Call control did not appear upon entering Call phase")
	}
}

func TestPhaseBarEnablesInitialCallOnlyForActiveViewer(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseRecovery
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	callButton := findButton(screen, string(model.PhaseCall))
	if callButton == nil {
		t.Fatal("phase bar does not contain Call")
	}
	if callButton.Disabled() {
		t.Fatal("Call transition is disabled for active player in initial Recovery")
	}
	for _, phase := range []model.Phase{
		model.PhaseRecovery,
		model.PhaseDraw,
		model.PhaseMain,
		model.PhaseBattle,
		model.PhaseEnd,
	} {
		button := findButton(screen, string(phase))
		if button == nil || !button.Disabled() {
			t.Fatalf("phase %q should be displayed but disabled", phase)
		}
	}

	test.Tap(callButton)
	if !completionRequested {
		t.Fatal("phase bar did not request completion of the current phase")
	}
}

func TestPhaseBarDisablesTransitionsForNonActiveViewer(t *testing.T) {
	match := testMatchView()
	match.ViewerID = match.Players[1].ID
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseRecovery
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) {}},
		nil,
	)

	callButton := findButton(screen, string(model.PhaseCall))
	if callButton == nil || !callButton.Disabled() {
		t.Fatal("Call transition is enabled for non-active player")
	}
}

func TestPhaseBarEnablesDrawForLaterRecovery(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 2
	match.Turn.Phase = model.PhaseRecovery
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	drawButton := findButton(screen, string(model.PhaseDraw))
	if drawButton == nil || drawButton.Disabled() {
		t.Fatal("Draw transition is not enabled during a later Recovery")
	}
	test.Tap(drawButton)
	if !completionRequested {
		t.Fatal("Draw transition did not request completion of Recovery")
	}
}

func TestPhaseBarEnablesCallDuringLaterDraw(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 2
	match.Turn.Phase = model.PhaseDraw
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	callButton := findButton(screen, string(model.PhaseCall))
	if callButton == nil || callButton.Disabled() {
		t.Fatal("Call transition is not enabled during Draw")
	}
	test.Tap(callButton)
	if !completionRequested {
		t.Fatal("Call transition did not request completion of Draw")
	}
}

func TestPhaseBarEnablesMainDuringCall(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	mainButton := findButton(screen, string(model.PhaseMain))
	if mainButton == nil || mainButton.Disabled() {
		t.Fatal("Main transition is not enabled during Call")
	}
	test.Tap(mainButton)
	if !completionRequested {
		t.Fatal("Main transition did not request completion of Call")
	}
}

func TestPhaseBarEnablesRemainingSkeletonTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current model.Phase
		target  model.Phase
		label   string
	}{
		{name: "Main to Battle", current: model.PhaseMain, target: model.PhaseBattle, label: "Battle"},
		{name: "Battle to End", current: model.PhaseBattle, target: model.PhaseEnd, label: "End"},
		{name: "End turn", current: model.PhaseEnd, target: model.PhaseEnd, label: "End Turn"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			match.Turn.Number = 1
			match.Turn.Phase = testCase.current
			completionRequested := false
			screen := NewBoardScreen(
				match,
				testDefinitions(),
				BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
				nil,
			)

			targetButton := findButton(screen, testCase.label)
			if targetButton == nil || targetButton.Disabled() {
				t.Fatalf("%q transition is not enabled during %q", testCase.target, testCase.current)
			}
			test.Tap(targetButton)
			if !completionRequested {
				t.Fatalf("%q transition did not request completion of %q", testCase.target, testCase.current)
			}
		})
	}
}

func TestBoardControllerUpdatesMetadataWithoutRebuildingPlayerFields(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	match.Revision = 2
	requestedRevision := model.Revision(0)
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			CompleteCurrentPhase: func(revision model.Revision) {
				requestedRevision = revision
			},
		},
		nil,
	)
	firstOpponentField := controller.boards.Objects[0]
	firstPlayerField := controller.boards.Objects[1]

	updated := match
	updated.Turn.Phase = model.PhaseMain
	updated.Revision = 3
	controller.Update(updated)

	if controller.boards.Objects[0] != firstOpponentField ||
		controller.boards.Objects[1] != firstPlayerField {
		t.Fatal("metadata-only update rebuilt player fields")
	}
	if !containsText(
		controller.Content(),
		"Turn 1 • Main • Revision 3 • Active player: player-one",
	) {
		t.Fatal("metadata-only update did not refresh status")
	}
	battleButton := findButton(controller.Content(), string(model.PhaseBattle))
	if battleButton == nil || battleButton.Disabled() {
		t.Fatal("metadata-only update did not enable Battle")
	}
	test.Tap(battleButton)
	if requestedRevision != 3 {
		t.Fatalf("phase action used revision %d; want updated revision 3", requestedRevision)
	}

	updated.Players[0].DeckCount--
	updated.Revision = 4
	controller.Update(updated)
	if controller.boards.Objects[0] != firstOpponentField {
		t.Fatal("player-zone update rebuilt the unaffected opponent field")
	}
	if controller.boards.Objects[1] == firstPlayerField {
		t.Fatal("player-zone update did not rebuild the affected player field")
	}
}

func containsText(object fyne.CanvasObject, text string) bool {
	if label, ok := object.(*widget.Label); ok && label.Text == text {
		return true
	}
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return true
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if containsText(child, text) {
				return true
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return containsText(scroll.Content, text)
	}
	return false
}

func containsTextPart(object fyne.CanvasObject, text string) bool {
	if label, ok := object.(*widget.Label); ok && strings.Contains(label.Text, text) {
		return true
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if containsTextPart(child, text) {
				return true
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return containsTextPart(scroll.Content, text)
	}
	return false
}

func findButton(object fyne.CanvasObject, text string) *widget.Button {
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return button
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if button := findButton(child, text); button != nil {
				return button
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findButton(scroll.Content, text)
	}
	return nil
}

func findCardTile(object fyne.CanvasObject, cardID string) *CardTile {
	if tile, ok := object.(*CardTile); ok && tile.Card.ID == cardID {
		return tile
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if tile := findCardTile(child, cardID); tile != nil {
				return tile
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findCardTile(scroll.Content, cardID)
	}
	return nil
}

func findCardTileByMatchID(object fyne.CanvasObject, matchID model.MatchCardID) *CardTile {
	if tile, ok := object.(*CardTile); ok && tile.View.MatchID == matchID {
		return tile
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if tile := findCardTileByMatchID(child, matchID); tile != nil {
				return tile
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findCardTileByMatchID(scroll.Content, matchID)
	}
	return nil
}

func findConcealedCardTile(object fyne.CanvasObject) *CardTile {
	if tile, ok := object.(*CardTile); ok &&
		!tile.View.ShowFace &&
		tile.View.CardID == "" &&
		tile.View.MatchID == "" {
		return tile
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if tile := findConcealedCardTile(child); tile != nil {
				return tile
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findConcealedCardTile(scroll.Content)
	}
	return nil
}

func testDefinitions() []cards.Card {
	return []cards.Card{
		{
			ID:      "1100",
			Name:    "Caster Token",
			Ability: "Produce one Aether.",
		},
		{
			ID:      "visible-hand-card",
			Name:    "Visible Hand Card",
			Type:    "Servant",
			Ability: "Visible ability.",
		},
		{
			ID:      "visible-servant-card",
			Name:    "Visible Servant",
			Type:    "Servant",
			Ability: "Servant ability.",
		},
		{
			ID:      "visible-grave-card",
			Name:    "Visible Graveyard Card",
			Type:    "Spell",
			Ability: "Resolved ability.",
		},
		{
			ID:      "visible-exile-card",
			Name:    "Visible Exiled Card",
			Type:    "Spell",
			Ability: "Exiled ability.",
		},
		{
			ID:      "face-down-caster-card",
			Name:    "Known Face-down Caster",
			Type:    "Caster",
			Ability: "Caster ability.",
		},
	}
}

func testMatchView() simulatorview.MatchView {
	return simulatorview.MatchView{
		ViewerID:    "player-one",
		MatchStatus: model.StatusSetup,
		Turn: model.TurnState{
			ActivePlayer: "player-one",
			Phase:        model.PhaseCall,
		},
		Players: [2]simulatorview.PlayerView{
			{
				ID:        "player-one",
				DeckCount: 36,
				Hand: []simulatorview.CardView{
					{
						MatchID:  "player-one-hand",
						CardID:   "visible-hand-card",
						ShowFace: true,
					},
				},
				Orbs: []simulatorview.CardView{
					{Face: model.CardFaceDown},
				},
				CasterZone: []simulatorview.CardView{
					{
						MatchID:     "player-one-token",
						CardID:      model.CasterTokenCardID,
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
					{
						MatchID:     "player-one-facedown-caster",
						CardID:      "face-down-caster-card",
						Face:        model.CardFaceDown,
						Orientation: model.OrientationRecovered,
						ShowFace:    false,
					},
				},
				ServantZone: []simulatorview.CardView{
					{
						MatchID:     "player-one-servant",
						CardID:      "visible-servant-card",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
				Graveyard: []simulatorview.CardView{
					{
						MatchID:     "player-one-grave",
						CardID:      "visible-grave-card",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
				Exile: []simulatorview.CardView{
					{
						MatchID:     "player-one-exile",
						CardID:      "visible-exile-card",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
			},
			{
				ID:        "player-two",
				DeckCount: 36,
				Hand: []simulatorview.CardView{
					{Face: model.CardFaceDown},
				},
				Orbs: []simulatorview.CardView{
					{Face: model.CardFaceDown},
				},
				CasterZone: []simulatorview.CardView{
					{
						MatchID:     "player-two-token",
						CardID:      model.CasterTokenCardID,
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
					{
						Face:        model.CardFaceDown,
						Orientation: model.OrientationRecovered,
						ShowFace:    false,
					},
				},
			},
		},
	}
}
