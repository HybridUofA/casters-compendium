package session

import (
	"fmt"
	"strings"
	"sync"

	"github.com/HybridUofA/casters-compendium/internal/simulator/engine"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

type LocalSession struct {
	state    model.MatchState
	viewerID model.PlayerID
}

type LocalMatch struct {
	mu      sync.RWMutex
	state   model.MatchState
	seed    engine.MatchSeed
	catalog rules.CardCatalog
}

type PlayerSession struct {
	match    *LocalMatch
	playerID model.PlayerID
}

func (session *LocalSession) View() (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session state cannot be nil")
	}
	out, err := view.ProjectMatch(session.state, session.viewerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("an error occurred during rendering: %w", err)
	}
	return out, nil
}

func (session *LocalSession) SubmitOpeningHandDecision(
	replace []model.MatchCardID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session state cannot be nil")
	}
	err := engine.ApplyOpeningHandDecision(&session.state, engine.OpeningHandDecision{
		PlayerID:         session.viewerID,
		Replace:          replace,
		ExpectedRevision: expectedRevision,
	},
	)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("an error submitting opening hand decisions occurred: %w", err)
	}
	finalView, err := view.ProjectMatch(session.state, session.viewerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("an error rendering the board state occurred: %w", err)
	}
	return finalView, nil
}

func NewLocalSession(
	state model.MatchState,
	viewerID model.PlayerID,
) (*LocalSession, error) {
	if strings.TrimSpace(string(viewerID)) == "" {
		return nil, fmt.Errorf("player ID cannot be blank")
	}
	_, err := view.ProjectMatch(state, viewerID)
	if err != nil {
		return nil, fmt.Errorf("error validating match state: %w", err)
	}
	session := &LocalSession{state: state, viewerID: viewerID}
	return session, nil
}

func NewLocalMatch(state model.MatchState, seed engine.MatchSeed, catalog rules.CardCatalog) (*LocalMatch, error) {
	if catalog == nil {
		return nil, fmt.Errorf("card catalog cannot be nil")
	}
	firstID := state.Players[0].ID
	secondID := state.Players[1].ID
	if strings.TrimSpace(string(firstID)) == "" || strings.TrimSpace(string(secondID)) == "" {
		return nil, fmt.Errorf("player IDs cannot be blank")
	}
	if firstID == secondID {
		return nil, fmt.Errorf("player IDs must be different")
	}
	_, err := view.ProjectMatch(state, firstID)
	if err != nil {
		return nil, fmt.Errorf("error displaying player one's state: %w", err)
	}
	_, err = view.ProjectMatch(state, secondID)
	if err != nil {
		return nil, fmt.Errorf("error displaying player two's state: %w", err)
	}
	localMatch := &LocalMatch{
		state:   state,
		seed:    seed,
		catalog: catalog,
	}
	return localMatch, nil
}

func NewPlayerSession(match *LocalMatch, playerID model.PlayerID) (*PlayerSession, error) {
	if match == nil {
		return nil, fmt.Errorf("match cannot be nil")
	}
	if strings.TrimSpace(string(playerID)) == "" {
		return nil, fmt.Errorf("player ID cannot be empty")
	}
	match.mu.RLock()
	defer match.mu.RUnlock()
	if _, err := view.ProjectMatch(match.state, playerID); err != nil {
		return nil, fmt.Errorf("error projecting match state: %w", err)
	}
	session := PlayerSession{
		match:    match,
		playerID: playerID,
	}
	return &session, nil
}

func (session *PlayerSession) View() (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.RLock()
	defer session.match.mu.RUnlock()
	matchProjection, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("error projecting match: %w", err)
	}
	return matchProjection, nil
}

func (session *PlayerSession) SubmitOpeningHandDecision(
	replace []model.MatchCardID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	err := engine.ApplyOpeningHandDecision(&session.match.state, engine.OpeningHandDecision{
		PlayerID:         session.playerID,
		Replace:          replace,
		ExpectedRevision: expectedRevision,
	})
	if err != nil {
		return view.MatchView{}, fmt.Errorf("submit opening-hand decision: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CallFaceDownLevelOne(
	cardID model.MatchCardID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	err := engine.CallFaceDownLevelOne(&session.match.state, session.playerID, cardID, expectedRevision)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("call face-down caster: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) GenerateNonElementalAether(
	cardID model.MatchCardID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	err := engine.GenerateNonElementalAether(&session.match.state, session.playerID, cardID, expectedRevision)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("generate non-elemental aether: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

// UseCasterToken removes the session player's starting token and returns the
// player's fresh private projection after the authoritative mutation.
func (session *PlayerSession) UseCasterToken(
	tokenID model.MatchCardID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.UseCasterToken(
		&session.match.state,
		session.playerID,
		tokenID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("use caster token: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CompleteCurrentPhase(
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	err := engine.CompleteCurrentPhase(
		&session.match.state,
		session.playerID,
		expectedRevision,
	)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("phase transition: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}
