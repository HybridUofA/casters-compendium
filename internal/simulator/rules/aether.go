package rules

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateGenerateNonElementalAether(
	state *model.MatchState,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("state in %q state, expected %q", state.MatchStatus, model.StatusInProgress)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	activeIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			activeIndex = index
		}
	}
	if activeIndex == -1 {
		return fmt.Errorf("acting player ID not found")
	}
	if exists := slices.Contains(state.Players[activeIndex].CasterZone, cardID); !exists {
		return fmt.Errorf("card must exist in caster zone: %q", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q does not exist in card instances", cardID)
	}
	if instance.Face != model.CardFaceDown {
		return fmt.Errorf("cannot generate non-elemental Aether: card is face up")
	}
	if instance.Orientation != model.OrientationRecovered {
		return fmt.Errorf("cannot rest an already rested caster")
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("cannot rest casters you do not control")
	}
	return nil
}

func ValidateUseCasterToken(state *model.MatchState, actingPlayerID model.PlayerID, tokenID model.MatchCardID) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("match status is %q, must be %q", state.MatchStatus, model.StatusInProgress)
	}
	actingIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingIndex = index
		}
	}
	if actingIndex == -1 {
		return fmt.Errorf("player %q not found in players", actingPlayerID)
	}
	player := &state.Players[actingIndex]
	cardIndex := slices.Index(player.CasterZone, tokenID)
	if cardIndex == -1 {
		return fmt.Errorf("caster token not found in caster zone")
	}
	instance, exists := state.CardInstances[tokenID]
	if !exists {
		return fmt.Errorf("caster token %q not found", tokenID)
	}
	if instance.CardID != model.CasterTokenCardID {
		return fmt.Errorf("card %q is not a caster token", tokenID)
	}
	if instance.CardCategory != model.CategoryTokenCard {
		return fmt.Errorf("card %q is not a token", tokenID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("owner %q is not active player %q", instance.Owner, actingPlayerID)
	}
	if instance.Controller != actingPlayerID {
		return fmt.Errorf("controller %q is not active player %q", instance.Controller, actingPlayerID)
	}
	if instance.Orientation != model.OrientationRecovered {
		return fmt.Errorf("caster token must be recovered")
	}
	if instance.Face != model.CardFaceUp {
		return fmt.Errorf("caster token must be face-up")
	}
	return nil
}

func CalculateCasterAether(catalog CardCatalog, cardID model.CardID) (model.Element, int, error) {
	card, err := ResolveCardDefinition(catalog, cardID)
	if err != nil {
		return "", 0, fmt.Errorf("card not found in catalog: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(string(card.Type))) != "caster" {
		return "", 0, fmt.Errorf("card is not a caster")
	}
	level, err := strconv.Atoi(strings.TrimSpace(card.CostLevel))
	if err != nil {
		return "", 0, fmt.Errorf("card level invalid: %w", err)
	}
	if level < 1 {
		return "", 0, fmt.Errorf("card level must be positive: %d", level)
	}
	element := strings.ToLower(strings.TrimSpace(card.Element))
	switch element {
	case "aes":
		return model.ElementAes, level, nil
	case "aqua":
		return model.ElementAqua, level, nil
	case "ignus":
		return model.ElementIgnus, level, nil
	case "luna":
		return model.ElementLuna, level, nil
	case "silva":
		return model.ElementSilva, level, nil
	case "solis":
		return model.ElementSolis, level, nil
	case "terra":
		return model.ElementTerra, level, nil
	case "void":
		return model.ElementVoid, level, nil
	default:
		return "", 0, fmt.Errorf("element blank or unknown: %q", card.Element)
	}
}

func ValidateGenerateCasterAether(state *model.MatchState, catalog CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID) (model.Element, int, error) {
	if state == nil {
		return "", 0, fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return "", 0, fmt.Errorf("status is %q, must be %q", state.MatchStatus, model.StatusInProgress)
	}
	actingIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingIndex = index
		}
	}
	if actingIndex == -1 {
		return "", 0, fmt.Errorf("player %q not found in players", actingPlayerID)
	}
	player := &state.Players[actingIndex]
	cardIndex := slices.Index(player.CasterZone, cardID)
	if cardIndex == -1 {
		return "", 0, fmt.Errorf("%q not found in caster zone", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return "", 0, fmt.Errorf("caster %q not found", cardID)
	}
	if instance.CardID == model.CasterTokenCardID {
		return "", 0, fmt.Errorf("card %q is a caster token", cardID)
	}
	if instance.CardCategory != model.CategoryPrintedCard {
		return "", 0, fmt.Errorf("card %q is not a printed card", cardID)
	}
	if instance.Controller != actingPlayerID {
		return "", 0, fmt.Errorf("controller %q is not acting player %q", instance.Controller, actingPlayerID)
	}
	if instance.Orientation != model.OrientationRecovered {
		return "", 0, fmt.Errorf("caster %q must be recovered", cardID)
	}
	if instance.Face != model.CardFaceUp {
		return "", 0, fmt.Errorf("caster %q must be face-up", cardID)
	}
	element, amount, err := CalculateCasterAether(catalog, instance.CardID)
	if err != nil {
		return "", 0, fmt.Errorf("error calculating aether generated: %w", err)
	}
	return element, amount, nil
}

