package ui

import (
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	cardimages "github.com/HybridUofA/casters-compendium/internal/carddata/images"
	"github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

type refreshTrackingRectangle struct {
	*canvas.Rectangle
	refreshes int
}

func (rectangle *refreshTrackingRectangle) Refresh() {
	rectangle.refreshes++
}

func TestNewBoardScreenContainsBothPlayerFieldsAndRequiredZones(t *testing.T) {
	match := testMatchView()
	match.Revision = 7
	screen := NewBoardScreen(match, testDefinitions(), BoardActions{}, nil)

	for _, text := range []string{
		"Opponent Field — player-two",
		"Player Field — player-one",
		"Orb Zone",
		"Deck Zone",
		"Graveyard",
		"Exile",
		"Servant Zone",
		"Barrier Zone",
		"Caster Zone",
		"Hand",
		"Card Information",
		"Aether Pools",
		"Card Preview",
		"Turn 0 • Call • Revision 7 • Active player: player-one • Priority:  • Chase: 0",
	} {
		if !containsText(screen, text) {
			t.Errorf("board screen does not contain %q", text)
		}
	}
}

func TestTurnStatusUsesHighContrastBoardText(t *testing.T) {
	controller := NewBoardController(testMatchView(), testDefinitions(), BoardActions{}, nil)
	if controller.status.Color != boardForeground {
		t.Fatalf("status color = %#v; want high-contrast board color %#v", controller.status.Color, boardForeground)
	}
	if !controller.status.TextStyle.Bold {
		t.Fatal("turn status is not rendered with bold emphasis")
	}
}

func TestBoardUsesCompactUnscrolledBattlefieldWithCenteredPhaseGuidance(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) {}},
		nil,
	)

	if _, isScroll := controller.boardArea.(*container.Scroll); isScroll {
		t.Fatal("complete battlefield is still wrapped in a scroll container")
	}
	if !containsText(controller.boardArea, "Current phase: Call  •  Select Main to continue") {
		t.Fatal("center phase band does not explain the current phase and legal transition")
	}
	if height := controller.boards.MinSize().Height; height > 765 {
		t.Fatalf("compact two-player battlefield minimum height = %.1f; want at most 765", height)
	}
}

func TestContainerStructureRefreshDoesNotRecursivelyRefreshChildren(t *testing.T) {
	child := &refreshTrackingRectangle{Rectangle: canvas.NewRectangle(color.Transparent)}
	target := container.NewWithoutLayout(child)

	refreshContainerStructure(target)

	if child.refreshes != 0 {
		t.Fatalf("structural refresh refreshed child %d times; want 0", child.refreshes)
	}
}

func TestCenteredPhaseGuidanceUpdatesWithMatchState(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) {}},
		nil,
	)

	updated := match
	updated.Turn.Phase = model.PhaseMain
	updated.Revision++
	controller.Update(updated)

	if !containsText(controller.boardArea, "Current phase: Main  •  Select Battle to continue") {
		t.Fatal("center phase guidance did not update after the phase changed")
	}
}

func TestSelectingZoneUpdatesCardInformationPanel(t *testing.T) {
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	servantButton := findButton(screen, "Servant Zone")
	if servantButton == nil {
		t.Fatal("board screen does not contain a Servant Zone button")
	}

	test.Tap(servantButton)

	if !containsText(screen, "Opponent — Servant Zone") {
		t.Fatal("selecting opponent Servant Zone did not update preview title")
	}
	if !containsText(screen, "Servants in play occupy this row.") {
		t.Fatal("selecting Servant Zone did not update preview description")
	}
}

func TestPreviewReusesRenderedCardUntilSelectionChanges(t *testing.T) {
	preview := newPreviewPanel()
	card := cards.Card{ID: "missing-preview-art", Name: "Preview Card"}

	preview.showCard(card)
	firstPreviewObject := preview.image.Objects[len(preview.image.Objects)-1]
	preview.showCard(card)

	if got := preview.image.Objects[len(preview.image.Objects)-1]; got != firstPreviewObject {
		t.Fatal("showing the same card rebuilt its preview image")
	}
	preview.showHiddenCard("Player", "Hand")
	preview.showCard(card)
	if got := preview.image.Objects[len(preview.image.Objects)-1]; got == firstPreviewObject {
		t.Fatal("changing away from a card did not invalidate its preview selection")
	}
}

