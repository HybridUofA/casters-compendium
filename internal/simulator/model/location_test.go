package model

import (
	"reflect"
	"testing"
)

func TestFindCardLocationSupportedZones(t *testing.T) {
	zones := []struct {
		zone Zone
		set  func(*PlayerState, []MatchCardID)
	}{
		{ZoneHand, func(p *PlayerState, ids []MatchCardID) { p.Hand = ids }},
		{ZoneDeck, func(p *PlayerState, ids []MatchCardID) { p.Deck = ids }},
		{ZoneOrbs, func(p *PlayerState, ids []MatchCardID) { p.Orbs = ids }},
		{ZoneCaster, func(p *PlayerState, ids []MatchCardID) { p.CasterZone = ids }},
		{ZoneServant, func(p *PlayerState, ids []MatchCardID) { p.ServantZone = ids }},
		{ZoneGraveyard, func(p *PlayerState, ids []MatchCardID) { p.Graveyard = ids }},
		{ZoneExile, func(p *PlayerState, ids []MatchCardID) { p.Exile = ids }},
	}
	for _, zone := range zones {
		t.Run(string(zone.zone), func(t *testing.T) {
			for playerIndex := 0; playerIndex < 2; playerIndex++ {
				for cardIndex := 0; cardIndex < 2; cardIndex++ {
					state := &MatchState{}
					ids := []MatchCardID{"other", "other"}
					ids[cardIndex] = "target"
					zone.set(&state.Players[playerIndex], ids)
					before := append([]MatchCardID(nil), ids...)
					got, err := FindCardLocation(state, "target")
					want := CardLocation{PlayerIndex: playerIndex, Zone: zone.zone, CardIndex: cardIndex}
					if err != nil || got != want {
						t.Fatalf("FindCardLocation = %v, %v; want %v", got, err, want)
					}
					if !reflect.DeepEqual(ids, before) {
						t.Fatal("lookup modified zone contents")
					}
				}
			}
		})
	}
}

func TestFindCardLocationRejectsInvalidOrMissingCard(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state *MatchState
		id    MatchCardID
	}{
		{"nil state", nil, "target"},
		{"empty ID", &MatchState{}, ""},
		{"blank ID", &MatchState{}, " \t"},
		{"absent card", &MatchState{}, "target"},
		{"instance without placement", &MatchState{CardInstances: map[MatchCardID]CardInstance{"target": {MatchID: "target"}}}, "target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := FindCardLocation(tc.state, tc.id); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestFindCardLocationRejectsDuplicatePlacement(t *testing.T) {
	for _, tc := range []struct {
		name    string
		players [2]PlayerState
	}{
		{"same zone", [2]PlayerState{{Hand: []MatchCardID{"target", "target"}}}},
		{"different zones", [2]PlayerState{{Hand: []MatchCardID{"target"}, Exile: []MatchCardID{"target"}}}},
		{"different players", [2]PlayerState{{Hand: []MatchCardID{"target"}}, {Deck: []MatchCardID{"target"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := FindCardLocation(&MatchState{Players: tc.players}, "target"); err == nil {
				t.Fatal("expected duplicate placement to be rejected")
			}
		})
	}
}
