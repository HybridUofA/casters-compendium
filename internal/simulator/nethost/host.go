package nethost

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
	"github.com/coder/websocket"
)

// MatchFactory builds the authoritative LocalMatch when a room becomes full.
// Optional: without it, create/join still work but commands cannot run.
type MatchFactory func(room *Room) (*session.LocalMatch, error)

type Host struct {
	Lobby        *Lobby
	MatchFactory MatchFactory

	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

type wsClient struct {
	conn       *websocket.Conn
	roomCode   string
	playerID   string
	name       string
	spectator  bool
}

func NewHost(lobby *Lobby) *Host {
	if lobby == nil {
		lobby = NewLobby()
	}
	return &Host{
		Lobby:   lobby,
		clients: make(map[*wsClient]struct{}),
	}
}

// HandleMessage decodes one protocol envelope and returns encoded reply bytes.
// Used by unit tests and as the shared dispatch core for WebSocket clients.
func (host *Host) HandleMessage(data []byte) ([]byte, error) {
	return host.dispatch(nil, data)
}

func (host *Host) dispatch(client *wsClient, data []byte) ([]byte, error) {
	if host == nil || host.Lobby == nil {
		return nil, fmt.Errorf("host lobby is required")
	}

	envelope, err := protocol.Decode(data)
	if err != nil {
		return encodeError("bad_request", err.Error())
	}

	switch envelope.Kind {
	case protocol.KindHello:
		return host.handleHello(client, envelope.Payload)
	case protocol.KindCreateRoom:
		return host.handleCreateRoom(client, envelope.Payload)
	case protocol.KindJoinRoom:
		return host.handleJoinRoom(client, envelope.Payload)
	case protocol.KindListRooms:
		return host.handleListRooms(client, envelope.Payload)
	case protocol.KindCommand:
		return host.handleCommand(client, envelope.Payload)
	default:
		return encodeError("unknown_kind", fmt.Sprintf("unsupported kind %q", envelope.Kind))
	}
}

func (host *Host) handleHello(client *wsClient, raw json.RawMessage) ([]byte, error) {
	var payload protocol.HelloPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return encodeError("bad_request", err.Error())
	}
	if payload.Version != 0 && payload.Version != protocol.ProtocolVersion {
		return encodeError("bad_version", fmt.Sprintf("unsupported client version %d", payload.Version))
	}
	if client != nil {
		client.name = payload.Name
	}
	return protocol.Encode(protocol.KindHello, protocol.HelloPayload{
		Name:    payload.Name,
		Version: protocol.ProtocolVersion,
	})
}

func (host *Host) handleCreateRoom(client *wsClient, raw json.RawMessage) ([]byte, error) {
	var payload protocol.CreateRoomPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return encodeError("bad_request", err.Error())
	}
	roomCode, playerID, roomName, err := host.Lobby.CreateRoom(payload.PlayerName, payload.RoomName, payload.Deck, payload.Password)
	if err != nil {
		return encodeError("create_room_failed", err.Error())
	}
	if client != nil {
		client.roomCode = roomCode
		client.playerID = playerID
		client.spectator = false
		if client.name == "" {
			client.name = payload.PlayerName
		}
	}
	return protocol.Encode(protocol.KindCreateRoom, protocol.CreateRoomPayload{
		RoomCode:   roomCode,
		RoomName:   roomName,
		PlayerID:   playerID,
		PlayerName: payload.PlayerName,
	})
}

func (host *Host) handleJoinRoom(client *wsClient, raw json.RawMessage) ([]byte, error) {
	var payload protocol.JoinRoomPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return encodeError("bad_request", err.Error())
	}

	var playerID string
	var err error
	if payload.Spectator {
		playerID, err = host.Lobby.JoinSpectator(payload.RoomCode, payload.PlayerName, payload.Password)
	} else {
		playerID, err = host.Lobby.JoinRoom(payload.RoomCode, payload.PlayerName, payload.Deck, payload.Password)
	}
	if err != nil {
		return encodeError("join_room_failed", err.Error())
	}
	if client != nil {
		client.roomCode = payload.RoomCode
		client.playerID = playerID
		client.spectator = payload.Spectator
		if client.name == "" {
			client.name = payload.PlayerName
		}
	}

	if !payload.Spectator {
		if err := host.maybeStartMatch(payload.RoomCode); err != nil {
			return encodeError("start_match_failed", err.Error())
		}
	}

	reply, err := protocol.Encode(protocol.KindJoinRoom, protocol.JoinRoomPayload{
		RoomCode:   payload.RoomCode,
		PlayerID:   playerID,
		PlayerName: payload.PlayerName,
		Spectator:  payload.Spectator,
	})
	if err != nil {
		return nil, err
	}

	host.pushViews(payload.RoomCode, playerID)
	return reply, nil
}

func (host *Host) handleListRooms(_ *wsClient, raw json.RawMessage) ([]byte, error) {
	if len(raw) > 0 && string(raw) != "null" {
		var payload protocol.ListRoomsPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return encodeError("bad_request", err.Error())
		}
	}
	return protocol.Encode(protocol.KindRoomList, protocol.RoomListPayload{
		Rooms: host.Lobby.ListRooms(),
	})
}

func (host *Host) maybeStartMatch(roomCode string) error {
	if host.MatchFactory == nil {
		return nil
	}
	host.Lobby.mu.Lock()
	room, ok := host.Lobby.rooms[roomCode]
	if !ok {
		host.Lobby.mu.Unlock()
		return fmt.Errorf("room not found")
	}
	if room.Match != nil || len(room.Players) != 2 {
		host.Lobby.mu.Unlock()
		return nil
	}
	// Copy pointer under lock; factory runs unlocked so it can do slow work.
	host.Lobby.mu.Unlock()

	match, err := host.MatchFactory(room)
	if err != nil {
		return err
	}
	return host.Lobby.AttachMatch(roomCode, match)
}

func (host *Host) register(client *wsClient) {
	host.mu.Lock()
	defer host.mu.Unlock()
	host.clients[client] = struct{}{}
}

func (host *Host) unregister(client *wsClient) {
	host.mu.Lock()
	defer host.mu.Unlock()
	delete(host.clients, client)
}

func (host *Host) clientsInRoom(roomCode string) []*wsClient {
	host.mu.Lock()
	defer host.mu.Unlock()
	out := make([]*wsClient, 0, 2)
	for client := range host.clients {
		if client.roomCode == roomCode {
			out = append(out, client)
		}
	}
	return out
}

func encodeError(code, message string) ([]byte, error) {
	return protocol.Encode(protocol.KindError, protocol.ErrorPayload{
		Code:    code,
		Message: message,
	})
}
