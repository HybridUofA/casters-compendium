package engine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/rules"
)

type PlayerSetup struct {
	ID   model.PlayerID
	Deck decks.Deck
}

type SetupInput struct {
	Players [2]PlayerSetup
	Random  RandomSource
	Catalog decks.CardCatalog
}

// OpeningHandDecision records whether a player keeps or replaces cards from
// their opening hand. An empty Replace slice means the player keeps every card.
type OpeningHandDecision struct {
	PlayerID         model.PlayerID
	Replace          []model.MatchCardID
	ExpectedRevision model.Revision
}

func (setup SetupInput) Validate() error {
	if setup.Players[0].ID == "" || setup.Players[1].ID == "" {
		return fmt.Errorf("player ID fields cannot be empty")
	}

	if setup.Players[0].ID == setup.Players[1].ID {
		return fmt.Errorf("player IDs must be different")
	}

	if setup.Random == nil {
		return fmt.Errorf("random source cannot be nil")
	}

	if setup.Catalog == nil {
		return fmt.Errorf("card catalog cannot be nil")
	}

	for _, player := range setup.Players {
		if err := player.Deck.Validate(); err != nil {
			return fmt.Errorf("%q is invalid: %w", player.ID, err)
		}
		for _, entry := range player.Deck.MainDeck {
			if strings.TrimSpace(entry.CardID) == "" {
				return fmt.Errorf("card ID cannot be empty")
			}
			if entry.Quantity <= 0 {
				return fmt.Errorf("card quantity cannot be less than 1")
			}
		}
		if player.Deck.MainTotal() != rules.StandardDeckSize {
			return fmt.Errorf("deck size is %d, must be %d", player.Deck.MainTotal(), rules.StandardDeckSize)
		}
		if err := player.Deck.ValidateCopyLimits(); err != nil {
			return fmt.Errorf("player %q deck failed copy-limit validation: %w", player.ID, err)
		}
		if err := player.Deck.ValidateCards(setup.Catalog); err != nil {
			return fmt.Errorf("%q deck failed card validation: %w", player.ID, err)
		}
	}
	return nil
}

func expandPlayerDeck(setup PlayerSetup, playerIndex int) (model.PlayerState, map[model.MatchCardID]model.CardInstance, error) {
	playerState := model.PlayerState{
		ID:   setup.ID,
		Deck: make([]model.MatchCardID, 0, rules.StandardDeckSize),
	}
	instances := make(
		map[model.MatchCardID]model.CardInstance, rules.StandardDeckSize,
	)
	cardCount := 0
	for _, entry := range setup.Deck.MainDeck {
		for range entry.Quantity {
			cardCount++
			matchID := model.MatchCardID(fmt.Sprintf("player-%d-card-%d", playerIndex+1, cardCount))
			instance := model.CardInstance{
				CardID:       model.CardID(entry.CardID),
				MatchID:      matchID,
				Owner:        setup.ID,
				Controller:   setup.ID,
				CardCategory: model.CategoryPrintedCard,
			}
			playerState.Deck = append(playerState.Deck, matchID)
			instances[matchID] = instance
		}
	}
	if len(playerState.Deck) != rules.StandardDeckSize || len(instances) != rules.StandardDeckSize {
		return model.PlayerState{}, nil, fmt.Errorf("deck count or instances do not match standard deck size: %d, %d", len(playerState.Deck), len(instances))
	}
	return playerState, instances, nil
}

func selectFirstPlayer(random RandomSource, players [2]PlayerSetup) (model.PlayerID, error) {
	index := random.RandInt(2)
	if index > 1 || index < 0 {
		return "", fmt.Errorf("player index out of bounds")
	}
	return players[index].ID, nil
}

func BeginSetup(setup SetupInput) (model.MatchState, error) {
	err := setup.Validate()
	if err != nil {
		return model.MatchState{}, fmt.Errorf("error in validation: %w", err)
	}
	matchState := model.MatchState{
		CardInstances: make(map[model.MatchCardID]model.CardInstance, rules.StandardDeckSize*2),
		MatchStatus:   model.StatusSetup,
	}
	for index, player := range setup.Players {
		playerState, playerInstances, err := expandPlayerDeck(player, index)
		if err != nil {
			return model.MatchState{}, fmt.Errorf("could not expand deck of player %d: %w", index+1, err)
		}
		token := generateCasterToken(player.ID, index)
		if _, exists := matchState.CardInstances[token.MatchID]; exists {
			return model.MatchState{}, fmt.Errorf("token already exists in match state for %q", player.ID)
		}
		matchState.CardInstances[token.MatchID] = token
		playerState.CasterZone = append(playerState.CasterZone, token.MatchID)
		for matchID, instance := range playerInstances {
			_, alreadyExists := matchState.CardInstances[matchID]
			if alreadyExists {
				return model.MatchState{}, fmt.Errorf("card %q already exists in player %d's deck", matchID, index+1)
			}
			matchState.CardInstances[matchID] = instance
		}
		if err := shuffleDeck(playerState.Deck, setup.Random); err != nil {
			return model.MatchState{}, fmt.Errorf("player %d's deck failed to shuffle: %w", index+1, err)
		}
		if playerState.Hand, playerState.Deck, err = takeTopCards(playerState.Deck, rules.OpeningHandSize); err != nil {
			return model.MatchState{}, fmt.Errorf("error placing cards into %q's hand: %w", player.ID, err)
		}
		playerState.Orbs, playerState.Deck, err = takeTopCards(playerState.Deck, rules.StartingOrbCount)
		if err != nil {
			return model.MatchState{}, fmt.Errorf("error placing cards into %q's orb zone: %w", player.ID, err)
		}
		matchState.Players[index] = playerState
	}
	firstID, err := selectFirstPlayer(setup.Random, setup.Players)
	if err != nil {
		return model.MatchState{}, fmt.Errorf("error selecting first player: %w", err)
	}
	matchState.FirstPlayer = firstID
	matchState.Turn.ActivePlayer = firstID
	return matchState, nil
}

