package deckbuilder

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	cards "github.com/HybridUofA/casters-compendium/internal/carddata/catalog"
	"github.com/HybridUofA/casters-compendium/internal/game/decks"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/netclient"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	simulatorui "github.com/HybridUofA/casters-compendium/internal/simulator/ui"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const (
	networkServerPreferenceKey     = "network.server_url"
	networkPlayerNamePreferenceKey = "network.player_name"
	defaultNetworkServerURL        = "ws://3.86.186.199:7474/"
	networkCommandTimeout          = 15 * time.Second
)

func showHostOnlineGame(
	window fyne.Window,
	libraryDirectory string,
	repository *cards.Repository,
	onBackToMenu func(),
) {
	showSimulatorSingleDeckSelection(
		window,
		libraryDirectory,
		repository,
		"Host Online Game — Choose Deck",
		"Continue",
		func(deck decks.Deck) {
			showNetworkConnectForm(window, "Host Online Game", false, true, true, func(serverURL, playerName, _, password, roomName string) {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
					defer cancel()

					client, err := netclient.Dial(ctx, serverURL)
					if err != nil {
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("connect: %w", err), window) })
						return
					}
					if err := client.Hello(ctx, playerName); err != nil {
						_ = client.Close()
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("hello: %w", err), window) })
						return
					}
					created, err := client.CreateRoom(ctx, playerName, roomName, deck, password)
					if err != nil {
						_ = client.Close()
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("create room: %w", err), window) })
						return
					}

					fyne.Do(func() {
						showNetworkWaitingRoom(
							window,
							repository,
							client,
							created.PlayerID,
							created.RoomCode,
							created.RoomName,
							playerName,
							false,
							onBackToMenu,
						)
					})
				}()
			})
		},
	)
}

func showJoinOnlineGame(
	window fyne.Window,
	libraryDirectory string,
	repository *cards.Repository,
	onBackToMenu func(),
) {
	showSimulatorSingleDeckSelection(
		window,
		libraryDirectory,
		repository,
		"Join Online Game — Choose Deck",
		"Continue",
		func(deck decks.Deck) {
			showNetworkConnectForm(window, "Join Online Game", true, true, false, func(serverURL, playerName, roomCode, password, _ string) {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
					defer cancel()

					client, err := netclient.Dial(ctx, serverURL)
					if err != nil {
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("connect: %w", err), window) })
						return
					}
					if err := client.Hello(ctx, playerName); err != nil {
						_ = client.Close()
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("hello: %w", err), window) })
						return
					}
					joined, err := client.JoinRoom(ctx, roomCode, playerName, deck, password)
					if err != nil {
						_ = client.Close()
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("join room: %w", err), window) })
						return
					}
					matchView, err := client.Command(ctx, joined.PlayerID, 0, "request_view", nil)
					if err != nil {
						_ = client.Close()
						fyne.Do(func() { dialog.ShowError(fmt.Errorf("load match: %w", err), window) })
						return
					}

					fyne.Do(func() {
						openNetworkBoard(
							window,
							repository,
							client,
							joined.PlayerID,
							joined.RoomCode,
							matchView,
							false,
							onBackToMenu,
						)
					})
				}()
			})
		},
	)
}

func showNetworkConnectForm(
	window fyne.Window,
	title string,
	requireRoomCode bool,
	allowPassword bool,
	allowRoomName bool,
	onSubmit func(serverURL, playerName, roomCode, password, roomName string),
) {
	prefs := fyne.CurrentApp().Preferences()
	serverEntry := widget.NewEntry()
	serverEntry.SetText(prefs.StringWithFallback(networkServerPreferenceKey, defaultNetworkServerURL))
	nameEntry := widget.NewEntry()
	nameEntry.SetText(prefs.StringWithFallback(networkPlayerNamePreferenceKey, "Player"))
	roomEntry := widget.NewEntry()
	roomEntry.SetPlaceHolder("ABCD")
	roomNameEntry := widget.NewEntry()
	roomNameEntry.SetPlaceHolder("e.g. Friday Night Casters")
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("optional")

	items := []*widget.FormItem{
		widget.NewFormItem("Server URL", serverEntry),
		widget.NewFormItem("Display Name", nameEntry),
	}
	if allowRoomName {
		items = append(items, widget.NewFormItem("Room Name", roomNameEntry))
	}
	if requireRoomCode {
		items = append(items, widget.NewFormItem("Room Code", roomEntry))
	}
	if allowPassword {
		items = append(items, widget.NewFormItem("Password", passwordEntry))
	}

	showScaledForm(title, "Connect", "Cancel", items, func(confirmed bool) {
		if !confirmed {
			return
		}
		serverURL := normalizeWebSocketURL(serverEntry.Text)
		playerName := strings.TrimSpace(nameEntry.Text)
		roomCode := strings.ToUpper(strings.TrimSpace(roomEntry.Text))
		if serverURL == "" {
			dialog.ShowError(fmt.Errorf("server URL is required"), window)
			return
		}
		if playerName == "" {
			dialog.ShowError(fmt.Errorf("display name is required"), window)
			return
		}
		if requireRoomCode && roomCode == "" {
			dialog.ShowError(fmt.Errorf("room code is required"), window)
			return
		}
		prefs.SetString(networkServerPreferenceKey, serverURL)
		prefs.SetString(networkPlayerNamePreferenceKey, playerName)
		onSubmit(serverURL, playerName, roomCode, passwordEntry.Text, strings.TrimSpace(roomNameEntry.Text))
	}, window, 0.5, 0.55, 560, 360, 960, 720)
}

