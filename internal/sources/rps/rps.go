// Package rps embeds Rock / Paper / Scissors choice art for simulator PendingChoice UI.
package rps

import _ "embed"

var (
	//go:embed rock.jpg
	RockJPG []byte

	//go:embed paper.jpg
	PaperJPG []byte

	//go:embed scissors.jpg
	ScissorsJPG []byte
)
