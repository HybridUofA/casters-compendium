package view

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestProjectMatchProtectsHiddenInformation(t *testing.T) {
	state := projectionStateForTest()
	state.Revision = 42

	result, err := ProjectMatch(state, state.Players[0].ID)
	if err != nil {
		t.Fatalf("ProjectMatch() returned unexpected error: %v", err)
	}

	if result.ViewerID != state.Players[0].ID ||
		result.Turn != state.Turn ||
		result.MatchStatus != state.MatchStatus ||
		result.Revision != state.Revision {
		t.Fatal("ProjectMatch() did not preserve public match metadata")
	}
	if result.Players[0].DeckCount != len(state.Players[0].Deck) ||
		result.Players[1].DeckCount != len(state.Players[1].Deck) {
		t.Fatal("ProjectMatch() did not preserve public deck counts")
	}
	if result.Players[0].Aether != state.Players[0].Aether ||
		result.Players[1].Aether != state.Players[1].Aether {
		t.Fatal("ProjectMatch() did not preserve public Aether pools")
	}

	assertVisibleCard(t, result.Players[0].Hand[0], "p1-hand", "card-p1-hand")
	assertConcealedCard(t, result.Players[1].Hand[0])
	assertConcealedCard(t, result.Players[0].Orbs[0])
	assertConcealedCard(t, result.Players[1].Orbs[0])

	assertVisibleCard(t, result.Players[0].CasterZone[0], "p1-token", model.CasterTokenCardID)
	assertVisibleCard(t, result.Players[1].CasterZone[0], "p2-token", model.CasterTokenCardID)

	ownFaceDown := result.Players[0].CasterZone[1]
	if ownFaceDown.ShowFace ||
		ownFaceDown.MatchID != "p1-facedown-caster" ||
		ownFaceDown.CardID != "card-p1-facedown-caster" {
		t.Fatalf("own face-down Caster projection = %#v; want known identity rendered face down", ownFaceDown)
	}

	opponentFaceDown := result.Players[1].CasterZone[1]
	assertConcealedCard(t, opponentFaceDown)
	if opponentFaceDown.Face != model.CardFaceDown ||
		opponentFaceDown.Orientation != model.OrientationRested {
		t.Fatalf("opponent face-down Caster projection lost public position: %#v", opponentFaceDown)
	}

	assertVisibleCard(t, result.Players[0].ServantZone[0], "p1-servant", "card-p1-servant")
	opponentFaceDownServant := result.Players[1].ServantZone[0]
	assertConcealedCard(t, opponentFaceDownServant)
	if opponentFaceDownServant.Face != model.CardFaceDown ||
		opponentFaceDownServant.Orientation != model.OrientationReversed {
		t.Fatalf("opponent face-down Servant projection lost public position: %#v", opponentFaceDownServant)
	}
	assertVisibleCard(t, result.Players[0].Graveyard[0], "p1-grave", "card-p1-grave")
	assertVisibleCard(t, result.Players[1].Graveyard[0], "p2-grave", "card-p2-grave")
	assertVisibleCard(t, result.Players[0].Exile[0], "p1-exile", "card-p1-exile")
	assertVisibleCard(t, result.Players[1].Exile[0], "p2-exile", "card-p2-exile")
}

func TestProjectMatchUsesViewerPerspectiveForEitherPlayer(t *testing.T) {
	state := projectionStateForTest()

	result, err := ProjectMatch(state, state.Players[1].ID)
	if err != nil {
		t.Fatalf("ProjectMatch() returned unexpected error: %v", err)
	}

	assertConcealedCard(t, result.Players[0].Hand[0])
	assertVisibleCard(t, result.Players[1].Hand[0], "p2-hand", "card-p2-hand")

	assertConcealedCard(t, result.Players[0].CasterZone[1])
	ownFaceDown := result.Players[1].CasterZone[1]
	if ownFaceDown.ShowFace ||
		ownFaceDown.MatchID != "p2-facedown-caster" ||
		ownFaceDown.CardID != "card-p2-facedown-caster" {
		t.Fatalf("player two face-down Caster projection = %#v; want known identity rendered face down", ownFaceDown)
	}
	assertVisibleCard(t, result.Players[0].ServantZone[0], "p1-servant", "card-p1-servant")
	ownFaceDownServant := result.Players[1].ServantZone[0]
	if ownFaceDownServant.ShowFace ||
		ownFaceDownServant.MatchID != "p2-facedown-servant" ||
		ownFaceDownServant.CardID != "card-p2-facedown-servant" {
		t.Fatalf("player two face-down Servant projection = %#v; want known identity rendered face down", ownFaceDownServant)
	}
}

func TestProjectMatchRejectsUnknownViewer(t *testing.T) {
	_, err := ProjectMatch(projectionStateForTest(), "spectator")
	if err == nil || !strings.Contains(err.Error(), "not in current player IDs") {
		t.Fatalf("ProjectMatch() error = %v; want unknown-viewer error", err)
	}
}

