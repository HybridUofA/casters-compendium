package session

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestPlayerSessionsShareMatchWithPrivateViews(t *testing.T) {
	state := sessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	playerOneView, err := playerOne.View()
	if err != nil {
		t.Fatalf("player one View() error = %v", err)
	}
	playerTwoView, err := playerTwo.View()
	if err != nil {
		t.Fatalf("player two View() error = %v", err)
	}

	assertSessionHandVisibility(t, playerOneView, 0)
	assertSessionHandVisibility(t, playerTwoView, 1)

	playerOneView, err = playerOne.SubmitOpeningHandDecision(nil, playerOneView.Revision)
	if err != nil {
		t.Fatalf("player one SubmitOpeningHandDecision() error = %v", err)
	}
	if !playerOneView.Players[0].OpeningHandFinalized {
		t.Fatal("player one decision did not finalize player one's hand")
	}
	if playerOneView.Players[1].OpeningHandFinalized {
		t.Fatal("player one decision finalized player two's hand")
	}
	if playerOneView.MatchStatus != model.StatusSetup {
		t.Fatalf("match status = %q after one decision; want setup", playerOneView.MatchStatus)
	}

	playerTwoView, err = playerTwo.View()
	if err != nil {
		t.Fatalf("player two refreshed View() error = %v", err)
	}
	playerTwoView, err = playerTwo.SubmitOpeningHandDecision(nil, playerTwoView.Revision)
	if err != nil {
		t.Fatalf("player two SubmitOpeningHandDecision() error = %v", err)
	}
	if !playerTwoView.Players[0].OpeningHandFinalized ||
		!playerTwoView.Players[1].OpeningHandFinalized {
		t.Fatal("both hands are not finalized after both decisions")
	}
	if playerTwoView.MatchStatus != model.StatusInProgress ||
		playerTwoView.Turn.Number != 1 ||
		playerTwoView.Turn.Phase != model.PhaseRecovery {
		t.Fatalf("match did not enter turn-one Recovery: %#v", playerTwoView)
	}

	playerOneView, err = playerOne.View()
	if err != nil {
		t.Fatalf("player one refreshed View() error = %v", err)
	}
	if playerOneView.MatchStatus != playerTwoView.MatchStatus ||
		playerOneView.Turn != playerTwoView.Turn {
		t.Fatal("player sessions do not observe the same authoritative progression")
	}
	assertSessionHandVisibility(t, playerOneView, 0)
	assertSessionHandVisibility(t, playerTwoView, 1)
}

func TestPlayerSessionsRejectOneSimultaneousOpeningHandDecisionAsStale(t *testing.T) {
	state := sessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	sessions := [2]*PlayerSession{}
	for index := range sessions {
		sessions[index], err = NewPlayerSession(localMatch, state.Players[index].ID)
		if err != nil {
			t.Fatalf("NewPlayerSession(player %d) error = %v", index+1, err)
		}
	}

	start := make(chan struct{})
	type submissionResult struct {
		index int
		err   error
	}
	results := make(chan submissionResult, len(sessions))
	var wait sync.WaitGroup
	for index, playerSession := range sessions {
		wait.Add(1)
		go func(playerIndex int, session *PlayerSession) {
			defer wait.Done()
			<-start
			_, submitErr := session.SubmitOpeningHandDecision(nil, 0)
			results <- submissionResult{index: playerIndex, err: submitErr}
		}(index, playerSession)
	}
	close(start)
	wait.Wait()
	close(results)

	successes := 0
	staleIndex := -1
	for result := range results {
		if result.err == nil {
			successes++
			continue
		}
		if !strings.Contains(result.err.Error(), "expected revision") {
			t.Fatalf("simultaneous submission error = %v; want stale-revision error", result.err)
		}
		staleIndex = result.index
	}
	if successes != 1 || staleIndex == -1 {
		t.Fatalf("simultaneous results = %d successes, stale index %d; want one of each", successes, staleIndex)
	}
	result, err := sessions[0].View()
	if err != nil {
		t.Fatalf("View() after simultaneous decisions error = %v", err)
	}
	if result.Revision != 1 || result.MatchStatus != model.StatusSetup {
		t.Fatalf("state after competing submissions = %#v; want revision 1 Setup", result)
	}
	refreshed, err := sessions[staleIndex].View()
	if err != nil {
		t.Fatalf("stale player refresh error = %v", err)
	}
	result, err = sessions[staleIndex].SubmitOpeningHandDecision(nil, refreshed.Revision)
	if err != nil {
		t.Fatalf("retried opening-hand decision error = %v", err)
	}
	if !result.Players[0].OpeningHandFinalized ||
		!result.Players[1].OpeningHandFinalized ||
		result.Turn.Phase != model.PhaseRecovery ||
		result.Revision != 2 {
		t.Fatalf("retried decisions produced incomplete state: %#v", result)
	}
}

