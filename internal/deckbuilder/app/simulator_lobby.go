package deckbuilder

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/netclient"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const (
	networkPriorityModePreferenceKey = "network.priority_mode"
	networkStopPhasesPreferenceKey   = "network.stop_phases"
	priorityModeFull                 = "full"
	priorityModeStops                = "stops"
	// priorityModeEmpty is Arena-style: auto-pass empty priority and Chase
	// priority when the viewer has no non-main-timing response (Quickcast or
	// non-aether activated abilities). Main/Battle still pause for the turn
	// player. Aether abilities alone do not stop (CR 9.7 / Arena mana analogue).
	priorityModeEmpty = "empty"
)

func showOnlineLobby(
	window fyne.Window,
	libraryDirectory string,
	repository *cards.Repository,
	onBackToMenu func(),
) {
	prefs := fyne.CurrentApp().Preferences()
	serverEntry := widget.NewEntry()
	serverEntry.SetText(prefs.StringWithFallback(networkServerPreferenceKey, defaultNetworkServerURL))
	nameEntry := widget.NewEntry()
	nameEntry.SetText(prefs.StringWithFallback(networkPlayerNamePreferenceKey, "Player"))

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search by room name, creator, or code")

	roomList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return widget.NewLabel("room")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	var (
		client       *netclient.Client
		allRooms     []protocol.RoomSummary
		visibleRooms []protocol.RoomSummary
	)

	refreshLabel := widget.NewLabel("Connect to a server, then refresh the open-room list.")
	refreshLabel.Wrapping = fyne.TextWrapWord

	applyFilter := func() {
		query := strings.ToLower(strings.TrimSpace(searchEntry.Text))
		visibleRooms = visibleRooms[:0]
		for _, room := range allRooms {
			if query == "" || roomMatchesSearch(room, query) {
				visibleRooms = append(visibleRooms, room)
			}
		}
		roomList.Length = func() int { return len(visibleRooms) }
		roomList.UpdateItem = func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(visibleRooms) {
				label.SetText("")
				return
			}
			room := visibleRooms[id]
			lock := ""
			if room.HasPassword {
				lock = " [password]"
			}
			state := "waiting"
			if room.MatchStarted {
				state = "in progress"
			}
			name := room.RoomName
			if strings.TrimSpace(name) == "" {
				name = room.RoomCode
			}
			label.SetText(fmt.Sprintf(
				"%s%s — created by %s — code %s — %d/2 players, %d spectators — %s",
				name, lock, room.HostName, room.RoomCode, room.PlayerCount, room.SpectatorCount, state,
			))
		}
		roomList.Refresh()
		switch {
		case len(allRooms) == 0:
			refreshLabel.SetText("No open rooms on this server.")
		case len(visibleRooms) == 0:
			refreshLabel.SetText(fmt.Sprintf("No rooms match %q (%d total).", searchEntry.Text, len(allRooms)))
		case query != "":
			refreshLabel.SetText(fmt.Sprintf("Showing %d of %d room(s).", len(visibleRooms), len(allRooms)))
		default:
			refreshLabel.SetText(fmt.Sprintf("%d open room(s).", len(allRooms)))
		}
	}

	setRooms := func(next []protocol.RoomSummary) {
		allRooms = next
		applyFilter()
	}
	searchEntry.OnChanged = func(string) { applyFilter() }

	ensureClient := func() (*netclient.Client, string, string, error) {
		serverURL := normalizeWebSocketURL(serverEntry.Text)
		playerName := strings.TrimSpace(nameEntry.Text)
		if serverURL == "" {
			return nil, "", "", fmt.Errorf("server URL is required")
		}
		if playerName == "" {
			return nil, "", "", fmt.Errorf("display name is required")
		}
		prefs.SetString(networkServerPreferenceKey, serverURL)
		prefs.SetString(networkPlayerNamePreferenceKey, playerName)
		if client != nil {
			return client, serverURL, playerName, nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
		defer cancel()
		dialed, err := netclient.Dial(ctx, serverURL)
		if err != nil {
			return nil, "", "", fmt.Errorf("connect: %w", err)
		}
		if err := dialed.Hello(ctx, playerName); err != nil {
			_ = dialed.Close()
			return nil, "", "", fmt.Errorf("hello: %w", err)
		}
		client = dialed
		return client, serverURL, playerName, nil
	}

	refreshRooms := func() {
		go func() {
			active, _, _, err := ensureClient()
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, window) })
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
			defer cancel()
			listed, err := active.ListRooms(ctx)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				setRooms(listed)
			})
		}()
	}

	promptPassword := func(hasPassword bool, done func(password string)) {
		if !hasPassword {
			done("")
			return
		}
		entry := widget.NewPasswordEntry()
		showScaledForm("Room Password", "Continue", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Password", entry),
		}, func(ok bool) {
			if ok {
				done(entry.Text)
			}
		}, window, 0.4, 0.3, 420, 220, 720, 420)
	}

	var selected *protocol.RoomSummary
	roomList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(visibleRooms) {
			selected = nil
			return
		}
		room := visibleRooms[id]
		selected = &room
	}

	joinSelected := func(asSpectator bool) {
		if selected == nil {
			dialog.ShowError(fmt.Errorf("select a room first"), window)
			return
		}
		room := *selected
		promptPassword(room.HasPassword, func(password string) {
			if asSpectator {
				go joinLobbyAsSpectator(window, repository, client, room.RoomCode, strings.TrimSpace(nameEntry.Text), password, onBackToMenu, ensureClient)
				return
			}
			if room.MatchStarted || room.OpenSeats <= 0 {
				dialog.ShowError(fmt.Errorf("that room has no open player seats; spectate instead"), window)
				return
			}
			showSimulatorSingleDeckSelection(
				window,
				libraryDirectory,
				repository,
				"Join Online Game — Choose Deck",
				"Join Seat",
				func(deck decks.Deck) {
					go joinLobbyAsPlayer(window, repository, room.RoomCode, strings.TrimSpace(nameEntry.Text), password, deck, onBackToMenu, ensureClient)
				},
			)
		})
	}

	hostButton := widget.NewButton("Host New Room", func() {
		if client != nil {
			_ = client.Close()
			client = nil
		}
		showHostOnlineGame(window, libraryDirectory, repository, onBackToMenu)
	})
	joinButton := widget.NewButton("Join as Player", func() { joinSelected(false) })
	spectateButton := widget.NewButton("Spectate", func() { joinSelected(true) })
	refreshButton := widget.NewButton("Refresh", refreshRooms)
	codeButton := widget.NewButton("Join by Code…", func() {
		showJoinOnlineGame(window, libraryDirectory, repository, onBackToMenu)
	})
	priorityButton := widget.NewButton("Priority Preferences…", func() {
		showPriorityPreferencesDialog(window)
	})

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Online Lobby", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			widget.NewForm(
				widget.NewFormItem("Server URL", serverEntry),
				widget.NewFormItem("Display Name", nameEntry),
				widget.NewFormItem("Search", searchEntry),
			),
			container.NewHBox(refreshButton, hostButton, joinButton, spectateButton, codeButton, priorityButton),
			refreshLabel,
		),
		widget.NewButton("Back to Main Menu", func() {
			if client != nil {
				_ = client.Close()
				client = nil
			}
			onBackToMenu()
		}),
		nil,
		nil,
		roomList,
	)
	setWindowContent(window, content)
	window.SetTitle(applicationName + " — Online Lobby")
}

