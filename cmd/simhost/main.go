package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/HybridUofA/casters-compendium/internal/simulator/matchboot"
	"github.com/HybridUofA/casters-compendium/internal/simulator/nethost"
)

func main() {
	addr := flag.String("addr", ":7474", "TCP listen address for the WebSocket host")
	catalog := flag.String("catalog", defaultCatalogPath(), "path to normalized cards.json")
	flag.Parse()

	repository, err := matchboot.LoadCatalog(*catalog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "simhost: %v\n", err)
		os.Exit(1)
	}

	host := nethost.NewHost(nethost.NewLobby())
	host.MatchFactory = matchboot.SubmittedDecksFactory(repository)

	fmt.Fprintf(os.Stderr, "simulator host listening on %s\n", *addr)
	fmt.Fprintf(os.Stderr, "catalog: %s (%d cards)\n", *catalog, len(repository.All()))
	fmt.Fprintf(os.Stderr, "connect: ws://127.0.0.1%s/ (or your LAN/public IP)\n", portSuffix(*addr))
	if err := host.ListenAndServe(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "simhost: %v\n", err)
		os.Exit(1)
	}
}

func defaultCatalogPath() string {
	candidates := []string{
		"data/cards.json",
		filepath.Join("..", "..", "data", "cards.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "data/cards.json"
}

func portSuffix(addr string) string {
	if addr == "" {
		return ":7474"
	}
	if addr[0] == ':' {
		return addr
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		return ":7474"
	}
	return ":" + port
}