func TestPlayerSessionCompleteCurrentPhaseUpdatesSharedMatch(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseRecovery
	state.Players[0].OpeningHandFinalized = true
	state.Players[1].OpeningHandFinalized = true
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	result, err := playerOne.CompleteCurrentPhase(state.Revision)
	if err != nil {
		t.Fatalf("CompleteCurrentPhase() error = %v", err)
	}
	if result.Turn.Phase != model.PhaseCall {
		t.Fatalf("requesting player's phase = %q; want Call", result.Turn.Phase)
	}
	otherView, err := playerTwo.View()
	if err != nil {
		t.Fatalf("player two View() error = %v", err)
	}
	if otherView.Turn.Phase != model.PhaseCall {
		t.Fatalf("other player's phase = %q; want shared Call phase", otherView.Turn.Phase)
	}

	result, err = playerOne.CompleteCurrentPhase(result.Revision)
	if err != nil {
		t.Fatalf("CompleteCurrentPhase(Call) error = %v", err)
	}
	if result.Turn.Phase != model.PhaseMain {
		t.Fatalf("requesting player's phase = %q; want Main", result.Turn.Phase)
	}
	otherView, err = playerTwo.View()
	if err != nil {
		t.Fatalf("player two View() after Call error = %v", err)
	}
	if otherView.Turn.Phase != model.PhaseMain {
		t.Fatalf("other player's phase = %q; want shared Main phase", otherView.Turn.Phase)
	}
}

func TestPlayerSessionCallFaceDownLevelOneUpdatesSharedPrivateViews(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseCall
	state.Revision = 4
	selectedID := state.Players[0].Hand[0]
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	activeView, err := playerOne.CallFaceDownLevelOne(selectedID, state.Revision)
	if err != nil {
		t.Fatalf("CallFaceDownLevelOne() error = %v", err)
	}
	if activeView.Revision != state.Revision+1 || !activeView.Turn.CallActionTaken {
		t.Fatalf("active view revision/turn = %d/%#v; want revision %d with Call recorded", activeView.Revision, activeView.Turn, state.Revision+1)
	}
	if len(activeView.Players[0].Hand) != len(state.Players[0].Hand)-1 ||
		len(activeView.Players[0].CasterZone) != 1 {
		t.Fatalf("active player's projected zones = %#v", activeView.Players[0])
	}
	calledCard := activeView.Players[0].CasterZone[0]
	if calledCard.MatchID != selectedID || calledCard.ShowFace || calledCard.Face != model.CardFaceDown {
		t.Fatalf("active player's called card projection = %#v", calledCard)
	}

	opponentView, err := playerTwo.View()
	if err != nil {
		t.Fatalf("player two View() error = %v", err)
	}
	if opponentView.Revision != activeView.Revision || !opponentView.Turn.CallActionTaken {
		t.Fatalf("opponent did not receive current shared turn state: %#v", opponentView)
	}
	if len(opponentView.Players[0].CasterZone) != 1 {
		t.Fatalf("opponent view Caster Zone = %#v; want one concealed card", opponentView.Players[0].CasterZone)
	}
	opponentCard := opponentView.Players[0].CasterZone[0]
	if opponentCard.ShowFace || opponentCard.MatchID != "" || opponentCard.CardID != "" {
		t.Fatalf("opponent projection exposed face-down Call: %#v", opponentCard)
	}
}

