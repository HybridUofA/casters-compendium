package rules

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
)

func ValidateCastServant(state *model.MatchState, catalog CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment, entryOrientation model.CardOrientation) error {
	err := ValidateAddChaseLink(state, actingPlayerID, cardID, model.ChaseLinkCardPlay)
	if err != nil {
		return fmt.Errorf("error occurred when adding the chase link: %w", err)
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("active player must be currently acting player")
	}
	if state.Turn.Phase != model.PhaseMain {
		return fmt.Errorf("phase must be Main, is currently %q", state.Turn.Phase)
	}
	if len(state.ChaseLinks) > 0 {
		return fmt.Errorf("chase must be empty to cast a servant")
	}
	actingPlayerIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingPlayerIndex = index
			break
		}
	}
	if actingPlayerIndex == -1 {
		return fmt.Errorf("acting player cannot be found")
	}
	cardIndex := -1
	for index, handCardID := range state.Players[actingPlayerIndex].Hand {
		if handCardID == cardID {
			cardIndex = index
			break
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card %q was not found in the acting player's hand", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q was not found in card instances", cardID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("servant being cast must be owned by acting player")
	}
	if instance.CardCategory != model.CategoryPrintedCard {
		return fmt.Errorf("card must be a printed card, is %q", instance.CardCategory)
	}
	if strings.TrimSpace(string(instance.CardID)) == "" {
		return fmt.Errorf("card id cannot be empty")
	}
	definition, err := ResolveCardDefinition(catalog, instance.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(definition.Type), "Servant") {
		return fmt.Errorf("card must be a servant, is %q", definition.Type)
	}
	cost, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	if err != nil {
		return fmt.Errorf("servant cost %q is not a valid integer: %w", definition.CostLevel, err)
	}
	if cost < 0 {
		return fmt.Errorf("servant cost cannot be negative: %d", cost)
	}
	var requiredElement model.Element
	switch strings.ToLower(strings.TrimSpace(definition.Element)) {
	case "aes":
		requiredElement = model.ElementAes
	case "aqua":
		requiredElement = model.ElementAqua
	case "ignus":
		requiredElement = model.ElementIgnus
	case "luna":
		requiredElement = model.ElementLuna
	case "silva":
		requiredElement = model.ElementSilva
	case "solis":
		requiredElement = model.ElementSolis
	case "terra":
		requiredElement = model.ElementTerra
	case "void":
		requiredElement = model.ElementVoid
	default:
		return fmt.Errorf("element %q is either blank or unsupported", definition.Element)
	}
	err = ValidateAetherPayment(
		state.Players[actingPlayerIndex].Aether,
		payment,
		cost,
		requiredElement,
	)
	if err != nil {
		return fmt.Errorf("error paying for servant: %w", err)
	}
	if entryOrientation != model.OrientationRecovered && entryOrientation != model.OrientationReversed {
		return fmt.Errorf("servant must be in either recovered or reversed position, in %q", entryOrientation)
	}
	return nil
}

func ValidateCastConjure(state *model.MatchState, catalog CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment) error {
	err := ValidateAddChaseLink(state, actingPlayerID, cardID, model.ChaseLinkCardPlay)
	if err != nil {
		return fmt.Errorf("error occurred when adding the chase link: %w", err)
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("active player must be currently acting player")
	}
	if state.Turn.Phase != model.PhaseMain {
		return fmt.Errorf("phase must be Main, is currently %q", state.Turn.Phase)
	}
	if len(state.ChaseLinks) > 0 {
		return fmt.Errorf("chase must be empty to cast a conjure")
	}
	actingPlayerIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingPlayerIndex = index
			break
		}
	}
	if actingPlayerIndex == -1 {
		return fmt.Errorf("acting player cannot be found")
	}
	cardIndex := -1
	for index, handCardID := range state.Players[actingPlayerIndex].Hand {
		if handCardID == cardID {
			cardIndex = index
			break
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card %q was not found in the acting player's hand", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q was not found in card instances", cardID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("conjure being cast must be owned by acting player")
	}
	if instance.CardCategory != model.CategoryPrintedCard {
		return fmt.Errorf("card must be a printed card, is %q", instance.CardCategory)
	}
	if strings.TrimSpace(string(instance.CardID)) == "" {
		return fmt.Errorf("card id cannot be empty")
	}
	definition, err := ResolveCardDefinition(catalog, instance.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(definition.Type), "Conjure") {
		return fmt.Errorf("card must be a conjure, is %q", definition.Type)
	}
	cost, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	if err != nil {
		return fmt.Errorf("conjure cost %q is not a valid integer: %w", definition.CostLevel, err)
	}
	if cost < 0 {
		return fmt.Errorf("conjure cost cannot be negative: %d", cost)
	}
	var requiredElement model.Element
	switch strings.ToLower(strings.TrimSpace(definition.Element)) {
	case "aes":
		requiredElement = model.ElementAes
	case "aqua":
		requiredElement = model.ElementAqua
	case "ignus":
		requiredElement = model.ElementIgnus
	case "luna":
		requiredElement = model.ElementLuna
	case "silva":
		requiredElement = model.ElementSilva
	case "solis":
		requiredElement = model.ElementSolis
	case "terra":
		requiredElement = model.ElementTerra
	case "void":
		requiredElement = model.ElementVoid
	default:
		return fmt.Errorf("element %q is either blank or unsupported", definition.Element)
	}
	err = ValidateAetherPayment(
		state.Players[actingPlayerIndex].Aether,
		payment,
		cost,
		requiredElement,
	)
	if err != nil {
		return fmt.Errorf("error paying for conjure: %w", err)
	}
	return nil
}