func TestPreviewPrefersCachedThumbnailOverFullArtwork(t *testing.T) {
	oldImages := cardimages.DefaultDirectory
	oldThumbnails := cardimages.ThumbnailDirectory
	t.Cleanup(func() { cardimages.ConfigureDirectories(oldImages, oldThumbnails) })

	root := t.TempDir()
	images := filepath.Join(root, "images")
	thumbnails := filepath.Join(root, "thumbnails")
	if err := os.MkdirAll(images, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(thumbnails, 0o755); err != nil {
		t.Fatal(err)
	}
	fullPath := filepath.Join(images, "42.png")
	thumbnailPath := filepath.Join(thumbnails, "42.jpg")
	if err := os.WriteFile(fullPath, []byte("full"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(thumbnailPath, []byte("thumbnail"), 0o600); err != nil {
		t.Fatal(err)
	}
	cardimages.ConfigureDirectories(images, thumbnails)

	got, found := previewImagePath("42")
	if !found || got != thumbnailPath {
		t.Fatalf("preview image = %q/%t; want thumbnail %q", got, found, thumbnailPath)
	}
}

func TestProjectedServantGraveyardAndExileCardsRender(t *testing.T) {
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	if findCardTile(screen, "visible-servant-card") == nil {
		t.Fatal("projected Servant Zone card was not rendered")
	}
	if findCardTile(screen, "visible-grave-card") == nil {
		t.Fatal("projected Graveyard card was not rendered")
	}
	if findCardTile(screen, "visible-exile-card") == nil {
		t.Fatal("projected Exile card was not rendered")
	}
}

func TestPersistentFieldCardsSplitIntoServantAndBarrierRows(t *testing.T) {
	field := []simulatorview.CardView{
		{MatchID: "servant", CardID: "servant-definition"},
		{MatchID: "barrier", CardID: "barrier-definition"},
		{MatchID: "unknown", CardID: "missing-definition"},
	}
	definitions := cardLookup{
		"servant-definition": {ID: "servant-definition", Type: " Servant "},
		"barrier-definition": {ID: "barrier-definition", Type: " BARRIER "},
	}

	servants, barriers := splitPersistentFieldCards(field, definitions)

	if len(servants) != 2 || servants[0].MatchID != "servant" || servants[1].MatchID != "unknown" {
		t.Fatalf("Servant row = %#v; want servant and safe unknown fallback", servants)
	}
	if len(barriers) != 1 || barriers[0].MatchID != "barrier" {
		t.Fatalf("Barrier row = %#v; want barrier", barriers)
	}
}

func TestPlayerUtilityZonesMirrorAcrossTheBattlefield(t *testing.T) {
	controller := NewBoardController(testMatchView(), testDefinitions(), BoardActions{}, nil)
	controller.Content().Resize(fyne.NewSize(1400, 850))

	opponent := controller.boards.Objects[0]
	opponentDeck, foundDeck := findButtonPosition(opponent, "Deck Zone", fyne.Position{})
	opponentOrbs, foundOrbs := findButtonPosition(opponent, "Orb Zone", fyne.Position{})
	if !foundDeck || !foundOrbs || opponentDeck.X >= opponentOrbs.X {
		t.Fatalf("opponent Deck/Orb positions = %v/%v; want Deck on the left and Orbs on the right", opponentDeck, opponentOrbs)
	}

	player := controller.boards.Objects[1]
	playerDeck, foundDeck := findButtonPosition(player, "Deck Zone", fyne.Position{})
	playerOrbs, foundOrbs := findButtonPosition(player, "Orb Zone", fyne.Position{})
	if !foundDeck || !foundOrbs || playerOrbs.X >= playerDeck.X {
		t.Fatalf("player Orb/Deck positions = %v/%v; want Orbs on the left and Deck on the right", playerOrbs, playerDeck)
	}
}

func TestOrbZoneUsesUnscrolledVerticalCardLayers(t *testing.T) {
	orbs := make([]simulatorview.CardView, 7)
	for index := range orbs {
		orbs[index] = simulatorview.CardView{Face: model.CardFaceDown}
	}
	zone := newLayeredOrbZone(
		"Player",
		orbs,
		cardLookup{},
		newPreviewPanel(),
		false,
	)
	if scroll := findScroll(zone); scroll != nil {
		t.Fatal("Orb zone contains a scrollbar")
	}
	stack := findVerticalCardStack(zone)
	if stack == nil {
		t.Fatal("Orb zone does not contain a vertical layered-card layout")
	}
	stack.Resize(fyne.NewSize(orbCardWidth+20, 260))

	tiles := collectCardTiles(stack)
	if len(tiles) != 7 {
		t.Fatalf("Orb tile count = %d; want 7", len(tiles))
	}
	for index := 1; index < len(tiles); index++ {
		if tiles[index].Position().X != tiles[0].Position().X {
			t.Fatalf("Orb %d X = %.1f; want vertically aligned X %.1f", index, tiles[index].Position().X, tiles[0].Position().X)
		}
		step := tiles[index].Position().Y - tiles[index-1].Position().Y
		if step <= 0 || step >= tiles[index].Size().Height {
			t.Fatalf("Orb %d vertical step = %.1f; want positive overlap below card height %.1f", index, step, tiles[index].Size().Height)
		}
	}
}

func TestAetherPoolsLiveBelowDescriptionAndUpdateWithoutRebuildingFields(t *testing.T) {
	match := testMatchView()
	match.Players[0].Aether.Aes = 2
	controller := NewBoardController(match, testDefinitions(), BoardActions{}, nil)
	firstOpponentField := controller.boards.Objects[0]
	firstPlayerField := controller.boards.Objects[1]

	if !containsText(controller.aetherPools, "Opponent — player-two") ||
		!containsText(controller.aetherPools, "You — player-one") ||
		!containsText(controller.aetherPools, "2") {
		t.Fatal("sidebar does not show both public Aether pools and the viewer's amount")
	}

	updated := match
	updated.Players[0].Aether.Aes = 3
	updated.Revision++
	controller.Update(updated)

	if controller.boards.Objects[0] != firstOpponentField || controller.boards.Objects[1] != firstPlayerField {
		t.Fatal("Aether-only update rebuilt a player battlefield")
	}
	if !containsText(controller.aetherPools, "3") {
		t.Fatal("sidebar Aether pool did not receive the updated amount")
	}
}

func TestAetherPoolRenderingOnlyIncludesPositiveTypes(t *testing.T) {
	pool := model.AetherPool{
		Aes:          2,
		Void:         1,
		NonElemental: 3,
	}
	entries := visibleAetherEntries(pool)
	want := []struct {
		name   string
		amount int
	}{
		{name: "Aes", amount: 2},
		{name: "Void", amount: 1},
		{name: "Non-elemental", amount: 3},
	}
	if len(entries) != len(want) {
		t.Fatalf("visible Aether entry count = %d; want %d", len(entries), len(want))
	}
	for index, expected := range want {
		if entries[index].name != expected.name || entries[index].amount != expected.amount {
			t.Fatalf("visible Aether entry %d = %s/%d; want %s/%d", index, entries[index].name, entries[index].amount, expected.name, expected.amount)
		}
		if len(entries[index].resource.Content()) == 0 {
			t.Fatalf("visible Aether entry %q has no embedded icon", entries[index].name)
		}
	}
}

func TestEmptyAetherPoolUsesCompactZeroState(t *testing.T) {
	display := newAetherPoolDisplay(model.AetherPool{})
	if !containsText(display, "Aether: 0") {
		t.Fatal("empty Aether pool did not render its compact zero state")
	}
}

func TestCardDescriptionUsesRemainingPreviewHeight(t *testing.T) {
	region := newPreviewRegion(newPreviewPanel(), container.NewVBox())
	region.Resize(fyne.NewSize(previewPanelWidth, 800))
	descriptionScroll := findScroll(region)
	if descriptionScroll == nil {
		t.Fatal("preview region does not contain a description scroller")
	}
	if descriptionScroll.Size().Height <= 145 {
		t.Fatalf("description height = %.1f; want it to expand beyond its 145 minimum", descriptionScroll.Size().Height)
	}
}

func TestGenerateNonElementalAetherSubmitsOwnCasterAndCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.ViewerID = "player-two"
	match.MatchStatus = model.StatusInProgress
	match.Revision = 15
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseBattle
	match.Turn.ActivePlayer = "player-one"
	match.Players[0].CasterZone[1].MatchID = ""
	match.Players[0].CasterZone[1].CardID = ""
	match.Players[1].CasterZone[1].MatchID = "player-two-facedown-caster"
	match.Players[1].CasterZone[1].CardID = "face-down-caster-card"
	generatedBy := model.MatchCardID("")
	generatedRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			GenerateNonElementalAether: func(cardID model.MatchCardID, revision model.Revision) {
				generatedBy = cardID
				generatedRevision = revision
			},
		},
		nil,
	)
	generateButton := findButton(screen, "Produce Aether")
	if generateButton == nil {
		t.Fatal("Aether control was not rendered for the non-active player's eligible Caster")
	}
	if generateButton.Visible() {
		t.Fatal("Aether control is visible before selecting a Caster")
	}
	caster := findCardTile(screen, "face-down-caster-card")
	if caster == nil {
		t.Fatal("viewer's known face-down Caster was not rendered")
	}

	test.Tap(caster)
	generateButton = findButton(screen, "Rest Selected for 1 Aether")
	if !generateButton.Visible() || generateButton.Disabled() {
		t.Fatal("Aether control did not appear and enable after selecting an eligible Caster")
	}
	test.Tap(caster)
	if generateButton.Visible() {
		t.Fatal("Aether control remained visible after deselecting the Caster")
	}
	test.Tap(caster)
	test.Tap(generateButton)

	if generatedBy != "player-two-facedown-caster" || generatedRevision != 15 {
		t.Fatalf("Aether action submitted Caster/revision %q/%d; want player-two-facedown-caster/15", generatedBy, generatedRevision)
	}
}

