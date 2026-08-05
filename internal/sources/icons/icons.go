// Package icons embeds the elemental symbols used by simulator presentation.
package icons

import _ "embed"

var (
	//go:embed aes.png
	AesPNG []byte

	//go:embed aqua.png
	AquaPNG []byte

	//go:embed ignis.png
	IgnusPNG []byte

	//go:embed luna.png
	LunaPNG []byte

	//go:embed silva.png
	SilvaPNG []byte

	//go:embed solis.png
	SolisPNG []byte

	//go:embed terra_icon.png
	TerraPNG []byte

	//go:embed void.png
	VoidPNG []byte

	//go:embed non_elemental.png
	NonElementalPNG []byte
)
