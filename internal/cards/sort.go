package cards

import (
	"sort"
	"strconv"
	"strings"
)

func searchTypeRank(cardType string) int {
	switch normalizeText(cardType) {
	case "caster":
		return 0
	case "servant":
		return 1
	case "conjure":
		return 2
	case "barrier":
		return 3
	default:
		return 99
	}
}

func searchCostRank(cost string) int {
	value, err := strconv.Atoi(
		strings.TrimSpace(cost),
	)
	if err != nil {
		return 999
	}

	return value
}

// SortForSearch applies a deterministic search-result order:
//
//	Type → cost/level → name → card number → ID
func SortForSearch(matches []Card) {
	sort.SliceStable(
		matches,
		func(i int, j int) bool {
			left := matches[i]
			right := matches[j]

			leftType := searchTypeRank(left.Type)
			rightType := searchTypeRank(right.Type)

			if leftType != rightType {
				return leftType < rightType
			}

			leftCost := searchCostRank(left.CostLevel)
			rightCost := searchCostRank(right.CostLevel)

			if leftCost != rightCost {
				return leftCost < rightCost
			}

			leftName := normalizeText(left.Name)
			rightName := normalizeText(right.Name)

			if leftName != rightName {
				return leftName < rightName
			}

			if left.CardNumber != right.CardNumber {
				return left.CardNumber < right.CardNumber
			}

			return left.ID < right.ID
		},
	)
}