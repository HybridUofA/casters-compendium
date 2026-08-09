package cards

import (
	"regexp"
	"strconv"
	"strings"
)

var casterLevelSuffix = regexp.MustCompile(`(?i)\s+lv\s*([0-9]+)\s*$`)

// NormalizeDefinition canonicalizes source-specific representation details
// before a printed card definition is used by deck or simulator rules.
func NormalizeDefinition(card Card) Card {
	if !strings.EqualFold(strings.TrimSpace(card.Type), "caster") {
		return card
	}
	level, err := strconv.Atoi(strings.TrimSpace(card.CostLevel))
	if err != nil || level <= 1 {
		return card
	}
	match := casterLevelSuffix.FindStringSubmatch(card.Name)
	if len(match) != 2 {
		return card
	}
	suffixLevel, err := strconv.Atoi(match[1])
	if err != nil || suffixLevel != level {
		return card
	}

	// The source sometimes repeats the level in the name even though the card
	// image prints it separately in the Lv emblem.
	card.Name = strings.TrimSpace(card.Name[:len(card.Name)-len(match[0])])
	return card
}
