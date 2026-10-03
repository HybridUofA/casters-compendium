package netclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
	"github.com/coder/websocket"
)

// Client is a WebSocket protocol client for the authoritative nethost.
type Client struct {
	conn *websocket.Conn

	OnView  func(simulatorview.MatchView)
	OnError func(protocol.ErrorPayload)

	mu      sync.Mutex
	pending chan protocol.Envelope
	closed  chan struct{}
	readErr error
}

// Dial connects to a host WebSocket URL such as ws://127.0.0.1:8080/.
func Dial(ctx context.Context, url string) (*Client, error) {
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, err
	}
	client := &Client{
		conn:   conn,
		closed: make(chan struct{}),
	}
	go client.readLoop()
	return client, nil
}

func (client *Client) Close() error {
	if client == nil || client.conn == nil {
		return nil
	}
	return client.conn.Close(websocket.StatusNormalClosure, "")
}

func (client *Client) Hello(ctx context.Context, name string) error {
	raw, err := protocol.Encode(protocol.KindHello, protocol.HelloPayload{
		Name:    name,
		Version: protocol.ProtocolVersion,
	})
	if err != nil {
		return err
	}
	env, err := client.roundTrip(ctx, raw)
	if err != nil {
		return err
	}
	if env.Kind != protocol.KindHello {
		return unexpectedKind(env)
	}
	return nil
}

func (client *Client) CreateRoom(
	ctx context.Context,
	playerName string,
	deck decks.Deck,
) (protocol.CreateRoomPayload, error) {
	raw, err := protocol.Encode(protocol.KindCreateRoom, protocol.CreateRoomPayload{
		PlayerName: playerName,
		Deck:       deck,
	})
	if err != nil {
		return protocol.CreateRoomPayload{}, err
	}
	env, err := client.roundTrip(ctx, raw)
	if err != nil {
		return protocol.CreateRoomPayload{}, err
	}
	if env.Kind != protocol.KindCreateRoom {
		return protocol.CreateRoomPayload{}, unexpectedKind(env)
	}
	var payload protocol.CreateRoomPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return protocol.CreateRoomPayload{}, err
	}
	return payload, nil
}

func (client *Client) JoinRoom(
	ctx context.Context,
	roomCode, playerName string,
	deck decks.Deck,
) (protocol.JoinRoomPayload, error) {
	raw, err := protocol.Encode(protocol.KindJoinRoom, protocol.JoinRoomPayload{
		RoomCode:   roomCode,
		PlayerName: playerName,
		Deck:       deck,
	})
	if err != nil {
		return protocol.JoinRoomPayload{}, err
	}
	env, err := client.roundTrip(ctx, raw)
	if err != nil {
		return protocol.JoinRoomPayload{}, err
	}
	if env.Kind != protocol.KindJoinRoom {
		return protocol.JoinRoomPayload{}, unexpectedKind(env)
	}
	var payload protocol.JoinRoomPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return protocol.JoinRoomPayload{}, err
	}
	return payload, nil
}

// Command sends a match command and returns the actor's updated private view.
func (client *Client) Command(
	ctx context.Context,
	playerID string,
	revision uint64,
	name string,
	args any,
) (simulatorview.MatchView, error) {
	var argsRaw json.RawMessage
	if args != nil {
		encoded, err := json.Marshal(args)
		if err != nil {
			return simulatorview.MatchView{}, err
		}
		argsRaw = encoded
	}
	raw, err := protocol.Encode(protocol.KindCommand, protocol.CommandPayload{
		PlayerID: playerID,
		Revision: revision,
		Name:     name,
		Args:     argsRaw,
	})
	if err != nil {
		return simulatorview.MatchView{}, err
	}
	env, err := client.roundTrip(ctx, raw)
	if err != nil {
		return simulatorview.MatchView{}, err
	}
	if env.Kind != protocol.KindView {
		return simulatorview.MatchView{}, unexpectedKind(env)
	}
	var payload protocol.ViewPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return simulatorview.MatchView{}, err
	}
	return payload.Match, nil
}

func (client *Client) roundTrip(ctx context.Context, data []byte) (protocol.Envelope, error) {
	client.mu.Lock()
	if client.pending != nil {
		client.mu.Unlock()
		return protocol.Envelope{}, fmt.Errorf("another request is in flight")
	}
	client.pending = make(chan protocol.Envelope, 1)
	pending := client.pending
	client.mu.Unlock()

	defer func() {
		client.mu.Lock()
		client.pending = nil
		client.mu.Unlock()
	}()

	if err := client.conn.Write(ctx, websocket.MessageText, data); err != nil {
		return protocol.Envelope{}, err
	}

	select {
	case env := <-pending:
		if env.Kind == protocol.KindError {
			var errPayload protocol.ErrorPayload
			_ = json.Unmarshal(env.Payload, &errPayload)
			if errPayload.Message == "" {
				errPayload.Message = "host returned an error"
			}
			return protocol.Envelope{}, fmt.Errorf("%s: %s", errPayload.Code, errPayload.Message)
		}
		return env, nil
	case <-ctx.Done():
		return protocol.Envelope{}, ctx.Err()
	case <-client.closed:
		if client.readErr != nil {
			return protocol.Envelope{}, client.readErr
		}
		return protocol.Envelope{}, fmt.Errorf("connection closed")
	}
}

func (client *Client) readLoop() {
	defer close(client.closed)
	ctx := context.Background()
	for {
		_, data, err := client.conn.Read(ctx)
		if err != nil {
			client.readErr = err
			return
		}
		env, err := protocol.Decode(data)
		if err != nil {
			continue
		}
		client.mu.Lock()
		pending := client.pending
		client.mu.Unlock()

		// A view can be either the reply to Command or an unsolicited peer push.
		// Prefer completing an in-flight RPC when one is waiting.
		if pending != nil && (env.Kind == protocol.KindView ||
			env.Kind == protocol.KindError ||
			env.Kind == protocol.KindHello ||
			env.Kind == protocol.KindCreateRoom ||
			env.Kind == protocol.KindJoinRoom) {
			select {
			case pending <- env:
			default:
			}
			continue
		}

		switch env.Kind {
		case protocol.KindView:
			var payload protocol.ViewPayload
			if err := json.Unmarshal(env.Payload, &payload); err != nil {
				continue
			}
			if client.OnView != nil {
				client.OnView(payload.Match)
			}
		case protocol.KindError:
			if client.OnError != nil {
				var payload protocol.ErrorPayload
				_ = json.Unmarshal(env.Payload, &payload)
				client.OnError(payload)
			}
		}
	}
}

func unexpectedKind(env protocol.Envelope) error {
	if env.Kind == protocol.KindError {
		var payload protocol.ErrorPayload
		_ = json.Unmarshal(env.Payload, &payload)
		return fmt.Errorf("%s: %s", payload.Code, payload.Message)
	}
	return fmt.Errorf("unexpected kind %q", env.Kind)
}

// DefaultTimeout is a convenience deadline for interactive client calls.
const DefaultTimeout = 5 * time.Second
