package cardupdate

import (
	"sort"
	"strconv"
	"strings"

	gamecards "github.com/HybridUofA/casters-compendium/internal/game/cards"
)

// LegacyIDMigration records one historical catalog ID carried onto its
// current canonical printing during a source refresh.
type LegacyIDMigration struct {
	LegacyID    string
	CanonicalID string
}

// CarryForwardLegacyIDs preserves identifiers from an older normalized
// catalog when Speedrobo replaces a record ID. A case-insensitive artwork URL
// match is strongest; card number and gameplay data provide a conservative
// fallback. Unmatched removals remain removed instead of being guessed.
func CarryForwardLegacyIDs(
	previous []gamecards.Card,
	current []gamecards.Card,
) ([]gamecards.Card, []LegacyIDMigration) {
	result := make([]gamecards.Card, len(current))
	currentIDs := make(map[string]int, len(current))
	for index, card := range current {
		card = gamecards.NormalizeDefinition(card)
		card.LegacyIDs = append([]string(nil), card.LegacyIDs...)
		result[index] = card
		currentIDs[card.ID] = index
	}

	migrations := make([]LegacyIDMigration, 0)
	for _, oldCard := range previous {
		oldCard = gamecards.NormalizeDefinition(oldCard)
		if index, found := currentIDs[oldCard.ID]; found {
			result[index].LegacyIDs = append(result[index].LegacyIDs, oldCard.LegacyIDs...)
			continue
		}

		replacementIndex, found := replacementFor(oldCard, result)
		if !found {
			continue
		}

		result[replacementIndex].LegacyIDs = append(
			result[replacementIndex].LegacyIDs,
			oldCard.ID,
		)
		result[replacementIndex].LegacyIDs = append(
			result[replacementIndex].LegacyIDs,
			oldCard.LegacyIDs...,
		)
		migrations = append(migrations, LegacyIDMigration{
			LegacyID:    oldCard.ID,
			CanonicalID: result[replacementIndex].ID,
		})
	}

	for index := range result {
		result[index] = gamecards.NormalizeDefinition(result[index])
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].LegacyID < migrations[j].LegacyID
	})
	return result, migrations
}

func replacementFor(oldCard gamecards.Card, current []gamecards.Card) (int, bool) {
	bestIndex := -1
	bestScore := 99
	for index, candidate := range current {
		score, matches := replacementScore(oldCard, candidate)
		if !matches {
			continue
		}
		if bestIndex == -1 || score < bestScore ||
			(score == bestScore && preferredReplacement(candidate, current[bestIndex])) {
			bestIndex = index
			bestScore = score
		}
	}
	return bestIndex, bestIndex >= 0
}

func replacementScore(oldCard gamecards.Card, candidate gamecards.Card) (int, bool) {
	if normalized(oldCard.Name) != normalized(candidate.Name) ||
		normalized(oldCard.Type) != normalized(candidate.Type) ||
		normalized(oldCard.CostLevel) != normalized(candidate.CostLevel) {
		return 0, false
	}

	oldImage := normalized(oldCard.ImageURL)
	if oldImage != "" && oldImage == normalized(candidate.ImageURL) {
		return 0, true
	}
	oldNumber := normalized(oldCard.CardNumber)
	if oldNumber != "" && oldNumber == normalized(candidate.CardNumber) {
		return 1, true
	}
	if sameGameplayDefinition(oldCard, candidate) {
		return 2, true
	}
	return 0, false
}

func sameGameplayDefinition(left gamecards.Card, right gamecards.Card) bool {
	return normalized(left.Name) == normalized(right.Name) &&
		normalized(left.Subname) == normalized(right.Subname) &&
		normalized(left.Type) == normalized(right.Type) &&
		normalized(left.Element) == normalized(right.Element) &&
		normalized(left.Traits) == normalized(right.Traits) &&
		normalized(left.CostLevel) == normalized(right.CostLevel) &&
		normalized(left.Attack) == normalized(right.Attack) &&
		normalized(left.Defense) == normalized(right.Defense) &&
		normalized(left.Ability) == normalized(right.Ability)
}

func preferredReplacement(left gamecards.Card, right gamecards.Card) bool {
	leftRank := replacementVariantRank(left)
	rightRank := replacementVariantRank(right)
	if leftRank != rightRank {
		return leftRank < rightRank
	}

	leftID, leftErr := strconv.Atoi(strings.TrimSpace(left.ID))
	rightID, rightErr := strconv.Atoi(strings.TrimSpace(right.ID))
	if leftErr == nil && rightErr == nil && leftID != rightID {
		return leftID < rightID
	}
	return normalized(left.ID) < normalized(right.ID)
}

func replacementVariantRank(card gamecards.Card) int {
	imageURL := normalized(card.ImageURL)
	cardNumber := normalized(card.CardNumber)
	if strings.Contains(cardNumber, "alt") || strings.Contains(imageURL, "alternate-art") {
		return 2
	}
	if strings.Contains(imageURL, "-rare.") ||
		strings.Contains(imageURL, "-super-rare.") ||
		strings.Contains(imageURL, "-hyper-rare.") {
		return 1
	}
	for key, value := range card.ExtraFields {
		if normalized(key) == "rarity" && normalized(value) != "" {
			return 1
		}
	}
	return 0
}

func normalized(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
