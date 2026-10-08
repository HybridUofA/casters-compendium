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
	random  engine.RandomSource
	catalog rules.CardCatalog
}

type PlayerSession struct {
	match    *LocalMatch
	playerID model.PlayerID
}

// AdjustAether manually adds or removes one unit from the bound player's pool.
func (session *PlayerSession) AdjustAether(
	command model.AdjustAetherCommand,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.AdjustAether(&session.match.state, session.playerID, command, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("adjust aether: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// MoveCard binds manual movement to the authenticated local player and returns
// only that player's projection. Hidden Exile needs explicit viewing grants.
func (session *PlayerSession) MoveCard(command model.MoveCardCommand, expectedRevision model.Revision) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if command.DestinationZone == model.ZoneExile && command.DestinationFace == model.CardFaceDown {
		return view.MatchView{}, fmt.Errorf("face-down Exile movement is not yet supported by the player interface")
	}
	if err := engine.MoveCard(&session.match.state, session.match.catalog, session.playerID, command, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("move card: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// DrawCards draws count cards from the bound player's deck into their hand.
func (session *PlayerSession) DrawCards(count int, expectedRevision model.Revision) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.ManualDrawCards(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		count,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("draw cards: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// ShuffleDeck shuffles the bound player's deck using the match random stream.
func (session *PlayerSession) ShuffleDeck(expectedRevision model.Revision) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.ShufflePlayerDeck(&session.match.state, session.match.random, session.playerID, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("shuffle deck: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// PeekDeckTops returns the top count cards of deckOwner's library for the bound
// player only. It does not mutate match state or bump revision.
func (session *PlayerSession) PeekDeckTops(
	deckOwnerID model.PlayerID,
	count int,
) (peeked []view.CardView, matchView view.MatchView, err error) {
	if session == nil || session.match == nil {
		return nil, view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.RLock()
	defer session.match.mu.RUnlock()
	tops, err := engine.PeekDeckTops(&session.match.state, deckOwnerID, count)
	if err != nil {
		return nil, view.MatchView{}, fmt.Errorf("peek deck: %w", err)
	}
	peeked = make([]view.CardView, 0, len(tops))
	for _, matchID := range tops {
		instance, exists := session.match.state.CardInstances[matchID]
		if !exists {
			return nil, view.MatchView{}, fmt.Errorf("peek deck: card %q missing", matchID)
		}
		peeked = append(peeked, view.CardView{
			MatchID:  matchID,
			CardID:   instance.CardID,
			Face:     model.CardFaceDown,
			ShowFace: true,
			Owner:    instance.Owner,
			HasStock: len(instance.Stock) > 0,
		})
	}
	matchView, err = view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return nil, view.MatchView{}, err
	}
	return peeked, matchView, nil
}

// MoveDeckTopToBottom puts the top card of deckOwner's deck on the bottom.
func (session *PlayerSession) MoveDeckTopToBottom(
	deckOwnerID model.PlayerID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.MoveDeckTopToBottom(
		&session.match.state,
		session.playerID,
		deckOwnerID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("move deck top to bottom: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// ResolveDeckDig keeps one peeked card into hand and bottoms the rest in order.
func (session *PlayerSession) ResolveDeckDig(
	keep model.MatchCardID,
	bottomOrder []model.MatchCardID,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.ResolveDeckDig(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		session.playerID,
		keep,
		bottomOrder,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("resolve deck dig: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// AcceptSageAdvice chooses the dig replacement for the next pending draw.
func (session *PlayerSession) AcceptSageAdvice(
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.AcceptSageAdvice(&session.match.state, session.playerID, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("accept sage advice: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// DeclineDrawReplacement draws normally instead of using Sage Advice.
func (session *PlayerSession) DeclineDrawReplacement(
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.DeclineDrawReplacement(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("decline draw replacement: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
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
		random:  engine.NewSeededRandom(seed),
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

// CallFaceUpLevelOne calls one eligible Level 1 Caster from the session
// player's hand and returns that player's fresh private projection.
func (session *PlayerSession) CallFaceUpLevelOne(
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
	if err := engine.CallFaceUpLevelOne(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		cardID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("call face-up Level 1 Caster: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

// LevelUpCaster places an eligible Caster from the session player's hand on
// top of a chosen face-up Caster and returns a fresh private projection.
func (session *PlayerSession) LevelUpCaster(
	upperCardID model.MatchCardID,
	targetCasterID model.MatchCardID,
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
	if err := engine.LevelUpCaster(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		upperCardID,
		targetCasterID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("level up Caster: %w", err)
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

// GenerateCasterAether rests one eligible face-up Caster and returns the
// session player's fresh private projection after the authoritative mutation.
func (session *PlayerSession) GenerateCasterAether(
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
	if err := engine.GenerateCasterAether(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		cardID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("generate Caster Aether: %w", err)
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
		session.match.catalog,
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

func (session *PlayerSession) CastServant(
	cardID model.MatchCardID,
	payment model.AetherPayment,
	orientation model.CardOrientation,
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
	err := engine.CastServant(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		cardID,
		payment,
		orientation,
		expectedRevision,
	)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("cast servant: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CastServantWithPlan(
	cardID model.MatchCardID,
	plan model.CastPaymentPlan,
	orientation model.CardOrientation,
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
	if err := engine.CastServantWithPlan(&session.match.state, session.match.catalog, session.playerID, cardID, plan, orientation, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("cast servant with payment plan: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CastConjure(
	cardID model.MatchCardID,
	payment model.AetherPayment,
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
	err := engine.CastConjure(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		cardID,
		payment,
		expectedRevision,
	)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("cast conjure: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CastConjureWithPlan(cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.CastConjureWithPlan(&session.match.state, session.match.catalog, session.playerID, cardID, plan, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("cast conjure with payment plan: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CastBarrier(
	cardID model.MatchCardID,
	payment model.AetherPayment,
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
	err := engine.CastBarrier(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		cardID,
		payment,
		expectedRevision,
	)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("cast barrier: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) CastBarrierWithPlan(cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) (view.MatchView, error) {
	if session == nil {
		return view.MatchView{}, fmt.Errorf("session cannot be nil")
	}
	if session.match == nil {
		return view.MatchView{}, fmt.Errorf("match cannot be nil")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.CastBarrierWithPlan(&session.match.state, session.match.catalog, session.playerID, cardID, plan, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("cast barrier with payment plan: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) PassPriority(
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
	err := engine.PassPriority(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		expectedRevision,
	)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("pass priority: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

func (session *PlayerSession) DeclareAttack(
	attackerID model.MatchCardID,
	targetKind model.AttackTargetKind,
	targetCardID model.MatchCardID,
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
	if err := engine.DeclareAttack(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		attackerID,
		targetKind,
		targetCardID,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("declare attack: %w", err)
	}
	updatedView, err := view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.MatchView{}, fmt.Errorf("project updated match: %w", err)
	}
	return updatedView, nil
}

// CorruptOrb chooses one enemy Orb to corrupt after a player-attack judgment.
func (session *PlayerSession) CorruptOrb(
	orbIndex int,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	return session.CorruptOrbs([]int{orbIndex}, expectedRevision)
}

// CorruptOrbs chooses one or more enemy Orbs to corrupt (Double Corrupt).
func (session *PlayerSession) CorruptOrbs(
	orbIndexes []int,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.CorruptOrbs(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		orbIndexes,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("corrupt orbs: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// SetGrantedDoubleCorrupt toggles a manual Double Corrupt marker on a Servant.
func (session *PlayerSession) SetGrantedDoubleCorrupt(
	cardID model.MatchCardID,
	enabled bool,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.SetGrantedDoubleCorrupt(
		&session.match.state,
		session.playerID,
		cardID,
		enabled,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("set granted double corrupt: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// PlayBreak uses the optional Break on a just-corrupted Orb card.
func (session *PlayerSession) PlayBreak(
	cardID model.MatchCardID,
	entryOrientation model.CardOrientation,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.PlayBreak(
		&session.match.state,
		session.match.catalog,
		session.playerID,
		cardID,
		entryOrientation,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("play break: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// DeclineBreak skips the optional Break for the currently offered card.
func (session *PlayerSession) DeclineBreak(
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if err := engine.DeclineBreak(&session.match.state, session.playerID, expectedRevision); err != nil {
		return view.MatchView{}, fmt.Errorf("decline break: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}

// PeekOrb looks at an Orb and remembers its identity for the bound player.
func (session *PlayerSession) PeekOrb(
	orbOwnerID model.PlayerID,
	orbIndex int,
	expectedRevision model.Revision,
) (peeked view.CardView, matchView view.MatchView, err error) {
	if session == nil || session.match == nil {
		return view.CardView{}, view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	orbID, err := engine.PeekOrb(
		&session.match.state,
		session.playerID,
		orbOwnerID,
		orbIndex,
		expectedRevision,
	)
	if err != nil {
		return view.CardView{}, view.MatchView{}, fmt.Errorf("peek orb: %w", err)
	}
	instance := session.match.state.CardInstances[orbID]
	peeked = view.CardView{
		MatchID:  orbID,
		CardID:   instance.CardID,
		Face:     model.CardFaceDown,
		ShowFace: true,
		Owner:    instance.Owner,
	}
	matchView, err = view.ProjectMatch(session.match.state, session.playerID)
	if err != nil {
		return view.CardView{}, view.MatchView{}, err
	}
	return peeked, matchView, nil
}

// RevealOrb shows one of the bound player's Orbs to the opponent permanently.
func (session *PlayerSession) RevealOrb(
	orbIndex int,
	expectedRevision model.Revision,
) (view.MatchView, error) {
	if session == nil || session.match == nil {
		return view.MatchView{}, fmt.Errorf("session and match must exist")
	}
	session.match.mu.Lock()
	defer session.match.mu.Unlock()
	if _, err := engine.RevealOrb(
		&session.match.state,
		session.playerID,
		orbIndex,
		expectedRevision,
	); err != nil {
		return view.MatchView{}, fmt.Errorf("reveal orb: %w", err)
	}
	return view.ProjectMatch(session.match.state, session.playerID)
}
