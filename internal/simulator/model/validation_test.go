package model

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateInitialState(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*MatchState)
		wantErrPart string
	}{
		{
			name: "valid initial state",
		},
		{
			name: "empty player ID",
			mutate: func(state *MatchState) {
				state.Players[0].ID = ""
			},
			wantErrPart: "player ID cannot be empty",
		},
		{
			name: "duplicate player IDs",
			mutate: func(state *MatchState) {
				state.Players[1].ID = state.Players[0].ID
			},
			wantErrPart: "player ID cannot be empty or the same",
		},
		{
			name: "unknown first player",
			mutate: func(state *MatchState) {
				state.FirstPlayer = "unknown-player"
			},
			wantErrPart: "no first player set",
		},
		{
			name: "unknown active player",
			mutate: func(state *MatchState) {
				state.Turn.ActivePlayer = "unknown-player"
			},
			wantErrPart: "no active player set",
		},
		{
			name: "active player is not first player",
			mutate: func(state *MatchState) {
				state.Turn.ActivePlayer = state.Players[1].ID
			},
			wantErrPart: "is the player going first",
		},
		{
			name: "zone references unknown card",
			mutate: func(state *MatchState) {
				state.Players[0].Deck[0] = "unknown-card"
			},
			wantErrPart: "references unknown card",
		},
		{
			name: "card appears in multiple zones",
			mutate: func(state *MatchState) {
				state.Players[0].Hand[0] = state.Players[0].Deck[0]
			},
			wantErrPart: "appears in both",
		},
		{
			name: "card is in another player's zone",
			mutate: func(state *MatchState) {
				first := state.Players[0].Deck[0]
				second := state.Players[1].Deck[0]
				state.Players[0].Deck[0] = second
				state.Players[1].Deck[0] = first
			},
			wantErrPart: "is owned by",
		},
		{
			name: "registered card is not assigned to a zone",
			mutate: func(state *MatchState) {
				matchID := MatchCardID("unplaced-card")
				state.CardInstances[matchID] = CardInstance{
					CardID:     "definition-unplaced",
					MatchID:    matchID,
					Owner:      state.Players[0].ID,
					Controller: state.Players[0].ID,
				}
			},
			wantErrPart: "is not assigned to any player zone",
		},
		{
			name: "empty match ID",
			mutate: func(state *MatchState) {
				oldID := state.Players[0].Deck[0]
				instance := state.CardInstances[oldID]
				delete(state.CardInstances, oldID)

				instance.MatchID = ""
				state.CardInstances[""] = instance
				state.Players[0].Deck[0] = ""
			},
			wantErrPart: "match ID cannot be empty",
		},
		{
			name: "empty printed card ID",
			mutate: func(state *MatchState) {
				matchID := state.Players[0].Deck[0]
				instance := state.CardInstances[matchID]
				instance.CardID = ""
				state.CardInstances[matchID] = instance
			},
			wantErrPart: "card ID cannot be empty",
		},
		{
			name: "instance ID differs from map key",
			mutate: func(state *MatchState) {
				matchID := state.Players[0].Deck[0]
				instance := state.CardInstances[matchID]
				instance.MatchID = "different-match-ID"
				state.CardInstances[matchID] = instance
			},
			wantErrPart: "not matching",
		},
		{
			name: "unknown card owner",
			mutate: func(state *MatchState) {
				matchID := state.Players[0].Deck[0]
				instance := state.CardInstances[matchID]
				instance.Owner = "unknown-player"
				state.CardInstances[matchID] = instance
			},
			wantErrPart: "is owned by",
		},
		{
			name: "controller differs from initial owner",
			mutate: func(state *MatchState) {
				matchID := state.Players[0].Deck[0]
				instance := state.CardInstances[matchID]
				instance.Controller = state.Players[1].ID
				state.CardInstances[matchID] = instance
			},
			wantErrPart: "has controller",
		},
		{
			name: "missing caster token",
			mutate: func(state *MatchState) {
				tokenID := state.Players[0].CasterZone[0]
				delete(state.CardInstances, tokenID)
				state.Players[0].CasterZone = nil
			},
			wantErrPart: "expected 1 card in caster zone",
		},
		{
			name: "more than one card in caster zone",
			mutate: func(state *MatchState) {
				state.Players[0].CasterZone = append(
					state.Players[0].CasterZone,
					state.Players[0].Deck[0],
				)
			},
			wantErrPart: "expected 1 card in caster zone",
		},
		{
			name: "caster token has wrong card ID",
			mutate: func(state *MatchState) {
				tokenID := state.Players[0].CasterZone[0]
				token := state.CardInstances[tokenID]
				token.CardID = "not-a-caster-token"
				state.CardInstances[tokenID] = token
			},
			wantErrPart: "caster token in unexpected state",
		},
		{
			name: "caster token has wrong category",
			mutate: func(state *MatchState) {
				tokenID := state.Players[0].CasterZone[0]
				token := state.CardInstances[tokenID]
				token.CardCategory = CategoryPrintedCard
				state.CardInstances[tokenID] = token
			},
			wantErrPart: "caster token in unexpected state",
		},
		{
			name: "caster token is face down",
			mutate: func(state *MatchState) {
				tokenID := state.Players[0].CasterZone[0]
				token := state.CardInstances[tokenID]
				token.Face = CardFaceDown
				state.CardInstances[tokenID] = token
			},
			wantErrPart: "caster token in unexpected state",
		},
		{
			name: "caster token is rested",
			mutate: func(state *MatchState) {
				tokenID := state.Players[0].CasterZone[0]
				token := state.CardInstances[tokenID]
				token.Orientation = OrientationRested
				state.CardInstances[tokenID] = token
			},
			wantErrPart: "caster token in unexpected state",
		},
		{
			name: "caster token controlled by opponent",
			mutate: func(state *MatchState) {
				tokenID := state.Players[0].CasterZone[0]
				token := state.CardInstances[tokenID]
				token.Controller = state.Players[1].ID
				state.CardInstances[tokenID] = token
			},
			wantErrPart: "caster token",
		},
		{
			name: "wrong orb count",
			mutate: func(state *MatchState) {
				state.Players[0].Orbs = state.Players[0].Orbs[:6]
			},
			wantErrPart: "must have 7 orbs",
		},
		{
			name: "wrong hand count",
			mutate: func(state *MatchState) {
				state.Players[0].Hand = state.Players[0].Hand[:6]
			},
			wantErrPart: "must have 7 cards in hand",
		},
		{
			name: "wrong total card count",
			mutate: func(state *MatchState) {
				state.Players[0].Deck = state.Players[0].Deck[:35]
			},
			wantErrPart: "total number of cards is not 50",
		},
		{
			name: "wrong match status",
			mutate: func(state *MatchState) {
				state.MatchStatus = StatusSetup
			},
			wantErrPart: "must be \"In Progress\"",
		},
		{
			name: "wrong turn count",
			mutate: func(state *MatchState) {
				state.Turn.Number = 2
			},
			wantErrPart: "must be 1",
		},
		{
			name: "wrong phase",
			mutate: func(state *MatchState) {
				state.Turn.Phase = PhaseDraw
			},
			wantErrPart: "must be \"Call\"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := validInitialStateForTest()
			if test.mutate != nil {
				test.mutate(&state)
			}

			err := state.ValidateInitialState()
			if test.wantErrPart == "" {
				if err != nil {
					t.Fatalf("ValidateInitialState() returned unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("ValidateInitialState() returned nil; want error containing %q", test.wantErrPart)
			}
			if !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("ValidateInitialState() error = %q; want it to contain %q", err, test.wantErrPart)
			}
		})
	}
}

