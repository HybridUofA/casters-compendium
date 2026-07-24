package deckbuilder

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	sourcebackgrounds "github.com/HybridUofA/casters-compendium/internal/sources/backgrounds"
)

const (
	appearanceThemePreferenceKey = "appearance.theme"
	appearanceThemeSystem        = "system"
	appearanceThemeLight         = "light"
	appearanceThemeDark          = "dark"
)

const (
	appearanceThemeSystemLabel = "System Default"
	appearanceThemeLightLabel  = "Light"
	appearanceThemeDarkLabel   = "Dark"
)

const (
	backgroundPreferenceKey = "appearance.background"
	backgroundNone          = "none"
	backgroundAcademyRift   = "academy-rift"
	backgroundCasterDuel    = "caster-duel"
)

const (
	backgroundNoneLabel        = "None"
	backgroundAcademyRiftLabel = "Academy Rift"
	backgroundCasterDuelLabel  = "Caster Duel"
)

var (
	academyRiftResource = fyne.NewStaticResource(
		"academy-rift.png",
		sourcebackgrounds.AcademyRiftPNG,
	)
	casterDuelResource = fyne.NewStaticResource(
		"caster-duel.png",
		sourcebackgrounds.CasterDuelPNG,
	)
)

// fixedVariantTheme delegates theme resources while forcing one color variant.
type fixedVariantTheme struct {
	base    fyne.Theme
	variant fyne.ThemeVariant
}

// Color returns the requested color from the fixed light or dark palette.
func (current fixedVariantTheme) Color(
	name fyne.ThemeColorName,
	_ fyne.ThemeVariant,
) color.Color {
	return current.base.Color(name, current.variant)
}

// Font delegates font selection to the standard Fyne theme.
func (current fixedVariantTheme) Font(style fyne.TextStyle) fyne.Resource {
	return current.base.Font(style)
}

// Icon delegates icon selection to the standard Fyne theme.
func (current fixedVariantTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return current.base.Icon(name)
}

// Size delegates size selection to the standard Fyne theme.
func (current fixedVariantTheme) Size(name fyne.ThemeSizeName) float32 {
	return current.base.Size(name)
}

// normalizeAppearanceTheme converts unknown stored values to the system default.
func normalizeAppearanceTheme(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case appearanceThemeLight:
		return appearanceThemeLight
	case appearanceThemeDark:
		return appearanceThemeDark
	default:
		return appearanceThemeSystem
	}
}

// appearanceThemeLabel converts a stored preference into its user-facing label.
func appearanceThemeLabel(value string) string {
	switch normalizeAppearanceTheme(value) {
	case appearanceThemeLight:
		return appearanceThemeLightLabel
	case appearanceThemeDark:
		return appearanceThemeDarkLabel
	default:
		return appearanceThemeSystemLabel
	}
}

// appearanceThemeValue converts a user-facing label into its stored preference.
func appearanceThemeValue(label string) string {
	switch strings.TrimSpace(label) {
	case appearanceThemeLightLabel:
		return appearanceThemeLight
	case appearanceThemeDarkLabel:
		return appearanceThemeDark
	default:
		return appearanceThemeSystem
	}
}

// normalizeBackground converts unknown stored values to the unobstructed default.
func normalizeBackground(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case backgroundAcademyRift:
		return backgroundAcademyRift
	case backgroundCasterDuel:
		return backgroundCasterDuel
	default:
		return backgroundNone
	}
}

// backgroundLabel converts a stored preference into its user-facing label.
func backgroundLabel(value string) string {
	switch normalizeBackground(value) {
	case backgroundAcademyRift:
		return backgroundAcademyRiftLabel
	case backgroundCasterDuel:
		return backgroundCasterDuelLabel
	default:
		return backgroundNoneLabel
	}
}

// backgroundValue converts a user-facing label into its stored preference.
func backgroundValue(label string) string {
	switch strings.TrimSpace(label) {
	case backgroundAcademyRiftLabel:
		return backgroundAcademyRift
	case backgroundCasterDuelLabel:
		return backgroundCasterDuel
	default:
		return backgroundNone
	}
}