func joinLobbyAsPlayer(
	window fyne.Window,
	repository *cards.Repository,
	roomCode, playerName, password string,
	deck decks.Deck,
	onBackToMenu func(),
	ensureClient func() (*netclient.Client, string, string, error),
) {
	active, _, _, err := ensureClient()
	if err != nil {
		fyne.Do(func() { dialog.ShowError(err, window) })
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
	defer cancel()
	joined, err := active.JoinRoom(ctx, roomCode, playerName, deck, password, loadSimulatorCosmetics())
	if err != nil {
		fyne.Do(func() { dialog.ShowError(fmt.Errorf("join room: %w", err), window) })
		return
	}
	matchView, err := active.Command(ctx, joined.PlayerID, 0, "request_view", nil)
	if err != nil {
		fyne.Do(func() { dialog.ShowError(fmt.Errorf("load match: %w", err), window) })
		return
	}
	fyne.Do(func() {
		openNetworkBoard(window, repository, active, joined.PlayerID, joined.RoomCode, matchView, false, onBackToMenu)
	})
}

func joinLobbyAsSpectator(
	window fyne.Window,
	repository *cards.Repository,
	existing *netclient.Client,
	roomCode, playerName, password string,
	onBackToMenu func(),
	ensureClient func() (*netclient.Client, string, string, error),
) {
	active := existing
	var err error
	if active == nil {
		active, _, _, err = ensureClient()
		if err != nil {
			fyne.Do(func() { dialog.ShowError(err, window) })
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
	defer cancel()
	joined, err := active.JoinAsSpectator(ctx, roomCode, playerName, password)
	if err != nil {
		fyne.Do(func() { dialog.ShowError(fmt.Errorf("spectate: %w", err), window) })
		return
	}
	matchView, err := active.Command(ctx, joined.PlayerID, 0, "request_view", nil)
	if err != nil {
		// Waiting room: match not started yet.
		fyne.Do(func() {
			showNetworkWaitingRoom(
				window,
				repository,
				active,
				joined.PlayerID,
				joined.RoomCode,
				joined.RoomCode,
				playerName,
				true,
				onBackToMenu,
			)
		})
		return
	}
	fyne.Do(func() {
		openNetworkBoard(window, repository, active, joined.PlayerID, joined.RoomCode, matchView, true, onBackToMenu)
	})
}

func showPriorityPreferencesDialog(window fyne.Window) {
	prefs := fyne.CurrentApp().Preferences()
	mode := prefs.StringWithFallback(networkPriorityModePreferenceKey, priorityModeEmpty)
	stops := prefs.StringWithFallback(networkStopPhasesPreferenceKey, "Main,Battle")

	const (
		labelEmpty = "Arena-style: auto-pass unless you can act or must decide"
		labelStops = "Auto-pass except selected stop phases"
		labelFull  = "Full control (stop every priority)"
	)
	modeSelect := widget.NewRadioGroup([]string{labelEmpty, labelStops, labelFull}, nil)
	switch mode {
	case priorityModeStops:
		modeSelect.SetSelected(labelStops)
	case priorityModeFull:
		modeSelect.SetSelected(labelFull)
	default:
		modeSelect.SetSelected(labelEmpty)
	}

	recovery := widget.NewCheck("Recovery", nil)
	draw := widget.NewCheck("Draw", nil)
	call := widget.NewCheck("Call", nil)
	main := widget.NewCheck("Main", nil)
	battle := widget.NewCheck("Battle", nil)
	end := widget.NewCheck("End", nil)
	checks := map[string]*widget.Check{
		"Recovery": recovery,
		"Draw":     draw,
		"Call":     call,
		"Main":     main,
		"Battle":   battle,
		"End":      end,
	}
	for _, phase := range strings.Split(stops, ",") {
		phase = strings.TrimSpace(phase)
		if check, ok := checks[phase]; ok {
			check.SetChecked(true)
		}
	}

	form := container.NewVBox(
		widget.NewLabel("Arena-style priority control"),
		modeSelect,
		widget.NewLabel("Arena-style keeps Main for the turn player, auto-skips empty Battle (no attack and no ability with a legal target), auto-passes empty stacks, and only stops on Chase when you may have a non-main-timing response (Quickcast, field activated abilities with legal targets, or a pending decision). Aether generation alone does not stop."),
		widget.NewLabel("Stop phases (used when “except selected stop phases” is enabled):"),
		recovery, draw, call, main, battle, end,
	)
	showScaledCustomConfirm("Priority Preferences", "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		switch modeSelect.Selected {
		case labelStops:
			prefs.SetString(networkPriorityModePreferenceKey, priorityModeStops)
		case labelFull:
			prefs.SetString(networkPriorityModePreferenceKey, priorityModeFull)
		default:
			prefs.SetString(networkPriorityModePreferenceKey, priorityModeEmpty)
		}
		selected := make([]string, 0, 6)
		for name, check := range checks {
			if check.Checked {
				selected = append(selected, name)
			}
		}
		prefs.SetString(networkStopPhasesPreferenceKey, strings.Join(selected, ","))
	}, window, 0.45, 0.6, 480, 420, 900, 780)
}

func roomMatchesSearch(room protocol.RoomSummary, query string) bool {
	fields := []string{room.RoomName, room.HostName, room.RoomCode}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

// makeShouldAutoPassPriority returns Arena-style auto-pass using printed
// ability text to detect CR-legal non-main-timing responses during Chase.
func makeShouldAutoPassPriority(definitions []cards.Card) func(simulatorview.MatchView) bool {
	abilityByID := make(map[model.CardID]string, len(definitions))
	for _, definition := range definitions {
		abilityByID[model.CardID(definition.ID)] = definition.Ability
	}
	return func(match simulatorview.MatchView) bool {
		return shouldAutoPassPriority(match, func(id model.CardID) string {
			return abilityByID[id]
		})
	}
}

// shouldAutoPassPriorityForView is the catalog-free default used when
// definitions are unavailable. Chase with unknown abilities auto-passes.
func shouldAutoPassPriorityForView(match simulatorview.MatchView) bool {
	return shouldAutoPassPriority(match, nil)
}

// shouldAutoPassPriority decides whether the board should auto-pass for this
// viewer-safe projection. Default Arena-style empty mode auto-passes empty
// stacks and Chase priority when the viewer has no CR-legal non-main-timing
// response; Main still pauses for the turn player. Empty Battle (no legal
// attack and no activated ability with a legal target) auto-skips.
//
// Authority: CR 1.2 §§6.4, 7.1.2, 7.4, 9.7, 11.5 — outside main timing a player
// with priority may pass, play activated abilities, or play Quickcast cards.
// Servant/Conjure/Barrier plays without Quickcast require main timing.
func shouldAutoPassPriority(
	match simulatorview.MatchView,
	abilityOf func(model.CardID) string,
) bool {
	if match.Spectator ||
		match.MatchStatus != model.StatusInProgress ||
		!match.PrioritySequenceOpen ||
		match.PriorityHolder != match.ViewerID ||
		match.Attack.Step != model.BattleStepIdle ||
		match.PendingDraw.Step != "" ||
		match.PendingBreak.PlayerID != "" ||
		match.PendingChoice.PlayerID != "" {
		return false
	}
	prefs := fyne.CurrentApp().Preferences()
	mode := prefs.StringWithFallback(networkPriorityModePreferenceKey, priorityModeEmpty)
	switch mode {
	case priorityModeFull:
		return false
	case priorityModeStops:
		return !phaseIsConfiguredStop(string(match.Turn.Phase), prefs.StringWithFallback(networkStopPhasesPreferenceKey, "Main,Battle"))
	default:
		if match.ChaseLinkCount > 0 {
			// Stop only when the viewer may have a non-main-timing response.
			return !viewerHasNonMainTimingResponse(match, abilityOf)
		}
		// Empty Chase: leave Main with the turn player.
		if match.Turn.ActivePlayer == match.ViewerID && match.Turn.Phase == model.PhaseMain {
			return false
		}
		// Empty Battle: auto-pass when nothing legal can be done (no attackers,
		// and Rest/activated lines that need missing targets do not count).
		if match.Turn.ActivePlayer == match.ViewerID && match.Turn.Phase == model.PhaseBattle {
			return !viewerHasUsableBattleAction(match, abilityOf)
		}
		return true
	}
}

// viewerHasNonMainTimingResponse reports whether the viewer may have a legal
// action during Chase / non-main-timing priority (CR 6.4.1 + Quickcast).
// Aether-only abilities are ignored so generated Aether does not force a stop.
// Activated lines that need a missing target (e.g. Lumielle with no enemy
// Servant, Glory Butterfly with no allied Servant) do not count.
func viewerHasNonMainTimingResponse(
	match simulatorview.MatchView,
	abilityOf func(model.CardID) string,
) bool {
	if abilityOf == nil {
		return false
	}
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			continue
		}
		for _, card := range player.Hand {
			if card.CardID == "" {
				continue
			}
			if cardDeclaresQuickcastAbility(abilityOf(card.CardID)) {
				return true
			}
		}
		for _, zone := range [][]simulatorview.CardView{player.CasterZone, player.ServantZone} {
			for _, card := range zone {
				if card.CardID == "" || !card.ShowFace {
					continue
				}
				if cardHasUsableActivatedAbility(abilityOf(card.CardID), card.Orientation, match) {
					return true
				}
			}
		}
	}
	return false
}