func TestPlayerSessionCallFaceDownLevelOneRejectsOpponentImpersonation(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseCall
	selectedID := state.Players[0].Hand[0]
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	_, err = playerTwo.CallFaceDownLevelOne(selectedID, state.Revision)
	if err == nil || !strings.Contains(err.Error(), "is not the active player") {
		t.Fatalf("CallFaceDownLevelOne() error = %v; want inactive-player error", err)
	}
	result, viewErr := playerTwo.View()
	if viewErr != nil {
		t.Fatalf("View() after rejected Call error = %v", viewErr)
	}
	if result.Revision != state.Revision || result.Turn.CallActionTaken || len(result.Players[0].CasterZone) != 0 {
		t.Fatalf("rejected opponent Call mutated shared state: %#v", result)
	}
}

func TestPlayerSessionCallFaceDownLevelOneRejectsStaleRevision(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseCall
	state.Revision = 9
	selectedID := state.Players[0].Hand[0]
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}

	_, err = playerOne.CallFaceDownLevelOne(selectedID, state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("CallFaceDownLevelOne() error = %v; want stale-revision error", err)
	}
	result, viewErr := playerOne.View()
	if viewErr != nil {
		t.Fatalf("View() after stale Call error = %v", viewErr)
	}
	if result.Revision != state.Revision || result.Turn.CallActionTaken || len(result.Players[0].CasterZone) != 0 {
		t.Fatalf("stale Call mutated shared state: %#v", result)
	}
}

func TestPlayerSessionGenerateNonElementalAetherAllowsNonActivePlayer(t *testing.T) {
	state, selectedID := aetherSessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, "player-one")
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, "player-two")
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	playerTwoView, err := playerTwo.GenerateNonElementalAether(selectedID, state.Revision)
	if err != nil {
		t.Fatalf("GenerateNonElementalAether() error = %v", err)
	}
	if playerTwoView.Players[1].Aether.NonElemental != 1 ||
		playerTwoView.Revision != state.Revision+1 {
		t.Fatalf("producing player's view = %#v; want one non-elemental Aether and next revision", playerTwoView)
	}
	calledCard := playerTwoView.Players[1].CasterZone[0]
	if calledCard.MatchID != selectedID || calledCard.Orientation != model.OrientationRested {
		t.Fatalf("producing player's Caster projection = %#v; want known Rested card", calledCard)
	}

	playerOneView, err := playerOne.View()
	if err != nil {
		t.Fatalf("player one View() error = %v", err)
	}
	if playerOneView.Players[1].Aether.NonElemental != 1 ||
		playerOneView.Players[1].CasterZone[0].Orientation != model.OrientationRested {
		t.Fatalf("active player did not observe public Aether/orientation update: %#v", playerOneView)
	}
	if playerOneView.Players[1].CasterZone[0].MatchID != "" ||
		playerOneView.Players[1].CasterZone[0].CardID != "" {
		t.Fatalf("active player's view exposed opponent face-down Caster: %#v", playerOneView.Players[1].CasterZone[0])
	}
}

func TestPlayerSessionGenerateNonElementalAetherRejectsStaleRevision(t *testing.T) {
	state, selectedID := aetherSessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, "player-two")
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	_, err = playerTwo.GenerateNonElementalAether(selectedID, state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("GenerateNonElementalAether() error = %v; want stale-revision error", err)
	}
	result, viewErr := playerTwo.View()
	if viewErr != nil {
		t.Fatalf("View() after stale Aether action error = %v", viewErr)
	}
	if result.Revision != state.Revision ||
		result.Players[1].Aether.NonElemental != 0 ||
		result.Players[1].CasterZone[0].Orientation != model.OrientationRecovered {
		t.Fatalf("stale Aether action mutated state: %#v", result)
	}
}

