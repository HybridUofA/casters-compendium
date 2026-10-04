package deckbuilder

import (
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
)

func TestRoomMatchesSearchByNameCreatorOrCode(t *testing.T) {
	room := protocol.RoomSummary{
		RoomCode: "ABCD",
		RoomName: "Friday Night Casters",
		HostName: "Hybrid",
	}
	tests := []struct {
		query string
		want  bool
	}{
		{query: "friday", want: true},
		{query: "hybrid", want: true},
		{query: "abcd", want: true},
		{query: "missing", want: false},
	}
	for _, testCase := range tests {
		if got := roomMatchesSearch(room, testCase.query); got != testCase.want {
			t.Fatalf("roomMatchesSearch(%q) = %v; want %v", testCase.query, got, testCase.want)
		}
	}
}