// viewerHasUsableBattleAction is true when the turn player can declare an
// attack or activate a field ability that still has a legal target.
func viewerHasUsableBattleAction(
	match simulatorview.MatchView,
	abilityOf func(model.CardID) string,
) bool {
	if viewerHasRecoveredAttacker(match) {
		return true
	}
	if abilityOf == nil {
		return false
	}
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			continue
		}
		for _, zone := range [][]simulatorview.CardView{player.CasterZone, player.ServantZone} {
			for _, card := range zone {
				if card.CardID == "" || !card.ShowFace {
					continue
				}
				if cardHasUsableActivatedAbility(abilityOf(card.CardID), card.Orientation, match) {
					return true
				}
			}
		}
	}
	return false
}

func viewerHasRecoveredAttacker(match simulatorview.MatchView) bool {
	for _, player := range match.Players {
		if player.ID != match.ViewerID {
			continue
		}
		for _, card := range player.ServantZone {
			if card.MatchID != "" && card.Orientation == model.OrientationRecovered {
				return true
			}
		}
	}
	return false
}

// cardHasUsableActivatedAbility is like cardHasNonMainTimingActivatedAbility
// but also skips Rest lines whose printed targets cannot exist right now.
func cardHasUsableActivatedAbility(
	ability string,
	orientation model.CardOrientation,
	match simulatorview.MatchView,
) bool {
	for _, line := range strings.Split(ability, "\n") {
		trimmed := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "•*-"))
		if trimmed == "" {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon <= 0 {
			continue
		}
		cost := strings.TrimSpace(trimmed[:colon])
		effect := strings.TrimSpace(trimmed[colon+1:])
		if cost == "" || effect == "" {
			continue
		}
		if label, ok := abilityKeywordLabelForAutoPass(line); ok {
			if strings.EqualFold(label, "Quickcast") ||
				strings.EqualFold(label, "Break") ||
				strings.EqualFold(label, "Enter") ||
				strings.EqualFold(label, "Last Words") ||
				strings.EqualFold(label, "Unity") ||
				strings.EqualFold(label, "United Front") ||
				strings.EqualFold(label, "Slow Start") ||
				strings.EqualFold(label, "Hubris") ||
				strings.EqualFold(label, "Double Corrupt") ||
				strings.EqualFold(label, "Triple Corrupt") {
				continue
			}
		}
		lowerEffect := strings.ToLower(effect)
		if strings.Contains(lowerEffect, "only during main timing") ||
			strings.Contains(lowerEffect, "only during your main") {
			continue
		}
		if isAetherAbilityEffect(effect) {
			continue
		}
		lowerCost := strings.ToLower(cost)
		requiresRest := strings.Contains(lowerCost, "rest")
		if requiresRest && orientation != model.OrientationRecovered {
			continue
		}
		if !(requiresRest ||
			strings.Contains(lowerCost, "banish") ||
			strings.Contains(lowerCost, "discard") ||
			strings.Contains(lowerCost, "return") ||
			strings.Contains(lowerCost, "pay")) {
			continue
		}
		if abilityEffectLacksLegalTarget(effect, match) {
			continue
		}
		return true
	}
	return false
}

