package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// ProtocolVersion 3 adds lobby listing, optional passwords, and spectators.
const ProtocolVersion = 3

type Kind string

const (
	KindHello      Kind = "hello"
	KindCreateRoom Kind = "create_room"
	KindJoinRoom   Kind = "join_room"
	KindListRooms  Kind = "list_rooms"
	KindRoomList   Kind = "room_list"
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
	RoomName   string     `json:"room_name,omitempty"`
	PlayerID   string     `json:"player_id,omitempty"`
	PlayerName string     `json:"player_name"`
	Password   string     `json:"password,omitempty"`
	Deck       decks.Deck `json:"deck"`
}

type JoinRoomPayload struct {
	RoomCode   string     `json:"room_code"`
	PlayerID   string     `json:"player_id,omitempty"`
	PlayerName string     `json:"player_name"`
	Password   string     `json:"password,omitempty"`
	Spectator  bool       `json:"spectator,omitempty"`
	Deck       decks.Deck `json:"deck,omitempty"`
}

type ListRoomsPayload struct{}

type RoomSummary struct {
	RoomCode       string `json:"room_code"`
	RoomName       string `json:"room_name"`
	HostName       string `json:"host_name"` // display name of the player who created the room
	PlayerCount    int    `json:"player_count"`
	SpectatorCount int    `json:"spectator_count"`
	HasPassword    bool   `json:"has_password"`
	MatchStarted   bool   `json:"match_started"`
	OpenSeats      int    `json:"open_seats"`
}

type RoomListPayload struct {
	Rooms []RoomSummary `json:"rooms"`
}

type CommandPayload struct {
	PlayerID string          `json:"player_id"`
	Revision uint64          `json:"revision"`
	Name     string          `json:"name"`
	Args     json.RawMessage `json:"args,omitempty"`
}

type ViewPayload struct {
	Match        simulatorview.MatchView `json:"match"`
	Private      *PrivateView            `json:"private,omitempty"`
	DisplayNames map[string]string       `json:"display_names,omitempty"`
}

// PrivateView carries actor-only information that must never be pushed to peers.
type PrivateView struct {
	DeckPeek      []simulatorview.CardView `json:"deck_peek,omitempty"`
	DeckPeekOwner string                   `json:"deck_peek_owner,omitempty"`
	OrbPeek       *simulatorview.CardView  `json:"orb_peek,omitempty"`
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
