package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	sourceicons "github.com/HybridUofA/casters-compendium/internal/sources/icons"
)

const aetherIconSize float32 = 28

type aetherDisplayEntry struct {
	name     string
	amount   int
	resource fyne.Resource
}

var aetherIconResources = struct {
	aes          fyne.Resource
	aqua         fyne.Resource
	ignus        fyne.Resource
	luna         fyne.Resource
	silva        fyne.Resource
	solis        fyne.Resource
	terra        fyne.Resource
	void         fyne.Resource
	nonElemental fyne.Resource
}{
	aes:          fyne.NewStaticResource("aether-aes.png", sourceicons.AesPNG),
	aqua:         fyne.NewStaticResource("aether-aqua.png", sourceicons.AquaPNG),
	ignus:        fyne.NewStaticResource("aether-ignus.png", sourceicons.IgnusPNG),
	luna:         fyne.NewStaticResource("aether-luna.png", sourceicons.LunaPNG),
	silva:        fyne.NewStaticResource("aether-silva.png", sourceicons.SilvaPNG),
	solis:        fyne.NewStaticResource("aether-solis.png", sourceicons.SolisPNG),
	terra:        fyne.NewStaticResource("aether-terra.png", sourceicons.TerraPNG),
	void:         fyne.NewStaticResource("aether-void.png", sourceicons.VoidPNG),
	nonElemental: fyne.NewStaticResource("aether-non-elemental.png", sourceicons.NonElementalPNG),
}

func visibleAetherEntries(pool model.AetherPool) []aetherDisplayEntry {
	entries := []aetherDisplayEntry{
		{name: "Aes", amount: pool.Aes, resource: aetherIconResources.aes},
		{name: "Aqua", amount: pool.Aqua, resource: aetherIconResources.aqua},
		{name: "Ignus", amount: pool.Ignus, resource: aetherIconResources.ignus},
		{name: "Luna", amount: pool.Luna, resource: aetherIconResources.luna},
		{name: "Silva", amount: pool.Silva, resource: aetherIconResources.silva},
		{name: "Solis", amount: pool.Solis, resource: aetherIconResources.solis},
		{name: "Terra", amount: pool.Terra, resource: aetherIconResources.terra},
		{name: "Void", amount: pool.Void, resource: aetherIconResources.void},
		{name: "Non-elemental", amount: pool.NonElemental, resource: aetherIconResources.nonElemental},
	}
	visible := make([]aetherDisplayEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.amount > 0 {
			visible = append(visible, entry)
		}
	}
	return visible
}

func newAetherPoolDisplay(pool model.AetherPool) fyne.CanvasObject {
	entries := visibleAetherEntries(pool)
	if len(entries) == 0 {
		label := widget.NewLabel("Aether: 0")
		label.Importance = widget.LowImportance
		return label
	}

	objects := make([]fyne.CanvasObject, 0, len(entries)+1)
	label := widget.NewLabel("Aether:")
	label.Importance = widget.LowImportance
	objects = append(objects, label)
	for _, entry := range entries {
		icon := canvas.NewImageFromResource(entry.resource)
		icon.FillMode = canvas.ImageFillContain
		icon.SetMinSize(fyne.NewSize(aetherIconSize, aetherIconSize))
		amount := widget.NewLabel(fmt.Sprintf("%d", entry.amount))
		amount.Importance = widget.LowImportance
		objects = append(objects, container.NewHBox(icon, amount))
	}
	return container.NewHBox(objects...)
}
