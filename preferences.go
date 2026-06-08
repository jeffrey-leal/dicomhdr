package main

import (
	"fmt"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// appTheme wraps a base Fyne theme and optionally overrides the font.
type appTheme struct {
	base     fyne.Theme
	font     fyne.Resource
	fontName string
	isDark   bool
}

func (t *appTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.base.Color(name, variant)
}

func (t *appTheme) Font(style fyne.TextStyle) fyne.Resource {
	if t.font != nil {
		return t.font
	}
	return t.base.Font(style)
}

func (t *appTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *appTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.base.Size(name)
}

func newAppTheme(isDark bool) *appTheme {
	t := &appTheme{isDark: isDark}
	if isDark {
		t.base = theme.DarkTheme()
	} else {
		t.base = theme.LightTheme()
	}
	return t
}

// systemFontDirs returns the OS-specific directories that contain font files.
func systemFontDirs() []string {
	switch runtime.GOOS {
	case "windows":
		windir := os.Getenv("WINDIR")
		if windir == "" {
			windir = `C:\Windows`
		}
		return []string{filepath.Join(windir, "Fonts")}
	case "darwin":
		return []string{"/Library/Fonts", "/System/Library/Fonts"}
	default:
		return []string{"/usr/share/fonts", "/usr/local/share/fonts"}
	}
}

// listSystemFonts returns a sorted list of font names found in the system font directories.
func listSystemFonts() []string {
	seen := map[string]bool{}
	var names []string
	for _, dir := range systemFontDirs() {
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if ext != ".ttf" && ext != ".otf" {
				return nil
			}
			name := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if !seen[strings.ToLower(name)] {
				seen[strings.ToLower(name)] = true
				names = append(names, name)
			}
			return nil
		})
	}
	sort.Strings(names)
	return names
}

