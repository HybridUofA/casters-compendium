// Package catalog loads, indexes, and filters the locally stored card database.
package catalog

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

// Card aliases the shared game card model for catalog consumers.
type Card = gamecards.Card

type Repository struct {
	cards       []Card
	searchCards []Card
	byID        map[string]Card
	byName      map[string][]Card
}

type Filter struct {
	Name           string
	Elements       []string
	Types          []string
	Traits         []string
	Keywords       []string
	CostLevels     []string
	Expansions     []string
	IncludeTesting bool
}

// PrintingGroup collects cards with identical gameplay data but different
// physical printings. Preferred is the ordinary printing shown by default.
type PrintingGroup struct {
	Preferred Card
	Printings []Card
}

// printingIdentity contains only fields that affect how a card plays. Fields
// such as the card ID, set number, expansion, image, rarity, flavor text, and
// artist identify a physical printing and intentionally do not participate.
type printingIdentity struct {
	name      string
	subname   string
	cardType  string
	element   string
	traits    string
	costLevel string
	attack    string
	defense   string
	ability   string
}

// LoadFile decodes and validates a normalized card database from disk.
func LoadFile(path string) (*Repository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read card database %q: %w", path, err)
	}

	var cards []Card
	if err := json.Unmarshal(data, &cards); err != nil {
		return nil, fmt.Errorf("decode card database %q: %w", path, err)
	}

	repository, err := NewRepository(cards)
	if err != nil {
		return nil, fmt.Errorf(
			"build repository from %q: %w",
			path,
			err,
		)
	}

	return repository, nil
}

// NewRepository validates cards and builds the lookup indexes used by searches and deck resolution.
func NewRepository(cards []Card) (*Repository, error) {
	if len(cards) == 0 {
		return nil, fmt.Errorf("card list cannot be empty")
	}

	repository := &Repository{
		cards:  make([]Card, 0, len(cards)),
		byID:   make(map[string]Card, len(cards)),
		byName: make(map[string][]Card),
	}

	for index, card := range cards {
		card = gamecards.NormalizeDefinition(card)
		card.Name = strings.TrimSpace(card.Name)

		if card.ID == "" {
			return nil, fmt.Errorf(
				"card at index %d has no ID",
				index,
			)
		}

		if card.Name == "" {
			return nil, fmt.Errorf(
				"card %q has no name",
				card.ID,
			)
		}

		if _, exists := repository.byID[card.ID]; exists {
			return nil, fmt.Errorf(
				"duplicate card ID %q",
				card.ID,
			)
		}

		repository.cards = append(repository.cards, card)
		repository.byID[card.ID] = card

		nameKey := normalizeText(card.Name)

		repository.byName[nameKey] = append(
			repository.byName[nameKey],
			card,
		)
	}

	for _, card := range repository.cards {
		for _, legacyID := range card.LegacyIDs {
			if _, exists := repository.byID[legacyID]; exists {
				return nil, fmt.Errorf(
					"legacy card ID %q for %q conflicts with another card ID",
					legacyID,
					card.ID,
				)
			}
			repository.byID[legacyID] = card
		}
	}
	repository.searchCards = preferredSearchPrintings(repository.cards)

	return repository, nil
}

// preferredSearchPrintings collapses records that use the exact same image
// asset, keeping the lowest card number. Duplicate source records remain in the
// repository indexes so existing decks and explicit printing imports resolve.
func preferredSearchPrintings(cardList []Card) []Card {
	preferred := make([]Card, 0, len(cardList))
	imageIndexes := make(map[string]int)

	for _, card := range cardList {
		imageURL := strings.TrimSpace(card.ImageURL)
		if imageURL == "" {
			preferred = append(preferred, card)
			continue
		}

		index, found := imageIndexes[imageURL]
		if !found {
			imageIndexes[imageURL] = len(preferred)
			preferred = append(preferred, card)
			continue
		}

		current := preferred[index]
		cardNumber := normalizeText(card.CardNumber)
		currentNumber := normalizeText(current.CardNumber)
		if cardNumber != "" &&
			(currentNumber == "" || cardNumber < currentNumber) {
			preferred[index] = card
		}
	}

	return preferred
}

// normalizeText trims and case-folds text for case-insensitive matching.
func normalizeText(text string) string {
	return strings.ToLower(strings.TrimSpace(text))
}

// FindByID resolves one exact card identifier.
func (repository *Repository) FindByID(id string) (Card, bool) {
	card, found := repository.byID[strings.TrimSpace(id)]
	return card, found
}

// FindByName returns all printings with an exact case-insensitive name match.
func (repository *Repository) FindByName(name string) []Card {
	matches := repository.byName[normalizeText(name)]

	result := make([]Card, len(matches))
	copy(result, matches)

	return result
}

