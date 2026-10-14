package ops

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

// ops_test.go holds tests for every engine/ops primitive. Append new op
// coverage here as each operation is added.

func TestDestroyServantMovesCardToOwnerGraveyard(t *testing.T) {
	state := opsFieldStateForTest()
	beforeGrave := append([]model.MatchCardID(nil), state.Players[0].Graveyard...)

	if err := DestroyServant(&state, "p1-servant"); err != nil {
		t.Fatalf("DestroyServant() error = %v", err)
	}

	if slicesContains(state.Players[0].ServantZone, "p1-servant") {
		t.Fatalf("ServantZone still contains destroyed card: %#v", state.Players[0].ServantZone)
	}
	wantGrave := append(append([]model.MatchCardID(nil), beforeGrave...), "p1-servant")
	if !reflect.DeepEqual(state.Players[0].Graveyard, wantGrave) {
		t.Fatalf("Graveyard = %#v; want %#v", state.Players[0].Graveyard, wantGrave)
	}

	got := state.CardInstances["p1-servant"]
	if got.Face != model.CardFaceUp ||
		got.Orientation != model.OrientationRecovered ||
		got.Controller != "player-one" ||
		got.GrantedDoubleCorrupt {
		t.Fatalf("destroyed instance = %#v", got)
	}
	if state.Revision != 1 {
		t.Fatalf("Revision = %d; ops must not bump revision", state.Revision)
	}
}

func TestDestroyServantReturnsControlAndClearsDoubleCorrupt(t *testing.T) {
	state := opsFieldStateForTest()
	state.Players[1].ServantZone = []model.MatchCardID{"stolen-servant"}
	state.CardInstances["stolen-servant"] = model.CardInstance{
		MatchID:              "stolen-servant",
		CardID:               "printed",
		Owner:                "player-one",
		Controller:           "player-two",
		CardCategory:         model.CategoryPrintedCard,
		Face:                 model.CardFaceUp,
		Orientation:          model.OrientationRested,
		GrantedDoubleCorrupt: true,
	}

	if err := DestroyServant(&state, "stolen-servant"); err != nil {
		t.Fatalf("DestroyServant() error = %v", err)
	}
	if slicesContains(state.Players[1].ServantZone, "stolen-servant") {
		t.Fatal("controller ServantZone still holds destroyed card")
	}
	if !reflect.DeepEqual(state.Players[0].Graveyard[len(state.Players[0].Graveyard)-1:], []model.MatchCardID{"stolen-servant"}) {
		t.Fatalf("owner Graveyard = %#v; want stolen-servant appended", state.Players[0].Graveyard)
	}
	got := state.CardInstances["stolen-servant"]
	if got.Controller != "player-one" || got.GrantedDoubleCorrupt || got.Orientation != model.OrientationRecovered {
		t.Fatalf("destroyed stolen instance = %#v", got)
	}
}

func TestDestroyServantRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		cardID  model.MatchCardID
		wantErr string
	}{
		{name: "missing card", cardID: "missing", wantErr: "does not exist"},
		{name: "caster zone", cardID: "p1-caster", wantErr: "not in a servant zone"},
		{name: "hand", cardID: "p1-hand", wantErr: "not in a servant zone"},
		{name: "already graveyard", cardID: "p1-grave", wantErr: "not in a servant zone"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := opsFieldStateForTest()
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			err = DestroyServant(&state, test.cardID)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("DestroyServant() error = %v; want containing %q", err, test.wantErr)
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected DestroyServant mutated state")
			}
		})
	}
}

func TestSetOrientationUpdatesServantInPlace(t *testing.T) {
	state := opsFieldStateForTest()
	beforeFace := state.CardInstances["p1-servant"].Face
	beforeController := state.CardInstances["p1-servant"].Controller
	beforeDC := state.CardInstances["p1-servant"].GrantedDoubleCorrupt
	beforeZone := append([]model.MatchCardID(nil), state.Players[0].ServantZone...)

	if err := SetOrientation(&state, "p1-servant", model.OrientationRested); err != nil {
		t.Fatalf("SetOrientation() error = %v", err)
	}

	got := state.CardInstances["p1-servant"]
	if got.Orientation != model.OrientationRested {
		t.Fatalf("Orientation = %q; want Rested", got.Orientation)
	}
	if got.Face != beforeFace || got.Controller != beforeController || got.GrantedDoubleCorrupt != beforeDC {
		t.Fatalf("SetOrientation mutated face/controller/DC: %#v", got)
	}
	if !reflect.DeepEqual(state.Players[0].ServantZone, beforeZone) {
		t.Fatalf("ServantZone = %#v; want unchanged %#v", state.Players[0].ServantZone, beforeZone)
	}
	if state.Revision != 1 {
		t.Fatalf("Revision = %d; ops must not bump revision", state.Revision)
	}
}