func TestGenerateNonElementalAetherControlRequiresEligibleCaster(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*simulatorview.MatchView)
	}{
		{
			name: "match is not in progress",
			mutate: func(match *simulatorview.MatchView) {
				match.MatchStatus = model.StatusSetup
			},
		},
		{
			name: "Caster is Rested",
			mutate: func(match *simulatorview.MatchView) {
				match.Players[0].CasterZone[1].Orientation = model.OrientationRested
			},
		},
		{
			name: "Caster is face up",
			mutate: func(match *simulatorview.MatchView) {
				match.Players[0].CasterZone[1].Face = model.CardFaceUp
				match.Players[0].CasterZone[1].ShowFace = true
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			testCase.mutate(&match)
			screen := NewBoardScreen(
				match,
				testDefinitions(),
				BoardActions{GenerateNonElementalAether: func(model.MatchCardID, model.Revision) {}},
				nil,
			)
			if findButton(screen, "Rest Selected for 1 Aether") != nil {
				t.Fatal("Aether control was rendered without an eligible Caster")
			}
		})
	}
}

func TestUseCasterTokenControlAppearsAfterSelectionAndSubmitsCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.ViewerID = "player-two"
	match.MatchStatus = model.StatusInProgress
	match.Revision = 16
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseBattle
	match.Turn.ActivePlayer = "player-one"
	usedToken := model.MatchCardID("")
	usedRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			UseCasterToken: func(tokenID model.MatchCardID, revision model.Revision) {
				usedToken = tokenID
				usedRevision = revision
			},
		},
		nil,
	)
	actionButton := findButton(screen, "Produce Aether")
	if actionButton == nil {
		t.Fatal("hidden Caster Token action was not constructed")
	}
	if actionButton.Visible() {
		t.Fatal("Caster Token action is visible before selecting the token")
	}
	token := findCardTileByMatchID(screen, "player-two-token")
	if token == nil {
		t.Fatal("viewer's Caster Token was not rendered")
	}

	test.Tap(token)
	actionButton = findButton(screen, "Remove Token for 1 Aether")
	if actionButton == nil || !actionButton.Visible() || actionButton.Disabled() {
		t.Fatal("Caster Token action did not appear and enable after selection")
	}
	test.Tap(actionButton)

	if usedToken != "player-two-token" || usedRevision != 16 {
		t.Fatalf("token action submitted token/revision %q/%d; want player-two-token/16", usedToken, usedRevision)
	}
}

func TestGenerateCasterAetherControlAppearsAfterSelectionAndSubmitsCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.ViewerID = "player-two"
	match.MatchStatus = model.StatusInProgress
	match.Revision = 18
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseBattle
	match.Turn.ActivePlayer = "player-one"
	match.Players[1].CasterZone[1] = simulatorview.CardView{
		MatchID:     "player-two-faceup-caster",
		CardID:      "visible-faceup-caster",
		Face:        model.CardFaceUp,
		Orientation: model.OrientationRecovered,
		ShowFace:    true,
	}
	generatedBy := model.MatchCardID("")
	generatedRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			GenerateCasterAether: func(cardID model.MatchCardID, revision model.Revision) {
				generatedBy = cardID
				generatedRevision = revision
			},
		},
		nil,
	)
	actionButton := findButton(screen, "Produce Aether")
	if actionButton == nil || actionButton.Visible() {
		t.Fatal("face-up Caster action should exist but remain hidden before selection")
	}
	caster := findCardTileByMatchID(screen, "player-two-faceup-caster")
	if caster == nil {
		t.Fatal("viewer's face-up Caster was not rendered")
	}

	test.Tap(caster)
	actionButton = findButton(screen, "Rest Selected for Elemental Aether")
	if actionButton == nil || !actionButton.Visible() || actionButton.Disabled() {
		t.Fatal("face-up Caster action did not appear and enable after selection")
	}
	test.Tap(actionButton)

	if generatedBy != "player-two-faceup-caster" || generatedRevision != 18 {
		t.Fatalf("Caster action submitted card/revision %q/%d; want player-two-faceup-caster/18", generatedBy, generatedRevision)
	}
}

func TestRestedFaceUpCasterRendersSidewaysAfterBoardUpdate(t *testing.T) {
	match := testMatchView()
	match.ViewerID = "player-two"
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseBattle
	match.Players[1].CasterZone[1] = simulatorview.CardView{
		MatchID:     "player-two-faceup-caster",
		CardID:      "visible-faceup-caster",
		Face:        model.CardFaceUp,
		Orientation: model.OrientationRecovered,
		ShowFace:    true,
	}
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{GenerateCasterAether: func(model.MatchCardID, model.Revision) {}},
		nil,
	)

	updated := match
	updated.Revision++
	updated.Players[1].CasterZone = append([]simulatorview.CardView(nil), match.Players[1].CasterZone...)
	updated.Players[1].CasterZone[1].Orientation = model.OrientationRested
	controller.Update(updated)

	caster := findCardTileByMatchID(controller.Content(), "player-two-faceup-caster")
	if caster == nil {
		t.Fatal("updated face-up Caster was not rendered")
	}
	if caster.MinSize().Width <= caster.MinSize().Height {
		t.Fatalf("Rested face-up Caster size = %v; want landscape orientation", caster.MinSize())
	}
}

func TestHoveringCasterTokenUpdatesCardInformationPanel(t *testing.T) {
	token := testDefinitions()[0]
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	tile := findCardTile(screen, token.ID)
	if tile == nil {
		t.Fatal("board screen does not contain the starting Caster Token")
	}

	tile.MouseIn(&desktop.MouseEvent{})

	if !containsText(screen, token.Name) {
		t.Fatal("hovering Caster Token did not update preview title")
	}
	if !containsTextPart(screen, token.Ability) {
		t.Fatal("hovering Caster Token did not update preview description")
	}
}