func ValidateCastBarrier(state *model.MatchState, catalog CardCatalog, actingPlayerID model.PlayerID, cardID model.MatchCardID, payment model.AetherPayment) error {
	err := ValidateAddChaseLink(state, actingPlayerID, cardID, model.ChaseLinkCardPlay)
	if err != nil {
		return fmt.Errorf("error occurred when adding the chase link: %w", err)
	}
	if actingPlayerID != state.Turn.ActivePlayer {
		return fmt.Errorf("active player must be currently acting player")
	}
	if state.Turn.Phase != model.PhaseMain {
		return fmt.Errorf("phase must be Main, is currently %q", state.Turn.Phase)
	}
	if len(state.ChaseLinks) > 0 {
		return fmt.Errorf("chase must be empty to cast a barrier")
	}
	actingPlayerIndex := -1
	for index := range state.Players {
		if state.Players[index].ID == actingPlayerID {
			actingPlayerIndex = index
			break
		}
	}
	if actingPlayerIndex == -1 {
		return fmt.Errorf("acting player cannot be found")
	}
	cardIndex := -1
	for index, handCardID := range state.Players[actingPlayerIndex].Hand {
		if handCardID == cardID {
			cardIndex = index
			break
		}
	}
	if cardIndex == -1 {
		return fmt.Errorf("card %q was not found in the acting player's hand", cardID)
	}
	instance, exists := state.CardInstances[cardID]
	if !exists {
		return fmt.Errorf("card %q was not found in card instances", cardID)
	}
	if instance.Owner != actingPlayerID {
		return fmt.Errorf("barrier being cast must be owned by acting player")
	}
	if instance.CardCategory != model.CategoryPrintedCard {
		return fmt.Errorf("card must be a printed card, is %q", instance.CardCategory)
	}
	if strings.TrimSpace(string(instance.CardID)) == "" {
		return fmt.Errorf("card id cannot be empty")
	}
	definition, err := ResolveCardDefinition(catalog, instance.CardID)
	if err != nil {
		return fmt.Errorf("error resolving card definition: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(definition.Type), "Barrier") {
		return fmt.Errorf("card must be a barrier, is %q", definition.Type)
	}
	cost, err := strconv.Atoi(strings.TrimSpace(definition.CostLevel))
	if err != nil {
		return fmt.Errorf("barrier cost %q is not a valid integer: %w", definition.CostLevel, err)
	}
	if cost < 0 {
		return fmt.Errorf("barrier cost cannot be negative: %d", cost)
	}
	var requiredElement model.Element
	switch strings.ToLower(strings.TrimSpace(definition.Element)) {
	case "aes":
		requiredElement = model.ElementAes
	case "aqua":
		requiredElement = model.ElementAqua
	case "ignus":
		requiredElement = model.ElementIgnus
	case "luna":
		requiredElement = model.ElementLuna
	case "silva":
		requiredElement = model.ElementSilva
	case "solis":
		requiredElement = model.ElementSolis
	case "terra":
		requiredElement = model.ElementTerra
	case "void":
		requiredElement = model.ElementVoid
	default:
		return fmt.Errorf("element %q is either blank or unsupported", definition.Element)
	}
	err = ValidateAetherPayment(
		state.Players[actingPlayerIndex].Aether,
		payment,
		cost,
		requiredElement,
	)
	if err != nil {
		return fmt.Errorf("error paying for barrier: %w", err)
	}
	return nil
}