func TestSetOrientationUpdatesCasterInPlace(t *testing.T) {
	state := opsFieldStateForTest()
	if err := SetOrientation(&state, "p1-caster", model.OrientationRested); err != nil {
		t.Fatalf("SetOrientation() error = %v", err)
	}
	if state.CardInstances["p1-caster"].Orientation != model.OrientationRested {
		t.Fatalf("caster Orientation = %q; want Rested", state.CardInstances["p1-caster"].Orientation)
	}
	if !reflect.DeepEqual(state.Players[0].CasterZone, []model.MatchCardID{"p1-caster"}) {
		t.Fatalf("CasterZone = %#v; want card still present", state.Players[0].CasterZone)
	}
}

func TestSetOrientationAcceptsAllValidOrientations(t *testing.T) {
	for _, orientation := range []model.CardOrientation{
		model.OrientationRecovered,
		model.OrientationRested,
		model.OrientationReversed,
	} {
		state := opsFieldStateForTest()
		if err := SetOrientation(&state, "p1-servant", orientation); err != nil {
			t.Fatalf("SetOrientation(%q) error = %v", orientation, err)
		}
		if state.CardInstances["p1-servant"].Orientation != orientation {
			t.Fatalf("Orientation = %q; want %q", state.CardInstances["p1-servant"].Orientation, orientation)
		}
	}
}

func TestModifyAttackAndDefenseAccumulateOnServant(t *testing.T) {
	state := opsFieldStateForTest()

	if err := ModifyAttack(&state, "p1-servant", 1000); err != nil {
		t.Fatalf("ModifyAttack() error = %v", err)
	}
	if err := ModifyAttack(&state, "p1-servant", 500); err != nil {
		t.Fatalf("second ModifyAttack() error = %v", err)
	}
	if err := ModifyDefense(&state, "p1-servant", -200); err != nil {
		t.Fatalf("ModifyDefense() error = %v", err)
	}

	got := state.CardInstances["p1-servant"]
	if got.AttackModifier != 1500 {
		t.Fatalf("AttackModifier = %d; want 1500", got.AttackModifier)
	}
	if got.DefenseModifier != -200 {
		t.Fatalf("DefenseModifier = %d; want -200", got.DefenseModifier)
	}
	if state.Revision != 1 {
		t.Fatalf("Revision = %d; ops must not bump revision", state.Revision)
	}
}

func TestModifyAttackAndDefenseRejectInvalidTargetsWithoutMutation(t *testing.T) {
	tests := []struct {
		name    string
		cardID  model.MatchCardID
		wantErr string
		attack  bool
	}{
		{name: "missing card attack", cardID: "missing", wantErr: "does not exist", attack: true},
		{name: "caster attack", cardID: "p1-caster", wantErr: "not in a servant zone", attack: true},
		{name: "hand defense", cardID: "p1-hand", wantErr: "not in a servant zone", attack: false},
		{name: "graveyard defense", cardID: "p1-grave", wantErr: "not in a servant zone", attack: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := opsFieldStateForTest()
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if test.attack {
				err = ModifyAttack(&state, test.cardID, 100)
			} else {
				err = ModifyDefense(&state, test.cardID, 100)
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v; want containing %q", err, test.wantErr)
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected modify mutated state")
			}
		})
	}
}

