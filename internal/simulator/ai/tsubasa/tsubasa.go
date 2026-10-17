// Package tsubasa is the Ignus Passion Wing solo persona: eager, earnest, naive.
package tsubasa

import (
	"strconv"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/ai"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// Agent plays like Tsubasa / classic Passion Wing: push damage, buff before
// swinging, pick simple targets, answer prompts aggressively.
//
// Important: casters are called face-up so they can produce Ignus aether.
// Face-down-only Call was causing pass-bot behavior in Main/Battle.
type Agent struct {
	Catalog ai.Catalog
}

func (a Agent) Name() string { return "Tsubasa (Passion Wing)" }

func (a Agent) Choose(match view.MatchView) (ai.Action, error) {
	viewer := match.ViewerID
	self := find(match, viewer)
	if self == nil {
		return ai.Action{Kind: ai.ActionWait}, nil
	}

	if !self.OpeningHandFinalized {
		return ai.Action{Kind: ai.ActionKeepOpeningHand}, nil
	}
	if match.PendingBreak.PlayerID == viewer && len(match.PendingBreak.CardIDs) > 0 {
		if match.PendingBreak.OfferPlay {
			return ai.Action{
				Kind:        ai.ActionPlayBreak,
				CardID:      match.PendingBreak.CardIDs[0],
				Orientation: model.OrientationRecovered,
			}, nil
		}
		return ai.Action{Kind: ai.ActionDeclineBreak}, nil
	}
	if match.PendingDraw.PlayerID == viewer && match.PendingDraw.Step != model.PendingDrawIdle {
		return ai.Action{Kind: ai.ActionDeclineSage}, nil
	}
	if match.PendingChoice.PlayerID == viewer {
		return aggressiveChoice(match.PendingChoice), nil
	}
	if match.Attack.Step == model.BattleStepAwaitingJudgment && match.Turn.ActivePlayer == viewer {
		return ai.Action{Kind: ai.ActionCorruptOrb, OrbIndexes: []int{0}}, nil
	}

	active := match.Turn.ActivePlayer == viewer
	holdsPriority := match.PrioritySequenceOpen && match.PriorityHolder == viewer

	// Call: level up Passion Wing, else call face-up Lv1 (so they can tap for aether).
	if active && holdsPriority && match.Turn.Phase == model.PhaseCall {
		if act, ok := a.tryLevelUpPassionWing(*self); ok {
			return act, nil
		}
		if act, ok := a.tryCallFaceUpLv1(*self); ok {
			return act, nil
		}
	}

	// Main: cast if affordable; else produce aether / token so a cast becomes possible.
	if active && holdsPriority && match.Turn.Phase == model.PhaseMain {
		if act, ok := a.tryCastServant(*self); ok {
			return act, nil
		}
		if act, ok := a.tryProduceAether(*self); ok {
			return act, nil
		}
		if act, ok := a.tryUseCasterToken(*self); ok {
			return act, nil
		}
	}

	if active && holdsPriority && match.Turn.Phase == model.PhaseBattle {
		if act, ok := a.tryPassionWingRest(*self); ok {
			return act, nil
		}
		if act, ok := a.tryAttack(match, *self); ok {
			return act, nil
		}
	}

	if holdsPriority {
		if active && match.Turn.Phase != model.PhaseMain && match.Turn.Phase != model.PhaseBattle {
			return ai.Action{Kind: ai.ActionCompletePhase}, nil
		}
		return ai.Action{Kind: ai.ActionPass}, nil
	}
	if active && !match.PrioritySequenceOpen {
		return ai.Action{Kind: ai.ActionCompletePhase}, nil
	}
	return ai.Action{Kind: ai.ActionWait}, nil
}

func aggressiveChoice(choice model.PendingChoice) ai.Action {
	if len(choice.Options) > 0 {
		prefer := []string{"bottom", "rock", "paper", "use", "confirm"}
		for _, want := range prefer {
			for _, opt := range choice.Options {
				if strings.EqualFold(opt.ID, want) {
					act := ai.Action{Kind: ai.ActionResolveChoice, OptionID: opt.ID}
					if len(choice.SelectableCardIDs) > 0 {
						act.Selected = []model.MatchCardID{choice.SelectableCardIDs[0]}
					}
					return act
				}
			}
		}
		act := ai.Action{Kind: ai.ActionResolveChoice, OptionID: choice.Options[0].ID}
		if len(choice.SelectableCardIDs) > 0 {
			act.Selected = []model.MatchCardID{choice.SelectableCardIDs[0]}
		}
		return act
	}
	if len(choice.SelectableCardIDs) > 0 {
		// Enter destroy / buff prompts are mandatory when targets exist — pick one.
		return ai.Action{
			Kind:     ai.ActionResolveChoice,
			Selected: []model.MatchCardID{choice.SelectableCardIDs[0]},
		}
	}
	if choice.AllowDecline {
		return ai.Action{Kind: ai.ActionDeclineChoice}
	}
	return ai.Action{Kind: ai.ActionResolveChoice}
}

func (a Agent) tryCallFaceUpLv1(self view.PlayerView) (ai.Action, bool) {
	if a.Catalog == nil {
		return ai.Action{}, false
	}
	var fallback view.CardView
	foundFallback := false
	for _, card := range self.Hand {
		if card.CardID == "" {
			continue
		}
		def, ok := a.Catalog.FindByID(string(card.CardID))
		if !ok || !strings.EqualFold(def.Type, "Caster") {
			continue
		}
		if strings.TrimSpace(def.CostLevel) != "1" {
			continue
		}
		// Prefer Passion Wing for the persona.
		if isPassionWing(def) {
			return ai.Action{Kind: ai.ActionCallFaceUpLv1, CardID: card.MatchID}, true
		}
		if !foundFallback {
			fallback = card
			foundFallback = true
		}
	}
	if foundFallback {
		return ai.Action{Kind: ai.ActionCallFaceUpLv1, CardID: fallback.MatchID}, true
	}
	return ai.Action{}, false
}

func (a Agent) tryLevelUpPassionWing(self view.PlayerView) (ai.Action, bool) {
	if a.Catalog == nil {
		return ai.Action{}, false
	}
	type fieldCaster struct {
		card  view.CardView
		level int
	}
	var field []fieldCaster
	for _, card := range self.CasterZone {
		if card.MatchID == "" || card.CardID == "" {
			continue
		}
		if card.Face == model.CardFaceDown {
			continue
		}
		def, ok := a.Catalog.FindByID(string(card.CardID))
		if !ok || !isPassionWing(def) {
			continue
		}
		field = append(field, fieldCaster{card: card, level: costLevel(def)})
	}
	if len(field) == 0 {
		return ai.Action{}, false
	}
	for _, card := range self.Hand {
		if card.CardID == "" {
			continue
		}
		def, ok := a.Catalog.FindByID(string(card.CardID))
		if !ok || !strings.EqualFold(def.Type, "Caster") || !isPassionWing(def) {
			continue
		}
		upperLevel := costLevel(def)
		for _, base := range field {
			if upperLevel > base.level {
				return ai.Action{
					Kind:         ai.ActionLevelUpCaster,
					CardID:       card.MatchID,
					TargetCardID: base.card.MatchID,
				}, true
			}
		}
	}
	return ai.Action{}, false
}

func (a Agent) tryCastServant(self view.PlayerView) (ai.Action, bool) {
	if a.Catalog == nil {
		return ai.Action{}, false
	}
	// Prefer Ignus servants (Passion Wing), still cheapest-first within that bias.
	servants := ai.HandServantsCheapestFirst(self.Hand, a.Catalog)
	var ignus, other []view.CardView
	for _, card := range servants {
		def, ok := a.Catalog.FindByID(string(card.CardID))
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(def.Element), "Ignus") {
			ignus = append(ignus, card)
		} else {
			other = append(other, card)
		}
	}
	for _, card := range append(ignus, other...) {
		def, ok := a.Catalog.FindByID(string(card.CardID))
		if !ok {
			continue
		}
		pay, ok := ai.NaivePayment(self.Aether, def)
		if !ok {
			continue
		}
		return ai.Action{
			Kind:        ai.ActionCastServant,
			CardID:      card.MatchID,
			Payment:     pay,
			Orientation: model.OrientationRecovered,
		}, true
	}
	return ai.Action{}, false
}

