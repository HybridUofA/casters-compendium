package ai

import (
	"strconv"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	"github.com/HybridUofA/casters-compendium/internal/simulator/view"
)

// Catalog is the printed-card lookup agents use for cost/type/name filters.
type Catalog interface {
	FindByID(id string) (gamecards.Card, bool)
}

func aetherTotal(pool model.AetherPool) int {
	return pool.Aes + pool.Aqua + pool.Ignus + pool.Luna + pool.Silva +
		pool.Solis + pool.Terra + pool.Void + pool.NonElemental
}

func cardCost(def gamecards.Card) int {
	n, err := strconv.Atoi(strings.TrimSpace(def.CostLevel))
	if err != nil {
		return 0
	}
	return n
}

// naivePayment builds a legal-looking payment: 1 of the card's element (if any)
// plus NonElemental for the rest, falling back to whatever is in the pool.
// NaivePayment builds a legal-looking payment from the pool for def's cost.
func NaivePayment(pool model.AetherPool, def gamecards.Card) (model.AetherPayment, bool) {
	cost := cardCost(def)
	if cost <= 0 {
		return model.AetherPayment{}, true
	}
	if aetherTotal(pool) < cost {
		return model.AetherPayment{}, false
	}
	element := strings.TrimSpace(def.Element)
	pay := model.AetherPayment{}
	remaining := cost
	tryTake := func(have *int, into *int, n int) int {
		if n <= 0 || *have <= 0 {
			return 0
		}
		take := n
		if take > *have {
			take = *have
		}
		*have -= take
		*into += take
		return take
	}
	// Prefer printed element first.
	switch strings.ToLower(element) {
	case "aes":
		remaining -= tryTake(&pool.Aes, &pay.Aes, 1)
	case "aqua":
		remaining -= tryTake(&pool.Aqua, &pay.Aqua, 1)
	case "ignus":
		remaining -= tryTake(&pool.Ignus, &pay.Ignus, 1)
	case "luna":
		remaining -= tryTake(&pool.Luna, &pay.Luna, 1)
	case "silva":
		remaining -= tryTake(&pool.Silva, &pay.Silva, 1)
	case "solis":
		remaining -= tryTake(&pool.Solis, &pay.Solis, 1)
	case "terra":
		remaining -= tryTake(&pool.Terra, &pay.Terra, 1)
	case "void":
		remaining -= tryTake(&pool.Void, &pay.Void, 1)
	}
	remaining -= tryTake(&pool.NonElemental, &pay.NonElemental, remaining)
	// Drain any element for the remainder (naive).
	for remaining > 0 {
		before := remaining
		remaining -= tryTake(&pool.Aes, &pay.Aes, remaining)
		remaining -= tryTake(&pool.Aqua, &pay.Aqua, remaining)
		remaining -= tryTake(&pool.Ignus, &pay.Ignus, remaining)
		remaining -= tryTake(&pool.Luna, &pay.Luna, remaining)
		remaining -= tryTake(&pool.Silva, &pay.Silva, remaining)
		remaining -= tryTake(&pool.Solis, &pay.Solis, remaining)
		remaining -= tryTake(&pool.Terra, &pay.Terra, remaining)
		remaining -= tryTake(&pool.Void, &pay.Void, remaining)
		if remaining == before {
			return model.AetherPayment{}, false
		}
	}
	return pay, true
}

func isServant(def gamecards.Card) bool {
	return strings.EqualFold(strings.TrimSpace(def.Type), "Servant")
}

func isCaster(def gamecards.Card) bool {
	return strings.EqualFold(strings.TrimSpace(def.Type), "Caster")
}

func nameContains(def gamecards.Card, needle string) bool {
	return strings.Contains(strings.ToLower(def.Name), strings.ToLower(needle))
}

func recoveredFieldCards(player view.PlayerView) []view.CardView {
	out := make([]view.CardView, 0, len(player.CasterZone)+len(player.ServantZone))
	for _, card := range player.CasterZone {
		if card.MatchID != "" && card.Orientation == model.OrientationRecovered {
			out = append(out, card)
		}
	}
	for _, card := range player.ServantZone {
		if card.MatchID != "" && card.Orientation == model.OrientationRecovered {
			out = append(out, card)
		}
	}
	return out
}

// HandServantsCheapestFirst returns hand servants ordered by ascending cost.
func HandServantsCheapestFirst(hand []view.CardView, catalog Catalog) []view.CardView {
	type scored struct {
		card view.CardView
		cost int
	}
	var list []scored
	for _, card := range hand {
		if card.CardID == "" || catalog == nil {
			continue
		}
		def, ok := catalog.FindByID(string(card.CardID))
		if !ok || !isServant(def) {
			continue
		}
		list = append(list, scored{card: card, cost: cardCost(def)})
	}
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].cost < list[i].cost {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	out := make([]view.CardView, len(list))
	for i, s := range list {
		out[i] = s.card
	}
	return out
}

func weakestEnemyServant(enemy view.PlayerView) (view.CardView, bool) {
	var best view.CardView
	found := false
	for _, card := range enemy.ServantZone {
		if card.MatchID == "" {
			continue
		}
		if !found {
			best = card
			found = true
			continue
		}
		// Naive: prefer rested, else first MatchID order (already "weak" heuristic).
		if card.Orientation == model.OrientationRested && best.Orientation != model.OrientationRested {
			best = card
		}
	}
	return best, found
}

func canAttackPlayer(enemy view.PlayerView) bool {
	for _, card := range enemy.ServantZone {
		if card.Orientation == model.OrientationReversed {
			return false
		}
	}
	return true
}

func selfShouldAct(match view.MatchView) bool {
	viewer := match.ViewerID
	self := findPlayer(match, viewer)
	if self == nil {
		return false
	}
	if match.MatchStatus == model.StatusFinished {
		return false
	}
	if !self.OpeningHandFinalized {
		return true
	}
	if match.PendingBreak.PlayerID == viewer ||
		(match.PendingDraw.PlayerID == viewer && match.PendingDraw.Step != model.PendingDrawIdle) ||
		match.PendingChoice.PlayerID == viewer {
		return true
	}
	if match.Attack.Step == model.BattleStepAwaitingJudgment && match.Turn.ActivePlayer == viewer {
		return true
	}
	if match.PrioritySequenceOpen && match.PriorityHolder == viewer {
		return true
	}
	if match.Turn.ActivePlayer == viewer && !match.PrioritySequenceOpen {
		return true
	}
	if match.Turn.ActivePlayer == viewer && match.PrioritySequenceOpen && match.PriorityHolder == viewer {
		return true
	}
	return false
}