func abilityEffectLacksLegalTarget(effect string, match simulatorview.MatchView) bool {
	lower := strings.ToLower(effect)
	alliedServants := 0
	enemyServants := 0
	for _, player := range match.Players {
		count := 0
		for _, card := range player.ServantZone {
			if card.MatchID != "" {
				count++
			}
		}
		if player.ID == match.ViewerID {
			alliedServants = count
		} else {
			enemyServants = count
		}
	}
	needsEnemy := strings.Contains(lower, "enemy servant") ||
		strings.Contains(lower, "target enemy") ||
		(strings.Contains(lower, "opponent") && strings.Contains(lower, "servant"))
	needsAllied := strings.Contains(lower, "allied servant") ||
		strings.Contains(lower, "target allied") ||
		strings.Contains(lower, "allied {") ||
		strings.Contains(lower, "an allied servant")
	// Glory Butterfly style: "Target allied servant gains…"
	if !needsAllied && strings.Contains(lower, "target") && strings.Contains(lower, "servant") &&
		!needsEnemy && (strings.Contains(lower, "gains") || strings.Contains(lower, "gain ")) {
		// Ambiguous "target servant" buffs usually mean allied when Rest on own caster.
		if !strings.Contains(lower, "enemy") {
			needsAllied = true
		}
	}
	if needsEnemy && enemyServants == 0 {
		return true
	}
	if needsAllied && alliedServants == 0 {
		return true
	}
	return false
}