func TestDestroyServantClearsStatModifiers(t *testing.T) {
	state := opsFieldStateForTest()
	if err := ModifyAttack(&state, "p1-servant", 800); err != nil {
		t.Fatalf("ModifyAttack() error = %v", err)
	}
	if err := ModifyDefense(&state, "p1-servant", 400); err != nil {
		t.Fatalf("ModifyDefense() error = %v", err)
	}
	if err := DestroyServant(&state, "p1-servant"); err != nil {
		t.Fatalf("DestroyServant() error = %v", err)
	}
	got := state.CardInstances["p1-servant"]
	if got.AttackModifier != 0 || got.DefenseModifier != 0 {
		t.Fatalf("modifiers after destroy = ATK %+d DEF %+d; want 0", got.AttackModifier, got.DefenseModifier)
	}
}

func TestDrawMovesTopDeckCardsToHand(t *testing.T) {
	state := opsFieldStateForTest()
	beforeHand := append([]model.MatchCardID(nil), state.Players[0].Hand...)
	beforeDeck := append([]model.MatchCardID(nil), state.Players[0].Deck...)

	if err := Draw(&state, "player-one", 2); err != nil {
		t.Fatalf("Draw() error = %v", err)
	}
	wantHand := append(append([]model.MatchCardID(nil), beforeHand...), beforeDeck[0], beforeDeck[1])
	if !reflect.DeepEqual(state.Players[0].Hand, wantHand) {
		t.Fatalf("Hand = %#v; want %#v", state.Players[0].Hand, wantHand)
	}
	if !reflect.DeepEqual(state.Players[0].Deck, beforeDeck[2:]) {
		t.Fatalf("Deck = %#v; want %#v", state.Players[0].Deck, beforeDeck[2:])
	}
	if state.Revision != 1 {
		t.Fatalf("Revision = %d; ops must not bump revision", state.Revision)
	}
}

func TestDrawRejectsInsufficientDeckWithoutMutation(t *testing.T) {
	state := opsFieldStateForTest()
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	err = Draw(&state, "player-one", len(state.Players[0].Deck)+1)
	if err == nil || !strings.Contains(err.Error(), "error taking cards") {
		t.Fatalf("Draw() error = %v; want taking-cards error", err)
	}
	after, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected Draw mutated state")
	}
}

func TestReturnToHandFromServantAndCaster(t *testing.T) {
	state := opsFieldStateForTest()
	if err := ModifyAttack(&state, "p1-servant", 300); err != nil {
		t.Fatalf("ModifyAttack() error = %v", err)
	}
	if err := ReturnToHand(&state, "p1-servant"); err != nil {
		t.Fatalf("ReturnToHand(servant) error = %v", err)
	}
	if slicesContains(state.Players[0].ServantZone, "p1-servant") {
		t.Fatal("servant remained in ServantZone")
	}
	if !slicesContains(state.Players[0].Hand, "p1-servant") {
		t.Fatalf("Hand = %#v; want p1-servant", state.Players[0].Hand)
	}
	servant := state.CardInstances["p1-servant"]
	if servant.AttackModifier != 0 || servant.GrantedDoubleCorrupt || servant.Orientation != model.OrientationRecovered {
		t.Fatalf("returned servant instance = %#v", servant)
	}

	if err := ReturnToHand(&state, "p1-caster"); err != nil {
		t.Fatalf("ReturnToHand(caster) error = %v", err)
	}
	if slicesContains(state.Players[0].CasterZone, "p1-caster") {
		t.Fatal("caster remained in CasterZone")
	}
	if !slicesContains(state.Players[0].Hand, "p1-caster") {
		t.Fatalf("Hand = %#v; want p1-caster", state.Players[0].Hand)
	}
}

func TestReturnToHandRejectsNonFieldWithoutMutation(t *testing.T) {
	state := opsFieldStateForTest()
	before, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	err = ReturnToHand(&state, "p1-hand")
	if err == nil || !strings.Contains(err.Error(), "not in a field zone") {
		t.Fatalf("ReturnToHand() error = %v; want field-zone error", err)
	}
	after, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("rejected ReturnToHand mutated state")
	}
}