func TestPlayerSessionUseCasterTokenUpdatesBothPublicViews(t *testing.T) {
	state, tokenID := casterTokenSessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, "player-one")
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, "player-two")
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	playerTwoView, err := playerTwo.UseCasterToken(tokenID, state.Revision)
	if err != nil {
		t.Fatalf("UseCasterToken() error = %v", err)
	}
	if len(playerTwoView.Players[1].CasterZone) != 0 ||
		len(playerTwoView.Players[1].Exile) != 0 ||
		playerTwoView.Players[1].Aether.NonElemental != 1 ||
		playerTwoView.Revision != state.Revision+1 {
		t.Fatalf("producing player's view = %#v; want disappeared token, one Aether, and next revision", playerTwoView)
	}
	if _, exists := localMatch.state.CardInstances[tokenID]; exists {
		t.Fatal("shared state retained the used Caster Token instance")
	}

	playerOneView, err := playerOne.View()
	if err != nil {
		t.Fatalf("player one View() error = %v", err)
	}
	if len(playerOneView.Players[1].CasterZone) != 0 ||
		playerOneView.Players[1].Aether.NonElemental != 1 ||
		playerOneView.Revision != state.Revision+1 {
		t.Fatalf("opponent did not observe the public token/Aether update: %#v", playerOneView)
	}
}

func TestPlayerSessionUseCasterTokenRejectsStaleRevision(t *testing.T) {
	state, tokenID := casterTokenSessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, "player-two")
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	_, err = playerTwo.UseCasterToken(tokenID, state.Revision-1)
	if err == nil || !strings.Contains(err.Error(), "does not match expected revision") {
		t.Fatalf("UseCasterToken() error = %v; want stale-revision error", err)
	}
	result, viewErr := playerTwo.View()
	if viewErr != nil {
		t.Fatalf("View() after stale token action error = %v", viewErr)
	}
	if result.Revision != state.Revision ||
		result.Players[1].Aether.NonElemental != 0 ||
		len(result.Players[1].CasterZone) != 1 {
		t.Fatalf("stale token action mutated shared state: %#v", result)
	}
}

func TestPlayerSessionCompletesLaterRecoveryAndDraw(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 2
	state.Turn.Phase = model.PhaseRecovery
	state.Turn.ActivePlayer = state.Players[1].ID
	topCard := state.Players[1].Deck[0]
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	activeSession, err := NewPlayerSession(localMatch, state.Turn.ActivePlayer)
	if err != nil {
		t.Fatalf("NewPlayerSession(active player) error = %v", err)
	}
	opponentSession, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(opponent) error = %v", err)
	}

	drawView, err := activeSession.CompleteCurrentPhase(state.Revision)
	if err != nil {
		t.Fatalf("CompleteCurrentPhase(Recovery) error = %v", err)
	}
	if drawView.Turn.Phase != model.PhaseDraw {
		t.Fatalf("phase = %q; want %q", drawView.Turn.Phase, model.PhaseDraw)
	}
	activePlayer := drawView.Players[1]
	if activePlayer.DeckCount != 3 || len(activePlayer.Hand) != 8 {
		t.Fatalf("active player after draw = %#v; want deck 3 and hand 8", activePlayer)
	}
	if drawn := activePlayer.Hand[len(activePlayer.Hand)-1].MatchID; drawn != topCard {
		t.Fatalf("drawn card = %q; want former deck top %q", drawn, topCard)
	}

	callView, err := activeSession.CompleteCurrentPhase(drawView.Revision)
	if err != nil {
		t.Fatalf("CompleteCurrentPhase(Draw) error = %v", err)
	}
	if callView.Turn.Phase != model.PhaseCall ||
		callView.Players[1].DeckCount != 3 ||
		len(callView.Players[1].Hand) != 8 {
		t.Fatalf("state after completing Draw = %#v", callView)
	}

	opponentView, err := opponentSession.View()
	if err != nil {
		t.Fatalf("opponent View() error = %v", err)
	}
	if opponentView.Turn.Phase != model.PhaseCall {
		t.Fatalf("opponent phase = %q; want %q", opponentView.Turn.Phase, model.PhaseCall)
	}
	for _, card := range opponentView.Players[1].Hand {
		if card.ShowFace || card.CardID != "" || card.MatchID != "" {
			t.Fatalf("opponent view exposed active player's hand card: %#v", card)
		}
	}
}

