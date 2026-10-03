package nethost

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/protocol"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

func (host *Host) handleCommand(client *wsClient, raw json.RawMessage) ([]byte, error) {
	var payload protocol.CommandPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return encodeError("bad_request", err.Error())
	}

	roomCode, playerID, err := host.commandIdentity(client, payload)
	if err != nil {
		return encodeError("unauthorized", err.Error())
	}

	playerSession, err := host.Lobby.Session(roomCode, playerID)
	if err != nil {
		return encodeError("no_match", err.Error())
	}

	matchView, err := host.applyCommand(playerSession, payload)
	if err != nil {
		return encodeError("command_failed", err.Error())
	}

	// Reply carries the actor's view; notify peers separately.
	host.pushViews(roomCode, playerID)

	return protocol.Encode(protocol.KindView, protocol.ViewPayload{Match: matchView})
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

func (host *Host) applyCommand(
	playerSession *session.PlayerSession,
	payload protocol.CommandPayload,
) (simulatorview.MatchView, error) {
	revision := model.Revision(payload.Revision)

	switch payload.Name {
	case "request_view":
		return playerSession.View()
	case "pass_priority":
		return playerSession.PassPriority(revision)
	case "complete_current_phase":
		return playerSession.CompleteCurrentPhase(revision)
	case "submit_opening_hand":
		var args struct {
			Replace []model.MatchCardID `json:"replace"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.SubmitOpeningHandDecision(args.Replace, revision)
	case "call_face_down_level_one":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CallFaceDownLevelOne(args.CardID, revision)
	case "call_face_up_level_one":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CallFaceUpLevelOne(args.CardID, revision)
	case "level_up_caster":
		var args struct {
			UpperCardID    model.MatchCardID `json:"upper_card_id"`
			TargetCasterID model.MatchCardID `json:"target_caster_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.LevelUpCaster(args.UpperCardID, args.TargetCasterID, revision)
	case "generate_non_elemental_aether":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.GenerateNonElementalAether(args.CardID, revision)
	case "generate_caster_aether":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.GenerateCasterAether(args.CardID, revision)
	case "use_caster_token":
		var args struct {
			CardID model.MatchCardID `json:"card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.UseCasterToken(args.CardID, revision)
	case "cast_servant":
		var args struct {
			CardID      model.MatchCardID     `json:"card_id"`
			Payment     model.AetherPayment   `json:"payment"`
			Orientation model.CardOrientation `json:"orientation"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CastServant(args.CardID, args.Payment, args.Orientation, revision)
	case "cast_conjure":
		var args struct {
			CardID  model.MatchCardID   `json:"card_id"`
			Payment model.AetherPayment `json:"payment"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CastConjure(args.CardID, args.Payment, revision)
	case "cast_barrier":
		var args struct {
			CardID  model.MatchCardID   `json:"card_id"`
			Payment model.AetherPayment `json:"payment"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CastBarrier(args.CardID, args.Payment, revision)
	case "cast_servant_with_plan":
		var args struct {
			CardID      model.MatchCardID     `json:"card_id"`
			Plan        model.CastPaymentPlan `json:"plan"`
			Orientation model.CardOrientation `json:"orientation"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CastServantWithPlan(args.CardID, args.Plan, args.Orientation, revision)
	case "cast_conjure_with_plan":
		var args struct {
			CardID model.MatchCardID     `json:"card_id"`
			Plan   model.CastPaymentPlan `json:"plan"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CastConjureWithPlan(args.CardID, args.Plan, revision)
	case "cast_barrier_with_plan":
		var args struct {
			CardID model.MatchCardID     `json:"card_id"`
			Plan   model.CastPaymentPlan `json:"plan"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.CastBarrierWithPlan(args.CardID, args.Plan, revision)
	case "move_card":
		var args model.MoveCardCommand
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.MoveCard(args, revision)
	case "declare_attack":
		var args struct {
			AttackerID   model.MatchCardID      `json:"attacker_id"`
			TargetKind   model.AttackTargetKind `json:"target_kind"`
			TargetCardID model.MatchCardID      `json:"target_card_id"`
		}
		if err := json.Unmarshal(payload.Args, &args); err != nil {
			return simulatorview.MatchView{}, err
		}
		return playerSession.DeclareAttack(args.AttackerID, args.TargetKind, args.TargetCardID, revision)
	default:
		return simulatorview.MatchView{}, fmt.Errorf("unknown command %q", payload.Name)
	}
}

func (host *Host) pushViews(roomCode, exceptPlayerID string) {
	if roomCode == "" {
		return
	}
	for _, client := range host.clientsInRoom(roomCode) {
		if client.playerID == "" || client.conn == nil || client.playerID == exceptPlayerID {
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
		raw, err := protocol.Encode(protocol.KindView, protocol.ViewPayload{Match: matchView})
		if err != nil {
			continue
		}
		_ = writeBytes(context.Background(), client.conn, raw)
	}
}