func TestPeekAndPutDeckTopOnBottom(t *testing.T) {
	state := opsFieldStateForTest()
	peeked, err := PeekDeckTop(&state, "player-one", 2)
	if err != nil {
		t.Fatalf("PeekDeckTop() error = %v", err)
	}
	if !reflect.DeepEqual(peeked, []model.MatchCardID{"p1-deck-1", "p1-deck-2"}) {
		t.Fatalf("PeekDeckTop() = %#v", peeked)
	}
	if !reflect.DeepEqual(state.Players[0].Deck, []model.MatchCardID{"p1-deck-1", "p1-deck-2", "p1-deck-3"}) {
		t.Fatal("PeekDeckTop mutated deck")
	}

	if err := PutDeckTopOnBottom(&state, "player-one"); err != nil {
		t.Fatalf("PutDeckTopOnBottom() error = %v", err)
	}
	if !reflect.DeepEqual(state.Players[0].Deck, []model.MatchCardID{"p1-deck-2", "p1-deck-3", "p1-deck-1"}) {
		t.Fatalf("Deck after PutDeckTopOnBottom = %#v", state.Players[0].Deck)
	}
}

func TestPutDeckCardOnBottomMovesChosenCard(t *testing.T) {
	state := opsFieldStateForTest()
	if err := PutDeckCardOnBottom(&state, "player-one", "p1-deck-2"); err != nil {
		t.Fatalf("PutDeckCardOnBottom() error = %v", err)
	}
	if !reflect.DeepEqual(state.Players[0].Deck, []model.MatchCardID{"p1-deck-1", "p1-deck-3", "p1-deck-2"}) {
		t.Fatalf("Deck = %#v", state.Players[0].Deck)
	}
}

func TestAddAndSpendAether(t *testing.T) {
	state := opsFieldStateForTest()
	if err := AddAether(&state, "player-one", "Aes", 2); err != nil {
		t.Fatalf("AddAether() error = %v", err)
	}
	if state.Players[0].Aether.Aes != 2 {
		t.Fatalf("Aes = %d; want 2", state.Players[0].Aether.Aes)
	}
	if err := SpendAether(&state, "player-one", "Aes", 1); err != nil {
		t.Fatalf("SpendAether() error = %v", err)
	}
	if state.Players[0].Aether.Aes != 1 {
		t.Fatalf("Aes = %d; want 1", state.Players[0].Aether.Aes)
	}
	if err := SpendAether(&state, "player-one", "Aes", 5); err == nil || !strings.Contains(err.Error(), "insufficient") {
		t.Fatalf("SpendAether() error = %v; want insufficient", err)
	}
}

func TestMoveOrbToHand(t *testing.T) {
	state := opsFieldStateForTest()
	if err := MoveOrbToHand(&state, "player-one", 0); err != nil {
		t.Fatalf("MoveOrbToHand() error = %v", err)
	}
	if slicesContains(state.Players[0].Orbs, "p1-orb-1") {
		t.Fatal("orb remained in Orbs")
	}
	if !slicesContains(state.Players[0].Hand, "p1-orb-1") {
		t.Fatalf("Hand = %#v; want p1-orb-1", state.Players[0].Hand)
	}
	if state.CardInstances["p1-orb-1"].Face != model.CardFaceDown {
		t.Fatal("corrupted orb should be face down")
	}
}

func TestExileAndMillAndDiscard(t *testing.T) {
	state := opsFieldStateForTest()
	if err := Exile(&state, "p1-servant"); err != nil {
		t.Fatalf("Exile() error = %v", err)
	}
	if !slicesContains(state.Players[0].Exile, "p1-servant") {
		t.Fatalf("Exile = %#v", state.Players[0].Exile)
	}
	if err := Mill(&state, "player-one", 1); err != nil {
		t.Fatalf("Mill() error = %v", err)
	}
	if !slicesContains(state.Players[0].Graveyard, "p1-deck-1") {
		t.Fatalf("Graveyard = %#v; want milled top deck", state.Players[0].Graveyard)
	}
	if err := Discard(&state, "p1-hand"); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if !slicesContains(state.Players[0].Graveyard, "p1-hand") {
		t.Fatalf("Graveyard = %#v; want discarded hand card", state.Players[0].Graveyard)
	}
}

