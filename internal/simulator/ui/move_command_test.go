package ui

import (
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestBuildDefaultMoveCommandHandToGraveyard(t *testing.T) {
	command, err := buildDefaultMoveCommand(
		simulatorview.CardView{MatchID: "c1", CardID: "servant", Owner: "player-one"},
		model.ZoneHand,
		"player-one",
		model.ZoneGraveyard,
		cards.Card{ID: "servant", Type: "Servant"},
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if command.DestinationZone != model.ZoneGraveyard || command.DestinationFace != model.CardFaceUp || command.DestinationPlayerID != "player-one" {
		t.Fatalf("command = %#v", command)
	}
}

func TestBuildDefaultMoveCommandHandToOrbs(t *testing.T) {
	command, err := buildDefaultMoveCommand(
		simulatorview.CardView{MatchID: "c1", CardID: "servant", Owner: "player-one"},
		model.ZoneHand,
		"player-one",
		model.ZoneOrbs,
		cards.Card{ID: "servant", Type: "Servant"},
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if command.DestinationZone != model.ZoneOrbs || command.DestinationFace != model.CardFaceDown {
		t.Fatalf("command = %#v", command)
	}
}

func TestBuildDefaultMoveCommandRejectsTokens(t *testing.T) {
	_, err := buildDefaultMoveCommand(
		simulatorview.CardView{MatchID: "tok", CardID: model.CasterTokenCardID, Owner: "player-one"},
		model.ZoneCaster,
		"player-one",
		model.ZoneGraveyard,
		cards.Card{},
		"",
	)
	if err == nil || !strings.Contains(err.Error(), "token") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildDefaultMoveCommandDeckPlacement(t *testing.T) {
	command, err := buildDefaultMoveCommand(
		simulatorview.CardView{MatchID: "c1", CardID: "servant", Owner: "player-one"},
		model.ZoneHand,
		"player-one",
		model.ZoneDeck,
		cards.Card{ID: "servant", Type: "Servant"},
		model.DeckPlacementBottom,
	)
	if err != nil {
		t.Fatal(err)
	}
	if command.Placement != model.DeckPlacementBottom || command.DestinationFace != model.CardFaceDown {
		t.Fatalf("command = %#v", command)
	}
}
