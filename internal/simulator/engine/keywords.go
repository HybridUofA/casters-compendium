package engine

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func cardDeclaresKeyword(ability string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, line := range strings.Split(ability, "\n") {
		keyword, found := abilityKeywordLabel(line)
		if found && strings.EqualFold(keyword, want) {
			return true
		}
	}
	return false
}

func cardDeclaresBreak(ability string) bool {
	return cardDeclaresKeyword(ability, "Break")
}

func cardDeclaresDoubleCorrupt(ability string) bool {
	return cardDeclaresKeyword(ability, "Double Corrupt")
}

// abilityKeywordLabel mirrors the catalog keyword extractor for rules labels.
func abilityKeywordLabel(line string) (string, bool) {
	line = strings.TrimSpace(line)
	line = strings.TrimSpace(strings.TrimLeft(line, "•*-"))
	if line == "" {
		return "", false
	}
	if strings.HasPrefix(line, "[") {
		if closing := strings.IndexRune(line, ']'); closing > 1 {
			return validAbilityKeyword(line[1:closing])
		}
	}
	separator := strings.IndexAny(line, "(:,→")
	if separator >= 0 {
		return validAbilityKeyword(line[:separator])
	}
	if strings.ContainsAny(line, ".!?;") {
		return "", false
	}
	return validAbilityKeyword(line)
}

func validAbilityKeyword(candidate string) (string, bool) {
	candidate = strings.Join(strings.Fields(candidate), " ")
	if candidate == "" || utf8.RuneCountInString(candidate) > 40 {
		return "", false
	}
	words := strings.Fields(candidate)
	if len(words) == 0 || len(words) > 5 {
		return "", false
	}
	for _, r := range candidate {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '\'' {
			continue
		}
		return "", false
	}
	if len(words) > 1 {
		for _, word := range words {
			first, _ := utf8.DecodeRuneInString(word)
			if !unicode.IsUpper(first) && !unicode.IsDigit(first) {
				return "", false
			}
		}
	}
	return candidate, true
}
