package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// dashboardTheme provides a dark color scheme matching the terminal dashboard.
type dashboardTheme struct{}

func (d *dashboardTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 30, G: 30, B: 46, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 205, G: 214, B: 244, A: 255}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 68, G: 138, B: 255, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 49, G: 50, B: 68, A: 255}
	case theme.ColorNameHeaderBackground:
		return color.NRGBA{R: 24, G: 24, B: 37, A: 255}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 69, G: 71, B: 90, A: 255}
	default:
		return theme.DefaultTheme().Color(name, theme.VariantDark)
	}
}

func (d *dashboardTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (d *dashboardTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (d *dashboardTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 13
	case theme.SizeNamePadding:
		return 6
	default:
		return theme.DefaultTheme().Size(name)
	}
}
