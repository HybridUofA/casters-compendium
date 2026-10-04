package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func TestParseDrawCount(t *testing.T) {
	count, err := parseDrawCount(" 3 ")
	if err != nil || count != 3 {
		t.Fatalf("parseDrawCount(3) = %d, %v", count, err)
	}
	if _, err := parseDrawCount("0"); err == nil || !strings.Contains(err.Error(), "at least 1") {
		t.Fatalf("parseDrawCount(0) error = %v", err)
	}
	if _, err := parseDrawCount("x"); err == nil {
		t.Fatal("parseDrawCount(x) accepted")
	}
}

func TestViewerDeckBeginDrawCardsPromptSubmitsCount(t *testing.T) {
	var drewCount int
	var drewRevision model.Revision
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Players[0].DeckCount = 5
	screen := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			DrawCards: func(count int, revision model.Revision) {
				drewCount = count
				drewRevision = revision
			},
		},
		nil,
	)
	window := test.NewWindow(screen.Content())
	defer window.Close()

	viewerBoard := screen.playerBoards[1]
	if findZoneInteractLayer(viewerBoard.deck) == nil {
		t.Fatal("viewer deck has no interact layer")
	}

	originalPrompt := promptDrawCardCount
	promptDrawCardCount = func(window fyne.Window, onConfirm func(int)) {
		if window == nil {
			t.Fatal("draw prompt missing window")
		}
		onConfirm(2)
	}
	t.Cleanup(func() { promptDrawCardCount = originalPrompt })

	viewerBoard.beginDrawCardsPrompt()
	if drewCount != 2 {
		t.Fatalf("DrawCards count = %d; want 2", drewCount)
	}
	if drewRevision != match.Revision {
		t.Fatalf("DrawCards revision = %d; want %d", drewRevision, match.Revision)
	}
}

func TestOpponentDeckHasNoDrawSecondaryTap(t *testing.T) {
	match := testMatchView()
	match.Players[1].DeckCount = 5
	screen := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{DrawCards: func(int, model.Revision) {}},
		nil,
	)
	opponentBoard := screen.playerBoards[0]
	if findZoneInteractLayer(opponentBoard.deck) != nil {
		t.Fatal("opponent deck should not offer draw context actions")
	}
}

func findZoneInteractLayer(object fyne.CanvasObject) *zoneInteractLayer {
	switch typed := object.(type) {
	case *zoneInteractLayer:
		return typed
	case *fyne.Container:
		for _, child := range typed.Objects {
			if found := findZoneInteractLayer(child); found != nil {
				return found
			}
		}
	}
	return nil
}