func TestConcealedCardsDoNotExposeDefinitions(t *testing.T) {
	screen := NewBoardScreen(testMatchView(), testDefinitions(), BoardActions{}, nil)
	tile := findConcealedCardTile(screen)
	if tile == nil {
		t.Fatal("board screen does not render concealed cards")
	}
	if tile.Card.ID != "" || tile.View.CardID != "" || tile.View.MatchID != "" {
		t.Fatalf("concealed tile leaked identifiers: Card=%#v View=%#v", tile.Card, tile.View)
	}

	tile.MouseIn(&desktop.MouseEvent{})

	if !containsText(screen, "Concealed Card") {
		t.Fatal("hovering concealed card did not show generic preview")
	}
	if containsTextPart(screen, "Produce one Aether.") {
		t.Fatal("hovering concealed card exposed a card ability")
	}
}

func TestOpeningHandSelectionSubmitsSelectedMatchIDs(t *testing.T) {
	var submitted []model.MatchCardID
	screen := NewBoardScreen(
		testMatchView(),
		testDefinitions(),
		BoardActions{
			SubmitOpeningHand: func(replace []model.MatchCardID, _ model.Revision) {
				submitted = append([]model.MatchCardID(nil), replace...)
			},
		},
		nil,
	)
	tile := findCardTile(screen, "visible-hand-card")
	if tile == nil {
		t.Fatal("viewer hand card was not rendered")
	}
	replaceButton := findButton(screen, "Replace Selected")
	if replaceButton == nil {
		t.Fatal("opening-hand replacement control was not rendered")
	}
	if !replaceButton.Disabled() {
		t.Fatal("replacement control is enabled before a card is selected")
	}

	test.Tap(tile)

	if replaceButton.Disabled() {
		t.Fatal("replacement control did not enable after selection")
	}
	test.Tap(replaceButton)

	want := model.MatchCardID("player-one-hand")
	if len(submitted) != 1 || submitted[0] != want {
		t.Fatalf("submitted replacement IDs = %v; want [%q]", submitted, want)
	}
}

func TestKeepHandSubmitsEmptyReplacement(t *testing.T) {
	submitted := []model.MatchCardID{"not-submitted"}
	screen := NewBoardScreen(
		testMatchView(),
		testDefinitions(),
		BoardActions{
			SubmitOpeningHand: func(replace []model.MatchCardID, _ model.Revision) {
				submitted = append([]model.MatchCardID(nil), replace...)
			},
		},
		nil,
	)
	keepButton := findButton(screen, "Keep Hand")
	if keepButton == nil {
		t.Fatal("Keep Hand control was not rendered")
	}

	test.Tap(keepButton)

	if len(submitted) != 0 {
		t.Fatalf("Keep Hand submitted replacements %v; want none", submitted)
	}
}

func TestFaceDownLevelOneCallSubmitsSelectedCardAndCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 12
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	match.Players[0].OpeningHandFinalized = true
	calledID := model.MatchCardID("")
	calledRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			CallFaceDownLevelOne: func(cardID model.MatchCardID, revision model.Revision) {
				calledID = cardID
				calledRevision = revision
			},
		},
		nil,
	)
	callButton := findButton(screen, "Call Selected Face Down")
	if callButton == nil {
		t.Fatal("face-down Call control was not rendered for the active player")
	}
	if !callButton.Disabled() {
		t.Fatal("face-down Call control is enabled before selecting a card")
	}
	handCard := findCardTile(screen, "visible-hand-card")
	if handCard == nil {
		t.Fatal("active player's visible hand card was not rendered")
	}

	test.Tap(handCard)
	if callButton.Disabled() {
		t.Fatal("face-down Call control did not enable after selecting a card")
	}
	test.Tap(callButton)

	if calledID != "player-one-hand" || calledRevision != 12 {
		t.Fatalf("face-down Call submitted card/revision %q/%d; want player-one-hand/12", calledID, calledRevision)
	}
}

func TestFaceUpLevelOneCallOnlyEnablesForEligibleSelectedCaster(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 14
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "servant-in-hand", CardID: "visible-hand-card", ShowFace: true},
		{MatchID: "caster-in-hand", CardID: "level-one-caster", ShowFace: true},
	}
	calledID := model.MatchCardID("")
	calledRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{
			CallFaceDownLevelOne: func(model.MatchCardID, model.Revision) {},
			CallFaceUpLevelOne: func(cardID model.MatchCardID, revision model.Revision) {
				calledID = cardID
				calledRevision = revision
			},
		},
		nil,
	)
	faceDownButton := findButton(screen, "Call Selected Face Down")
	faceUpButton := findButton(screen, "Call Selected Face Up")
	if faceDownButton == nil || faceUpButton == nil {
		t.Fatal("both Level 1 Call controls were not rendered")
	}
	if !faceDownButton.Disabled() || !faceUpButton.Disabled() {
		t.Fatal("Call controls were enabled before selecting a card")
	}

	servant := findCardTile(screen, "visible-hand-card")
	if servant == nil {
		t.Fatal("ineligible hand card was not rendered")
	}
	test.Tap(servant)
	if faceDownButton.Disabled() {
		t.Fatal("face-down Call did not enable for an arbitrary hand card")
	}
	if !faceUpButton.Disabled() {
		t.Fatal("face-up Call enabled for a non-Caster")
	}

	caster := findCardTile(screen, "level-one-caster")
	if caster == nil {
		t.Fatal("eligible Level 1 Caster was not rendered")
	}
	test.Tap(caster)
	if faceUpButton.Disabled() {
		t.Fatal("face-up Call did not enable for a Level 1 Caster")
	}
	test.Tap(faceUpButton)
	if calledID != "caster-in-hand" || calledRevision != 14 {
		t.Fatalf("face-up Call submitted card/revision %q/%d; want caster-in-hand/14", calledID, calledRevision)
	}
}