func ValidateAetherPayment(pool model.AetherPool, payment model.AetherPayment, cost int, reqElem model.Element) error {
	total := 0
	amounts := []struct {
		name      string
		amount    int
		available int
	}{
		{name: "Aes", amount: payment.Aes, available: pool.Aes},
		{name: "Aqua", amount: payment.Aqua, available: pool.Aqua},
		{name: "Ignus", amount: payment.Ignus, available: pool.Ignus},
		{name: "Luna", amount: payment.Luna, available: pool.Luna},
		{name: "Silva", amount: payment.Silva, available: pool.Silva},
		{name: "Solis", amount: payment.Solis, available: pool.Solis},
		{name: "Terra", amount: payment.Terra, available: pool.Terra},
		{name: "Void", amount: payment.Void, available: pool.Void},
		{name: "NonElemental", amount: payment.NonElemental, available: pool.NonElemental},
	}
	if cost < 0 {
		return fmt.Errorf("cost cannot be negative")
	}
	for _, entry := range amounts {
		if entry.amount < 0 {
			return fmt.Errorf("%s payment cannot be negative", entry.name)
		}
		if entry.amount > entry.available {
			return fmt.Errorf("%s payment of %d exceeds available amount %d", entry.name, entry.amount, entry.available)
		}
		total += entry.amount
	}
	if total != cost {
		return fmt.Errorf("payment total is %d, expected exactly %d", total, cost)
	}
	switch reqElem {
	case model.ElementAes:
		if cost > 0 && payment.Aes < 1 {
			return fmt.Errorf("payment requires 1 Aes aether, has %d", payment.Aes)
		}
	case model.ElementAqua:
		if cost > 0 && payment.Aqua < 1 {
			return fmt.Errorf("payment requires 1 Aqua aether, has %d", payment.Aqua)
		}
	case model.ElementIgnus:
		if cost > 0 && payment.Ignus < 1 {
			return fmt.Errorf("payment requires 1 Ignus aether, has %d", payment.Ignus)
		}
	case model.ElementLuna:
		if cost > 0 && payment.Luna < 1 {
			return fmt.Errorf("payment requires 1 Luna aether, has %d", payment.Luna)
		}
	case model.ElementSilva:
		if cost > 0 && payment.Silva < 1 {
			return fmt.Errorf("payment requires 1 Silva aether, has %d", payment.Silva)
		}
	case model.ElementSolis:
		if cost > 0 && payment.Solis < 1 {
			return fmt.Errorf("payment requires 1 Solis aether, has %d", payment.Solis)
		}
	case model.ElementTerra:
		if cost > 0 && payment.Terra < 1 {
			return fmt.Errorf("payment requires 1 Terra aether, has %d", payment.Terra)
		}
	case model.ElementVoid:
		if cost > 0 && payment.Void < 1 {
			return fmt.Errorf("payment requires 1 Void aether, has %d", payment.Void)
		}
	default:
		return fmt.Errorf("Unknown element: %s", reqElem)
	}
	return nil
}
