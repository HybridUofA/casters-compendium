package nethost

import (
	"encoding/json"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
)

func TestHandleMessageHelloCreateAndJoin(t *testing.T) {
	host := NewHost(NewLobby())

	helloReq, err := protocol.Encode(protocol.KindHello, protocol.HelloPayload{
		Name:    "Hybrid",
		Version: protocol.ProtocolVersion,
	})
	if err != nil {
		t.Fatalf("encode hello: %v", err)
	}
	helloRaw, err := host.HandleMessage(helloReq)
	if err != nil {
		t.Fatalf("HandleMessage(hello) error = %v", err)
	}
	helloEnv, err := protocol.Decode(helloRaw)
	if err != nil {
		t.Fatalf("decode hello reply: %v", err)
	}
	if helloEnv.Kind != protocol.KindHello {
		t.Fatalf("hello kind = %q", helloEnv.Kind)
	}

	createReq, err := protocol.Encode(protocol.KindCreateRoom, protocol.CreateRoomPayload{
		PlayerName: "Host",
		Deck:       decks.Deck{SchemaVersion: 1, Name: "Host Deck"},
	})
	if err != nil {
		t.Fatalf("encode create: %v", err)
	}
	createRaw, err := host.HandleMessage(createReq)
	if err != nil {
		t.Fatalf("HandleMessage(create) error = %v", err)
	}
	createEnv, err := protocol.Decode(createRaw)
	if err != nil {
		t.Fatalf("decode create reply: %v", err)
	}
	var created protocol.CreateRoomPayload
	if err := json.Unmarshal(createEnv.Payload, &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	if created.RoomCode == "" || created.PlayerID != "player-one" {
		t.Fatalf("create reply = %#v", created)
	}

	joinReq, err := protocol.Encode(protocol.KindJoinRoom, protocol.JoinRoomPayload{
		RoomCode:   created.RoomCode,
		PlayerName: "Guest",
		Deck:       decks.Deck{SchemaVersion: 1, Name: "Guest Deck"},
	})
	if err != nil {
		t.Fatalf("encode join: %v", err)
	}
	joinRaw, err := host.HandleMessage(joinReq)
	if err != nil {
		t.Fatalf("HandleMessage(join) error = %v", err)
	}
	joinEnv, err := protocol.Decode(joinRaw)
	if err != nil {
		t.Fatalf("decode join reply: %v", err)
	}
	var joined protocol.JoinRoomPayload
	if err := json.Unmarshal(joinEnv.Payload, &joined); err != nil {
		t.Fatalf("unmarshal join: %v", err)
	}
	if joined.PlayerID != "player-two" || joined.RoomCode != created.RoomCode {
		t.Fatalf("join reply = %#v", joined)
	}
}

func TestHandleMessageListRooms(t *testing.T) {
	host := NewHost(NewLobby())
	createReq, err := protocol.Encode(protocol.KindCreateRoom, protocol.CreateRoomPayload{
		PlayerName: "Host",
		RoomName:   "Secret Match",
		Password:   "secret",
		Deck:       decks.Deck{SchemaVersion: 1, Name: "Host Deck"},
	})
	if err != nil {
		t.Fatalf("encode create: %v", err)
	}
	if _, err := host.HandleMessage(createReq); err != nil {
		t.Fatalf("HandleMessage(create) error = %v", err)
	}

	listReq, err := protocol.Encode(protocol.KindListRooms, protocol.ListRoomsPayload{})
	if err != nil {
		t.Fatalf("encode list: %v", err)
	}
	listRaw, err := host.HandleMessage(listReq)
	if err != nil {
		t.Fatalf("HandleMessage(list) error = %v", err)
	}
	listEnv, err := protocol.Decode(listRaw)
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if listEnv.Kind != protocol.KindRoomList {
		t.Fatalf("kind = %q; want room_list", listEnv.Kind)
	}
	var payload protocol.RoomListPayload
	if err := json.Unmarshal(listEnv.Payload, &payload); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(payload.Rooms) != 1 ||
		!payload.Rooms[0].HasPassword ||
		payload.Rooms[0].HostName != "Host" ||
		payload.Rooms[0].RoomName != "Secret Match" {
		t.Fatalf("rooms = %#v", payload.Rooms)
	}
}

func TestHandleMessageUnknownKindReturnsErrorEnvelope(t *testing.T) {
	host := NewHost(NewLobby())
	req, err := protocol.Encode(protocol.Kind("nope"), map[string]string{})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	raw, err := host.HandleMessage(req)
	if err != nil {
		t.Fatalf("HandleMessage() error = %v", err)
	}
	env, err := protocol.Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Kind != protocol.KindError {
		t.Fatalf("kind = %q; want error", env.Kind)
	}
}
