package ai

import (
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// ActionKind identifies one PlayerSession-backed command the driver can apply.
type ActionKind string

const (
	ActionWait                 ActionKind = "wait"
	ActionPass                 ActionKind = "pass"
	ActionCompletePhase        ActionKind = "complete_phase"
	ActionKeepOpeningHand      ActionKind = "keep_opening_hand"
	ActionCallFaceDownLv1      ActionKind = "call_face_down_lv1"
	ActionCallFaceUpLv1        ActionKind = "call_face_up_lv1"
	ActionLevelUpCaster        ActionKind = "level_up_caster"
	ActionGenerateCasterAether ActionKind = "generate_caster_aether"
	ActionGenerateNonElemental ActionKind = "generate_non_elemental"
	ActionUseCasterToken       ActionKind = "use_caster_token"
	ActionCastServant          ActionKind = "cast_servant"
	ActionDeclareAttack        ActionKind = "declare_attack"
	ActionCorruptOrb           ActionKind = "corrupt_orb"
	ActionPlayBreak            ActionKind = "play_break"
	ActionDeclineBreak         ActionKind = "decline_break"
	ActionResolveChoice        ActionKind = "resolve_choice"
	ActionDeclineChoice        ActionKind = "decline_choice"
	ActionAcceptSage           ActionKind = "accept_sage"
	ActionDeclineSage          ActionKind = "decline_sage"
	ActionActivateRest         ActionKind = "activate_rest"
)

// Action is one decision for the AI seat.
type Action struct {
	Kind ActionKind

	CardID       model.MatchCardID
	TargetCardID model.MatchCardID
	TargetKind   model.AttackTargetKind
	Payment      model.AetherPayment
	Orientation  model.CardOrientation
	OptionID     string
	Selected     []model.MatchCardID
	OrbIndexes   []int
}

// Agent chooses the next action for its seat from a viewer-safe MatchView.
type Agent interface {
	Name() string
	Choose(match view.MatchView) (Action, error)
}

// PassAgent always passes priority or completes the current phase when active.
// Useful for driver smoke tests.
type PassAgent struct{}

func (PassAgent) Name() string { return "PassBot" }

func (PassAgent) Choose(match view.MatchView) (Action, error) {
	if match.MatchStatus != model.StatusInProgress && match.MatchStatus != model.StatusSetup {
		return Action{Kind: ActionWait}, nil
	}
	viewer := match.ViewerID
	self := findPlayer(match, viewer)
	if self == nil {
		return Action{Kind: ActionWait}, nil
	}
	if !self.OpeningHandFinalized {
		return Action{Kind: ActionKeepOpeningHand}, nil
	}
	if match.PendingBreak.PlayerID == viewer {
		return Action{Kind: ActionDeclineBreak}, nil
	}
	if match.PendingDraw.PlayerID == viewer && match.PendingDraw.Step != model.PendingDrawIdle {
		return Action{Kind: ActionDeclineSage}, nil
	}
	if match.PendingChoice.PlayerID == viewer {
		if match.PendingChoice.AllowDecline {
			return Action{Kind: ActionDeclineChoice}, nil
		}
		act := Action{Kind: ActionResolveChoice}
		if len(match.PendingChoice.Options) > 0 {
			act.OptionID = match.PendingChoice.Options[0].ID
		}
		if len(match.PendingChoice.SelectableCardIDs) > 0 {
			act.Selected = []model.MatchCardID{match.PendingChoice.SelectableCardIDs[0]}
		}
		return act, nil
	}
	if match.Attack.Step == model.BattleStepAwaitingJudgment && match.Turn.ActivePlayer == viewer {
		return Action{Kind: ActionCorruptOrb, OrbIndexes: []int{0}}, nil
	}
	if match.PrioritySequenceOpen && match.PriorityHolder == viewer {
		return Action{Kind: ActionPass}, nil
	}
	if match.Turn.ActivePlayer == viewer {
		return Action{Kind: ActionCompletePhase}, nil
	}
	return Action{Kind: ActionWait}, nil
}

func findPlayer(match view.MatchView, id model.PlayerID) *view.PlayerView {
	for i := range match.Players {
		if match.Players[i].ID == id {
			return &match.Players[i]
		}
	}
	return nil
}

func opponentOf(match view.MatchView, id model.PlayerID) *view.PlayerView {
	for i := range match.Players {
		if match.Players[i].ID != id && match.Players[i].ID != "" {
			return &match.Players[i]
		}
	}
	return nil
}