func TestGrantDoubleCorruptAndAttachStock(t *testing.T) {
	state := opsFieldStateForTest()
	if err := GrantDoubleCorrupt(&state, "p1-servant", true); err != nil {
		t.Fatalf("GrantDoubleCorrupt() error = %v", err)
	}
	if !state.CardInstances["p1-servant"].GrantedDoubleCorrupt {
		t.Fatal("expected granted double corrupt")
	}
	if err := AttachStock(&state, "p1-servant", "p1-hand"); err != nil {
		t.Fatalf("AttachStock() error = %v", err)
	}
	if slicesContains(state.Players[0].Hand, "p1-hand") {
		t.Fatal("stock card remained in hand")
	}
	if !reflect.DeepEqual(state.CardInstances["p1-servant"].Stock, []model.MatchCardID{"p1-hand"}) {
		t.Fatalf("Stock = %#v", state.CardInstances["p1-servant"].Stock)
	}
}

func TestMoveFromDeckToHandAndReturnToDeck(t *testing.T) {
	state := opsFieldStateForTest()
	if err := MoveFromDeckToHand(&state, "player-one", "p1-deck-2"); err != nil {
		t.Fatalf("MoveFromDeckToHand() error = %v", err)
	}
	if slicesContains(state.Players[0].Deck, "p1-deck-2") || !slicesContains(state.Players[0].Hand, "p1-deck-2") {
		t.Fatalf("deck/hand after MoveFromDeckToHand = %#v / %#v", state.Players[0].Deck, state.Players[0].Hand)
	}
	if err := ReturnToDeck(&state, "p1-deck-2", model.DeckPlacementTop); err != nil {
		t.Fatalf("ReturnToDeck() error = %v", err)
	}
	if state.Players[0].Deck[0] != "p1-deck-2" {
		t.Fatalf("Deck top = %q; want p1-deck-2", state.Players[0].Deck[0])
	}
	if err := PutDeckCardOnTop(&state, "player-one", "p1-deck-3"); err != nil {
		t.Fatalf("PutDeckCardOnTop() error = %v", err)
	}
	if state.Players[0].Deck[0] != "p1-deck-3" {
		t.Fatalf("Deck top = %q; want p1-deck-3", state.Players[0].Deck[0])
	}
}

func TestBanishFieldAndStock(t *testing.T) {
	state := opsFieldStateForTest()
	if err := AttachStock(&state, "p1-servant", "p1-hand"); err != nil {
		t.Fatalf("AttachStock() error = %v", err)
	}
	if err := BanishStock(&state, "p1-hand"); err != nil {
		t.Fatalf("BanishStock() error = %v", err)
	}
	if len(state.CardInstances["p1-servant"].Stock) != 0 {
		t.Fatalf("Stock = %#v; want empty after banish", state.CardInstances["p1-servant"].Stock)
	}
	if !slicesContains(state.Players[0].Graveyard, "p1-hand") {
		t.Fatalf("Graveyard = %#v; want banished stock", state.Players[0].Graveyard)
	}
	if err := Banish(&state, "p1-other-servant"); err != nil {
		t.Fatalf("Banish() error = %v", err)
	}
	if slicesContains(state.Players[0].ServantZone, "p1-other-servant") {
		t.Fatal("banished servant remained on field")
	}
	if !slicesContains(state.Players[0].Graveyard, "p1-other-servant") {
		t.Fatalf("Graveyard = %#v; want banished servant", state.Players[0].Graveyard)
	}
}

func TestGainControlAndExchangeControl(t *testing.T) {
	state := opsFieldStateForTest()
	if err := GainControl(&state, "p2-servant", "player-one"); err != nil {
		t.Fatalf("GainControl() error = %v", err)
	}
	if !slicesContains(state.Players[0].ServantZone, "p2-servant") {
		t.Fatalf("P1 servants = %#v; want stolen servant", state.Players[0].ServantZone)
	}
	if state.CardInstances["p2-servant"].Controller != "player-one" {
		t.Fatalf("Controller = %q", state.CardInstances["p2-servant"].Controller)
	}
	if state.CardInstances["p2-servant"].Owner != "player-two" {
		t.Fatalf("Owner changed to %q", state.CardInstances["p2-servant"].Owner)
	}

	state = opsFieldStateForTest()
	if err := ExchangeControl(&state, "p1-servant", "p2-servant"); err != nil {
		t.Fatalf("ExchangeControl() error = %v", err)
	}
	if state.CardInstances["p1-servant"].Controller != "player-two" || state.CardInstances["p2-servant"].Controller != "player-one" {
		t.Fatalf("controllers = %q / %q", state.CardInstances["p1-servant"].Controller, state.CardInstances["p2-servant"].Controller)
	}
	if !slicesContains(state.Players[1].ServantZone, "p1-servant") || !slicesContains(state.Players[0].ServantZone, "p2-servant") {
		t.Fatalf("zones after exchange = %#v / %#v", state.Players[0].ServantZone, state.Players[1].ServantZone)
	}
}