func TestPlayerSessionsRunPhaseSkeletonAndRollTurn(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseCall
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player one) error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	expectedRevision := state.Revision
	for _, wantPhase := range []model.Phase{
		model.PhaseMain,
		model.PhaseBattle,
		model.PhaseEnd,
		model.PhaseRecovery,
	} {
		result, completeErr := playerOne.CompleteCurrentPhase(expectedRevision)
		if completeErr != nil {
			t.Fatalf("player one completion toward %q error = %v", wantPhase, completeErr)
		}
		if result.Turn.Phase != wantPhase {
			t.Fatalf("phase = %q; want %q", result.Turn.Phase, wantPhase)
		}
		expectedRevision = result.Revision
	}

	rolledView, err := playerTwo.View()
	if err != nil {
		t.Fatalf("player two View() after rollover error = %v", err)
	}
	if rolledView.Turn.ActivePlayer != state.Players[1].ID ||
		rolledView.Turn.Number != 2 ||
		rolledView.Turn.Phase != model.PhaseRecovery {
		t.Fatalf("rolled player-two view = %#v", rolledView)
	}
	if _, err = playerOne.CompleteCurrentPhase(rolledView.Revision); err == nil ||
		!strings.Contains(err.Error(), "not the current active player") {
		t.Fatalf("former active player completion error = %v; want inactive-player error", err)
	}

	drawView, err := playerTwo.CompleteCurrentPhase(rolledView.Revision)
	if err != nil {
		t.Fatalf("player two CompleteCurrentPhase(Recovery) error = %v", err)
	}
	if drawView.Turn.Phase != model.PhaseDraw ||
		drawView.Players[1].DeckCount != 3 ||
		len(drawView.Players[1].Hand) != 8 {
		t.Fatalf("player two Draw entry view = %#v", drawView)
	}
}

func TestPlayerSessionCompleteCurrentPhaseRejectsNonActivePlayerWithoutMutation(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseRecovery
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerTwo, err := NewPlayerSession(localMatch, state.Players[1].ID)
	if err != nil {
		t.Fatalf("NewPlayerSession(player two) error = %v", err)
	}

	_, err = playerTwo.CompleteCurrentPhase(state.Revision)
	if err == nil || !strings.Contains(err.Error(), "not the current active player") {
		t.Fatalf("CompleteCurrentPhase() error = %v; want non-active-player error", err)
	}
	result, viewErr := playerTwo.View()
	if viewErr != nil {
		t.Fatalf("View() error = %v", viewErr)
	}
	if result.Turn.Phase != model.PhaseRecovery {
		t.Fatalf("rejected request changed phase to %q", result.Turn.Phase)
	}
}

