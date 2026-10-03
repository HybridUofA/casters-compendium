package rules

import (
	"encoding/json"
	"testing"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestValidateMoveCardPlacement(t *testing.T) {
	for _, tc := range []struct {
		name        string
		cardType    string
		zone        model.Zone
		face        model.CardFace
		orientation model.CardOrientation
		valid       bool
	}{
		{"servant becomes caster", "Servant", model.ZoneCaster, model.CardFaceDown, model.OrientationRecovered, true},
		{"rested face-down caster", "Conjure", model.ZoneCaster, model.CardFaceDown, model.OrientationRested, true},
		{"face-up caster", " cAsTeR ", model.ZoneCaster, model.CardFaceUp, model.OrientationRecovered, true},
		{"face-up non-caster", "Servant", model.ZoneCaster, model.CardFaceUp, model.OrientationRecovered, false},
		{"reversed caster", "Caster", model.ZoneCaster, model.CardFaceUp, model.OrientationReversed, false},
		{"missing caster orientation", "Caster", model.ZoneCaster, model.CardFaceUp, "", false},
		{"invalid destination face", "Caster", model.ZoneCaster, "invalid", model.OrientationRecovered, false},
		{"recovered servant", "Servant", model.ZoneServant, model.CardFaceUp, model.OrientationRecovered, true},
		{"rested servant", "Servant", model.ZoneServant, model.CardFaceUp, model.OrientationRested, true},
		{"reversed servant", " sErVaNt ", model.ZoneServant, model.CardFaceUp, model.OrientationReversed, true},
		{"invalid servant orientation", "Servant", model.ZoneServant, model.CardFaceUp, "invalid", false},
		{"barrier", " bArRiEr ", model.ZoneServant, model.CardFaceUp, model.OrientationRecovered, true},
		{"rested barrier", "Barrier", model.ZoneServant, model.CardFaceUp, model.OrientationRested, true},
		{"reversed barrier", "Barrier", model.ZoneServant, model.CardFaceUp, model.OrientationReversed, false},
		{"conjure in servant zone", "Conjure", model.ZoneServant, model.CardFaceUp, model.OrientationRecovered, false},
		{"face-down servant", "Servant", model.ZoneServant, model.CardFaceDown, model.OrientationRecovered, false},
		{"hand", "Servant", model.ZoneHand, model.CardFaceDown, "", true},
		{"deck", "Servant", model.ZoneDeck, model.CardFaceDown, "", true},
		{"orbs", "Servant", model.ZoneOrbs, model.CardFaceDown, "", true},
		{"graveyard", "Servant", model.ZoneGraveyard, model.CardFaceUp, "", true},
		{"face-down graveyard", "Servant", model.ZoneGraveyard, model.CardFaceDown, "", false},
		{"public exile", "Servant", model.ZoneExile, model.CardFaceUp, "", true},
		{"hidden exile", "Servant", model.ZoneExile, model.CardFaceDown, "", true},
		{"invalid exile face", "Servant", model.ZoneExile, "invalid", "", false},
		{"non-field orientation", "Servant", model.ZoneHand, model.CardFaceDown, model.OrientationRecovered, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, originalFace := range []model.CardFace{model.CardFaceUp, model.CardFaceDown} {
				state, catalog := moveStateForTest(tc.cardType)
				instance := state.CardInstances["card"]
				instance.Face = originalFace
				state.CardInstances["card"] = instance
				// Use a different source so every destination represents a move.
				state.Players[0].Graveyard = []model.MatchCardID{"card"}
				if tc.zone == model.ZoneGraveyard {
					state.Players[0].Graveyard = nil
					state.Players[0].Hand = []model.MatchCardID{"card"}
				}
				command := model.MoveCardCommand{CardID: "card", DestinationPlayerID: "owner", DestinationZone: tc.zone, DestinationFace: tc.face, EntryOrientation: tc.orientation}
				if tc.zone == model.ZoneDeck {
					command.Placement = model.DeckPlacementTop
				}
				before, err := json.Marshal(state)
				if err != nil {
					t.Fatal(err)
				}
				err = ValidateMoveCard(&state, catalog, "owner", command)
				if (err == nil) != tc.valid {
					t.Fatalf("original face %q: error = %v, want valid=%v", originalFace, err, tc.valid)
				}
				after, err := json.Marshal(state)
				if err != nil {
					t.Fatal(err)
				}
				if string(before) != string(after) {
					t.Fatal("validator mutated state")
				}
			}
		})
	}
}

