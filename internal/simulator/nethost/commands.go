package nethost

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const peerPushTimeout = 2 * time.Second

type commandResult struct {
	view    simulatorview.MatchView
	private *protocol.PrivateView
	mutate  bool
}

func (host *Host) handleCommand(client *wsClient, raw json.RawMessage) ([]byte, error) {
	var payload protocol.CommandPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return encodeError("bad_request", err.Error())
	}

	roomCode, playerID, err := host.commandIdentity(client, payload)
	if err != nil {
		return encodeError("unauthorized", err.Error())
	}

	if host.isSpectator(roomCode, playerID, client) && payload.Name != "request_view" {
		return encodeError("forbidden", "spectators may only request views")
	}

	playerSession, err := host.Lobby.Session(roomCode, playerID)
	if err != nil {
		return encodeError("no_match", err.Error())
	}

	result, err := host.applyCommand(playerSession, payload)
	if err != nil {
		return encodeError("command_failed", err.Error())
	}

	reply, err := protocol.Encode(protocol.KindView, protocol.ViewPayload{
		Match:        host.withDisplayNames(roomCode, result.view),
		Private:      result.private,
		DisplayNames: host.displayNames(roomCode),
	})
	if err != nil {
		return nil, err
	}

	// Reply to the acting player first. Peer fan-out must not block their
	// command round-trip (a stalled peer write previously froze online play).
	if result.mutate {
		go host.pushViews(roomCode, playerID)
	}
	return reply, nil
}

func (host *Host) isSpectator(roomCode, playerID string, client *wsClient) bool {
	if client != nil && client.spectator {
		return true
	}
	host.Lobby.mu.Lock()
	defer host.Lobby.mu.Unlock()
	room, ok := host.Lobby.rooms[roomCode]
	if !ok {
		return false
	}
	return room.IsSpectator(playerID)
}

func (host *Host) displayNames(roomCode string) map[string]string {
	host.Lobby.mu.Lock()
	defer host.Lobby.mu.Unlock()
	room, ok := host.Lobby.rooms[roomCode]
	if !ok {
		return nil
	}
	return room.DisplayNames()
}

func (host *Host) withDisplayNames(roomCode string, matchView simulatorview.MatchView) simulatorview.MatchView {
	names := host.displayNames(roomCode)
	if len(names) == 0 {
		return matchView
	}
	matchView.DisplayNames = make(map[model.PlayerID]string, len(names))
	for id, name := range names {
		matchView.DisplayNames[model.PlayerID(id)] = name
	}
	return matchView
}

func (host *Host) commandIdentity(client *wsClient, payload protocol.CommandPayload) (roomCode, playerID string, err error) {
	if client != nil {
		if client.roomCode == "" || client.playerID == "" {
			return "", "", fmt.Errorf("join a room before sending commands")
		}
		if payload.PlayerID != "" && payload.PlayerID != client.playerID {
			return "", "", fmt.Errorf("player_id does not match connection seat")
		}
		return client.roomCode, client.playerID, nil
	}
	// Test path without a websocket client: require explicit player id and a
	// single occupied room that already has a match.
	if payload.PlayerID == "" {
		return "", "", fmt.Errorf("player_id is required")
	}
	host.Lobby.mu.Lock()
	defer host.Lobby.mu.Unlock()
	for code, room := range host.Lobby.rooms {
		if room.Match == nil {
			continue
		}
		if _, ok := room.Sessions[payload.PlayerID]; ok {
			return code, payload.PlayerID, nil
		}
	}
	return "", "", fmt.Errorf("session not found")
}

func mutatingView(matchView simulatorview.MatchView, err error) (commandResult, error) {
	if err != nil {
		return commandResult{}, err
	}
	return commandResult{view: matchView, mutate: true}, nil
}

