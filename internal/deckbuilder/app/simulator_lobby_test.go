package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2/test"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
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

func TestShouldAutoPassPriorityForViewDefaultsToEmptyChase(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()
	prefs.SetString(networkPriorityModePreferenceKey, "") // force default

	base := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn: model.TurnState{
			ActivePlayer: "player-one",
			Phase:        model.PhaseDraw,
		},
	}
	if !shouldAutoPassPriorityForView(base) {
		t.Fatal("default empty mode should auto-pass Draw with an empty Chase")
	}

	withChase := base
	withChase.ChaseLinkCount = 1
	if !shouldAutoPassPriorityForView(withChase) {
		t.Fatal("default empty mode should auto-pass Chase when no response is known")
	}

	mainTurn := base
	mainTurn.Turn.Phase = model.PhaseMain
	if shouldAutoPassPriorityForView(mainTurn) {
		t.Fatal("default empty mode must keep Main priority for the turn player")
	}

	mainOpponent := mainTurn
	mainOpponent.Turn.ActivePlayer = "player-two"
	if !shouldAutoPassPriorityForView(mainOpponent) {
		t.Fatal("default empty mode should auto-pass Main for the non-turn player")
	}

	emptyBattle := base
	emptyBattle.Turn.Phase = model.PhaseBattle
	emptyBattle.Players = [2]simulatorview.PlayerView{
		{ID: "player-one"},
		{ID: "player-two"},
	}
	if !shouldAutoPassPriorityForView(emptyBattle) {
		t.Fatal("default empty mode should auto-pass empty Battle for the turn player")
	}
}

func TestShouldAutoPassEmptyBattleIgnoresUntriggerableRest(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()
	prefs.SetString(networkPriorityModePreferenceKey, priorityModeEmpty)

	autoPass := makeShouldAutoPassPriority([]cards.Card{
		{
			ID:      "lumielle",
			Name:    "Lumielle",
			Type:    "Caster",
			Ability: "• Rest: Swap the ATK and DEF of target enemy servant until end of turn.",
		},
		{
			ID:      "glory",
			Name:    "Glory Butterfly",
			Type:    "Caster",
			Ability: "• Rest: Target allied servant gains +500 DEF until end of turn.",
		},
	})

	base := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn: model.TurnState{
			ActivePlayer: "player-one",
			Phase:        model.PhaseBattle,
		},
		Players: [2]simulatorview.PlayerView{
			{
				ID: "player-one",
				CasterZone: []simulatorview.CardView{{
					MatchID:     "lumielle-1",
					CardID:      "lumielle",
					ShowFace:    true,
					Orientation: model.OrientationRecovered,
				}},
			},
			{ID: "player-two"},
		},
	}
	if !autoPass(base) {
		t.Fatal("Lumielle with no enemy Servant should not keep empty Battle open")
	}

	gloryOnly := base
	gloryOnly.Players[0].CasterZone = []simulatorview.CardView{{
		MatchID:     "glory-1",
		CardID:      "glory",
		ShowFace:    true,
		Orientation: model.OrientationRecovered,
	}}
	if !autoPass(gloryOnly) {
		t.Fatal("Glory Butterfly with no allied Servant should not keep empty Battle open")
	}

	withEnemy := base
	withEnemy.Players[1].ServantZone = []simulatorview.CardView{{
		MatchID:     "enemy-s1",
		CardID:      "enemy",
		ShowFace:    true,
		Orientation: model.OrientationRecovered,
	}}
	if autoPass(withEnemy) {
		t.Fatal("Lumielle with an enemy Servant must keep Battle priority")
	}
}

func TestShouldAutoPassPriorityStopsForQuickcastDuringChase(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()
	prefs.SetString(networkPriorityModePreferenceKey, priorityModeEmpty)

	autoPass := makeShouldAutoPassPriority([]cards.Card{
		{
			ID:      "quick-def",
			Name:    "Quick Spell",
			Type:    "Conjure",
			Ability: "• Quickcast (You may play this card any time you have priority.)\n• Draw a card.",
		},
		{
			ID:      "slow-def",
			Name:    "Slow Spell",
			Type:    "Conjure",
			Ability: "• Draw a card.",
		},
	})

	base := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		ChaseLinkCount:       1,
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseMain,
		},
		Players: [2]simulatorview.PlayerView{
			{
				ID: "player-one",
				Hand: []simulatorview.CardView{
					{MatchID: "hand-1", CardID: "slow-def", ShowFace: true},
				},
			},
			{ID: "player-two"},
		},
	}
	if !autoPass(base) {
		t.Fatal("Chase with only non-Quickcast hand should auto-pass")
	}

	withQuickcast := base
	withQuickcast.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "hand-1", CardID: "quick-def", ShowFace: true},
	}
	if autoPass(withQuickcast) {
		t.Fatal("Chase with Quickcast in hand must not auto-pass")
	}
}