// SearchByName returns cards whose names contain a case-insensitive query.
func (repository *Repository) SearchByName(query string) []Card {
	normalizedQuery := normalizeText(query)

	if normalizedQuery == "" {
		return []Card{}
	}

	var matches []Card

	for _, card := range repository.searchCards {
		normalizedCardName := normalizeText(card.Name)

		if strings.Contains(normalizedCardName, normalizedQuery) {
			matches = append(matches, card)
		}
	}

	return matches
}

// matchesAnySelected reports whether a card field contains any selected filter value.
func matchesAnySelected(
	cardValue string,
	selectedValues []string,
) bool {
	normalizedCard := normalizeText(cardValue)

	for _, selected := range selectedValues {
		normalizedSelected := normalizeText(selected)

		if normalizedSelected == "" {
			continue
		}

		if strings.Contains(
			normalizedCard,
			normalizedSelected,
		) {
			return true
		}
	}

	return false
}

// Filter applies all populated search filters and playtesting visibility to the repository.
func (repository *Repository) Filter(options Filter) []Card {
	normalizedName := normalizeText(options.Name)

	var matches []Card

	for _, card := range repository.searchCards {
		if normalizedName != "" &&
			!strings.Contains(normalizeText(card.Name), normalizedName) {
			continue
		}

		if len(options.Elements) > 0 &&
			!matchesAnySelected(card.Element, options.Elements) {
			continue
		}

		if len(options.Types) > 0 &&
			!matchesAnySelected(card.Type, options.Types) {
			continue
		}

		if len(options.Traits) > 0 &&
			!matchesAnySelected(card.Traits, options.Traits) {
			continue
		}

		if len(options.Keywords) > 0 &&
			!matchesAnyKeyword(card.Ability, options.Keywords) {
			continue
		}

		if len(options.CostLevels) > 0 &&
			!matchesAnySelected(card.CostLevel, options.CostLevels) {
			continue
		}

		if len(options.Expansions) > 0 &&
			!matchesAnySelected(card.Expansion, options.Expansions) {
			continue
		}

		if card.IsPlaytesting && !options.IncludeTesting {
			continue
		}

		matches = append(matches, card)
	}

	return matches
}

// FilterPrintingGroups applies the ordinary card filters and then collapses
// gameplay-identical artwork into one search result. Every matching distinct
// artwork remains available in the returned group, including alternate art.
func (repository *Repository) FilterPrintingGroups(options Filter) []PrintingGroup {
	matches := repository.Filter(options)
	SortForSearch(matches)

	groups := make([]PrintingGroup, 0, len(matches))
	groupIndexes := make(map[printingIdentity]int, len(matches))

	for _, card := range matches {
		identity := gameplayIdentity(card)
		index, found := groupIndexes[identity]
		if !found {
			groupIndexes[identity] = len(groups)
			groups = append(groups, PrintingGroup{
				Preferred: card,
				Printings: []Card{card},
			})
			continue
		}

		groups[index].Printings = append(groups[index].Printings, card)
	}

	for index := range groups {
		printings := groups[index].Printings
		sort.SliceStable(printings, func(i, j int) bool {
			return preferredPrintingLess(printings[i], printings[j])
		})
		groups[index].Preferred = printings[0]
	}

	return groups
}

func gameplayIdentity(card Card) printingIdentity {
	return printingIdentity{
		name:      normalizeText(card.Name),
		subname:   normalizeText(card.Subname),
		cardType:  normalizeText(card.Type),
		element:   normalizeText(card.Element),
		traits:    normalizedTraitIdentity(card.Traits),
		costLevel: normalizeText(card.CostLevel),
		attack:    normalizeText(card.Attack),
		defense:   normalizeText(card.Defense),
		ability:   normalizeText(card.Ability),
	}
}

func normalizedTraitIdentity(value string) string {
	traits := splitTraits(value)
	for index := range traits {
		traits[index] = normalizeText(traits[index])
	}
	sort.Strings(traits)
	return strings.Join(traits, "\x00")
}

// preferredPrintingLess places an ordinary printing before explicitly marked
// alternate-art and rarity variants, then uses stable catalog metadata.
func preferredPrintingLess(left Card, right Card) bool {
	leftRank := printingVariantRank(left)
	rightRank := printingVariantRank(right)
	if leftRank != rightRank {
		return leftRank < rightRank
	}

	leftNumber := normalizeText(left.CardNumber)
	rightNumber := normalizeText(right.CardNumber)
	if leftNumber != rightNumber {
		return leftNumber < rightNumber
	}

	return normalizeText(left.ID) < normalizeText(right.ID)
}