func TestLevelUpCasterSelectsUpperAndTargetAtCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 21
	match.Turn.Number = 3
	match.Turn.Phase = model.PhaseCall
	match.Players[0].OpeningHandFinalized = true
	match.Players[0].Hand = []simulatorview.CardView{
		{MatchID: "upper-in-hand", CardID: "level-three-aria", ShowFace: true},
	}
	match.Players[0].CasterZone = append(match.Players[0].CasterZone, simulatorview.CardView{
		MatchID: "target-on-field", CardID: "level-two-aria", Face: model.CardFaceUp,
		Orientation: model.OrientationRested, ShowFace: true,
	})
	gotUpper := model.MatchCardID("")
	gotTarget := model.MatchCardID("")
	gotRevision := model.Revision(0)
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{LevelUpCaster: func(upper, target model.MatchCardID, revision model.Revision) {
			gotUpper = upper
			gotTarget = target
			gotRevision = revision
		}},
		nil,
	)
	levelButton := findButton(screen, "Level Up Selected")
	targetSelect := findSelect(screen)
	if levelButton == nil || targetSelect == nil {
		t.Fatal("level-up controls were not rendered")
	}
	if levelButton.Visible() || targetSelect.Visible() {
		t.Fatal("level-up controls appeared before selecting an upper Caster")
	}
	upper := findCardTile(screen, "level-three-aria")
	if upper == nil {
		t.Fatal("upper Caster was not rendered in hand")
	}

	test.Tap(upper)
	if !levelButton.Visible() || !targetSelect.Visible() || len(targetSelect.Options) != 1 {
		t.Fatalf("eligible upper selection produced button/select/options = %t/%t/%v", levelButton.Visible(), targetSelect.Visible(), targetSelect.Options)
	}
	if !levelButton.Disabled() {
		t.Fatal("level-up button enabled before choosing a target")
	}
	targetSelect.SetSelected(targetSelect.Options[0])
	if levelButton.Disabled() {
		t.Fatal("level-up button did not enable after choosing a target")
	}
	test.Tap(levelButton)

	if gotUpper != "upper-in-hand" || gotTarget != "target-on-field" || gotRevision != 21 {
		t.Fatalf("Level Up submitted %q/%q/%d; want upper-in-hand/target-on-field/21", gotUpper, gotTarget, gotRevision)
	}
}

func TestFaceDownLevelOneCallControlOnlyAppearsWhenAvailable(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*simulatorview.MatchView)
	}{
		{
			name: "non-active viewer",
			mutate: func(match *simulatorview.MatchView) {
				match.ViewerID = "player-two"
			},
		},
		{
			name: "wrong phase",
			mutate: func(match *simulatorview.MatchView) {
				match.Turn.Phase = model.PhaseMain
			},
		},
		{
			name: "Call already taken",
			mutate: func(match *simulatorview.MatchView) {
				match.Turn.CallActionTaken = true
			},
		},
		{
			name: "match not in progress",
			mutate: func(match *simulatorview.MatchView) {
				match.MatchStatus = model.StatusSetup
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			match.Turn.Number = 1
			match.Turn.Phase = model.PhaseCall
			match.Players[0].OpeningHandFinalized = true
			testCase.mutate(&match)
			screen := NewBoardScreen(
				match,
				testDefinitions(),
				BoardActions{CallFaceDownLevelOne: func(model.MatchCardID, model.Revision) {}},
				nil,
			)
			if findButton(screen, "Call Selected Face Down") != nil {
				t.Fatal("face-down Call control was rendered when the action was unavailable")
			}
		})
	}
}

func TestBoardControllerUpdatesOnlyViewerHandWhenCallBecomesAvailable(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseRecovery
	match.Players[0].OpeningHandFinalized = true
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{CallFaceDownLevelOne: func(model.MatchCardID, model.Revision) {}},
		nil,
	)
	firstOpponentField := controller.boards.Objects[0]
	firstPlayerField := controller.boards.Objects[1]
	firstPlayerHand := controller.playerBoards[1].hand.Objects[0]
	if findButton(controller.Content(), "Call Selected Face Down") != nil {
		t.Fatal("face-down Call control appeared during Recovery")
	}

	updated := match
	updated.Turn.Phase = model.PhaseCall
	updated.Revision++
	controller.Update(updated)

	if controller.boards.Objects[0] != firstOpponentField {
		t.Fatal("Call availability update rebuilt the opponent field")
	}
	if controller.boards.Objects[1] != firstPlayerField {
		t.Fatal("Call availability update replaced the persistent viewer field")
	}
	if controller.playerBoards[1].hand.Objects[0] == firstPlayerHand {
		t.Fatal("Call availability update did not replace the viewer's Hand zone")
	}
	if findButton(controller.Content(), "Call Selected Face Down") == nil {
		t.Fatal("face-down Call control did not appear upon entering Call phase")
	}
}

func TestPhaseBarEnablesInitialCallOnlyForActiveViewer(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseRecovery
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	callButton := findButton(screen, string(model.PhaseCall))
	if callButton == nil {
		t.Fatal("phase bar does not contain Call")
	}
	if callButton.Disabled() {
		t.Fatal("Call transition is disabled for active player in initial Recovery")
	}
	for _, phase := range []model.Phase{
		model.PhaseRecovery,
		model.PhaseDraw,
		model.PhaseMain,
		model.PhaseBattle,
		model.PhaseEnd,
	} {
		button := findButton(screen, string(phase))
		if button == nil || !button.Disabled() {
			t.Fatalf("phase %q should be displayed but disabled", phase)
		}
	}

	test.Tap(callButton)
	if !completionRequested {
		t.Fatal("phase bar did not request completion of the current phase")
	}
}

func TestPassPriorityButtonRequiresPriorityAndUsesCurrentRevision(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Revision = 23
	match.PriorityHolder = match.ViewerID
	passedRevision := model.Revision(0)
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{PassPriority: func(revision model.Revision) { passedRevision = revision }},
		nil,
	)

	passButton := findButton(controller.Content(), "Pass Priority")
	if passButton == nil || passButton.Disabled() {
		t.Fatal("Pass Priority is not enabled for the priority holder")
	}
	test.Tap(passButton)
	if passedRevision != match.Revision {
		t.Fatalf("Pass Priority submitted revision %d; want %d", passedRevision, match.Revision)
	}

	updated := match
	updated.Revision++
	updated.PriorityHolder = updated.Players[1].ID
	controller.Update(updated)
	if !passButton.Disabled() {
		t.Fatal("Pass Priority remained enabled after priority transferred")
	}
}