func showNetworkWaitingRoom(
	window fyne.Window,
	repository *cards.Repository,
	client *netclient.Client,
	playerID string,
	roomCode string,
	roomName string,
	playerName string,
	spectator bool,
	onBackToMenu func(),
) {
	if strings.TrimSpace(roomName) == "" {
		roomName = roomCode
	}
	waitText := fmt.Sprintf(
		"%s\nCode %s\nWaiting for opponent to join…\n\nShare the room code with your opponent.",
		roomName,
		roomCode,
	)
	title := "Hosting Online Match"
	if spectator {
		waitText = fmt.Sprintf("%s\nCode %s\nWaiting for the match to start…", roomName, roomCode)
		title = "Spectating Online Match"
	}
	status := widget.NewLabel(waitText)
	status.Alignment = fyne.TextAlignCenter
	status.Wrapping = fyne.TextWrapWord

	var (
		openMu sync.Mutex
		opened bool
		stop   = make(chan struct{})
	)
	openOnce := func(matchView simulatorview.MatchView) {
		openMu.Lock()
		defer openMu.Unlock()
		if opened {
			return
		}
		opened = true
		close(stop)
		openNetworkBoard(window, repository, client, playerID, roomCode, matchView, spectator, onBackToMenu)
	}

	setWindowContent(window, container.NewCenter(container.NewVBox(
		widget.NewLabelWithStyle(title, fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		status,
		widget.NewLabel("You: "+playerName),
		widget.NewButton("Cancel", func() {
			openMu.Lock()
			if !opened {
				opened = true
				close(stop)
			}
			openMu.Unlock()
			_ = client.Close()
			onBackToMenu()
		}),
	)))
	window.SetTitle(applicationName + " — " + roomName)

	client.OnView = func(matchView simulatorview.MatchView) {
		fyne.Do(func() { openOnce(matchView) })
	}
	client.OnError = func(payload protocol.ErrorPayload) {
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf("%s: %s", payload.Code, payload.Message), window)
		})
	}

	// Poll in case the match-start view arrived before OnView was attached.
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
				matchView, err := client.Command(ctx, playerID, 0, "request_view", nil)
				cancel()
				if err != nil {
					continue
				}
				fyne.Do(func() { openOnce(matchView) })
				return
			}
		}
	}()
}

