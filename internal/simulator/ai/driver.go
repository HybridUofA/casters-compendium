package ai

import (
	"fmt"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/session"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

const maxDriverSteps = 32

// Driver applies Agent decisions through a PlayerSession.
type Driver struct {
	Session *session.PlayerSession
	Agent   Agent
}

// Step asks the agent once and applies the resulting action.
// Returns the updated view and whether the AI still expects to act.
func (d *Driver) Step() (view.MatchView, bool, error) {
	if d == nil || d.Session == nil || d.Agent == nil {
		return view.MatchView{}, false, fmt.Errorf("ai driver requires session and agent")
	}
	match, err := d.Session.View()
	if err != nil {
		return view.MatchView{}, false, err
	}
	if !selfShouldAct(match) {
		return match, false, nil
	}
	action, err := d.Agent.Choose(match)
	if err != nil {
		return match, false, err
	}
	if action.Kind == ActionWait {
		return match, false, nil
	}
	updated, err := d.apply(action, match.Revision)
	if err != nil {
		// Naive safety net: try pass, then complete phase.
		if updated, passErr := d.Session.PassPriority(match.Revision); passErr == nil {
			return updated, selfShouldAct(updated), nil
		}
		if updated, phaseErr := d.Session.CompleteCurrentPhase(match.Revision); phaseErr == nil {
			return updated, selfShouldAct(updated), nil
		}
		return match, false, err
	}
	return updated, selfShouldAct(updated), nil
}

// RunUntilIdle keeps stepping while the AI seat must act (bounded).
func (d *Driver) RunUntilIdle() (view.MatchView, error) {
	var last view.MatchView
	var err error
	for i := 0; i < maxDriverSteps; i++ {
		var cont bool
		last, cont, err = d.Step()
		if err != nil {
			return last, err
		}
		if !cont {
			return last, nil
		}
	}
	return last, fmt.Errorf("ai driver exceeded %d steps", maxDriverSteps)
}

func (d *Driver) apply(action Action, revision model.Revision) (view.MatchView, error) {
	switch action.Kind {
	case ActionWait:
		return d.Session.View()
	case ActionKeepOpeningHand:
		return d.Session.SubmitOpeningHandDecision(nil, revision)
	case ActionPass:
		return d.Session.PassPriority(revision)
	case ActionCompletePhase:
		return d.Session.CompleteCurrentPhase(revision)
	case ActionCallFaceDownLv1:
		return d.Session.CallFaceDownLevelOne(action.CardID, revision)
	case ActionCallFaceUpLv1:
		return d.Session.CallFaceUpLevelOne(action.CardID, revision)
	case ActionLevelUpCaster:
		return d.Session.LevelUpCaster(action.CardID, action.TargetCardID, revision)
	case ActionGenerateCasterAether:
		return d.Session.GenerateCasterAether(action.CardID, revision)
	case ActionGenerateNonElemental:
		return d.Session.GenerateNonElementalAether(action.CardID, revision)
	case ActionUseCasterToken:
		return d.Session.UseCasterToken(action.CardID, revision)
	case ActionCastServant:
		orient := action.Orientation
		if orient == "" {
			orient = model.OrientationRecovered
		}
		return d.Session.CastServant(action.CardID, action.Payment, orient, revision)
	case ActionDeclareAttack:
		return d.Session.DeclareAttack(action.CardID, action.TargetKind, action.TargetCardID, revision)
	case ActionCorruptOrb:
		indexes := action.OrbIndexes
		if len(indexes) == 0 {
			indexes = []int{0}
		}
		return d.Session.CorruptOrbs(indexes, revision)
	case ActionPlayBreak:
		orient := action.Orientation
		if orient == "" {
			orient = model.OrientationRecovered
		}
		return d.Session.PlayBreak(action.CardID, orient, revision)
	case ActionDeclineBreak:
		return d.Session.DeclineBreak(revision)
	case ActionResolveChoice:
		return d.Session.ResolvePendingChoice(action.OptionID, action.Selected, revision)
	case ActionDeclineChoice:
		return d.Session.DeclinePendingChoice(revision)
	case ActionAcceptSage:
		return d.Session.AcceptSageAdvice(revision)
	case ActionDeclineSage:
		return d.Session.DeclineDrawReplacement(revision)
	case ActionActivateRest:
		return d.Session.ActivateRegisteredRest(action.CardID, revision)
	default:
		return view.MatchView{}, fmt.Errorf("unsupported ai action %q", action.Kind)
	}
}