func TestPlayerSessionRejectsStalePhaseCompletionWithoutMutation(t *testing.T) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseRecovery
	state.Revision = 6
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}
	playerOne, err := NewPlayerSession(localMatch, state.Turn.ActivePlayer)
	if err != nil {
		t.Fatalf("NewPlayerSession(active player) error = %v", err)
	}

	_, err = playerOne.CompleteCurrentPhase(5)
	if err == nil || !strings.Contains(err.Error(), "expected revision 5") {
		t.Fatalf("CompleteCurrentPhase(stale) error = %v; want stale-revision error", err)
	}
	result, err := playerOne.View()
	if err != nil {
		t.Fatalf("View() after stale completion error = %v", err)
	}
	if result.Revision != 6 || result.Turn.Phase != model.PhaseRecovery {
		t.Fatalf("stale completion changed view to revision %d phase %q", result.Revision, result.Turn.Phase)
	}
}

func TestNewLocalMatchRejectsInvalidPlayersAndState(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.MatchState)
	}{
		{
			name: "blank first player ID",
			mutate: func(state *model.MatchState) {
				state.Players[0].ID = " "
			},
		},
		{
			name: "duplicate player IDs",
			mutate: func(state *model.MatchState) {
				state.Players[1].ID = state.Players[0].ID
			},
		},
		{
			name: "unprojectable state",
			mutate: func(state *model.MatchState) {
				state.Players[0].Hand[0] = "missing-card"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := sessionStateForTest()
			test.mutate(&state)

			localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
			if err == nil {
				t.Fatal("NewLocalMatch() returned nil error")
			}
			if localMatch != nil {
				t.Fatalf("NewLocalMatch() returned invalid match %#v", localMatch)
			}
		})
	}
}

func TestNewLocalMatchRejectsNilCatalog(t *testing.T) {
	localMatch, err := NewLocalMatch(sessionStateForTest(), matchSeedForTest(), nil)
	if err == nil || !strings.Contains(err.Error(), "catalog cannot be nil") {
		t.Fatalf("NewLocalMatch(nil catalog) error = %v; want nil-catalog error", err)
	}
	if localMatch != nil {
		t.Fatalf("NewLocalMatch(nil catalog) returned invalid match %#v", localMatch)
	}
}

func TestNewPlayerSessionRejectsInvalidSeat(t *testing.T) {
	state := sessionStateForTest()
	localMatch, err := NewLocalMatch(state, matchSeedForTest(), sessionCardCatalog{})
	if err != nil {
		t.Fatalf("NewLocalMatch() error = %v", err)
	}

	tests := []struct {
		name     string
		match    *LocalMatch
		playerID model.PlayerID
	}{
		{name: "nil match", playerID: state.Players[0].ID},
		{name: "blank player ID", match: localMatch},
		{name: "unknown player ID", match: localMatch, playerID: "unknown-player"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			playerSession, sessionErr := NewPlayerSession(test.match, test.playerID)
			if sessionErr == nil {
				t.Fatal("NewPlayerSession() returned nil error")
			}
			if playerSession != nil {
				t.Fatalf("NewPlayerSession() returned invalid session %#v", playerSession)
			}
		})
	}
}