func printingVariantRank(card Card) int {
	cardNumber := normalizeText(card.CardNumber)
	imageURL := normalizeText(card.ImageURL)
	if strings.Contains(cardNumber, "alt") || strings.Contains(imageURL, "alternate-art") {
		return 2
	}
	if strings.Contains(imageURL, "-rare.") ||
		strings.Contains(imageURL, "-super-rare.") ||
		strings.Contains(imageURL, "-hyper-rare.") {
		return 1
	}

	for key, value := range card.ExtraFields {
		if normalizeText(key) == "rarity" && normalizeText(value) != "" {
			return 1
		}
	}

	return 0
}

// All returns a defensive copy of every card in repository order.
func (repository *Repository) All() []Card {
	result := make([]Card, len(repository.cards))
	copy(result, repository.cards)

	return result
}

// uniqueSortedValues removes blank and case-insensitive duplicate values before sorting them.
func uniqueSortedValues(values []string) []string {
	unique := make(map[string]string)

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		key := normalizeText(value)
		if _, exists := unique[key]; !exists {
			unique[key] = value
		}
	}

	results := make([]string, 0, len(unique))

	for _, value := range unique {
		results = append(results, value)
	}

	sort.Strings(results)

	return results
}

// SortForSearch applies the shared deterministic card ordering to search results.
func SortForSearch(matches []Card) {
	gamecards.SortForSearch(matches)
}

// Elements returns the distinct element values available for filtering.
func (repository *Repository) Elements() []string {
	values := make([]string, 0, len(repository.cards))

	for _, card := range repository.cards {
		values = append(values, card.Element)
	}

	return uniqueSortedValues(values)
}

// Types returns the distinct card-type values available for filtering.
func (repository *Repository) Types() []string {
	values := make([]string, 0, len(repository.cards))

	for _, card := range repository.cards {
		values = append(values, card.Type)
	}

	return uniqueSortedValues(values)
}

// Traits returns distinct parsed trait names while preserving their first observed spelling.
func (repository *Repository) Traits() []string {
	unique := make(map[string]string)

	for _, card := range repository.cards {
		for _, trait := range splitTraits(card.Traits) {
			key := normalizeText(trait)

			if key == "" {
				continue
			}

			// Preserve the first spelling encountered.
			if _, exists := unique[key]; !exists {
				unique[key] = trait
			}
		}
	}

	traits := make(
		[]string,
		0,
		len(unique),
	)

	for _, trait := range unique {
		traits = append(traits, trait)
	}

	sort.Slice(
		traits,
		func(i, j int) bool {
			return strings.ToLower(traits[i]) <
				strings.ToLower(traits[j])
		},
	)

	return traits
}

// CostLevels returns the distinct cost or level values available for filtering.
func (repository *Repository) CostLevels() []string {
	values := make([]string, 0, len(repository.cards))

	for _, card := range repository.cards {
		values = append(values, card.CostLevel)
	}

	return uniqueSortedValues(values)
}

// Expansions returns the distinct expansion names available for filtering.
func (repository *Repository) Expansions() []string {
	values := make([]string, 0, len(repository.cards))

	for _, card := range repository.cards {
		values = append(values, card.Expansion)
	}

	return uniqueSortedValues(values)
}

// matchesAnyExact reports whether target equals one of values after normalization.
func matchesAnyExact(values []string, target string) bool {
	normalizedTarget := normalizeText(target)

	for _, value := range values {
		if normalizeText(value) == normalizedTarget {
			return true
		}
	}

	return false
}

// matchesAnyContained reports whether target contains one of values after normalization.
func matchesAnyContained(values []string, target string) bool {
	normalizedTarget := normalizeText(target)

	for _, value := range values {
		if strings.Contains(
			normalizedTarget,
			normalizeText(value),
		) {
			return true
		}
	}

	return false
}

// splitTraits parses bracketed traits first and falls back to comma or semicolon separation.
func splitTraits(value string) []string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil
	}

	traits := make([]string, 0)
	remaining := raw

	for {
		open := strings.Index(remaining, "[")
		if open == -1 {
			break
		}

		closeOffset := strings.Index(
			remaining[open+1:],
			"]",
		)
		if closeOffset == -1 {
			break
		}

		close := open + 1 + closeOffset

		trait := strings.TrimSpace(
			remaining[open+1 : close],
		)
		if trait != "" {
			traits = append(traits, trait)
		}
		remaining = remaining[close+1:]
	}
	if len(traits) > 0 {
		return traits
	}

	for _, trait := range strings.FieldsFunc(
		raw,
		func(r rune) bool {
			return r == ',' || r == ';'
		},
	) {
		trait = strings.TrimSpace(trait)

		if trait != "" {
			traits = append(traits, trait)
		}
	}

	return traits
}
