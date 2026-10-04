package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/simulator/matchboot"
	"github.com/HybridUofA/casters-compendium/internal/simulator/netclient"
)

// simsmoke dials a running simhost, creates a room, joins it, and requests a view.
func main() {
	url := flag.String("url", "ws://127.0.0.1:7474/", "WebSocket URL of simhost")
	catalog := flag.String("catalog", "data/cards.json", "path to cards.json used for smoke decks")
	flag.Parse()

	repository, err := matchboot.LoadCatalog(*catalog)
	if err != nil {
		fail("%v", err)
	}
	deck, err := matchboot.BuildPrototypeDeck(repository)
	if err != nil {
		fail("prototype deck: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	hostClient, err := netclient.Dial(ctx, *url)
	if err != nil {
		fail("dial host: %v", err)
	}
	defer hostClient.Close()

	guestClient, err := netclient.Dial(ctx, *url)
	if err != nil {
		fail("dial guest: %v", err)
	}
	defer guestClient.Close()

	if err := hostClient.Hello(ctx, "SmokeHost"); err != nil {
		fail("host hello: %v", err)
	}
	created, err := hostClient.CreateRoom(ctx, "SmokeHost", "Smoke Room", deck, "")
	if err != nil {
		fail("create room: %v", err)
	}
	fmt.Printf("room %s created; host seat %s\n", created.RoomCode, created.PlayerID)

	if err := guestClient.Hello(ctx, "SmokeGuest"); err != nil {
		fail("guest hello: %v", err)
	}
	joined, err := guestClient.JoinRoom(ctx, created.RoomCode, "SmokeGuest", deck, "")
	if err != nil {
		fail("join room: %v", err)
	}
	fmt.Printf("guest joined as %s\n", joined.PlayerID)

	view, err := guestClient.Command(ctx, joined.PlayerID, 0, "request_view", nil)
	if err != nil {
		fail("request_view: %v", err)
	}
	fmt.Printf("ok: viewer=%s status=%s revision=%d\n", view.ViewerID, view.MatchStatus, view.Revision)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "simsmoke: "+format+"\n", args...)
	os.Exit(1)
}
