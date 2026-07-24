package deckbuilder

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// TestNormalizeAppearanceTheme verifies only supported persisted values survive.
func TestNormalizeAppearanceTheme(t *testing.T) {
	tests := map[string]string{
		"system":  appearanceThemeSystem,
		" LIGHT ": appearanceThemeLight,
		"Dark":    appearanceThemeDark,
		"unknown": appearanceThemeSystem,
		"":        appearanceThemeSystem,
	}
	for input, want := range tests {
		if got := normalizeAppearanceTheme(input); got != want {
			t.Errorf("normalizeAppearanceTheme(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestNormalizeBackground verifies only bundled artwork choices survive.
func TestNormalizeBackground(t *testing.T) {
	tests := map[string]string{
		"none":           backgroundNone,
		" ACADEMY-RIFT ": backgroundAcademyRift,
		"Caster-Duel":    backgroundCasterDuel,
		"unknown":        backgroundNone,
		"":               backgroundNone,
	}
	for input, want := range tests {
		if got := normalizeBackground(input); got != want {
			t.Errorf("normalizeBackground(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestBackgroundLabelsRoundTrip verifies settings labels preserve every choice.
func TestBackgroundLabelsRoundTrip(t *testing.T) {
	for _, value := range []string{
		backgroundNone,
		backgroundAcademyRift,
		backgroundCasterDuel,
	} {
		if got := backgroundValue(backgroundLabel(value)); got != value {
			t.Errorf("background preference %q round-tripped as %q", value, got)
		}
	}
}

// TestWrapWithBackgroundKeepsDefaultPlainAndLayersArtwork checks both layouts.
func TestWrapWithBackgroundKeepsDefaultPlainAndLayersArtwork(t *testing.T) {
	foreground := widget.NewLabel("foreground")
	if got := wrapWithBackground(foreground, backgroundNone); got != foreground {
		t.Fatal("no-background preference unexpectedly wrapped content")
	}

	got := wrapWithBackground(foreground, backgroundAcademyRift)
	surface, ok := got.(*fyne.Container)
	if !ok {
		t.Fatalf("artwork content type = %T, want renderer-traversable *fyne.Container", got)
	}
	if len(surface.Objects) != 3 {
		t.Fatalf("background surface does not retain three ordered layers")
	}
	image, ok := surface.Objects[0].(*canvas.Image)
	if !ok || image.FillMode != canvas.ImageFillCover {
		t.Fatalf("first layer = %T with fill %v, want cover image", surface.Objects[0], image.FillMode)
	}
	if surface.Objects[2] != foreground {
		t.Fatal("foreground is not the top background-surface layer")
	}
}

// TestAppearanceThemeLabelsRoundTrip verifies settings labels preserve every supported value.
func TestAppearanceThemeLabelsRoundTrip(t *testing.T) {
	for _, value := range []string{
		appearanceThemeSystem,
		appearanceThemeLight,
		appearanceThemeDark,
	} {
		if got := appearanceThemeValue(appearanceThemeLabel(value)); got != value {
			t.Errorf("theme preference %q round-tripped as %q", value, got)
		}
	}
}

// TestApplyAppearanceThemeForcesSelectedPalette verifies the saved choice overrides OS colors.
func TestApplyAppearanceThemeForcesSelectedPalette(t *testing.T) {
	guiApp := test.NewApp()
	defer guiApp.Quit()

	applyAppearanceTheme(guiApp, appearanceThemeDark)
	got := guiApp.Settings().Theme().Color(
		theme.ColorNameBackground,
		theme.VariantLight,
	)
	want := theme.DefaultTheme().Color(
		theme.ColorNameBackground,
		theme.VariantDark,
	)
	if got != want {
		t.Fatalf("dark background = %v, want %v", got, want)
	}

	applyAppearanceTheme(guiApp, appearanceThemeLight)
	got = guiApp.Settings().Theme().Color(
		theme.ColorNameBackground,
		theme.VariantDark,
	)
	want = theme.DefaultTheme().Color(
		theme.ColorNameBackground,
		theme.VariantLight,
	)
	if got != want {
		t.Fatalf("light background = %v, want %v", got, want)
	}
}