func TestProjectMatchRejectsBrokenZoneReferences(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*model.MatchState)
		wantErrPart string
	}{
		{
			name: "hand",
			mutate: func(state *model.MatchState) {
				state.Players[0].Hand[0] = "missing-hand-card"
			},
			wantErrPart: "hand",
		},
		{
			name: "orbs",
			mutate: func(state *model.MatchState) {
				state.Players[0].Orbs[0] = "missing-orb"
			},
			wantErrPart: "orbs",
		},
		{
			name: "caster zone",
			mutate: func(state *model.MatchState) {
				state.Players[0].CasterZone[0] = "missing-caster"
			},
			wantErrPart: "caster zone",
		},
		{
			name: "servant zone",
			mutate: func(state *model.MatchState) {
				state.Players[0].ServantZone[0] = "missing-servant"
			},
			wantErrPart: "servant zone",
		},
		{
			name: "graveyard",
			mutate: func(state *model.MatchState) {
				state.Players[0].Graveyard[0] = "missing-graveyard-card"
			},
			wantErrPart: "graveyard",
		},
		{
			name: "exile",
			mutate: func(state *model.MatchState) {
				state.Players[0].Exile[0] = "missing-exiled-card"
			},
			wantErrPart: "removed from game zone",
		},
		{
			name: "invalid caster face",
			mutate: func(state *model.MatchState) {
				id := state.Players[0].CasterZone[0]
				instance := state.CardInstances[id]
				instance.Face = "invalid"
				state.CardInstances[id] = instance
			},
			wantErrPart: "invalid face state",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := projectionStateForTest()
			test.mutate(&state)

			_, err := ProjectMatch(state, state.Players[0].ID)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("ProjectMatch() error = %v; want containing %q", err, test.wantErrPart)
			}
		})
	}
}

func assertVisibleCard(
	t *testing.T,
	card CardView,
	wantMatchID model.MatchCardID,
	wantCardID model.CardID,
) {
	t.Helper()
	if !card.ShowFace || card.MatchID != wantMatchID || card.CardID != wantCardID {
		t.Fatalf("visible CardView = %#v; want match ID %q and card ID %q", card, wantMatchID, wantCardID)
	}
}

func assertConcealedCard(t *testing.T, card CardView) {
	t.Helper()
	if card.ShowFace || card.MatchID != "" || card.CardID != "" {
		t.Fatalf("concealed CardView leaked information: %#v", card)
	}
}

func projectionStateForTest() model.MatchState {
	state := model.MatchState{
		CardInstances: make(map[model.MatchCardID]model.CardInstance),
		Players: [2]model.PlayerState{
			{
				ID: "player-one",
				Aether: model.AetherPool{
					Aes: 2,
				},
				Deck:        []model.MatchCardID{"p1-deck-1", "p1-deck-2"},
				Hand:        []model.MatchCardID{"p1-hand"},
				Orbs:        []model.MatchCardID{"p1-orb"},
				CasterZone:  []model.MatchCardID{"p1-token", "p1-facedown-caster"},
				ServantZone: []model.MatchCardID{"p1-servant"},
				Graveyard:   []model.MatchCardID{"p1-grave"},
				Exile:       []model.MatchCardID{"p1-exile"},
			},
			{
				ID: "player-two",
				Aether: model.AetherPool{
					Void:         1,
					NonElemental: 3,
				},
				Deck:        []model.MatchCardID{"p2-deck-1"},
				Hand:        []model.MatchCardID{"p2-hand"},
				Orbs:        []model.MatchCardID{"p2-orb"},
				CasterZone:  []model.MatchCardID{"p2-token", "p2-facedown-caster"},
				ServantZone: []model.MatchCardID{"p2-facedown-servant"},
				Graveyard:   []model.MatchCardID{"p2-grave"},
				Exile:       []model.MatchCardID{"p2-exile"},
			},
		},
		MatchStatus: model.StatusSetup,
		Turn: model.TurnState{
			ActivePlayer:    "player-one",
			Phase:           model.PhaseCall,
			CallActionTaken: true,
		},
	}

	addInstance := func(
		matchID model.MatchCardID,
		cardID model.CardID,
		owner model.PlayerID,
		face model.CardFace,
		orientation model.CardOrientation,
	) {
		state.CardInstances[matchID] = model.CardInstance{
			MatchID:      matchID,
			CardID:       cardID,
			Owner:        owner,
			Controller:   owner,
			CardCategory: model.CategoryPrintedCard,
			Face:         face,
			Orientation:  orientation,
		}
	}

	addInstance("p1-hand", "card-p1-hand", "player-one", "", "")
	addInstance("p1-orb", "card-p1-orb", "player-one", "", "")
	addInstance("p2-hand", "card-p2-hand", "player-two", "", "")
	addInstance("p2-orb", "card-p2-orb", "player-two", "", "")
	addInstance("p1-token", model.CasterTokenCardID, "player-one", model.CardFaceUp, model.OrientationRecovered)
	addInstance("p2-token", model.CasterTokenCardID, "player-two", model.CardFaceUp, model.OrientationRecovered)
	addInstance("p1-facedown-caster", "card-p1-facedown-caster", "player-one", model.CardFaceDown, model.OrientationRecovered)
	addInstance("p2-facedown-caster", "card-p2-facedown-caster", "player-two", model.CardFaceDown, model.OrientationRested)
	addInstance("p1-servant", "card-p1-servant", "player-one", model.CardFaceUp, model.OrientationRested)
	addInstance("p2-facedown-servant", "card-p2-facedown-servant", "player-two", model.CardFaceDown, model.OrientationReversed)
	addInstance("p1-grave", "card-p1-grave", "player-one", model.CardFaceUp, model.OrientationRecovered)
	addInstance("p2-grave", "card-p2-grave", "player-two", model.CardFaceUp, model.OrientationRecovered)
	addInstance("p1-exile", "card-p1-exile", "player-one", model.CardFaceUp, model.OrientationRecovered)
	addInstance("p2-exile", "card-p2-exile", "player-two", model.CardFaceUp, model.OrientationRecovered)

	return state
}
