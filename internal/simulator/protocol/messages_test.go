package protocol

import (
	"encoding/json"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func TestEncodeDecodeCreateRoomRoundTrip(t *testing.T) {
	raw, err := Encode(KindCreateRoom, CreateRoomPayload{
		RoomCode:   "ABCD",
		PlayerName: "Hybrid",
		Deck:       decks.Deck{SchemaVersion: 1, Name: "Hybrid Deck"},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	envelope, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if envelope.Version != ProtocolVersion || envelope.Kind != KindCreateRoom {
		t.Fatalf("envelope = %#v; want version %d kind %q", envelope, ProtocolVersion, KindCreateRoom)
	}

	var payload CreateRoomPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload error = %v", err)
	}
	if payload.RoomCode != "ABCD" || payload.PlayerName != "Hybrid" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestEncodeDecodeCommandWithArgsRoundTrip(t *testing.T) {
	args, err := json.Marshal(map[string]string{
		"attacker_id":    "p1-attacker",
		"target_kind":    "Servant",
		"target_card_id": "p2-defender",
	})
	if err != nil {
		t.Fatalf("marshal args error = %v", err)
	}

	raw, err := Encode(KindCommand, CommandPayload{
		PlayerID: "player-one",
		Revision: 12,
		Name:     "declare_attack",
		Args:     args,
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	envelope, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	var payload CommandPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal command error = %v", err)
	}
	if payload.PlayerID != "player-one" || payload.Revision != 12 || payload.Name != "declare_attack" {
		t.Fatalf("command payload = %#v", payload)
	}

	var decodedArgs map[string]string
	if err := json.Unmarshal(payload.Args, &decodedArgs); err != nil {
		t.Fatalf("unmarshal args error = %v", err)
	}
	if decodedArgs["attacker_id"] != "p1-attacker" ||
		decodedArgs["target_kind"] != "Servant" ||
		decodedArgs["target_card_id"] != "p2-defender" {
		t.Fatalf("args = %#v", decodedArgs)
	}
}

func TestEncodeDecodeViewRoundTrip(t *testing.T) {
	raw, err := Encode(KindView, ViewPayload{
		Match: simulatorview.MatchView{
			ViewerID:    "player-one",
			MatchStatus: model.StatusInProgress,
			Revision:    9,
			Turn: model.TurnState{
				Number:       2,
				ActivePlayer: "player-one",
				Phase:        model.PhaseBattle,
			},
			PrioritySequenceOpen: true,
			PriorityHolder:       "player-one",
		},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	envelope, err := Decode(raw)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	var payload ViewPayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatalf("unmarshal view error = %v", err)
	}
	if payload.Match.ViewerID != "player-one" ||
		payload.Match.Revision != 9 ||
		payload.Match.Turn.Phase != model.PhaseBattle ||
		!payload.Match.PrioritySequenceOpen {
		t.Fatalf("view payload = %#v", payload.Match)
	}
}

func TestDecodeRejectsUnsupportedVersion(t *testing.T) {
	raw, err := json.Marshal(Envelope{
		Version: ProtocolVersion + 1,
		Kind:    KindHello,
		Payload: []byte(`{"name":"x","version":1}`),
	})
	if err != nil {
		t.Fatalf("marshal envelope error = %v", err)
	}

	_, err = Decode(raw)
	if err == nil {
		t.Fatal("Decode() error = nil; want unsupported-version error")
	}
}