// backgroundResource returns the bundled artwork selected by the preference.
func backgroundResource(value string) fyne.Resource {
	switch normalizeBackground(value) {
	case backgroundAcademyRift:
		return academyRiftResource
	case backgroundCasterDuel:
		return casterDuelResource
	default:
		return nil
	}
}

// wrapWithBackground layers optional cover-scaled artwork and a dark readability
// scrim behind one application screen.
func wrapWithBackground(content fyne.CanvasObject, value string) fyne.CanvasObject {
	resource := backgroundResource(value)
	if resource == nil {
		return content
	}

	image := canvas.NewImageFromResource(resource)
	image.FillMode = canvas.ImageFillCover
	image.ScaleMode = canvas.ImageScaleSmooth
	scrim := canvas.NewRectangle(color.NRGBA{A: 112})

	// Fyne's renderer traverses children only when the concrete object is a
	// *fyne.Container or a widget. Do not wrap this container in a custom type.
	return container.NewStack(image, scrim, content)
}

// setWindowContent applies the saved artwork to every application screen.
func setWindowContent(window fyne.Window, content fyne.CanvasObject) {
	value := fyne.CurrentApp().Preferences().StringWithFallback(
		backgroundPreferenceKey,
		backgroundNone,
	)
	window.SetContent(wrapWithBackground(content, value))
}

// applyAppearanceTheme installs the selected adaptive or fixed-variant theme.
func applyAppearanceTheme(guiApp fyne.App, value string) {
	base := theme.DefaultTheme()
	var selected fyne.Theme = base
	switch normalizeAppearanceTheme(value) {
	case appearanceThemeLight:
		selected = fixedVariantTheme{base: base, variant: theme.VariantLight}
	case appearanceThemeDark:
		selected = fixedVariantTheme{base: base, variant: theme.VariantDark}
	}
	guiApp.Settings().SetTheme(selected)
}

// loadAppearanceTheme restores the saved preference before the first window is created.
func loadAppearanceTheme(guiApp fyne.App) {
	value := guiApp.Preferences().StringWithFallback(
		appearanceThemePreferenceKey,
		appearanceThemeSystem,
	)
	applyAppearanceTheme(guiApp, value)
}

// showSettingsDialog lets the user save and immediately apply appearance settings.
func showSettingsDialog(window fyne.Window, guiApp fyne.App, onSaved func()) {
	selection := widget.NewRadioGroup(
		[]string{
			appearanceThemeSystemLabel,
			appearanceThemeLightLabel,
			appearanceThemeDarkLabel,
		},
		nil,
	)
	selection.Horizontal = true
	selection.Required = true
	selection.SetSelected(appearanceThemeLabel(
		guiApp.Preferences().StringWithFallback(
			appearanceThemePreferenceKey,
			appearanceThemeSystem,
		),
	))

	backgroundSelection := widget.NewRadioGroup(
		[]string{
			backgroundNoneLabel,
			backgroundAcademyRiftLabel,
			backgroundCasterDuelLabel,
		},
		nil,
	)
	backgroundSelection.Horizontal = true
	backgroundSelection.Required = true
	backgroundSelection.SetSelected(backgroundLabel(
		guiApp.Preferences().StringWithFallback(
			backgroundPreferenceKey,
			backgroundNone,
		),
	))

	dialog.NewCustomConfirm(
		"Settings",
		"Save",
		"Cancel",
		container.NewVBox(
			widget.NewLabelWithStyle(
				"Appearance",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			widget.NewLabel("Theme"),
			selection,
			widget.NewSeparator(),
			widget.NewLabelWithStyle(
				"Background",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			widget.NewLabel("Artwork"),
			backgroundSelection,
		),
		func(saved bool) {
			if !saved {
				return
			}
			value := appearanceThemeValue(selection.Selected)
			guiApp.Preferences().SetString(appearanceThemePreferenceKey, value)
			applyAppearanceTheme(guiApp, value)
			background := backgroundValue(backgroundSelection.Selected)
			guiApp.Preferences().SetString(backgroundPreferenceKey, background)
			if onSaved != nil {
				onSaved()
			}
		},
		window,
	).Show()
}
