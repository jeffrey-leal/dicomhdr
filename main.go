package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	sqweekdialog "github.com/sqweek/dialog"
	"github.com/suyashkumar/dicom"
)

const version = "1.0.2"
const dicomDictEdition = "DICOM 2024b"

// buildDate is injected at link time: -ldflags "-X main.buildDate=YYYY-MM-DD"
var buildDate string

// rowLayout adds vertical padding around a single child, keeping it centred.
type rowLayout struct{}

func (rowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pad := theme.TextSize() / 4
	for _, o := range objects {
		o.Move(fyne.NewPos(0, pad))
		o.Resize(fyne.NewSize(size.Width, size.Height-pad*2))
	}
}

func (rowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	pad := theme.TextSize() / 4
	if len(objects) == 0 {
		return fyne.NewSize(0, pad*2)
	}
	s := objects[0].MinSize()
	return fyne.NewSize(s.Width, s.Height+pad*2)
}

func main() {
	a := app.NewWithID("com.jeffreyleal.dicomhdr")
	w := a.NewWindow("dicomhdr")
	w.Resize(fyne.NewSize(800, 600))

	// Ensure ~/.dcomhdr/settings.json exists, then load it.
	ensureDefaultSettings()
	cfg := loadSettings()

	currentTheme := newAppTheme(cfg.DarkTheme)
	if cfg.FontName != "" {
		if path := fontPathByName(cfg.FontName); path != "" {
			if res, err := loadFontResource(path); err == nil {
				currentTheme.font = res
				currentTheme.fontName = cfg.FontName
			}
		}
	}
	a.Settings().SetTheme(currentTheme)

	var italicPrivate atomic.Bool
	italicPrivate.Store(cfg.ItalicPrivate)

	var malformedColor atomic.Value
	malformedColor.Store(hexToColor(cfg.MalformedColor))

	var profiles atomic.Value
	profiles.Store(cfg.Profiles)

	model := newDicomTreeModel()
	var filesLoaded atomic.Int64
	var errCount atomic.Int64
	var loadErrMu sync.Mutex
	var loadErrors []string

	statusLabel := widget.NewLabel("v" + version)
	clockLabel := widget.NewLabel("")

	updateStatus := func() {
		n := filesLoaded.Load()
		e := errCount.Load()
		if n == 0 && e == 0 {
			statusLabel.SetText("v" + version)
			return
		}
		text := fmt.Sprintf("Files loaded: %d", n)
		if e > 0 {
			text += fmt.Sprintf("  |  Errors: %d", e)
		}
		statusLabel.SetText(text)
	}

	// selectedNodeID tracks the tree node the user most recently clicked.
	// Accessed only from the Fyne event goroutine.
	var selectedNodeID string

	// onMenu is called when a tree row is right-clicked; shows a copy popup.
	onMenu := func(id string, pos fyne.Position) {
		label := model.labelFor(id)
		copyRow := fyne.NewMenuItem("Copy row", func() {
			w.Clipboard().SetContent(label)
		})
		copyValue := fyne.NewMenuItem("Copy value", func() {
			if i := strings.Index(label, ": "); i >= 0 {
				w.Clipboard().SetContent(label[i+2:])
			} else {
				w.Clipboard().SetContent(label)
			}
		})
		popup := widget.NewPopUpMenu(fyne.NewMenu("", copyRow, copyValue), w.Canvas())
		popup.ShowAtPosition(pos)
	}

	tree := widget.NewTree(
		model.childUIDs,
		model.isBranch,
		func(branch bool) fyne.CanvasObject {
			return newTreeRow(w.Canvas(), onMenu)
		},
		func(id widget.TreeNodeID, branch bool, node fyne.CanvasObject) {
			row := node.(*treeRow)
			row.nodeID = id
			row.tooltipText = model.tooltipFor(id)
			row.ct.Text = model.labelFor(id)
			row.ct.TextSize = theme.TextSize()
			row.ct.TextStyle = fyne.TextStyle{Italic: italicPrivate.Load() && model.isPrivateNode(id)}
			elmTag, hasTag := model.nodeTag(id)
			activeProfiles, _ := profiles.Load().([]TagProfile)
			if model.isMalformedNode(id) {
				row.ct.Color, _ = malformedColor.Load().(color.Color)
			} else if c, ok := matchProfile(elmTag, hasTag, activeProfiles); ok {
				row.ct.Color = c
			} else {
				row.ct.Color = theme.Color(theme.ColorNameForeground)
			}
			row.Refresh()
		},
	)

	tree.OnSelected = func(id widget.TreeNodeID) {
		selectedNodeID = id
	}

	// Ctrl+C copies the full label of the currently selected tree row.
	w.Canvas().AddShortcut(&fyne.ShortcutCopy{}, func(_ fyne.Shortcut) {
		if selectedNodeID != "" {
			w.Clipboard().SetContent(model.labelFor(selectedNodeID))
		}
	})

	// Search bar: filters the tree when Search is clicked or Enter is pressed.
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search tags…")

	doSearch := func() {
		model.setFilter(searchEntry.Text)
		if searchEntry.Text != "" {
			tree.OpenAllBranches()
		}
		tree.Refresh()
	}
	doClear := func() {
		searchEntry.SetText("")
		model.setFilter("")
		tree.Refresh()
	}

	searchEntry.OnSubmitted = func(_ string) { doSearch() }

	searchBtn := widget.NewButton("Search", doSearch)
	clearBtn := widget.NewButton("Clear", doClear)
	expandBtn := widget.NewButton("Expand All", func() { tree.OpenAllBranches() })
	collapseBtn := widget.NewButton("Collapse All", func() { tree.CloseAllBranches() })
	searchBar := container.NewBorder(
		nil, nil,
		container.NewHBox(expandBtn, collapseBtn),
		container.NewHBox(searchBtn, clearBtn),
		searchEntry,
	)

	var progressBar *widget.ProgressBar

	loadPath := func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			return
		}
		if info.IsDir() {
			// Phase 1: discover all DICOM files so we know the total.
			var paths []string
			filepath.WalkDir(path, func(p string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() || !isDICOMFile(p) {
					return nil
				}
				paths = append(paths, p)
				return nil
			})
			total := len(paths)
			if total == 0 {
				return
			}
			fyne.Do(func() { progressBar.SetValue(0); progressBar.Show() })

			// Phase 2: parse each file, updating progress after every file.
			for i, p := range paths {
				ds, err := dicom.ParseFile(p, nil)
				frac := float64(i+1) / float64(total)
				if err != nil {
					loadErrMu.Lock()
					loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", p, err))
					loadErrMu.Unlock()
					errCount.Add(1)
					fyne.Do(func() { updateStatus(); progressBar.SetValue(frac) })
					continue
				}
				model.addDataset(ds)
				filesLoaded.Add(1)
				fyne.Do(func() {
					updateStatus()
					tree.Refresh()
					progressBar.SetValue(frac)
				})
			}
			fyne.Do(func() { progressBar.Hide() })
		} else {
			if !isDICOMFile(path) {
				return
			}
			ds, err := dicom.ParseFile(path, nil)
			if err != nil {
				loadErrMu.Lock()
				loadErrors = append(loadErrors, fmt.Sprintf("%s: %v", path, err))
				loadErrMu.Unlock()
				errCount.Add(1)
				fyne.Do(func() { updateStatus() })
				return
			}
			model.addDataset(ds)
			filesLoaded.Add(1)
			fyne.Do(func() {
				updateStatus()
				tree.Refresh()
			})
		}
	}

	openFile := func() {
		go func() {
			path, err := sqweekdialog.File().Load()
			if err != nil {
				return
			}
			loadPath(path)
		}()
	}

	loadFolder := func() {
		go func() {
			folderPath, err := sqweekdialog.Directory().Browse()
			if err != nil {
				return
			}
			loadPath(folderPath)
		}()
	}

	// Keyboard shortcuts — openFile and loadFolder must be defined first.
	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierShortcutDefault},
		func(_ fyne.Shortcut) { openFile() },
	)
	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		func(_ fyne.Shortcut) { loadFolder() },
	)
	w.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault},
		func(_ fyne.Shortcut) { w.Canvas().Focus(searchEntry) },
	)

	doExport := func(title, ext string, writeFn func(string, *dicomTreeModel) error) {
		if filesLoaded.Load() == 0 {
			dialog.ShowInformation("Export", "No data is loaded to export.", w)
			return
		}
		go func() {
			path, err := sqweekdialog.File().
				Title(title).
				Filter(strings.ToUpper(ext)+" files", ext).
				Save()
			if err != nil {
				return // cancelled
			}
			if !strings.HasSuffix(strings.ToLower(path), "."+ext) {
				path += "." + ext
			}
			if err := writeFn(path, model); err != nil {
				fyne.Do(func() { dialog.ShowError(err, w) })
			}
		}()
	}

	showLoadErrors := func() {
		loadErrMu.Lock()
		errs := append([]string(nil), loadErrors...)
		loadErrMu.Unlock()
		if len(errs) == 0 {
			dialog.ShowInformation("Load Errors", "No errors recorded.", w)
			return
		}
		content := widget.NewMultiLineEntry()
		content.SetText(strings.Join(errs, "\n"))
		content.Disable()
		scroll := container.NewScroll(content)
		scroll.SetMinSize(fyne.NewSize(500, 300))
		dialog.ShowCustom("Load Errors", "Close", scroll, w)
	}

	optionsMenu := fyne.NewMenu("Options",
		fyne.NewMenuItem("Refresh", func() {
			fyne.Do(func() { tree.Refresh() })
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Show load errors…", showLoadErrors),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Clear all", func() {
			model.clear()
			filesLoaded.Store(0)
			errCount.Store(0)
			loadErrMu.Lock()
			loadErrors = loadErrors[:0]
			loadErrMu.Unlock()
			searchEntry.SetText("")
			fyne.Do(func() {
				updateStatus()
				tree.Refresh()
			})
		}),
	)

	exportItem := fyne.NewMenuItem("Export", nil)
	exportItem.ChildMenu = fyne.NewMenu("",
		fyne.NewMenuItem("Plain Text…", func() { doExport("Export as Plain Text", "txt", writeText) }),
		fyne.NewMenuItem("CSV…", func() { doExport("Export as CSV", "csv", writeCSV) }),
		fyne.NewMenuItem("JSON…", func() { doExport("Export as JSON", "json", writeJSON) }),
	)

	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Load File", openFile),
		fyne.NewMenuItem("Load Folder", loadFolder),
		exportItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Preferences", func() {
			showPreferencesDialog(a, w, currentTheme, &italicPrivate, &malformedColor, &profiles, func() { tree.Refresh() })
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { a.Quit() }),
	)
	bd := buildDate
	if bd == "" {
		bd = "unknown"
	}
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			lbl := widget.NewLabel(fmt.Sprintf(
				"dicomhdr  v%s  (built %s)\n"+
					"DICOM file inspector — browse and inspect DICOM tag hierarchies.\n"+
					"Data dictionary: %s\n\n"+
					"Developer\n"+
					"  Jeffrey Leal  <jeffrey.leal@gmail.com>\n"+
					"  https://github.com/jeffrey-leal\n\n"+
					"AI Assistance\n"+
					"  Claude Sonnet 4.6 by Anthropic  (https://anthropic.com)\n"+
					"  Architecture, code generation, and DICOM standard research.\n\n"+
					"DICOM Standard Reference\n"+
					"  DICOM PS3 (2024b) — https://dicom.nema.org/medical/dicom/current\n\n"+
					"Open-Source Libraries\n"+
					"  fyne.io/fyne/v2 v2.7.3          Fyne.io — GUI framework (BSD 3-Clause)\n"+
					"  github.com/suyashkumar/dicom     Suyash Kumar — DICOM parsing (MIT)\n"+
					"  github.com/sqweek/dialog         sqweek — native file dialogs (ISC)\n\n"+
					"Full credits: CREDITS.md in the project repository.",
				version, bd, dicomDictEdition))
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			d := dialog.NewCustom("About dicomhdr", "OK", container.NewPadded(lbl), w)
			d.Resize(fyne.NewSize(580, 0))
			d.Show()
		}),
	)

	w.SetMainMenu(fyne.NewMainMenu(fileMenu, optionsMenu, helpMenu))

	w.SetOnDropped(func(_ fyne.Position, uris []fyne.URI) {
		go func() {
			for _, uri := range uris {
				// Fyne represents Windows paths as /C:/foo/bar — strip the leading slash.
				p := uri.Path()
				if len(p) > 2 && p[0] == '/' && p[2] == ':' {
					p = p[1:]
				}
				loadPath(filepath.FromSlash(p))
			}
		}()
	})

	go func() {
		for {
			fyne.Do(func() {
				clockLabel.SetText(time.Now().Format("2006-01-02  15:04:05"))
			})
			time.Sleep(time.Second)
		}
	}()

	progressBar = widget.NewProgressBar()
	progressBar.Hide()

	statusBar := container.NewVBox(
		container.NewHBox(statusLabel, layout.NewSpacer(), clockLabel),
		progressBar,
	)

	w.SetContent(container.NewBorder(
		searchBar, statusBar, nil, nil,
		container.NewScroll(tree),
	))

	w.ShowAndRun()
}
