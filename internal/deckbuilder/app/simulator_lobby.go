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
	// priorityModeEmpty auto-passes empty priority stacks and stops when a Chase
	// starts. Main/Battle still pause for the turn player so they can act.
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
	joined, err := active.JoinRoom(ctx, roomCode, playerName, deck, password)
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
		labelEmpty = "Auto-pass empty priority (stop when Chase starts)"
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
		widget.NewLabel("Empty-priority mode keeps Main/Battle for the turn player so they can cast or attack. Chase links always require a manual pass."),
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

// shouldAutoPassPriorityForView decides whether the board should auto-pass for
// this viewer-safe projection. Default is empty-stack auto-pass until a Chase
// starts; Main/Battle still pause for the turn player.
func shouldAutoPassPriorityForView(match simulatorview.MatchView) bool {
	if match.Spectator ||
		match.MatchStatus != model.StatusInProgress ||
		!match.PrioritySequenceOpen ||
		match.PriorityHolder != match.ViewerID ||
		match.ChaseLinkCount > 0 ||
		match.Attack.Step != model.BattleStepIdle ||
		match.PendingDraw.Step != "" ||
		match.PendingBreak.PlayerID != "" {
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
		// Empty-chase default: leave Main/Battle with the turn player.
		if match.Turn.ActivePlayer == match.ViewerID &&
			(match.Turn.Phase == model.PhaseMain || match.Turn.Phase == model.PhaseBattle) {
			return false
		}
		return true
	}
}

func phaseIsConfiguredStop(matchPhase, stopsCSV string) bool {
	for _, phase := range strings.Split(stopsCSV, ",") {
		if strings.EqualFold(strings.TrimSpace(phase), matchPhase) {
			return true
		}
	}
	return false
}