func TestNilPlayerSessionMethodsReturnErrors(t *testing.T) {
	var playerSession *PlayerSession

	if _, err := playerSession.View(); err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil PlayerSession.View() error = %v; want nil-session error", err)
	}
	if _, err := playerSession.SubmitOpeningHandDecision(nil, 0); err == nil ||
		!strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil PlayerSession.SubmitOpeningHandDecision() error = %v; want nil-session error", err)
	}
	if _, err := playerSession.CallFaceDownLevelOne("card", 0); err == nil ||
		!strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil PlayerSession.CallFaceDownLevelOne() error = %v; want nil-session error", err)
	}
	if _, err := playerSession.GenerateNonElementalAether("card", 0); err == nil ||
		!strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil PlayerSession.GenerateNonElementalAether() error = %v; want nil-session error", err)
	}
	if _, err := playerSession.UseCasterToken("token", 0); err == nil ||
		!strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil PlayerSession.UseCasterToken() error = %v; want nil-session error", err)
	}

	empty := &PlayerSession{}
	if _, err := empty.View(); err == nil || !strings.Contains(err.Error(), "match cannot be nil") {
		t.Fatalf("empty PlayerSession.View() error = %v; want nil-match error", err)
	}
	if _, err := empty.SubmitOpeningHandDecision(nil, 0); err == nil ||
		!strings.Contains(err.Error(), "match cannot be nil") {
		t.Fatalf("empty PlayerSession.SubmitOpeningHandDecision() error = %v; want nil-match error", err)
	}
	if _, err := empty.CallFaceDownLevelOne("card", 0); err == nil ||
		!strings.Contains(err.Error(), "match cannot be nil") {
		t.Fatalf("empty PlayerSession.CallFaceDownLevelOne() error = %v; want nil-match error", err)
	}
	if _, err := empty.GenerateNonElementalAether("card", 0); err == nil ||
		!strings.Contains(err.Error(), "match cannot be nil") {
		t.Fatalf("empty PlayerSession.GenerateNonElementalAether() error = %v; want nil-match error", err)
	}
	if _, err := empty.UseCasterToken("token", 0); err == nil ||
		!strings.Contains(err.Error(), "match cannot be nil") {
		t.Fatalf("empty PlayerSession.UseCasterToken() error = %v; want nil-match error", err)
	}
}

func aetherSessionStateForTest() (model.MatchState, model.MatchCardID) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 3
	state.Turn.Phase = model.PhaseBattle
	state.Revision = 6
	selectedID := state.Players[1].Hand[0]
	state.Players[1].Hand = state.Players[1].Hand[1:]
	state.Players[1].CasterZone = []model.MatchCardID{selectedID}
	instance := state.CardInstances[selectedID]
	instance.Face = model.CardFaceDown
	instance.Orientation = model.OrientationRecovered
	state.CardInstances[selectedID] = instance
	return state, selectedID
}

func casterTokenSessionStateForTest() (model.MatchState, model.MatchCardID) {
	state := sessionStateForTest()
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 3
	state.Turn.Phase = model.PhaseBattle
	state.Revision = 8
	tokenID := model.MatchCardID("p2-token")
	state.Players[1].CasterZone = []model.MatchCardID{tokenID}
	state.CardInstances[tokenID] = model.CardInstance{
		CardID:       model.CasterTokenCardID,
		MatchID:      tokenID,
		Owner:        "player-two",
		Controller:   "player-two",
		CardCategory: model.CategoryTokenCard,
		Face:         model.CardFaceUp,
		Orientation:  model.OrientationRecovered,
	}
	return state, tokenID
}

func TestNewLocalSessionProjectsViewerSafeState(t *testing.T) {
	state := sessionStateForTest()

	localSession, err := NewLocalSession(state, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewLocalSession() error = %v", err)
	}
	result, err := localSession.View()
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}

	if result.ViewerID != state.Players[0].ID {
		t.Fatalf("viewer ID = %q; want %q", result.ViewerID, state.Players[0].ID)
	}
	if result.Players[0].Hand[0].CardID == "" {
		t.Fatal("session concealed the viewer's own hand")
	}
	if result.Players[1].Hand[0].CardID != "" ||
		result.Players[1].Hand[0].MatchID != "" {
		t.Fatal("session exposed the opponent's hand")
	}
}

func assertSessionHandVisibility(
	t *testing.T,
	result simulatorview.MatchView,
	viewerIndex int,
) {
	t.Helper()
	for playerIndex, player := range result.Players {
		for _, card := range player.Hand {
			if playerIndex == viewerIndex {
				if !card.ShowFace || card.CardID == "" || card.MatchID == "" {
					t.Fatalf(
						"viewer %d received concealed own-hand card: %#v",
						viewerIndex,
						card,
					)
				}
				continue
			}
			if card.ShowFace || card.CardID != "" || card.MatchID != "" {
				t.Fatalf(
					"viewer %d received exposed opponent-hand card: %#v",
					viewerIndex,
					card,
				)
			}
		}
	}
}