func (a Agent) tryProduceAether(self view.PlayerView) (ai.Action, bool) {
	// Only worth resting a caster if we still have something to cast afterward.
	if a.Catalog != nil && len(ai.HandServantsCheapestFirst(self.Hand, a.Catalog)) == 0 {
		return ai.Action{}, false
	}
	var fallback view.CardView
	found := false
	for _, card := range self.CasterZone {
		if card.MatchID == "" || card.Orientation != model.OrientationRecovered {
			continue
		}
		if card.Face == model.CardFaceDown {
			continue
		}
		// Prefer Passion Wing for Ignus aether.
		if card.CardID != "" && a.Catalog != nil {
			if def, ok := a.Catalog.FindByID(string(card.CardID)); ok && isPassionWing(def) {
				return ai.Action{Kind: ai.ActionGenerateCasterAether, CardID: card.MatchID}, true
			}
		}
		if !found {
			fallback = card
			found = true
		}
	}
	if found {
		return ai.Action{Kind: ai.ActionGenerateCasterAether, CardID: fallback.MatchID}, true
	}
	return ai.Action{}, false
}

func (a Agent) tryUseCasterToken(self view.PlayerView) (ai.Action, bool) {
	if a.Catalog != nil && len(ai.HandServantsCheapestFirst(self.Hand, a.Catalog)) == 0 {
		return ai.Action{}, false
	}
	for _, card := range self.CasterZone {
		if card.MatchID == "" ||
			card.Orientation != model.OrientationRecovered ||
			card.CardID != model.CasterTokenCardID {
			continue
		}
		return ai.Action{Kind: ai.ActionUseCasterToken, CardID: card.MatchID}, true
	}
	return ai.Action{}, false
}