func (host *Host) applyCommand(
	playerSession *session.PlayerSession,
	payload protocol.CommandPayload,
) (commandResult, error) {
	revision := model.Revision(payload.Revision)

	switch payload.Name {
	case "request_view":
		matchView, err := playerSession.View()
		if err != nil {
			return commandResult{}, err
		}
		return commandResult{view: matchView}, nil
	case "pass_priority":
		return mutatingView(playerSession.PassPriority(revision))
	case "complete_current_phase":
		return mutatingView(playerSession.CompleteCurrentPhase(revision))
	case "submit_opening_hand":
		var args struct {
			Replace []model.MatchCardID `json:"replace"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.SubmitOpeningHandDecision(args.Replace, revision))
	case "call_face_down_level_one":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CallFaceDownLevelOne(args.CardID, revision))
	case "call_face_up_level_one":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CallFaceUpLevelOne(args.CardID, revision))
	case "level_up_caster":
		var args struct {
			UpperCardID    model.MatchCardID `json:"upper_card_id"`
			TargetCasterID model.MatchCardID `json:"target_caster_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.LevelUpCaster(args.UpperCardID, args.TargetCasterID, revision))
	case "generate_non_elemental_aether":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.GenerateNonElementalAether(args.CardID, revision))
	case "generate_caster_aether":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.GenerateCasterAether(args.CardID, revision))
	case "use_caster_token":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.UseCasterToken(args.CardID, revision))
	case "cast_servant":
		var args struct {
			CardID      model.MatchCardID     `json:"card_id"`
			Payment     model.AetherPayment   `json:"payment"`
			Orientation model.CardOrientation `json:"orientation"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CastServant(args.CardID, args.Payment, args.Orientation, revision))
	case "cast_conjure":
		var args struct {
			CardID  model.MatchCardID   `json:"card_id"`
			Payment model.AetherPayment `json:"payment"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CastConjure(args.CardID, args.Payment, revision))
	case "cast_barrier":
		var args struct {
			CardID  model.MatchCardID   `json:"card_id"`
			Payment model.AetherPayment `json:"payment"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CastBarrier(args.CardID, args.Payment, revision))
	case "cast_servant_with_plan":
		var args struct {
			CardID      model.MatchCardID     `json:"card_id"`
			Plan        model.CastPaymentPlan `json:"plan"`
			Orientation model.CardOrientation `json:"orientation"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CastServantWithPlan(args.CardID, args.Plan, args.Orientation, revision))
	case "cast_conjure_with_plan":
		var args struct {
			CardID model.MatchCardID     `json:"card_id"`
			Plan   model.CastPaymentPlan `json:"plan"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CastConjureWithPlan(args.CardID, args.Plan, revision))
	case "cast_barrier_with_plan":
		var args struct {
			CardID model.MatchCardID     `json:"card_id"`
			Plan   model.CastPaymentPlan `json:"plan"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.CastBarrierWithPlan(args.CardID, args.Plan, revision))
	case "move_card":
		var args model.MoveCardCommand
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.MoveCard(args, revision))
	case "adjust_aether":
		var args model.AdjustAetherCommand
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.AdjustAether(args, revision))
	case "draw_cards":
		var args struct {
			Count int `json:"count"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.DrawCards(args.Count, revision))
	case "shuffle_deck":
		return mutatingView(playerSession.ShuffleDeck(revision))
	case "peek_deck_tops":
		var args struct {
			OwnerID model.PlayerID `json:"owner_id"`
			Count   int            `json:"count"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		peeked, matchView, err := playerSession.PeekDeckTops(args.OwnerID, args.Count)
		if err != nil {
			return commandResult{}, err
		}
		return commandResult{
			view: matchView,
			private: &protocol.PrivateView{
				DeckPeek:      peeked,
				DeckPeekOwner: string(args.OwnerID),
			},
		}, nil
	case "move_deck_top_to_bottom":
		var args struct {
			OwnerID model.PlayerID `json:"owner_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.MoveDeckTopToBottom(args.OwnerID, revision))
	case "resolve_deck_dig":
		var args struct {
			Keep        model.MatchCardID   `json:"keep"`
			BottomOrder []model.MatchCardID `json:"bottom_order"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.ResolveDeckDig(args.Keep, args.BottomOrder, revision))
	case "declare_attack":
		var args struct {
			AttackerID   model.MatchCardID      `json:"attacker_id"`
			TargetKind   model.AttackTargetKind `json:"target_kind"`
			TargetCardID model.MatchCardID      `json:"target_card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.DeclareAttack(args.AttackerID, args.TargetKind, args.TargetCardID, revision))
	case "corrupt_orb":
		var args struct {
			OrbIndex   int   `json:"orb_index"`
			OrbIndexes []int `json:"orb_indexes"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		indexes := args.OrbIndexes
		if len(indexes) == 0 {
			indexes = []int{args.OrbIndex}
		}
		return mutatingView(playerSession.CorruptOrbs(indexes, revision))
	case "set_granted_double_corrupt":
		var args struct {
			CardID  model.MatchCardID `json:"card_id"`
			Enabled bool              `json:"enabled"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.SetGrantedDoubleCorrupt(args.CardID, args.Enabled, revision))
	case "play_break":
		var args struct {
			CardID           model.MatchCardID     `json:"card_id"`
			EntryOrientation model.CardOrientation `json:"orientation,omitempty"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.PlayBreak(args.CardID, args.EntryOrientation, revision))
	case "decline_break":
		return mutatingView(playerSession.DeclineBreak(revision))
	case "accept_sage_advice":
		return mutatingView(playerSession.AcceptSageAdvice(revision))
	case "decline_draw_replacement":
		return mutatingView(playerSession.DeclineDrawReplacement(revision))
	case "peek_orb":
		var args struct {
			OwnerID  model.PlayerID `json:"owner_id"`
			OrbIndex int            `json:"orb_index"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		peeked, matchView, err := playerSession.PeekOrb(args.OwnerID, args.OrbIndex, revision)
		if err != nil {
			return commandResult{}, err
		}
		return commandResult{
			view:   matchView,
			mutate: true,
			private: &protocol.PrivateView{
				OrbPeek: &peeked,
			},
		}, nil
	case "reveal_orb":
		var args struct {
			OrbIndex int `json:"orb_index"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return commandResult{}, err
		}
		return mutatingView(playerSession.RevealOrb(args.OrbIndex, revision))
	default:
		return commandResult{}, fmt.Errorf("unknown command %q", payload.Name)
	}
}

func (host *Host) pushViews(roomCode, exceptPlayerID string) {
	if roomCode == "" {
		return
	}
	names := host.displayNames(roomCode)
	for _, client := range host.clientsInRoom(roomCode) {
		if client.playerID == "" || client.playerID == exceptPlayerID {
			continue
		}
		if client.conn == nil && client.writeFn == nil {
			continue
		}
		playerSession, err := host.Lobby.Session(roomCode, client.playerID)
		if err != nil {
			continue
		}
		matchView, err := playerSession.View()
		if err != nil {
			continue
		}
		matchView = host.withDisplayNames(roomCode, matchView)
		raw, err := protocol.Encode(protocol.KindView, protocol.ViewPayload{
			Match:        matchView,
			DisplayNames: names,
		})
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), peerPushTimeout)
		_ = client.write(ctx, raw)
		cancel()
	}
}
