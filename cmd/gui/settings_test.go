package main

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
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