// fontPathByName searches the system font directories for a file matching name.
func fontPathByName(name string) string {
	for _, dir := range systemFontDirs() {
		for _, ext := range []string{".ttf", ".otf", ".TTF", ".OTF"} {
			p := filepath.Join(dir, name+ext)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

// loadFontResource reads a font file from disk and returns a Fyne resource.
func loadFontResource(path string) (fyne.Resource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return fyne.NewStaticResource(filepath.Base(path), data), nil
}

// colorToHex encodes a color as an 8-character RRGGBBAA hex string.
func colorToHex(c color.Color) string {
	r, g, b, a := c.RGBA()
	return fmt.Sprintf("%02X%02X%02X%02X", uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
}

// hexToColor decodes an 8-character RRGGBBAA hex string; returns a default red on parse failure.
func hexToColor(s string) color.Color {
	if len(s) == 8 {
		if v, err := strconv.ParseUint(s, 16, 32); err == nil {
			return color.RGBA{R: uint8(v >> 24), G: uint8(v >> 16), B: uint8(v >> 8), A: uint8(v)}
		}
	}
	return color.RGBA{R: 0xE5, G: 0x45, B: 0x45, A: 0xFF}
}

// cardBorderColor is the theme foreground at reduced opacity — a darker, clearly
// visible card outline in light mode and a light one in dark mode.
func cardBorderColor() color.Color {
	r, g, b, _ := theme.Color(theme.ColorNameForeground).RGBA()
	return color.NRGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 0xB0}
}

// sectionCard wraps content in a titled, bordered panel. Unlike widget.Card it
// uses a body-size bold heading (rather than the oversized heading text size)
// and a defined, darker border instead of a soft shadow.
func sectionCard(title string, content fyne.CanvasObject) fyne.CanvasObject {
	heading := canvas.NewText(title, theme.Color(theme.ColorNameForeground))
	heading.TextStyle = fyne.TextStyle{Bold: true}
	heading.TextSize = theme.TextSize()

	box := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	box.StrokeColor = cardBorderColor()
	box.StrokeWidth = 1
	box.CornerRadius = theme.Size(theme.SizeNameInputRadius)

	body := container.NewPadded(container.NewVBox(heading, content))
	return container.NewStack(box, body)
}

// showPreferencesDialog opens the preferences dialog, applying changes on confirm.
func showPreferencesDialog(a fyne.App, w fyne.Window, current *appTheme, italicPrivate *atomic.Bool, malformedColor *atomic.Value, profiles *atomic.Value, treeRefresh func()) {
	themeLabel := "Light"
	if current.isDark {
		themeLabel = "Dark"
	}
	themeSelect := widget.NewRadioGroup([]string{"Light", "Dark"}, nil)
	themeSelect.SetSelected(themeLabel)

	fontOptions := append([]string{"(default)"}, listSystemFonts()...)
	fontSelect := widget.NewSelect(fontOptions, nil)
	if current.fontName != "" {
		fontSelect.SetSelected(current.fontName)
	} else {
		fontSelect.SetSelected("(default)")
	}

	italicCheck := widget.NewCheck("Italicize", nil)
	italicCheck.SetChecked(italicPrivate.Load())

	pendingColor, _ := malformedColor.Load().(color.Color)
	if pendingColor == nil {
		pendingColor = color.RGBA{R: 0xE5, G: 0x45, B: 0x45, A: 0xFF}
	}
	swatch := canvas.NewRectangle(pendingColor)
	swatch.SetMinSize(fyne.NewSize(48, theme.TextSize()+8))
	changeColorBtn := widget.NewButton("Change…", func() {
		dialog.ShowColorPicker("Malformed tag color", "", func(c color.Color) {
			if c != nil {
				// Normalize to color.RGBA so atomic.Value.Store always receives the
				// same concrete type as the initial value (also color.RGBA).
				// The color picker may return color.NRGBA or a Fyne-internal type;
				// storing a different type panics atomic.Value and crashes the app.
				r, g, b, a := c.RGBA()
				rgba := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
				pendingColor = rgba
				swatch.FillColor = rgba
				swatch.Refresh()
			}
		}, w)
	})

	// Profiles — work on a mutable copy; committed to the atomic on Apply.
	existing, _ := profiles.Load().([]TagProfile)
	pendingProfiles := append([]TagProfile(nil), existing...)

	profileList := container.NewVBox()
	var buildProfileList func()
	buildProfileList = func() {
		rows := make([]fyne.CanvasObject, len(pendingProfiles))
		for i := range pendingProfiles {
			i := i
			check := widget.NewCheck("", func(enabled bool) {
				pendingProfiles[i].Enabled = enabled
			})
			check.SetChecked(pendingProfiles[i].Enabled)
			nameLabel := widget.NewLabel(pendingProfiles[i].Name)
			editBtn := widget.NewButton("Edit", func() {
				showProfileEditor(w, pendingProfiles[i], func(updated TagProfile) {
					updated.Enabled = pendingProfiles[i].Enabled
					pendingProfiles[i] = updated
					buildProfileList()
				})
			})
			deleteBtn := widget.NewButton("Delete", func() {
				pendingProfiles = append(pendingProfiles[:i], pendingProfiles[i+1:]...)
				buildProfileList()
			})
			rows[i] = container.NewBorder(nil, nil,
				container.NewHBox(check, nameLabel),
				container.NewHBox(editBtn, deleteBtn),
			)
		}
		profileList.Objects = rows
		profileList.Refresh()
	}
	buildProfileList()

	addProfileBtn := widget.NewButton("Add profile…", func() {
		newP := TagProfile{
			Name:    "New Profile",
			Color:   color.RGBA{R: 0x00, G: 0x80, B: 0xFF, A: 0xFF},
			Enabled: true,
		}
		showProfileEditor(w, newP, func(added TagProfile) {
			pendingProfiles = append(pendingProfiles, added)
			buildProfileList()
		})
	})

	profileScroll := container.NewScroll(profileList)
	profileScroll.SetMinSize(fyne.NewSize(0, 120))

	// Each logical group is a titled, bordered card so the sections are visually
	// distinct rather than a flat stack of labels.
	uiCard := sectionCard("Appearance", widget.NewForm(
		widget.NewFormItem("Theme", themeSelect),
		widget.NewFormItem("Tree font", fontSelect),
	))

	hlCard := sectionCard("Tag Highlights", widget.NewForm(
		widget.NewFormItem("Private tags", italicCheck),
		widget.NewFormItem("Malformed tag", container.NewHBox(swatch, changeColorBtn)),
	))

	profileCard := sectionCard("Tag Profiles",
		container.NewBorder(nil, addProfileBtn, nil, nil, profileScroll),
	)

	// Button row: separator at top, buttons pinned to the bottom of a taller area.
	var d dialog.Dialog
	cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })
	applyBtn := widget.NewButton("Apply", func() {
		current.isDark = themeSelect.Selected == "Dark"
		if current.isDark {
			current.base = theme.DarkTheme()
		} else {
			current.base = theme.LightTheme()
		}
		if fontSelect.Selected == "(default)" {
			current.font = nil
			current.fontName = ""
		} else {
			if path := fontPathByName(fontSelect.Selected); path != "" {
				if res, err := loadFontResource(path); err == nil {
					current.font = res
					current.fontName = fontSelect.Selected
				}
			}
		}
		italicPrivate.Store(italicCheck.Checked)
		malformedColor.Store(pendingColor)
		profiles.Store(pendingProfiles)
		saveSettings(Settings{
			DarkTheme:      current.isDark,
			FontName:       current.fontName,
			ItalicPrivate:  italicCheck.Checked,
			MalformedColor: colorToHex(pendingColor),
			Profiles:       pendingProfiles,
		})
		treeRefresh()
		a.Settings().SetTheme(current)
		d.Hide()
	})

	fill := canvas.NewRectangle(color.Transparent)
	fill.SetMinSize(fyne.NewSize(0, 24))
	buttonRow := container.NewBorder(
		widget.NewSeparator(),
		nil, nil, nil,
		container.NewStack(
			fill,
			container.NewCenter(container.NewHBox(cancelBtn, applyBtn)),
		),
	)

	content := container.NewVBox(uiCard, hlCard, profileCard, buttonRow)
	d = dialog.NewCustomWithoutButtons("Preferences", content, w)
	d.Show()
}
