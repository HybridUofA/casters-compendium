package deckbuilder

import (
	"image/color"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	deckexport "github.com/HybridUofA/casters-compendium/internal/deckbuilder/export"
	"github.com/HybridUofA/casters-compendium/internal/simulator/model"
	simulatorview "github.com/HybridUofA/casters-compendium/internal/simulator/view"
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

const (
	simulatorCardBackPreferenceKey    = "simulator.card_back_url"
	simulatorCasterTokenPreferenceKey = "simulator.caster_token_url"
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
	// Content swaps can raise Fyne's minimum size; pull back above the taskbar
	// before pointer hit-testing desyncs on Windows.
	scheduleWorkAreaClamp(window)
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

// loadSimulatorCosmetics reads the saved allowlisted custom-asset URLs.
func loadSimulatorCosmetics() simulatorview.PlayerCosmetics {
	prefs := fyne.CurrentApp().Preferences()
	return simulatorview.PlayerCosmetics{
		CardBackURL: strings.TrimSpace(prefs.StringWithFallback(
			simulatorCardBackPreferenceKey,
			"",
		)),
		CasterTokenURL: strings.TrimSpace(prefs.StringWithFallback(
			simulatorCasterTokenPreferenceKey,
			"",
		)),
	}
}

// applySimulatorCosmetics stamps prefs cosmetics onto both local seats.
func applySimulatorCosmetics(match *simulatorview.MatchView) {
	if match == nil {
		return
	}
	cosmetics := loadSimulatorCosmetics()
	if cosmetics.CardBackURL == "" && cosmetics.CasterTokenURL == "" {
		return
	}
	if match.Cosmetics == nil {
		match.Cosmetics = make(map[model.PlayerID]simulatorview.PlayerCosmetics, 2)
	}
	for _, player := range match.Players {
		if player.ID == "" {
			continue
		}
		match.Cosmetics[player.ID] = cosmetics
	}
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

	cardBackEntry := widget.NewEntry()
	cardBackEntry.SetText(guiApp.Preferences().StringWithFallback(simulatorCardBackPreferenceKey, ""))
	cardBackEntry.SetPlaceHolder("Leave blank for the official card back")
	cardBackEntry.Validator = deckexport.ValidateSimulatorCardBackURL

	casterTokenEntry := widget.NewEntry()
	casterTokenEntry.SetText(guiApp.Preferences().StringWithFallback(simulatorCasterTokenPreferenceKey, ""))
	casterTokenEntry.SetPlaceHolder("Leave blank for the official caster token")
	casterTokenEntry.Validator = deckexport.ValidateSimulatorCasterTokenURL

	uploadURL, err := url.Parse(deckexport.CustomAssetUploadURL)
	var uploadLink fyne.CanvasObject = widget.NewLabel(deckexport.CustomAssetUploadURL)
	if err == nil {
		uploadLink = widget.NewHyperlink("Upload and copy a custom image URL", uploadURL)
	}

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
			widget.NewSeparator(),
			widget.NewLabelWithStyle(
				"Simulator Cosmetics",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			uploadLink,
			widget.NewLabel("Card back URL"),
			cardBackEntry,
			widget.NewLabel("Caster token URL"),
			casterTokenEntry,
		),
		func(saved bool) {
			if !saved {
				return
			}
			cardBackURL := strings.TrimSpace(cardBackEntry.Text)
			casterTokenURL := strings.TrimSpace(casterTokenEntry.Text)
			if err := deckexport.ValidateSimulatorCardBackURL(cardBackURL); err != nil {
				dialog.ShowError(err, window)
				return
			}
			if err := deckexport.ValidateSimulatorCasterTokenURL(casterTokenURL); err != nil {
				dialog.ShowError(err, window)
				return
			}
			value := appearanceThemeValue(selection.Selected)
			guiApp.Preferences().SetString(appearanceThemePreferenceKey, value)
			applyAppearanceTheme(guiApp, value)
			background := backgroundValue(backgroundSelection.Selected)
			guiApp.Preferences().SetString(backgroundPreferenceKey, background)
			guiApp.Preferences().SetString(simulatorCardBackPreferenceKey, cardBackURL)
			guiApp.Preferences().SetString(simulatorCasterTokenPreferenceKey, casterTokenURL)
			if onSaved != nil {
				onSaved()
			}
		},
		window,
	).Show()
}
