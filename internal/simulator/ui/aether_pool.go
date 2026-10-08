package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	sourceicons "github.com/HybridUofA/casters-compendium/internal/sources/icons"
)

const aetherIconSize float32 = 28

type aetherDisplayEntry struct {
	name     string
	element  string
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

func allAetherEntries(pool model.AetherPool) []aetherDisplayEntry {
	return []aetherDisplayEntry{
		{name: "Aes", element: "Aes", amount: pool.Aes, resource: aetherIconResources.aes},
		{name: "Aqua", element: "Aqua", amount: pool.Aqua, resource: aetherIconResources.aqua},
		{name: "Ignus", element: "Ignus", amount: pool.Ignus, resource: aetherIconResources.ignus},
		{name: "Luna", element: "Luna", amount: pool.Luna, resource: aetherIconResources.luna},
		{name: "Silva", element: "Silva", amount: pool.Silva, resource: aetherIconResources.silva},
		{name: "Solis", element: "Solis", amount: pool.Solis, resource: aetherIconResources.solis},
		{name: "Terra", element: "Terra", amount: pool.Terra, resource: aetherIconResources.terra},
		{name: "Void", element: "Void", amount: pool.Void, resource: aetherIconResources.void},
		{name: "Non-elemental", element: "NonElemental", amount: pool.NonElemental, resource: aetherIconResources.nonElemental},
	}
}

func visibleAetherEntries(pool model.AetherPool) []aetherDisplayEntry {
	entries := allAetherEntries(pool)
	visible := make([]aetherDisplayEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.amount > 0 {
			visible = append(visible, entry)
		}
	}
	return visible
}

func newAetherPoolDisplay(
	pool model.AetherPool,
	editable bool,
	onAdjust func(element string, delta int),
) fyne.CanvasObject {
	entries := visibleAetherEntries(pool)
	if editable {
		entries = allAetherEntries(pool)
	}
	if len(entries) == 0 {
		return canvas.NewText("Aether: 0", boardForeground)
	}

	objects := make([]fyne.CanvasObject, 0, len(entries)+1)
	label := canvas.NewText("Aether:", boardForeground)
	label.TextStyle = fyne.TextStyle{Bold: true}
	objects = append(objects, label)
	for _, entry := range entries {
		icon := canvas.NewImageFromResource(entry.resource)
		icon.FillMode = canvas.ImageFillContain
		icon.SetMinSize(fyne.NewSize(aetherIconSize, aetherIconSize))
		amount := canvas.NewText(fmt.Sprintf("%d", entry.amount), boardForeground)
		amount.TextStyle = fyne.TextStyle{Bold: true}
		row := container.NewHBox(icon, amount)
		if editable && onAdjust != nil {
			element := entry.element
			objects = append(objects, newZoneInteractLayer(
				row,
				func() { onAdjust(element, 1) },
				func(*fyne.PointEvent) { onAdjust(element, -1) },
			))
			continue
		}
		objects = append(objects, row)
	}
	return container.NewHBox(objects...)
}
