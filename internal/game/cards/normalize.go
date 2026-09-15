package cards

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	casterLevelSuffix          = regexp.MustCompile(`(?i)\s+lv\s*([0-9]+)\s*$`)
	decoratedCasterLevelSuffix = regexp.MustCompile(
		`(?i)\s+(?:(?:super|hyper)\s+)?lv\s*([0-9]+)(?:\s+(?:super|hyper))?\s+rare\s*$`,
	)
	printingVariantSuffix = regexp.MustCompile(
		`(?i)\s+(?:alternate art|sample|(?:(?:super|hyper)\s+)?rare)\s*$`,
	)
)

// NormalizeSourceName removes labels that Speedrobo uses to distinguish a
// level or physical printing but which are not part of the card's rules name.
// It is suitable for comparing source summaries, which do not include the
// complete Type and Cost/Lv fields available to NormalizeDefinition.
func NormalizeSourceName(name string) string {
	name = strings.TrimSpace(name)
	name = decoratedCasterLevelSuffix.ReplaceAllString(name, "")
	name = casterLevelSuffix.ReplaceAllString(name, "")
	name = printingVariantSuffix.ReplaceAllString(name, "")
	return strings.TrimSpace(name)
}

// DecklistName returns the card name used by Speedrobo-compatible text
// decklists. Caster levels above one are part of that interchange name even
// though NormalizeDefinition removes the source-only level suffix from the
// rules name stored in the shared catalog.
func DecklistName(card Card) string {
	card = NormalizeDefinition(card)
	name := strings.TrimSpace(card.Name)
	if !strings.EqualFold(strings.TrimSpace(card.Type), "caster") {
		return name
	}

	level, err := strconv.Atoi(strings.TrimSpace(card.CostLevel))
	if err != nil || level <= 1 {
		return name
	}
	return name + " Lv" + strconv.Itoa(level)
}

// NormalizeDefinition canonicalizes source-specific representation details
// before a printed card definition is used by deck or simulator rules.
func NormalizeDefinition(card Card) Card {
	card.ID = strings.TrimSpace(card.ID)
	seenIDs := map[string]struct{}{card.ID: {}}
	legacyIDs := make([]string, 0, len(card.LegacyIDs))
	for _, legacyID := range card.LegacyIDs {
		legacyID = strings.TrimSpace(legacyID)
		if legacyID == "" {
			continue
		}
		if _, duplicate := seenIDs[legacyID]; duplicate {
			continue
		}
		seenIDs[legacyID] = struct{}{}
		legacyIDs = append(legacyIDs, legacyID)
	}
	sort.Strings(legacyIDs)
	card.LegacyIDs = legacyIDs

	card.Name = strings.TrimSpace(card.Name)
	if strings.EqualFold(strings.TrimSpace(card.Type), "caster") {
		level, err := strconv.Atoi(strings.TrimSpace(card.CostLevel))
		if err == nil && level > 1 {
			for _, suffix := range []*regexp.Regexp{
				decoratedCasterLevelSuffix,
				casterLevelSuffix,
			} {
				match := suffix.FindStringSubmatch(card.Name)
				if len(match) != 2 {
					continue
				}
				suffixLevel, parseErr := strconv.Atoi(match[1])
				if parseErr == nil && suffixLevel == level {
					card.Name = strings.TrimSpace(card.Name[:len(card.Name)-len(match[0])])
					break
				}
			}
		}
	}

	// Alternate-art, sample, and rarity labels identify a printing rather than
	// forming part of the rules name. Printing identity remains available via
	// the card ID, card number, image, expansion, and rarity metadata.
	card.Name = strings.TrimSpace(printingVariantSuffix.ReplaceAllString(card.Name, ""))
	return card
}