func TestSubmitOpeningHandDecisionUpdatesSessionAndReturnsFreshView(t *testing.T) {
	state := sessionStateForTest()
	localSession, err := NewLocalSession(state, state.Players[0].ID)
	if err != nil {
		t.Fatalf("NewLocalSession() error = %v", err)
	}
	replaced := state.Players[0].Hand[0]
	drawn := state.Players[0].Deck[0]

	result, err := localSession.SubmitOpeningHandDecision([]model.MatchCardID{replaced}, state.Revision)
	if err != nil {
		t.Fatalf("SubmitOpeningHandDecision() error = %v", err)
	}

	if !result.Players[0].OpeningHandFinalized {
		t.Fatal("returned view does not show finalized opening hand")
	}
	if result.Players[0].Hand[len(result.Players[0].Hand)-1].MatchID != drawn {
		t.Fatalf("replacement draw is not the former top card %q", drawn)
	}
	if localSession.state.Players[0].Deck[len(localSession.state.Players[0].Deck)-1] != replaced {
		t.Fatalf("replaced card %q was not placed on the bottom", replaced)
	}
}

func TestNewLocalSessionRejectsInvalidViewer(t *testing.T) {
	tests := []model.PlayerID{"", "  ", "unknown-player"}
	for _, viewerID := range tests {
		t.Run(fmt.Sprintf("viewer_%q", viewerID), func(t *testing.T) {
			session, err := NewLocalSession(sessionStateForTest(), viewerID)
			if err == nil {
				t.Fatal("NewLocalSession() returned nil error")
			}
			if session != nil {
				t.Fatalf("NewLocalSession() returned invalid session %#v", session)
			}
		})
	}
}

func TestNilLocalSessionMethodsReturnErrors(t *testing.T) {
	var localSession *LocalSession

	if _, err := localSession.View(); err == nil || !strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil View() error = %v; want nil-session error", err)
	}
	if _, err := localSession.SubmitOpeningHandDecision(nil, 0); err == nil ||
		!strings.Contains(err.Error(), "cannot be nil") {
		t.Fatalf("nil SubmitOpeningHandDecision() error = %v; want nil-session error", err)
	}
}

func sessionStateForTest() model.MatchState {
	state := model.MatchState{
		CardInstances: make(map[model.MatchCardID]model.CardInstance),
		Players: [2]model.PlayerState{
			{
				ID:   "player-one",
				Hand: sessionMatchIDs("p1-hand", 7),
				Deck: sessionMatchIDs("p1-deck", 4),
			},
			{
				ID:   "player-two",
				Hand: sessionMatchIDs("p2-hand", 7),
				Deck: sessionMatchIDs("p2-deck", 4),
			},
		},
		FirstPlayer: "player-one",
		MatchStatus: model.StatusSetup,
		Turn: model.TurnState{
			ActivePlayer: "player-one",
		},
	}
	for _, player := range state.Players {
		for _, matchID := range append(
			append([]model.MatchCardID(nil), player.Hand...),
			player.Deck...,
		) {
			state.CardInstances[matchID] = model.CardInstance{
				MatchID:    matchID,
				CardID:     model.CardID("definition-" + string(matchID)),
				Owner:      player.ID,
				Controller: player.ID,
			}
		}
	}
	return state
}

type sessionCardCatalog map[string]gamecards.Card

func (catalog sessionCardCatalog) FindByID(id string) (gamecards.Card, bool) {
	card, found := catalog[id]
	return card, found
}

func sessionMatchIDs(prefix string, count int) []model.MatchCardID {
	result := make([]model.MatchCardID, count)
	for index := range result {
		result[index] = model.MatchCardID(fmt.Sprintf("%s-%d", prefix, index+1))
	}
	return result
}

func matchSeedForTest() engine.MatchSeed {
	return engine.MatchSeed{First: 1, Second: 2}
}
