// Package backgrounds contains artwork bundled with the desktop application.
package backgrounds

import _ "embed"

var (
	// AcademyRiftPNG is the academy courtyard and dimensional-rift background.
	//
	//go:embed caster-chronicles-background-1.png
	AcademyRiftPNG []byte

	// CasterDuelPNG is the two-Caster illustrated background.
	//
	//go:embed caster-chronicles-background-2.png
	CasterDuelPNG []byte
)