func TestSwitchSetFaceAndPlace(t *testing.T) {
	state := opsFieldStateForTest()
	if err := Switch(&state, "p1-servant"); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	if state.CardInstances["p1-servant"].Orientation != model.OrientationReversed {
		t.Fatalf("orientation = %q; want reversed", state.CardInstances["p1-servant"].Orientation)
	}
	if err := Switch(&state, "p1-other-servant"); err != nil {
		t.Fatalf("Switch(rested) error = %v", err)
	}
	if state.CardInstances["p1-other-servant"].Orientation != model.OrientationRested {
		t.Fatal("rested servant should be unchanged by Switch")
	}
	if err := SetFace(&state, "p1-caster", model.CardFaceUp); err != nil {
		t.Fatalf("SetFace() error = %v", err)
	}
	if state.CardInstances["p1-caster"].Face != model.CardFaceUp {
		t.Fatal("caster face not updated")
	}
	if err := PutAsOrb(&state, "p1-grave", "player-one"); err != nil {
		t.Fatalf("PutAsOrb() error = %v", err)
	}
	if !slicesContains(state.Players[0].Orbs, "p1-grave") || slicesContains(state.Players[0].Graveyard, "p1-grave") {
		t.Fatalf("orb/grave after PutAsOrb = %#v / %#v", state.Players[0].Orbs, state.Players[0].Graveyard)
	}
	if err := MoveToHand(&state, "p1-deck-1"); err != nil {
		t.Fatalf("MoveToHand() error = %v", err)
	}
	if !slicesContains(state.Players[0].Hand, "p1-deck-1") {
		t.Fatalf("Hand = %#v", state.Players[0].Hand)
	}
	if err := PutAsCaster(&state, "p1-deck-2", "player-one", model.CardFaceDown, model.OrientationRested); err != nil {
		t.Fatalf("PutAsCaster() error = %v", err)
	}
	if !slicesContains(state.Players[0].CasterZone, "p1-deck-2") {
		t.Fatalf("CasterZone = %#v", state.Players[0].CasterZone)
	}
	if err := PutAsServant(&state, "p1-deck-3", "player-one", model.CardFaceUp, model.OrientationRecovered); err != nil {
		t.Fatalf("PutAsServant() error = %v", err)
	}
	if !slicesContains(state.Players[0].ServantZone, "p1-deck-3") {
		t.Fatalf("ServantZone = %#v", state.Players[0].ServantZone)
	}
}

func TestReorderShuffleAndCombatModifiers(t *testing.T) {
	state := opsFieldStateForTest()
	if err := ReorderDeckTops(&state, "player-one", []model.MatchCardID{"p1-deck-3", "p1-deck-1", "p1-deck-2"}); err != nil {
		t.Fatalf("ReorderDeckTops() error = %v", err)
	}
	if !reflect.DeepEqual(state.Players[0].Deck, []model.MatchCardID{"p1-deck-3", "p1-deck-1", "p1-deck-2"}) {
		t.Fatalf("Deck = %#v", state.Players[0].Deck)
	}
	fixed := fixedRandom{values: []int{0, 0, 0}}
	if err := ShuffleDeck(&state, "player-one", &fixed); err != nil {
		t.Fatalf("ShuffleDeck() error = %v", err)
	}
	if err := SetCombatModifiers(&state, "p1-servant", 500, -200); err != nil {
		t.Fatalf("SetCombatModifiers() error = %v", err)
	}
	got := state.CardInstances["p1-servant"]
	if got.AttackModifier != 500 || got.DefenseModifier != -200 {
		t.Fatalf("modifiers = %d/%d", got.AttackModifier, got.DefenseModifier)
	}
	if err := SwapCombatModifiers(&state, "p1-servant"); err != nil {
		t.Fatalf("SwapCombatModifiers() error = %v", err)
	}
	got = state.CardInstances["p1-servant"]
	if got.AttackModifier != -200 || got.DefenseModifier != 500 {
		t.Fatalf("swapped modifiers = %d/%d", got.AttackModifier, got.DefenseModifier)
	}
}

