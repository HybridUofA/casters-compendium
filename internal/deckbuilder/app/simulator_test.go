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