func TestCastControlsSubmitSelectedCardPaymentAndOrientation(t *testing.T) {
	tests := []struct {
		name     string
		cardType string
	}{
		{name: "Servant", cardType: "Servant"},
		{name: "Conjure", cardType: "Conjure"},
		{name: "Barrier", cardType: "Barrier"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			match.Revision = 31
			match.Turn = model.TurnState{Number: 2, ActivePlayer: match.ViewerID, Phase: model.PhaseMain}
			match.PriorityHolder = match.ViewerID
			match.Players[0].OpeningHandFinalized = true
			match.Players[0].Aether = model.AetherPool{Aes: 1}
			match.Players[0].Hand = []simulatorview.CardView{{
				MatchID:  "cast-card",
				CardID:   "cast-definition",
				ShowFace: true,
			}}
			definitions := append(testDefinitions(), cards.Card{
				ID:        "cast-definition",
				Name:      "Cast Test",
				Type:      testCase.cardType,
				Element:   "Aes",
				CostLevel: "1",
			})
			var submittedID model.MatchCardID
			var submittedPayment model.AetherPayment
			var submittedOrientation model.CardOrientation
			var submittedRevision model.Revision
			actions := BoardActions{
				CastServant: func(id model.MatchCardID, payment model.AetherPayment, orientation model.CardOrientation, revision model.Revision) {
					submittedID, submittedPayment, submittedOrientation, submittedRevision = id, payment, orientation, revision
				},
				CastConjure: func(id model.MatchCardID, payment model.AetherPayment, revision model.Revision) {
					submittedID, submittedPayment, submittedRevision = id, payment, revision
				},
				CastBarrier: func(id model.MatchCardID, payment model.AetherPayment, revision model.Revision) {
					submittedID, submittedPayment, submittedRevision = id, payment, revision
				},
			}
			controller := NewBoardController(match, definitions, actions, nil)
			castButton := findButton(controller.Content(), "Cast Selected")
			if castButton == nil || !castButton.Disabled() {
				t.Fatal("Cast Selected must begin disabled")
			}
			card := findCardTileByMatchID(controller.Content(), "cast-card")
			if card == nil {
				t.Fatal("castable hand card was not rendered")
			}
			test.Tap(card)
			for _, selection := range findSelects(controller.Content()) {
				if reflect.DeepEqual(selection.Options, []string{"0", "1"}) {
					selection.SetSelected("1")
				}
				if testCase.cardType == "Servant" && reflect.DeepEqual(selection.Options, []string{"Recovered", "Reversed"}) {
					selection.SetSelected("Reversed")
				}
			}
			if castButton.Disabled() {
				t.Fatal("Cast Selected did not enable after a legal Aether allocation")
			}
			test.Tap(castButton)
			if submittedID != "cast-card" || submittedPayment.Aes != 1 || submittedRevision != match.Revision {
				t.Fatalf("cast submitted ID/payment/revision %q/%#v/%d", submittedID, submittedPayment, submittedRevision)
			}
			if testCase.cardType == "Servant" && submittedOrientation != model.OrientationReversed {
				t.Fatalf("Servant orientation = %q; want Reversed", submittedOrientation)
			}
		})
	}
}

func TestCastPaymentControlsRefreshWhenAetherPoolChanges(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn = model.TurnState{Number: 2, ActivePlayer: match.ViewerID, Phase: model.PhaseMain}
	match.PriorityHolder = match.ViewerID
	match.Players[0].OpeningHandFinalized = true
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{CastServant: func(model.MatchCardID, model.AetherPayment, model.CardOrientation, model.Revision) {}},
		nil,
	)
	for _, selection := range findSelects(controller.Content()) {
		if reflect.DeepEqual(selection.Options, []string{"0", "1"}) {
			t.Fatal("payment selector exists before Aether is available")
		}
	}

	updated := match
	updated.Players[0].Aether.Aes = 1
	controller.Update(updated)
	foundPayment := false
	for _, selection := range findSelects(controller.Content()) {
		if reflect.DeepEqual(selection.Options, []string{"0", "1"}) {
			foundPayment = true
		}
	}
	if !foundPayment {
		t.Fatal("payment selectors did not refresh after the Aether pool changed")
	}
}

func TestPhaseBarDisablesTransitionsForNonActiveViewer(t *testing.T) {
	match := testMatchView()
	match.ViewerID = match.Players[1].ID
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseRecovery
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) {}},
		nil,
	)

	callButton := findButton(screen, string(model.PhaseCall))
	if callButton == nil || !callButton.Disabled() {
		t.Fatal("Call transition is enabled for non-active player")
	}
}

func TestPhaseBarEnablesDrawForLaterRecovery(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 2
	match.Turn.Phase = model.PhaseRecovery
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	drawButton := findButton(screen, string(model.PhaseDraw))
	if drawButton == nil || drawButton.Disabled() {
		t.Fatal("Draw transition is not enabled during a later Recovery")
	}
	test.Tap(drawButton)
	if !completionRequested {
		t.Fatal("Draw transition did not request completion of Recovery")
	}
}

func TestPhaseBarEnablesCallDuringLaterDraw(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 2
	match.Turn.Phase = model.PhaseDraw
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	callButton := findButton(screen, string(model.PhaseCall))
	if callButton == nil || callButton.Disabled() {
		t.Fatal("Call transition is not enabled during Draw")
	}
	test.Tap(callButton)
	if !completionRequested {
		t.Fatal("Call transition did not request completion of Draw")
	}
}

func TestPhaseBarEnablesMainDuringCall(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	completionRequested := false
	screen := NewBoardScreen(
		match,
		testDefinitions(),
		BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
		nil,
	)

	mainButton := findButton(screen, string(model.PhaseMain))
	if mainButton == nil || mainButton.Disabled() {
		t.Fatal("Main transition is not enabled during Call")
	}
	test.Tap(mainButton)
	if !completionRequested {
		t.Fatal("Main transition did not request completion of Call")
	}
}

func TestPhaseBarEnablesRemainingSkeletonTransitions(t *testing.T) {
	tests := []struct {
		name    string
		current model.Phase
		target  model.Phase
		label   string
	}{
		{name: "Main to Battle", current: model.PhaseMain, target: model.PhaseBattle, label: "Battle"},
		{name: "Battle to End", current: model.PhaseBattle, target: model.PhaseEnd, label: "End"},
		{name: "End turn", current: model.PhaseEnd, target: model.PhaseEnd, label: "End Turn"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			match := testMatchView()
			match.MatchStatus = model.StatusInProgress
			match.Turn.Number = 1
			match.Turn.Phase = testCase.current
			completionRequested := false
			screen := NewBoardScreen(
				match,
				testDefinitions(),
				BoardActions{CompleteCurrentPhase: func(model.Revision) { completionRequested = true }},
				nil,
			)

			targetButton := findButton(screen, testCase.label)
			if targetButton == nil || targetButton.Disabled() {
				t.Fatalf("%q transition is not enabled during %q", testCase.target, testCase.current)
			}
			test.Tap(targetButton)
			if !completionRequested {
				t.Fatalf("%q transition did not request completion of %q", testCase.target, testCase.current)
			}
		})
	}
}