func validInitialStateForTest() MatchState {
	const cardsPerPlayer = 50

	state := MatchState{
		CardInstances: make(map[MatchCardID]CardInstance, cardsPerPlayer*2+2),
		Players: [2]PlayerState{
			{ID: "player-one"},
			{ID: "player-two"},
		},
		FirstPlayer: "player-one",
		MatchStatus: StatusInProgress,
		Turn: TurnState{
			Number:       1,
			ActivePlayer: "player-one",
			Phase:        PhaseCall,
		},
	}

	for playerIndex := range state.Players {
		player := &state.Players[playerIndex]
		for cardIndex := 0; cardIndex < cardsPerPlayer; cardIndex++ {
			matchID := MatchCardID(fmt.Sprintf("player-%d-card-%02d", playerIndex+1, cardIndex+1))
			state.CardInstances[matchID] = CardInstance{
				CardID:       CardID(fmt.Sprintf("definition-%02d", cardIndex+1)),
				MatchID:      matchID,
				Owner:        player.ID,
				Controller:   player.ID,
				CardCategory: CategoryPrintedCard,
			}

			switch {
			case cardIndex < 36:
				player.Deck = append(player.Deck, matchID)
			case cardIndex < 43:
				player.Hand = append(player.Hand, matchID)
			default:
				player.Orbs = append(player.Orbs, matchID)
			}
		}

		tokenID := MatchCardID(fmt.Sprintf("player-%d-%s", playerIndex+1, CasterTokenCardID))
		state.CardInstances[tokenID] = CardInstance{
			CardID:       CasterTokenCardID,
			MatchID:      tokenID,
			Owner:        player.ID,
			Controller:   player.ID,
			CardCategory: CategoryTokenCard,
			Face:         CardFaceUp,
			Orientation:  OrientationRecovered,
		}
		player.CasterZone = []MatchCardID{tokenID}
	}

	return state
}
