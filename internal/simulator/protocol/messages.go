package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// ProtocolVersion 2 requires each create/join to include a validated deck list.
const ProtocolVersion = 2

type Kind string

const (
	KindHello      Kind = "hello"
	KindCreateRoom Kind = "create_room"
	KindJoinRoom   Kind = "join_room"
	KindCommand    Kind = "command"
	KindView       Kind = "view"
	KindError      Kind = "error"
)

type Envelope struct {
	Version int             `json:"version"`
	Kind    Kind            `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type HelloPayload struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

type CreateRoomPayload struct {
	RoomCode   string     `json:"room_code,omitempty"`
	PlayerID   string     `json:"player_id,omitempty"`
	PlayerName string     `json:"player_name"`
	Deck       decks.Deck `json:"deck"`
}

type JoinRoomPayload struct {
	RoomCode   string     `json:"room_code"`
	PlayerID   string     `json:"player_id,omitempty"`
	PlayerName string     `json:"player_name"`
	Deck       decks.Deck `json:"deck"`
}

type CommandPayload struct {
	PlayerID string          `json:"player_id"`
	Revision uint64          `json:"revision"`
	Name     string          `json:"name"`
	Args     json.RawMessage `json:"args,omitempty"`
}

type ViewPayload struct {
	Match simulatorview.MatchView `json:"match"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Encode(kind Kind, payload any) ([]byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.Marshal(Envelope{
		Version: ProtocolVersion,
		Kind:    kind,
		Payload: raw,
	})
}

func Decode(data []byte) (Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Envelope{}, err
	}
	if envelope.Version != ProtocolVersion {
		return Envelope{}, fmt.Errorf("unsupported protocol version %d", envelope.Version)
	}
	return envelope, nil
}