func TestValidateMoveCardControlTransfer(t *testing.T) {
	for _, zone := range []model.Zone{model.ZoneCaster, model.ZoneServant} {
		for _, change := range []string{"none", "face", "orientation"} {
			t.Run(string(zone)+"/"+change, func(t *testing.T) {
				cardType := "Servant"
				if zone == model.ZoneCaster {
					cardType = "Caster"
				}
				state, catalog := moveStateForTest(cardType)
				if zone == model.ZoneCaster {
					state.Players[0].CasterZone = []model.MatchCardID{"card"}
				} else {
					state.Players[0].ServantZone = []model.MatchCardID{"card"}
				}
				command := model.MoveCardCommand{CardID: "card", DestinationPlayerID: "opponent", DestinationZone: zone, DestinationFace: model.CardFaceUp, EntryOrientation: model.OrientationRested}
				if change == "face" {
					command.DestinationFace = model.CardFaceDown
				}
				if change == "orientation" {
					command.EntryOrientation = model.OrientationRecovered
				}
				err := ValidateMoveCard(&state, catalog, "owner", command)
				if (err == nil) != (change == "none") {
					t.Fatalf("error = %v for %s", err, change)
				}
			})
		}
	}
}

func TestValidateMoveCardDeckPlacement(t *testing.T) {
	for _, zone := range []model.Zone{model.ZoneDeck, model.ZoneHand, model.ZoneGraveyard, model.ZoneExile, model.ZoneOrbs, model.ZoneCaster, model.ZoneServant} {
		for _, placement := range []model.DeckPlacement{"", model.DeckPlacementTop, model.DeckPlacementBottom, "invalid"} {
			t.Run(string(zone)+"/"+string(placement), func(t *testing.T) {
				state, catalog := moveStateForTest("Servant")
				state.Players[0].Exile = []model.MatchCardID{"card"}
				if zone == model.ZoneExile {
					state.Players[0].Exile = nil
					state.Players[0].Hand = []model.MatchCardID{"card"}
				}
				command := model.MoveCardCommand{CardID: "card", DestinationPlayerID: "owner", DestinationZone: zone, DestinationFace: model.CardFaceDown, Placement: placement}
				switch zone {
				case model.ZoneCaster:
					command.EntryOrientation = model.OrientationRecovered
				case model.ZoneServant:
					command.EntryOrientation = model.OrientationRecovered
					command.DestinationFace = model.CardFaceUp
				case model.ZoneGraveyard:
					command.DestinationFace = model.CardFaceUp
				}
				wantValid := placement == ""
				if zone == model.ZoneDeck {
					wantValid = placement == model.DeckPlacementTop || placement == model.DeckPlacementBottom
				}
				err := ValidateMoveCard(&state, catalog, "owner", command)
				if (err == nil) != wantValid {
					t.Fatalf("error = %v; want valid=%v", err, wantValid)
				}
			})
		}
	}
}

func moveStateForTest(cardType string) (model.MatchState, definitionCatalogForTest) {
	return model.MatchState{
		MatchStatus: model.StatusInProgress,
		Players:     [2]model.PlayerState{{ID: "owner"}, {ID: "opponent"}},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"card": {MatchID: "card", CardID: "printed", Owner: "owner", Controller: "owner", CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRested},
		},
	}, definitionCatalogForTest{"printed": gamecards.Card{ID: "printed", Type: cardType}}
}
