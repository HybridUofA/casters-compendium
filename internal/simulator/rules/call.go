package rules

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateFaceDownLevelOneCall(
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
	if state.Turn.Phase != model.PhaseCall {
		return fmt.Errorf("phase is %q, expected %q", state.Turn.Phase, model.PhaseCall)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("acting player %q is not the active player %q", actingPlayerID, state.Turn.ActivePlayer)
	}
	if state.Turn.CallActionTaken {
		return fmt.Errorf("call action has already been taken")
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
	cardIndex := -1
	for index := range state.Players[activeIndex].Hand {
		if state.Players[activeIndex].Hand[index] == cardID {
			cardIndex = index
		}
	}
	if _, exists := state.CardInstances[cardID]; !exists {
		return fmt.Errorf("card ID %q not found in card instances", cardID)
	}
	if cardIndex == -1 {
		return fmt.Errorf("card not found in hand")
	}
	return nil
}

func ValidateFaceUpLevelOneCall(
	state *model.MatchState,
	catalog CardCatalog,
	actingPlayerID model.PlayerID,
	cardID model.MatchCardID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("state in %q state, expected %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Phase != model.PhaseCall {
		return fmt.Errorf("phase is %q, expected %q", state.Turn.Phase, model.PhaseCall)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("acting player %q is not the active player %q", actingPlayerID, state.Turn.ActivePlayer)
	}
	if state.Turn.CallActionTaken {
		return fmt.Errorf("call action has already been taken")
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
	cardIndex := -1
	for index := range state.Players[activeIndex].Hand {
		if state.Players[activeIndex].Hand[index] == cardID {
			cardIndex = index
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card not found in hand")
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card ID %q not found in card instances", cardID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("card %q is not owned by %q", cardID, actingPlayerID)
	}
	if instance.CardCategory != model.CategoryPrintedCard {
		return fmt.Errorf("card is not a printed card")
	}
	definition, err := ResolveCardDefinition(catalog, instance.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(string(definition.Type))) != "caster" {
		return fmt.Errorf("card %q is not a caster", cardID)
	}
	level, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	if err != nil {
		return fmt.Errorf("error level is malformed: %q", definition.CostLevel)
	}
	if level != 1 {
		return fmt.Errorf("card level must be 1, is %d", level)
	}
	for _, existingCardID := range state.Players[activeIndex].CasterZone {
		instance, exists := state.CardInstances[existingCardID]
		if !exists {
			return fmt.Errorf("card cannot be resolved: %q", existingCardID)
		}
		if instance.CardCategory == model.CategoryTokenCard && instance.CardID == model.CasterTokenCardID {
			continue
		}
		if instance.CardCategory != model.CategoryPrintedCard {
			return fmt.Errorf("card not a printed card: %q", instance.CardCategory)
		}
		if instance.Controller != actingPlayerID {
			return fmt.Errorf("card controller %q is not acting player %q", instance.Controller, actingPlayerID)
		}
		if instance.Face == model.CardFaceDown {
			continue
		}
		if instance.Face != model.CardFaceUp {
			return fmt.Errorf("invalid face: %q", instance.Face)
		}
		card, err := ResolveCardDefinition(catalog, instance.CardID)
		if err != nil {
			return fmt.Errorf("error resolving definition: %w", err)
		}
		if normalizeCasterIdentity(card.Type) != "caster" {
			return fmt.Errorf("card in an invalid zone for its type: %q", card.Type)
		}
		conflict := casterDefinitionsConflict(definition, card)
		if conflict {
			return fmt.Errorf("cannot call, card already in zone")
		}
	}
	return nil
}

func ValidateLevelUpCaster(
	state *model.MatchState,
	catalog CardCatalog,
	actingPlayerID model.PlayerID,
	upperCardID model.MatchCardID,
	targetCasterID model.MatchCardID,
) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.MatchStatus != model.StatusInProgress {
		return fmt.Errorf("state in %q state, expected %q", state.MatchStatus, model.StatusInProgress)
	}
	if state.Turn.Phase != model.PhaseCall {
		return fmt.Errorf("phase is %q, expected %q", state.Turn.Phase, model.PhaseCall)
	}
	if strings.TrimSpace(string(actingPlayerID)) == "" {
		return fmt.Errorf("player ID cannot be empty")
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("acting player %q is not the active player %q", actingPlayerID, state.Turn.ActivePlayer)
	}
	if state.Turn.CallActionTaken {
		return fmt.Errorf("call action has already been taken")
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
	cardIndex := -1
	for index := range state.Players[activeIndex].Hand {
		if state.Players[activeIndex].Hand[index] == upperCardID {
			cardIndex = index
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card not found in hand")
	}
	instance, exists := state.CardInstances[upperCardID]
	if !exists {
		return fmt.Errorf("card ID %q not found in card instances", upperCardID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("card %q is not owned by %q", upperCardID, actingPlayerID)
	}
	if instance.CardCategory != model.CategoryPrintedCard {
		return fmt.Errorf("card is not a printed card")
	}
	if len(instance.Stock) != 0 {
		return fmt.Errorf("cards in hand cannot have Stock")
	}
	definition, err := ResolveCardDefinition(catalog, instance.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(string(definition.Type))) != "caster" {
		return fmt.Errorf("card %q is not a caster", upperCardID)
	}
	upperLevel, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	if err != nil {
		return fmt.Errorf("error level is malformed: %q", definition.CostLevel)
	}
	player := state.Players[activeIndex]
	if !slices.Contains(player.CasterZone, targetCasterID) {
		return fmt.Errorf("target caster %q is not in the acting player's Caster Zone", targetCasterID)
	}
	target, exists := state.CardInstances[targetCasterID]
	if !exists {
		return fmt.Errorf("card cannot be resolved: %q", targetCasterID)
	}
	if target.CardCategory != model.CategoryPrintedCard {
		return fmt.Errorf("card cannot be a token")
	}
	if target.Controller != actingPlayerID {
		return fmt.Errorf("caster not controlled by active player")
	}
	if target.Face != model.CardFaceUp {
		return fmt.Errorf("card must be face-up to level up")
	}
	targetDefinition, err := ResolveCardDefinition(catalog, target.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if normalizeCasterIdentity(string(targetDefinition.Type)) != "caster" {
		return fmt.Errorf("card %q must be a caster", targetDefinition.Type)
	}
	upperName := normalizeCasterIdentity(string(definition.Name))
	targetName := normalizeCasterIdentity(string(targetDefinition.Name))
	if targetName != upperName || targetName == "" || upperName == "" {
		return fmt.Errorf("caster %q and %q must share a name", targetDefinition.Name, definition.Name)
	}
	targetLevel, err := strconv.Atoi(strings.TrimSpace(targetDefinition.CostLevel))
	if err != nil {
		return fmt.Errorf("error parsing target caster level: %w", err)
	}
	if upperLevel != targetLevel+1 {
		return fmt.Errorf("cannot level up - %d must be one level higher than the target level %d", upperLevel, targetLevel)
	}
	return nil
}

func normalizeCasterIdentity(value string) string {
	return strings.ToLower(strings.TrimSpace(string(value)))
}

func casterTraitSet(value string) map[string]struct{} {
	traits := make(map[string]struct{})

	cleaned := strings.TrimSpace(value)
	cleaned = strings.TrimPrefix(cleaned, "[")
	cleaned = strings.TrimSuffix(cleaned, "]")
	for _, rawTrait := range strings.Split(cleaned, "/") {
		trait := normalizeCasterIdentity(rawTrait)
		if trait != "" {
			traits[trait] = struct{}{}
		}
	}
	return traits
}

func casterDefinitionsConflict(
	selected gamecards.Card,
	existing gamecards.Card,
) bool {
	selectedName := normalizeCasterIdentity(selected.Name)
	existingName := normalizeCasterIdentity(existing.Name)
	selectedSubname := normalizeCasterIdentity(selected.Subname)
	existingSubname := normalizeCasterIdentity(existing.Subname)
	selectedTraits := casterTraitSet(selected.Traits)
	existingTraits := casterTraitSet(existing.Traits)
	if selectedName != "" && selectedName == existingName && selectedSubname == existingSubname && maps.Equal(selectedTraits, existingTraits) {
		return true
	}
	return false
}