func takeTopCards(deck []model.MatchCardID, count int) (taken []model.MatchCardID, remaining []model.MatchCardID, err error) {
	if count < 0 {
		return nil, nil, fmt.Errorf("count cannot be negative: %d", count)
	}
	if count > len(deck) {
		return nil, nil, fmt.Errorf("count cannot be greater than amount of cards in deck (%d): %d", len(deck), count)
	}
	taken = slices.Clone(deck[:count])
	remaining = slices.Clone(deck[count:])
	return taken, remaining, nil
}

func ApplyOpeningHandDecision(state *model.MatchState, decision OpeningHandDecision) error {
	if state == nil {
		return fmt.Errorf("match state cannot be nil")
	}
	if state.MatchStatus != model.StatusSetup {
		return fmt.Errorf("match state must be in the set up phase to make opening hand decisions")
	}
	if decision.PlayerID == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if decision.ExpectedRevision != state.Revision {
		return fmt.Errorf("expected revision %d does not match current revision %d", decision.ExpectedRevision, state.Revision)
	}
	playerIndex := -1
	for index, player := range state.Players {
		if player.ID == decision.PlayerID {
			playerIndex = index
			break
		}
	}
	if playerIndex == -1 {
		return fmt.Errorf("unknown player ID: %q", decision.PlayerID)
	}
	if state.Players[playerIndex].OpeningHandFinalized {
		return fmt.Errorf("opening hand decision has already been finalized.")
	}

	player := state.Players[playerIndex]
	if len(decision.Replace) > len(player.Hand) {
		return fmt.Errorf("cannot replace more cards (%d) than cards in hand %d", len(decision.Replace), len(player.Hand))
	}
	selected := make(
		map[model.MatchCardID]struct{},
		len(decision.Replace),
	)
	for _, cardID := range decision.Replace {
		if _, alreadySelected := selected[cardID]; alreadySelected {
			return fmt.Errorf("same card may not be selected twice")
		}
		if !slices.Contains(player.Hand, cardID) {
			return fmt.Errorf("card is not in %q's hand", player.ID)
		}
		selected[cardID] = struct{}{}
	}
	var keptHand []model.MatchCardID
	for _, card := range player.Hand {
		_, isSelected := selected[card]
		if isSelected {
			continue
		}
		keptHand = append(keptHand, card)
	}
	candidateDeck := slices.Clone(player.Deck)
	candidateDeck = append(candidateDeck, decision.Replace...)
	drawnCards, remaining, err := takeTopCards(candidateDeck, len(decision.Replace))
	if err != nil {
		return fmt.Errorf("error replacing cards from deck: %w", err)
	}
	completedHand := slices.Clone(keptHand)
	completedHand = append(completedHand, drawnCards...)
	if len(completedHand) != rules.OpeningHandSize {
		return fmt.Errorf("hand size is %d, needs to be %d", len(completedHand), rules.OpeningHandSize)
	}
	state.Players[playerIndex].Deck = remaining
	state.Players[playerIndex].Hand = completedHand
	state.Players[playerIndex].OpeningHandFinalized = true
	state.Revision++
	for _, player := range state.Players {
		if player.OpeningHandFinalized == false {
			return nil
		}
	}
	state.MatchStatus = model.StatusInProgress
	state.Turn.Number = 1
	state.Turn.Phase = model.PhaseRecovery
	return nil
}

func generateCasterToken(playerID model.PlayerID, playerIndex int) model.CardInstance {
	matchID := model.MatchCardID(fmt.Sprintf("player-%d-%s", playerIndex+1, model.CasterTokenCardID))
	casterToken := model.CardInstance{
		CardID:       model.CasterTokenCardID,
		MatchID:      matchID,
		Owner:        playerID,
		Controller:   playerID,
		CardCategory: model.CategoryTokenCard,
		Face:         model.CardFaceUp,
		Orientation:  model.OrientationRecovered,
	}
	return casterToken
}
