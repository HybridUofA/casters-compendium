// Package releasenotes exposes the reviewed release notes as bundled,
// offline application data.
package releasenotes

import (
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Note is one versioned Markdown release note.
type Note struct {
	Version  string
	Markdown string
}

//go:embed *.md
var noteFiles embed.FS

// All returns every bundled release note in descending semantic-version order.
func All() ([]Note, error) {
	entries, err := noteFiles.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("read bundled release notes: %w", err)
	}
	notes := make([]Note, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		markdown, readErr := noteFiles.ReadFile(entry.Name())
		if readErr != nil {
			return nil, fmt.Errorf("read bundled release note %q: %w", entry.Name(), readErr)
		}
		notes = append(notes, Note{
			Version:  strings.TrimSuffix(entry.Name(), ".md"),
			Markdown: string(markdown),
		})
	}
	sort.SliceStable(notes, func(left, right int) bool {
		return compareVersions(notes[left].Version, notes[right].Version) > 0
	})
	return notes, nil
}

type parsedVersion struct {
	core       [3]int
	prerelease []string
}

func parseVersion(value string) (parsedVersion, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.SplitN(value, "-", 2)
	coreParts := strings.Split(parts[0], ".")
	if len(coreParts) != 3 {
		return parsedVersion{}, false
	}
	var parsed parsedVersion
	for index, part := range coreParts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return parsedVersion{}, false
		}
		parsed.core[index] = number
	}
	if len(parts) == 2 && parts[1] != "" {
		parsed.prerelease = strings.Split(parts[1], ".")
	}
	return parsed, true
}

func compareVersions(left, right string) int {
	leftVersion, leftOK := parseVersion(left)
	rightVersion, rightOK := parseVersion(right)
	if !leftOK || !rightOK {
		return strings.Compare(left, right)
	}
	for index := range leftVersion.core {
		if leftVersion.core[index] != rightVersion.core[index] {
			if leftVersion.core[index] > rightVersion.core[index] {
				return 1
			}
			return -1
		}
	}
	if len(leftVersion.prerelease) == 0 || len(rightVersion.prerelease) == 0 {
		if len(leftVersion.prerelease) == len(rightVersion.prerelease) {
			return 0
		}
		if len(leftVersion.prerelease) == 0 {
			return 1
		}
		return -1
	}
	limit := min(len(leftVersion.prerelease), len(rightVersion.prerelease))
	for index := 0; index < limit; index++ {
		leftPart := leftVersion.prerelease[index]
		rightPart := rightVersion.prerelease[index]
		leftNumber, leftNumeric := numericIdentifier(leftPart)
		rightNumber, rightNumeric := numericIdentifier(rightPart)
		switch {
		case leftNumeric && rightNumeric && leftNumber != rightNumber:
			if leftNumber > rightNumber {
				return 1
			}
			return -1
		case leftNumeric != rightNumeric:
			if leftNumeric {
				return -1
			}
			return 1
		case leftPart != rightPart:
			return strings.Compare(leftPart, rightPart)
		}
	}
	if len(leftVersion.prerelease) == len(rightVersion.prerelease) {
		return 0
	}
	if len(leftVersion.prerelease) > len(rightVersion.prerelease) {
		return 1
	}
	return -1
}

func numericIdentifier(value string) (int, bool) {
	number, err := strconv.Atoi(value)
	return number, err == nil
}
