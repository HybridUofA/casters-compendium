package nethost

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/netclient"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
)

func TestWebSocketCreateJoinAndRequestView(t *testing.T) {
	host := NewHost(NewLobby())
	host.MatchFactory = func(room *Room) (*session.LocalMatch, error) {
		return localMatchForTest(t), nil
	}

	addr, server, err := host.Listen("127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "ws://" + addr.String()
	hostClient, err := netclient.Dial(ctx, url)
	if err != nil {
		t.Fatalf("host Dial() error = %v", err)
	}
	defer hostClient.Close()

	guestClient, err := netclient.Dial(ctx, url)
	if err != nil {
		t.Fatalf("guest Dial() error = %v", err)
	}
	defer guestClient.Close()

	if err := hostClient.Hello(ctx, "Host"); err != nil {
		t.Fatalf("host Hello() error = %v", err)
	}
	created, err := hostClient.CreateRoom(ctx, "Host", "Host Room", decks.Deck{SchemaVersion: 1, Name: "Host Deck"}, "")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if created.RoomCode == "" || created.PlayerID != "player-one" {
		t.Fatalf("CreateRoom payload = %#v", created)
	}

	if err := guestClient.Hello(ctx, "Guest"); err != nil {
		t.Fatalf("guest Hello() error = %v", err)
	}
	joined, err := guestClient.JoinRoom(ctx, created.RoomCode, "Guest", decks.Deck{SchemaVersion: 1, Name: "Guest Deck"}, "")
	if err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if joined.PlayerID != "player-two" {
		t.Fatalf("JoinRoom payload = %#v", joined)
	}

	view, err := guestClient.Command(ctx, joined.PlayerID, 0, "request_view", nil)
	if err != nil {
		t.Fatalf("request_view error = %v", err)
	}
	if view.ViewerID != "player-two" {
		t.Fatalf("ViewerID = %q", view.ViewerID)
	}
}

func TestServeHTTPNilHost(t *testing.T) {
	var host *Host
	recorder := &statusRecorder{status: 200}
	host.ServeHTTP(recorder, &http.Request{})
	if recorder.status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", recorder.status)
	}
}

type statusRecorder struct {
	status int
	header http.Header
}

func (r *statusRecorder) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}
	return r.header
}
func (r *statusRecorder) Write(b []byte) (int, error) { return len(b), nil }
func (r *statusRecorder) WriteHeader(status int)      { r.status = status }