func openNetworkBoard(
	window fyne.Window,
	repository *cards.Repository,
	client *netclient.Client,
	playerID string,
	roomCode string,
	initialView simulatorview.MatchView,
	spectator bool,
	onBackToMenu func(),
) {
	var screen *simulatorui.BoardScreen

	applyView := func(matchView simulatorview.MatchView) {
		if screen == nil {
			return
		}
		screen.Update(matchView)
	}

	client.OnView = func(matchView simulatorview.MatchView) {
		fyne.Do(func() { applyView(matchView) })
	}
	client.OnError = func(payload protocol.ErrorPayload) {
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf("%s: %s", payload.Code, payload.Message), window)
		})
	}

	runCommand := func(name string, revision model.Revision, args any) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
			defer cancel()
			matchView, err := client.Command(ctx, playerID, uint64(revision), name, args)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, window)
					return
				}
				applyView(matchView)
			})
		}()
	}
	runCommandWithPrivate := func(
		name string,
		revision model.Revision,
		args any,
		done func(*protocol.PrivateView, error),
	) {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), networkCommandTimeout)
			defer cancel()
			matchView, private, err := client.CommandWithPrivate(ctx, playerID, uint64(revision), name, args)
			fyne.Do(func() {
				if err != nil {
					if done != nil {
						done(nil, err)
					} else {
						dialog.ShowError(err, window)
					}
					return
				}
				applyView(matchView)
				if done != nil {
					done(private, nil)
				}
			})
		}()
	}

	actions := simulatorui.BoardActions{BackLabel: "Leave Match"}
	if !spectator {
		actions = networkBoardActions(runCommand, runCommandWithPrivate)
		actions.ShouldAutoPassPriority = func(match simulatorview.MatchView) bool {
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
			return shouldAutoPassPriority(string(match.Turn.Phase))
		}
	}

	screen = simulatorui.NewBoardController(
		initialView,
		repository.All(),
		actions,
		func() {
			_ = client.Close()
			window.SetTitle(applicationName)
			onBackToMenu()
		},
	)
	role := playerID
	if spectator {
		role = "spectator"
	}
	window.SetTitle(fmt.Sprintf("%s — Online (%s) Room %s", applicationName, role, roomCode))
	setWindowContent(window, screen.Content())
	simulatorui.BindBoardKeyboard(window, screen)
}