func TestBoardControllerUpdatesMetadataWithoutRebuildingPlayerFields(t *testing.T) {
	match := testMatchView()
	match.MatchStatus = model.StatusInProgress
	match.Turn.Number = 1
	match.Turn.Phase = model.PhaseCall
	match.Revision = 2
	requestedRevision := model.Revision(0)
	controller := NewBoardController(
		match,
		testDefinitions(),
		BoardActions{
			CompleteCurrentPhase: func(revision model.Revision) {
				requestedRevision = revision
			},
		},
		nil,
	)
	firstOpponentField := controller.boards.Objects[0]
	firstPlayerField := controller.boards.Objects[1]
	firstPlayerDeck := controller.playerBoards[1].deck.Objects[0]

	updated := match
	updated.Turn.Phase = model.PhaseMain
	updated.Revision = 3
	controller.Update(updated)

	if controller.boards.Objects[0] != firstOpponentField ||
		controller.boards.Objects[1] != firstPlayerField {
		t.Fatal("metadata-only update rebuilt player fields")
	}
	if !containsText(
		controller.Content(),
		"Turn 1 • Main • Revision 3 • Active player: player-one • Priority:  • Chase: 0",
	) {
		t.Fatal("metadata-only update did not refresh status")
	}
	battleButton := findButton(controller.Content(), string(model.PhaseBattle))
	if battleButton == nil || battleButton.Disabled() {
		t.Fatal("metadata-only update did not enable Battle")
	}
	test.Tap(battleButton)
	if requestedRevision != 3 {
		t.Fatalf("phase action used revision %d; want updated revision 3", requestedRevision)
	}

	updated.Players[0].DeckCount--
	updated.Revision = 4
	controller.Update(updated)
	if controller.boards.Objects[0] != firstOpponentField {
		t.Fatal("player-zone update rebuilt the unaffected opponent field")
	}
	if controller.boards.Objects[1] != firstPlayerField {
		t.Fatal("player-zone update replaced the persistent player field")
	}
	if controller.playerBoards[1].deck.Objects[0] == firstPlayerDeck {
		t.Fatal("Deck update did not replace the affected Deck zone")
	}
}

func TestDrawProjectionReplacesOnlyDeckAndHandZones(t *testing.T) {
	match := testMatchView()
	controller := NewBoardController(match, testDefinitions(), BoardActions{}, nil)
	playerBoard := controller.playerBoards[1]
	root := playerBoard.root
	deck := playerBoard.deck.Objects[0]
	hand := playerBoard.hand.Objects[0]
	orbs := playerBoard.orbs.Objects[0]
	caster := playerBoard.caster.Objects[0]
	servants := playerBoard.servants.Objects[0]
	barriers := playerBoard.barriers.Objects[0]
	graveyard := playerBoard.graveyard.Objects[0]
	exile := playerBoard.exile.Objects[0]

	updated := match
	updated.Players[0].DeckCount--
	updated.Players[0].Hand = append(
		append([]simulatorview.CardView(nil), match.Players[0].Hand...),
		simulatorview.CardView{MatchID: "drawn", CardID: "visible-hand-card", ShowFace: true},
	)
	updated.Revision++
	controller.Update(updated)

	if controller.playerBoards[1].root != root {
		t.Fatal("draw projection replaced the player board root")
	}
	if playerBoard.deck.Objects[0] == deck || playerBoard.hand.Objects[0] == hand {
		t.Fatal("draw projection did not replace both Deck and Hand")
	}
	unchanged := map[string]struct {
		got  fyne.CanvasObject
		want fyne.CanvasObject
	}{
		"Orbs":      {playerBoard.orbs.Objects[0], orbs},
		"Caster":    {playerBoard.caster.Objects[0], caster},
		"Servants":  {playerBoard.servants.Objects[0], servants},
		"Barriers":  {playerBoard.barriers.Objects[0], barriers},
		"Graveyard": {playerBoard.graveyard.Objects[0], graveyard},
		"Exile":     {playerBoard.exile.Objects[0], exile},
	}
	for name, objects := range unchanged {
		if objects.got != objects.want {
			t.Errorf("draw projection replaced unchanged %s zone", name)
		}
	}
}

func BenchmarkBoardDrawProjectionUpdate(b *testing.B) {
	oldImages := cardimages.DefaultDirectory
	oldThumbnails := cardimages.ThumbnailDirectory
	b.Cleanup(func() { cardimages.ConfigureDirectories(oldImages, oldThumbnails) })
	cardimages.ConfigureDirectories("../../../data/images", "../../../data/thumbnails")

	base := testMatchView()
	base.Players[0].OpeningHandFinalized = true
	base.Players[0].Hand = []simulatorview.CardView{{MatchID: "hand-259", CardID: "259", ShowFace: true}}
	drawn := base
	drawn.Players[0].DeckCount--
	drawn.Players[0].Hand = append(
		append([]simulatorview.CardView(nil), base.Players[0].Hand...),
		simulatorview.CardView{MatchID: "drawn", CardID: "259", ShowFace: true},
	)
	definitions := append(testDefinitions(), cards.Card{ID: "259", Name: "Benchmark Card"})
	controller := NewBoardController(base, definitions, BoardActions{}, nil)

	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if iteration%2 == 0 {
			controller.Update(drawn)
		} else {
			controller.Update(base)
		}
	}
}

func BenchmarkBoardCasterRestProjectionUpdate(b *testing.B) {
	base := testMatchView()
	base.MatchStatus = model.StatusInProgress
	base.Players[0].CasterZone = []simulatorview.CardView{{
		MatchID:     "face-up-caster",
		CardID:      "visible-faceup-caster",
		Face:        model.CardFaceUp,
		Orientation: model.OrientationRecovered,
		ShowFace:    true,
	}}
	rested := base
	rested.Players[0].CasterZone = append([]simulatorview.CardView(nil), base.Players[0].CasterZone...)
	rested.Players[0].CasterZone[0].Orientation = model.OrientationRested
	rested.Players[0].Aether.Aqua = 1
	controller := NewBoardController(
		base,
		testDefinitions(),
		BoardActions{GenerateCasterAether: func(model.MatchCardID, model.Revision) {}},
		nil,
	)

	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		if iteration%2 == 0 {
			controller.Update(rested)
		} else {
			controller.Update(base)
		}
	}
}

func containsText(object fyne.CanvasObject, text string) bool {
	if label, ok := object.(*widget.Label); ok && label.Text == text {
		return true
	}
	if canvasText, ok := object.(*canvas.Text); ok && canvasText.Text == text {
		return true
	}
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return true
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if containsText(child, text) {
				return true
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return containsText(scroll.Content, text)
	}
	return false
}

func containsTextPart(object fyne.CanvasObject, text string) bool {
	if label, ok := object.(*widget.Label); ok && strings.Contains(label.Text, text) {
		return true
	}
	if canvasText, ok := object.(*canvas.Text); ok && strings.Contains(canvasText.Text, text) {
		return true
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if containsTextPart(child, text) {
				return true
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return containsTextPart(scroll.Content, text)
	}
	return false
}

func findScroll(object fyne.CanvasObject) *container.Scroll {
	if scroll, ok := object.(*container.Scroll); ok {
		return scroll
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if scroll := findScroll(child); scroll != nil {
				return scroll
			}
		}
	}
	return nil
}

