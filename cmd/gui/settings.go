package main

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
func showSettingsDialog(window fyne.Window, guiApp fyne.App) {
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
		),
		func(saved bool) {
			if !saved {
				return
			}
			value := appearanceThemeValue(selection.Selected)
			guiApp.Preferences().SetString(appearanceThemePreferenceKey, value)
			applyAppearanceTheme(guiApp, value)
		},
		window,
	).Show()
}