func cardDeclaresQuickcastAbility(ability string) bool {
	for _, line := range strings.Split(ability, "\n") {
		label, ok := abilityKeywordLabelForAutoPass(line)
		if ok && strings.EqualFold(label, "Quickcast") {
			return true
		}
	}
	return false
}

func abilityKeywordLabelForAutoPass(line string) (string, bool) {
	trimmed := strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "•*-"))
	if trimmed == "" {
		return "", false
	}
	if strings.HasPrefix(trimmed, "[") {
		if closing := strings.IndexRune(trimmed, ']'); closing > 1 {
			return strings.TrimSpace(trimmed[1:closing]), true
		}
	}
	if separator := strings.IndexAny(trimmed, "(:,→"); separator >= 0 {
		return strings.TrimSpace(trimmed[:separator]), true
	}
	if strings.ContainsAny(trimmed, ".!?;") {
		return "", false
	}
	return strings.TrimSpace(trimmed), true
}

// cardHasNonMainTimingActivatedAbility detects printable activated abilities
// that remain legal outside main timing (CR 6.4.1.a / 7.4). Prefer
// cardHasUsableActivatedAbility when a MatchView is available so missing
// targets (Lumielle / Glory Butterfly) do not falsely stop priority.
func cardHasNonMainTimingActivatedAbility(ability string, orientation model.CardOrientation) bool {
	return cardHasUsableActivatedAbility(ability, orientation, simulatorview.MatchView{})
}

func isAetherAbilityEffect(effect string) bool {
	lower := strings.ToLower(effect)
	return strings.Contains(lower, "produce") && strings.Contains(lower, "aether")
}

func phaseIsConfiguredStop(matchPhase, stopsCSV string) bool {
	for _, phase := range strings.Split(stopsCSV, ",") {
		if strings.EqualFold(strings.TrimSpace(phase), matchPhase) {
			return true
		}
	}
	return false
}
