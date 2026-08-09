package deckbuilder

import (
	"fmt"
	"testing"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
)

func TestBuildSimulatorPrototypeSessionsUsePrivateProjections(t *testing.T) {
	definitions := make([]cards.Card, 0, 14)
	definitions = append(definitions, cards.Card{
		ID:   "1100",
		Name: "Caster Token",
	})
	for index := 0; index < 13; index++ {
		definitions = append(definitions, cards.Card{
			ID:   fmt.Sprintf("card-%02d", index+1),
			Name: fmt.Sprintf("Card %02d", index+1),
		})
	}
	repository, err := cards.NewRepository(definitions)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	playerSessions, err := buildSimulatorPrototypeSessions(repository)
	if err != nil {
		t.Fatalf("buildSimulatorPrototypeSessions() error = %v", err)
	}

	for viewerIndex, playerSession := range playerSessions {
		result, viewErr := playerSession.View()
		if viewErr != nil {
			t.Fatalf("PlayerSession %d View() error = %v", viewerIndex+1, viewErr)
		}
		if len(result.Players[0].Hand) != 7 || len(result.Players[1].Hand) != 7 {
			t.Fatal("prototype view does not contain both opening hands")
		}
		for playerIndex, player := range result.Players {
			for _, card := range player.Hand {
				if playerIndex == viewerIndex {
					if !card.ShowFace || card.CardID == "" || card.MatchID == "" {
						t.Fatalf("viewer %d hand contains concealed card: %#v", viewerIndex+1, card)
					}
				} else if card.ShowFace || card.CardID != "" || card.MatchID != "" {
					t.Fatalf("viewer %d received opponent card identity: %#v", viewerIndex+1, card)
				}
			}
			if player.DeckCount != 36 || len(player.Orbs) != 7 || len(player.CasterZone) != 1 {
				t.Fatalf("prototype player zones = %#v; want deck 36, orbs 7, Caster Zone 1", player)
			}
		}
	}
}

func TestBuildSimulatorPrototypeSessionsDealsArthurLevelUpPair(t *testing.T) {
	definitions := []cards.Card{
		{
			ID:        "arthur-level-one",
			Name:      "Arthur",
			Type:      "Caster",
			CostLevel: "1",
		},
		{
			ID:        "arthur-level-two",
			Name:      "Arthur Lv2",
			Type:      "Caster",
			CostLevel: "2",
		},
		{
			ID:   "1100",
			Name: "Caster Token",
		},
	}
	for index := 0; index < 11; index++ {
		definitions = append(definitions, cards.Card{
			ID:   fmt.Sprintf("filler-%02d", index+1),
			Name: fmt.Sprintf("Filler %02d", index+1),
		})
	}
	repository, err := cards.NewRepository(definitions)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	playerSessions, err := buildSimulatorPrototypeSessions(repository)
	if err != nil {
		t.Fatalf("buildSimulatorPrototypeSessions() error = %v", err)
	}
	playerOneView, err := playerSessions[0].View()
	if err != nil {
		t.Fatalf("Player One View() error = %v", err)
	}
	if playerOneView.Turn.ActivePlayer != "player-one" {
		t.Fatalf("active player = %q; want player-one", playerOneView.Turn.ActivePlayer)
	}

	handContains := map[string]bool{}
	for _, card := range playerOneView.Players[0].Hand {
		handContains[string(card.CardID)] = true
	}
	if !handContains["arthur-level-one"] || !handContains["arthur-level-two"] {
		t.Fatalf("Player One opening hand IDs = %#v; want both Arthur levels", handContains)
	}
}