func networkBoardActions(
	runCommand func(name string, revision model.Revision, args any),
	runCommandWithPrivate func(
		name string,
		revision model.Revision,
		args any,
		done func(*protocol.PrivateView, error),
	),
) simulatorui.BoardActions {
	return simulatorui.BoardActions{
		BackLabel: "Leave Match",
		UseCasterToken: func(tokenID model.MatchCardID, expectedRevision model.Revision) {
			runCommand("use_caster_token", expectedRevision, map[string]any{"card_id": tokenID})
		},
		GenerateCasterAether: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			runCommand("generate_caster_aether", expectedRevision, map[string]any{"card_id": cardID})
		},
		GenerateNonElementalAether: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			runCommand("generate_non_elemental_aether", expectedRevision, map[string]any{"card_id": cardID})
		},
		CallFaceDownLevelOne: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			runCommand("call_face_down_level_one", expectedRevision, map[string]any{"card_id": cardID})
		},
		CallFaceUpLevelOne: func(cardID model.MatchCardID, expectedRevision model.Revision) {
			runCommand("call_face_up_level_one", expectedRevision, map[string]any{"card_id": cardID})
		},
		LevelUpCaster: func(upperCardID, targetCasterID model.MatchCardID, expectedRevision model.Revision) {
			runCommand("level_up_caster", expectedRevision, map[string]any{
				"upper_card_id":    upperCardID,
				"target_caster_id": targetCasterID,
			})
		},
		CastServant: func(
			cardID model.MatchCardID,
			payment model.AetherPayment,
			orientation model.CardOrientation,
			expectedRevision model.Revision,
		) {
			runCommand("cast_servant", expectedRevision, map[string]any{
				"card_id":     cardID,
				"payment":     payment,
				"orientation": orientation,
			})
		},
		CastConjure: func(cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) {
			runCommand("cast_conjure", expectedRevision, map[string]any{
				"card_id": cardID,
				"payment": payment,
			})
		},
		CastBarrier: func(cardID model.MatchCardID, payment model.AetherPayment, expectedRevision model.Revision) {
			runCommand("cast_barrier", expectedRevision, map[string]any{
				"card_id": cardID,
				"payment": payment,
			})
		},
		CastServantWithPlan: func(
			cardID model.MatchCardID,
			plan model.CastPaymentPlan,
			orientation model.CardOrientation,
			expectedRevision model.Revision,
		) {
			runCommand("cast_servant_with_plan", expectedRevision, map[string]any{
				"card_id":     cardID,
				"plan":        plan,
				"orientation": orientation,
			})
		},
		CastConjureWithPlan: func(cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) {
			runCommand("cast_conjure_with_plan", expectedRevision, map[string]any{
				"card_id": cardID,
				"plan":    plan,
			})
		},
		CastBarrierWithPlan: func(cardID model.MatchCardID, plan model.CastPaymentPlan, expectedRevision model.Revision) {
			runCommand("cast_barrier_with_plan", expectedRevision, map[string]any{
				"card_id": cardID,
				"plan":    plan,
			})
		},
		MoveCard: func(command model.MoveCardCommand, expectedRevision model.Revision) {
			runCommand("move_card", expectedRevision, command)
		},
		DrawCards: func(count int, expectedRevision model.Revision) {
			runCommand("draw_cards", expectedRevision, map[string]any{"count": count})
		},
		ShuffleDeck: func(expectedRevision model.Revision) {
			runCommand("shuffle_deck", expectedRevision, nil)
		},
		PeekDeckTops: func(
			ownerID model.PlayerID,
			count int,
			done func([]simulatorview.CardView, error),
		) {
			runCommandWithPrivate(
				"peek_deck_tops",
				0,
				map[string]any{"owner_id": ownerID, "count": count},
				func(private *protocol.PrivateView, err error) {
					if done == nil {
						return
					}
					if err != nil {
						done(nil, err)
						return
					}
					if private == nil {
						done(nil, fmt.Errorf("peek returned no cards"))
						return
					}
					done(private.DeckPeek, nil)
				},
			)
		},
		MoveDeckTopToBottom: func(ownerID model.PlayerID, expectedRevision model.Revision) {
			runCommand("move_deck_top_to_bottom", expectedRevision, map[string]any{"owner_id": ownerID})
		},
		ResolveDeckDig: func(
			keep model.MatchCardID,
			bottomOrder []model.MatchCardID,
			expectedRevision model.Revision,
		) {
			runCommand("resolve_deck_dig", expectedRevision, map[string]any{
				"keep":         keep,
				"bottom_order": bottomOrder,
			})
		},
		PassPriority: func(expectedRevision model.Revision) {
			runCommand("pass_priority", expectedRevision, nil)
		},
		DeclareAttack: func(
			attackerID model.MatchCardID,
			targetKind model.AttackTargetKind,
			targetCardID model.MatchCardID,
			expectedRevision model.Revision,
		) {
			runCommand("declare_attack", expectedRevision, map[string]any{
				"attacker_id":    attackerID,
				"target_kind":    targetKind,
				"target_card_id": targetCardID,
			})
		},
		CorruptOrbs: func(orbIndexes []int, expectedRevision model.Revision) {
			runCommand("corrupt_orb", expectedRevision, map[string]any{"orb_indexes": orbIndexes})
		},
		SetGrantedDoubleCorrupt: func(cardID model.MatchCardID, enabled bool, expectedRevision model.Revision) {
			runCommand("set_granted_double_corrupt", expectedRevision, map[string]any{
				"card_id": cardID,
				"enabled": enabled,
			})
		},
		PlayBreak: func(cardID model.MatchCardID, orientation model.CardOrientation, expectedRevision model.Revision) {
			runCommand("play_break", expectedRevision, map[string]any{
				"card_id":     cardID,
				"orientation": orientation,
			})
		},
		DeclineBreak: func(expectedRevision model.Revision) {
			runCommand("decline_break", expectedRevision, nil)
		},
		AcceptSageAdvice: func(expectedRevision model.Revision) {
			runCommand("accept_sage_advice", expectedRevision, nil)
		},
		DeclineDrawReplacement: func(expectedRevision model.Revision) {
			runCommand("decline_draw_replacement", expectedRevision, nil)
		},
		PeekOrb: func(
			ownerID model.PlayerID,
			orbIndex int,
			expectedRevision model.Revision,
			done func(simulatorview.CardView, error),
		) {
			runCommandWithPrivate(
				"peek_orb",
				expectedRevision,
				map[string]any{"owner_id": ownerID, "orb_index": orbIndex},
				func(private *protocol.PrivateView, err error) {
					if done == nil {
						return
					}
					if err != nil {
						done(simulatorview.CardView{}, err)
						return
					}
					if private == nil || private.OrbPeek == nil {
						done(simulatorview.CardView{}, fmt.Errorf("peek returned no orb"))
						return
					}
					done(*private.OrbPeek, nil)
				},
			)
		},
		RevealOrb: func(orbIndex int, expectedRevision model.Revision) {
			runCommand("reveal_orb", expectedRevision, map[string]any{"orb_index": orbIndex})
		},
		CompleteCurrentPhase: func(expectedRevision model.Revision) {
			runCommand("complete_current_phase", expectedRevision, nil)
		},
		SubmitOpeningHand: func(replace []model.MatchCardID, expectedRevision model.Revision) {
			runCommand("submit_opening_hand", expectedRevision, map[string]any{"replace": replace})
		},
	}
}

func normalizeWebSocketURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "ws://" + raw
	}
	if !strings.HasSuffix(raw, "/") {
		raw += "/"
	}
	return raw
}