func TestShouldAutoPassPriorityStopsForActivatedAbilityDuringChase(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()
	prefs.SetString(networkPriorityModePreferenceKey, priorityModeEmpty)

	autoPass := makeShouldAutoPassPriority([]cards.Card{
		{
			ID:      "hilde",
			Name:    "Hilde Willow",
			Type:    "Caster",
			Ability: "• Rest: Look at the top card of an enemy deck. You may put that card on the bottom of that enemy deck.",
		},
		{
			ID:      "aether-only",
			Name:    "Sheer Gray",
			Type:    "Caster",
			Ability: "• Rest: Produce one aether of any element. Spend this aether only to play [Talent] cards.",
		},
		{
			ID:      "main-only",
			Name:    "Citra Piece",
			Type:    "Caster",
			Ability: "• Rest, banish a \"Citra Cayce\" from my Stock: Search your deck. Play this ability only during main timing.",
		},
	})

	base := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		ChaseLinkCount:       1,
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseMain,
		},
		Players: [2]simulatorview.PlayerView{
			{ID: "player-one"},
			{ID: "player-two"},
		},
	}

	withRestAbility := base
	withRestAbility.Players[0].CasterZone = []simulatorview.CardView{{
		MatchID:     "hilde-1",
		CardID:      "hilde",
		ShowFace:    true,
		Orientation: model.OrientationRecovered,
	}}
	if autoPass(withRestAbility) {
		t.Fatal("Chase with recovered Rest activated ability must not auto-pass")
	}

	rested := withRestAbility
	rested.Players[0].CasterZone[0].Orientation = model.OrientationRested
	if !autoPass(rested) {
		t.Fatal("Chase with only a rested Rest ability should auto-pass")
	}

	aetherOnly := base
	aetherOnly.Players[0].CasterZone = []simulatorview.CardView{{
		MatchID:     "sheer-1",
		CardID:      "aether-only",
		ShowFace:    true,
		Orientation: model.OrientationRecovered,
	}}
	if !autoPass(aetherOnly) {
		t.Fatal("Chase with only an aether ability should auto-pass")
	}

	mainOnly := base
	mainOnly.Players[0].CasterZone = []simulatorview.CardView{{
		MatchID:     "citra-1",
		CardID:      "main-only",
		ShowFace:    true,
		Orientation: model.OrientationRecovered,
	}}
	if !autoPass(mainOnly) {
		t.Fatal("Chase with only a main-timing-restricted ability should auto-pass")
	}
}

func TestShouldAutoPassPriorityStopsForPendingDecisions(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()
	prefs.SetString(networkPriorityModePreferenceKey, priorityModeEmpty)

	base := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseMain,
		},
	}
	if !shouldAutoPassPriorityForView(base) {
		t.Fatal("empty opponent Main should auto-pass")
	}

	withBreak := base
	withBreak.PendingBreak.PlayerID = "player-one"
	if shouldAutoPassPriorityForView(withBreak) {
		t.Fatal("PendingBreak must not auto-pass")
	}

	withDraw := base
	withDraw.PendingDraw.Step = model.PendingDrawOffer
	withDraw.PendingDraw.PlayerID = "player-one"
	if shouldAutoPassPriorityForView(withDraw) {
		t.Fatal("PendingDraw must not auto-pass")
	}

	withChoice := base
	withChoice.PendingChoice.PlayerID = "player-one"
	if shouldAutoPassPriorityForView(withChoice) {
		t.Fatal("PendingChoice must not auto-pass")
	}
}

func TestShouldAutoPassPriorityForViewHonorsFullAndStopsModes(t *testing.T) {
	application := test.NewApp()
	t.Cleanup(application.Quit)
	prefs := application.Preferences()

	match := simulatorview.MatchView{
		ViewerID:             "player-one",
		MatchStatus:          model.StatusInProgress,
		PrioritySequenceOpen: true,
		PriorityHolder:       "player-one",
		Turn: model.TurnState{
			ActivePlayer: "player-two",
			Phase:        model.PhaseDraw,
		},
	}

	prefs.SetString(networkPriorityModePreferenceKey, priorityModeFull)
	if shouldAutoPassPriorityForView(match) {
		t.Fatal("full control must never auto-pass")
	}

	prefs.SetString(networkPriorityModePreferenceKey, priorityModeStops)
	prefs.SetString(networkStopPhasesPreferenceKey, "Main,Battle")
	if !shouldAutoPassPriorityForView(match) {
		t.Fatal("stops mode should auto-pass Draw")
	}
	match.Turn.Phase = model.PhaseMain
	if shouldAutoPassPriorityForView(match) {
		t.Fatal("stops mode must not auto-pass configured Main stop")
	}
}

func TestCardDeclaresQuickcastAbility(t *testing.T) {
	if !cardDeclaresQuickcastAbility("• Quickcast (You may play this card any time you have priority.)\n• Draw a card.") {
		t.Fatal("expected Quickcast reminder text to match")
	}
	if !cardDeclaresQuickcastAbility("• Quickcast\n• Draw a card.") {
		t.Fatal("expected bare Quickcast label to match")
	}
	if cardDeclaresQuickcastAbility("• Cards you control lose Quickcast.") {
		t.Fatal("prose mentioning Quickcast must not count as declaring it")
	}
}