func (a Agent) tryPassionWingRest(self view.PlayerView) (ai.Action, bool) {
	if a.Catalog == nil {
		return ai.Action{}, false
	}
	hasAlly := false
	for _, s := range self.ServantZone {
		if s.MatchID != "" {
			hasAlly = true
			break
		}
	}
	if !hasAlly {
		return ai.Action{}, false
	}
	for _, card := range self.CasterZone {
		if card.MatchID == "" || card.Orientation != model.OrientationRecovered {
			continue
		}
		if card.CardID == "" || card.Face == model.CardFaceDown {
			continue
		}
		def, ok := a.Catalog.FindByID(string(card.CardID))
		if !ok {
			continue
		}
		if isPassionWing(def) {
			return ai.Action{Kind: ai.ActionActivateRest, CardID: card.MatchID}, true
		}
	}
	return ai.Action{}, false
}

func isPassionWing(def gamecards.Card) bool {
	return strings.Contains(strings.ToLower(def.Name), "passion wing")
}

func costLevel(def gamecards.Card) int {
	n, err := strconv.Atoi(strings.TrimSpace(def.CostLevel))
	if err != nil {
		return 0
	}
	return n
}

func (a Agent) tryAttack(match view.MatchView, self view.PlayerView) (ai.Action, bool) {
	var attacker view.CardView
	found := false
	for _, card := range self.ServantZone {
		if card.MatchID != "" && card.Orientation == model.OrientationRecovered {
			attacker = card
			found = true
			break
		}
	}
	if !found {
		return ai.Action{}, false
	}
	enemy := findOpponent(match, self.ID)
	if enemy == nil {
		return ai.Action{}, false
	}
	if canHitPlayer(*enemy) {
		return ai.Action{
			Kind:       ai.ActionDeclareAttack,
			CardID:     attacker.MatchID,
			TargetKind: model.AttackTargetPlayer,
		}, true
	}
	for _, card := range enemy.ServantZone {
		if card.MatchID != "" {
			return ai.Action{
				Kind:         ai.ActionDeclareAttack,
				CardID:       attacker.MatchID,
				TargetKind:   model.AttackTargetServant,
				TargetCardID: card.MatchID,
			}, true
		}
	}
	return ai.Action{}, false
}

func find(match view.MatchView, id model.PlayerID) *view.PlayerView {
	for i := range match.Players {
		if match.Players[i].ID == id {
			return &match.Players[i]
		}
	}
	return nil
}

func findOpponent(match view.MatchView, id model.PlayerID) *view.PlayerView {
	for i := range match.Players {
		if match.Players[i].ID != id && match.Players[i].ID != "" {
			return &match.Players[i]
		}
	}
	return nil
}

func canHitPlayer(enemy view.PlayerView) bool {
	for _, card := range enemy.ServantZone {
		if card.Orientation == model.OrientationReversed {
			return false
		}
	}
	return true
}