type fixedRandom struct {
	values []int
	next   int
}

func (r *fixedRandom) RandInt(upperBound int) int {
	if r.next >= len(r.values) {
		return 0
	}
	v := r.values[r.next]
	r.next++
	if v < 0 || v >= upperBound {
		return 0
	}
	return v
}

func TestSetOrientationRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		cardID      model.MatchCardID
		orientation model.CardOrientation
		wantErr     string
	}{
		{name: "missing card", cardID: "missing", orientation: model.OrientationRested, wantErr: "does not exist"},
		{name: "invalid orientation", cardID: "p1-servant", orientation: "Sideways", wantErr: "invalid orientation"},
		{name: "hand card", cardID: "p1-hand", orientation: model.OrientationRested, wantErr: "not in a field zone"},
		{name: "graveyard card", cardID: "p1-grave", orientation: model.OrientationRested, wantErr: "not in a field zone"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := opsFieldStateForTest()
			before, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			err = SetOrientation(&state, test.cardID, test.orientation)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("SetOrientation() error = %v; want containing %q", err, test.wantErr)
			}
			after, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected SetOrientation mutated state")
			}
		})
	}
}

func opsFieldStateForTest() model.MatchState {
	return model.MatchState{
		Revision: 1,
		Players: [2]model.PlayerState{
			{
				ID:          "player-one",
				Hand:        []model.MatchCardID{"p1-hand"},
				Deck:        []model.MatchCardID{"p1-deck-1", "p1-deck-2", "p1-deck-3"},
				Orbs:        []model.MatchCardID{"p1-orb-1"},
				CasterZone:  []model.MatchCardID{"p1-caster"},
				ServantZone: []model.MatchCardID{"p1-servant", "p1-other-servant"},
				Graveyard:   []model.MatchCardID{"p1-grave"},
			},
			{
				ID:          "player-two",
				Deck:        []model.MatchCardID{"p2-deck-1"},
				ServantZone: []model.MatchCardID{"p2-servant"},
			},
		},
		CardInstances: map[model.MatchCardID]model.CardInstance{
			"p1-hand": {
				MatchID: "p1-hand", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRecovered,
			},
			"p1-deck-1": {
				MatchID: "p1-deck-1", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown, Orientation: model.OrientationRecovered,
			},
			"p1-deck-2": {
				MatchID: "p1-deck-2", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown, Orientation: model.OrientationRecovered,
			},
			"p1-deck-3": {
				MatchID: "p1-deck-3", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown, Orientation: model.OrientationRecovered,
			},
			"p1-orb-1": {
				MatchID: "p1-orb-1", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown, Orientation: model.OrientationRecovered,
			},
			"p1-caster": {
				MatchID: "p1-caster", CardID: "printed-caster", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown, Orientation: model.OrientationRecovered,
			},
			"p1-servant": {
				MatchID: "p1-servant", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRecovered,
				GrantedDoubleCorrupt: true,
			},
			"p1-other-servant": {
				MatchID: "p1-other-servant", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRested,
			},
			"p1-grave": {
				MatchID: "p1-grave", CardID: "printed", Owner: "player-one", Controller: "player-one",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRecovered,
			},
			"p2-deck-1": {
				MatchID: "p2-deck-1", CardID: "printed", Owner: "player-two", Controller: "player-two",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceDown, Orientation: model.OrientationRecovered,
			},
			"p2-servant": {
				MatchID: "p2-servant", CardID: "printed", Owner: "player-two", Controller: "player-two",
				CardCategory: model.CategoryPrintedCard, Face: model.CardFaceUp, Orientation: model.OrientationRecovered,
			},
		},
	}
}

func slicesContains(ids []model.MatchCardID, want model.MatchCardID) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}
