package engine

import (
	"encoding/json"
	"reflect"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestMoveCardUpdatesZonesAndInstance(t *testing.T) {
	for _, tc := range []struct {
		name        string
		zone        model.Zone
		placement   model.DeckPlacement
		face        model.CardFace
		orientation model.CardOrientation
		destination model.PlayerID
	}{
		{"bounce stolen servant", model.ZoneHand, "", model.CardFaceDown, "", "owner"},
		{"deck top", model.ZoneDeck, model.DeckPlacementTop, model.CardFaceDown, "", "owner"},
		{"deck bottom", model.ZoneDeck, model.DeckPlacementBottom, model.CardFaceDown, "", "owner"},
		{"graveyard", model.ZoneGraveyard, "", model.CardFaceUp, "", "owner"},
		{"exile", model.ZoneExile, "", model.CardFaceUp, "", "owner"},
		{"hidden exile", model.ZoneExile, "", model.CardFaceDown, "", "owner"},
		{"convert servant to caster", model.ZoneCaster, "", model.CardFaceDown, model.OrientationRested, "owner"},
		{"transfer control", model.ZoneServant, "", model.CardFaceUp, model.OrientationRested, "owner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, catalog := moveEngineFixture()
			destination, err := playerZoneCards(&state.Players[0], tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			*destination = []model.MatchCardID{"existing"}
			command := model.MoveCardCommand{CardID: "moving", DestinationPlayerID: tc.destination, DestinationZone: tc.zone, DestinationFace: tc.face, EntryOrientation: tc.orientation, Placement: tc.placement}
			if err := MoveCard(&state, catalog, "controller", command, 7); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(state.Players[1].ServantZone, []model.MatchCardID{"before", "after"}) {
				t.Fatalf("source = %v", state.Players[1].ServantZone)
			}
			wantIDs := []model.MatchCardID{"existing", "moving"}
			if tc.placement == model.DeckPlacementTop {
				wantIDs = []model.MatchCardID{"moving", "existing"}
			}
			if !reflect.DeepEqual(*destination, wantIDs) {
				t.Fatalf("destination = %v, want %v", *destination, wantIDs)
			}
			instance := state.CardInstances["moving"]
			if instance.Owner != "owner" || instance.Controller != tc.destination || instance.Face != tc.face || instance.Orientation != tc.orientation || instance.MatchID != "moving" || instance.CardID != "printed" {
				t.Fatalf("unexpected instance: %+v", instance)
			}
			if state.Revision != 8 {
				t.Fatalf("revision = %d; want 8", state.Revision)
			}
			if len(state.CardInstances) != 4 {
				t.Fatal("move added or deleted a card instance")
			}
		})
	}
}

func TestMoveCardRejectionsDoNotMutateState(t *testing.T) {
	for _, reason := range []string{"stale revision", "wrong controller", "missing card", "same zone", "token", "stock", "deck placement", "transfer orientation"} {
		t.Run(reason, func(t *testing.T) {
			state, catalog := moveEngineFixture()
			command := model.MoveCardCommand{CardID: "moving", DestinationPlayerID: "owner", DestinationZone: model.ZoneHand, DestinationFace: model.CardFaceDown}
			actor := model.PlayerID("controller")
			revision := model.Revision(7)
			instance := state.CardInstances["moving"]
			switch reason {
			case "stale revision":
				revision = 6
			case "wrong controller":
				actor = "owner"
			case "missing card":
				command.CardID = "missing"
			case "same zone":
				command.DestinationPlayerID = "controller"
				command.DestinationZone = model.ZoneServant
			case "token":
				instance.CardCategory = model.CategoryTokenCard
			case "stock":
				instance.Stock = []model.MatchCardID{"stock"}
			case "deck placement":
				command.DestinationZone = model.ZoneDeck
			case "transfer orientation":
				command.DestinationZone = model.ZoneServant
				command.DestinationFace = model.CardFaceUp
				command.EntryOrientation = model.OrientationRecovered
			}
			state.CardInstances["moving"] = instance
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := MoveCard(&state, catalog, actor, command, revision); err == nil {
				t.Fatal("expected rejection")
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("rejected move mutated state")
			}
		})
	}
}

func TestMoveCardHandToOrbMarksOwnerKnowledge(t *testing.T) {
	state, catalog := moveEngineFixture()
	state.Players[0].Hand = []model.MatchCardID{"hand-orb"}
	state.CardInstances["hand-orb"] = model.CardInstance{
		MatchID: "hand-orb", CardID: "printed", Owner: "owner", Controller: "owner",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	command := model.MoveCardCommand{
		CardID: "hand-orb", DestinationPlayerID: "owner",
		DestinationZone: model.ZoneOrbs, DestinationFace: model.CardFaceDown,
	}
	if err := MoveCard(&state, catalog, "owner", command, state.Revision); err != nil {
		t.Fatal(err)
	}
	if !model.ViewerKnowsCard(&state, "owner", "hand-orb") {
		t.Fatal("owner should know Orbs placed from hand")
	}
	if model.ViewerKnowsCard(&state, "controller", "hand-orb") {
		t.Fatal("opponent should not learn hand-placed Orbs")
	}
	if !reflect.DeepEqual(state.Players[0].Orbs, []model.MatchCardID{"hand-orb"}) {
		t.Fatalf("Orbs = %#v", state.Players[0].Orbs)
	}
}

func TestMoveCardDeckToOrbStaysUnknown(t *testing.T) {
	state, catalog := moveEngineFixture()
	state.Players[0].Deck = []model.MatchCardID{"deck-orb"}
	state.CardInstances["deck-orb"] = model.CardInstance{
		MatchID: "deck-orb", CardID: "printed", Owner: "owner", Controller: "owner",
		CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown,
	}
	command := model.MoveCardCommand{
		CardID: "deck-orb", DestinationPlayerID: "owner",
		DestinationZone: model.ZoneOrbs, DestinationFace: model.CardFaceDown,
	}
	if err := MoveCard(&state, catalog, "owner", command, state.Revision); err != nil {
		t.Fatal(err)
	}
	if model.ViewerKnowsCard(&state, "owner", "deck-orb") {
		t.Fatal("deck-top Orbs must stay unknown (Compensation-style)")
	}
}

func moveEngineFixture() (model.MatchState, casterAetherCatalogForTest) {
	state := model.MatchState{
		MatchStatus: model.StatusInProgress, Revision: 7,
		Players:       [2]model.PlayerState{{ID: "owner"}, {ID: "controller", ServantZone: []model.MatchCardID{"before", "moving", "after"}}},
		CardInstances: map[model.MatchCardID]model.CardInstance{},
	}
	for _, id := range []model.MatchCardID{"before", "moving", "after", "existing"} {
		state.CardInstances[id] = model.CardInstance{MatchID: id, CardID: "printed", Owner: "owner", Controller: "controller", CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRested}
	}
	return state, casterAetherCatalogForTest{"printed": gamecards.Card{ID: "printed", Type: "Servant"}}
}