func findVerticalCardStack(object fyne.CanvasObject) *fyne.Container {
	if fyneContainer, ok := object.(*fyne.Container); ok {
		if _, layered := fyneContainer.Layout.(*verticalCardStackLayout); layered {
			return fyneContainer
		}
		for _, child := range fyneContainer.Objects {
			if result := findVerticalCardStack(child); result != nil {
				return result
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findVerticalCardStack(scroll.Content)
	}
	return nil
}

func collectCardTiles(object fyne.CanvasObject) []*CardTile {
	if tile, ok := object.(*CardTile); ok {
		return []*CardTile{tile}
	}
	result := make([]*CardTile, 0)
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			result = append(result, collectCardTiles(child)...)
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		result = append(result, collectCardTiles(scroll.Content)...)
	}
	return result
}

func findButton(object fyne.CanvasObject, text string) *widget.Button {
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return button
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if button := findButton(child, text); button != nil {
				return button
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findButton(scroll.Content, text)
	}
	return nil
}

func findButtonPosition(
	object fyne.CanvasObject,
	text string,
	origin fyne.Position,
) (fyne.Position, bool) {
	position := fyne.NewPos(origin.X+object.Position().X, origin.Y+object.Position().Y)
	if button, ok := object.(*widget.Button); ok && button.Text == text {
		return position, true
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if result, found := findButtonPosition(child, text, position); found {
				return result, true
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findButtonPosition(scroll.Content, text, position)
	}
	return fyne.Position{}, false
}

func findSelect(object fyne.CanvasObject) *widget.Select {
	if selection, ok := object.(*widget.Select); ok {
		return selection
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if selection := findSelect(child); selection != nil {
				return selection
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findSelect(scroll.Content)
	}
	return nil
}

func findSelects(object fyne.CanvasObject) []*widget.Select {
	result := make([]*widget.Select, 0)
	if selection, ok := object.(*widget.Select); ok {
		result = append(result, selection)
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			result = append(result, findSelects(child)...)
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		result = append(result, findSelects(scroll.Content)...)
	}
	return result
}

func findCardTile(object fyne.CanvasObject, cardID string) *CardTile {
	if tile, ok := object.(*CardTile); ok && tile.Card.ID == cardID {
		return tile
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			if tile := findCardTile(child, cardID); tile != nil {
				return tile
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findCardTile(scroll.Content, cardID)
	}
	return nil
}

func findCardTileByMatchID(object fyne.CanvasObject, matchID model.MatchCardID) *CardTile {
	if tile, ok := object.(*CardTile); ok && tile.View.MatchID == matchID {
		return tile
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if tile := findCardTileByMatchID(child, matchID); tile != nil {
				return tile
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findCardTileByMatchID(scroll.Content, matchID)
	}
	return nil
}

func findConcealedCardTile(object fyne.CanvasObject) *CardTile {
	if tile, ok := object.(*CardTile); ok &&
		!tile.View.ShowFace &&
		tile.View.CardID == "" &&
		tile.View.MatchID == "" {
		return tile
	}
	if fyneContainer, ok := object.(*fyne.Container); ok {
		for _, child := range fyneContainer.Objects {
			if tile := findConcealedCardTile(child); tile != nil {
				return tile
			}
		}
	}
	if scroll, ok := object.(*container.Scroll); ok {
		return findConcealedCardTile(scroll.Content)
	}
	return nil
}

func testDefinitions() []cards.Card {
	return []cards.Card{
		{
			ID:      "1100",
			Name:    "Caster Token",
			Ability: "Produce one Aether.",
		},
		{
			ID:      "visible-hand-card",
			Name:    "Visible Hand Card",
			Type:    "Servant",
			Ability: "Visible ability.",
		},
		{
			ID:        "level-one-caster",
			Name:      "Level One Caster",
			Type:      " CASTER ",
			CostLevel: " 1 ",
			Ability:   "Eligible face-up Call.",
		},
		{
			ID:        "level-two-aria",
			Name:      "Aria",
			Subname:   "Dawn",
			Type:      "Caster",
			CostLevel: "2",
		},
		{
			ID:        "level-three-aria",
			Name:      " aria ",
			Subname:   "Ascendant",
			Type:      " CASTER ",
			CostLevel: " 3 ",
		},
		{
			ID:      "visible-servant-card",
			Name:    "Visible Servant",
			Type:    "Servant",
			Ability: "Servant ability.",
		},
		{
			ID:      "visible-grave-card",
			Name:    "Visible Graveyard Card",
			Type:    "Spell",
			Ability: "Resolved ability.",
		},
		{
			ID:      "visible-exile-card",
			Name:    "Visible Exiled Card",
			Type:    "Spell",
			Ability: "Exiled ability.",
		},
		{
			ID:      "face-down-caster-card",
			Name:    "Known Face-down Caster",
			Type:    "Caster",
			Ability: "Caster ability.",
		},
		{
			ID:        "visible-faceup-caster",
			Name:      "Visible Face-up Caster",
			Type:      "Caster",
			Element:   "Aqua",
			CostLevel: "2",
			Ability:   "Face-up Caster ability.",
		},
	}
}

func testMatchView() simulatorview.MatchView {
	return simulatorview.MatchView{
		ViewerID:    "player-one",
		MatchStatus: model.StatusSetup,
		Turn: model.TurnState{
			ActivePlayer: "player-one",
			Phase:        model.PhaseCall,
		},
		Players: [2]simulatorview.PlayerView{
			{
				ID:        "player-one",
				DeckCount: 36,
				Hand: []simulatorview.CardView{
					{
						MatchID:  "player-one-hand",
						CardID:   "visible-hand-card",
						ShowFace: true,
					},
				},
				Orbs: []simulatorview.CardView{
					{Face: model.CardFaceDown},
				},
				CasterZone: []simulatorview.CardView{
					{
						MatchID:     "player-one-token",
						CardID:      model.CasterTokenCardID,
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
					{
						MatchID:     "player-one-facedown-caster",
						CardID:      "face-down-caster-card",
						Face:        model.CardFaceDown,
						Orientation: model.OrientationRecovered,
						ShowFace:    false,
					},
				},
				ServantZone: []simulatorview.CardView{
					{
						MatchID:     "player-one-servant",
						CardID:      "visible-servant-card",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
				Graveyard: []simulatorview.CardView{
					{
						MatchID:     "player-one-grave",
						CardID:      "visible-grave-card",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
				Exile: []simulatorview.CardView{
					{
						MatchID:     "player-one-exile",
						CardID:      "visible-exile-card",
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
				},
			},
			{
				ID:        "player-two",
				DeckCount: 36,
				Hand: []simulatorview.CardView{
					{Face: model.CardFaceDown},
				},
				Orbs: []simulatorview.CardView{
					{Face: model.CardFaceDown},
				},
				CasterZone: []simulatorview.CardView{
					{
						MatchID:     "player-two-token",
						CardID:      model.CasterTokenCardID,
						Face:        model.CardFaceUp,
						Orientation: model.OrientationRecovered,
						ShowFace:    true,
					},
					{
						Face:        model.CardFaceDown,
						Orientation: model.OrientationRecovered,
						ShowFace:    false,
					},
				},
			},
		},
	}
}
