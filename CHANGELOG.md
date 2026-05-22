# Changelog

All notable changes to dicomhdr are documented here.

## [1.0.3] - 2026-05-22

### Changed
- `CREDITS.md` expanded with itemised list of AI assistance contributions,
  new Build Toolchain section (Go, CGo, MSYS2 MinGW64 GCC), library table
  split into Direct and Notable Indirect dependencies.

---

## [1.0.2] - 2026-05-22

### Added
- `CREDITS.md` — full attribution for developer, AI assistance, DICOM standard
  reference, and all open-source libraries
- About dialog updated to match dicomqr style: monospace layout with developer,
  AI assistance, DICOM standard reference, and library credits

---

## [1.0.1] - 2026-04-29

### Added
- **Search and filter bar** — text field at the top of the window narrows the tag tree to rows whose label contains the typed string (case-insensitive). Ancestor branches containing a match are always kept visible. Activated by clicking Search or pressing Enter; cleared by clicking Clear.
- **Expand All / Collapse All** — toolbar buttons that open or close every branch in the tree with a single click.
- **Keyboard shortcuts** — Ctrl+O (Load File), Ctrl+Shift+O (Load Folder), Ctrl+F (focus search bar), Ctrl+C (copy selected row).
- **Copy to clipboard** — Ctrl+C copies the full label of the selected tag row. Right-clicking any row shows a context menu with Copy row (full label) and Copy value (text after the first ": ").
- **Export** — File > Export submenu writes the full tag tree to Plain Text (.txt, indented), CSV (.csv, with Depth / Tag / VR / Name / Value columns), or JSON (.json, hierarchical).
- **Tag tooltips** — hovering over a tag row for ~0.6 s displays a popup with the DICOM 2024b dictionary entry for that tag (tag address, name, keyword, VR, VM). Structural nodes and private/unknown tags show no tooltip.
- **Failed-file log** — files that fail to parse increment an Errors counter in the status bar. Options > Show load errors… lists every failed path and its error message.
- **Help > About dialog** — displays the program name, version, DICOM data dictionary edition (DICOM 2024b), and build date (injected at compile time via `-X main.buildDate`).
- **Progress bar** — a progress bar appears in the status area during folder loads, advancing from 0 % to 100 % as each file is parsed, then dismissing automatically. Not shown for single-file loads.

### Changed
- Status bar now shows `Files loaded: N` (and `| Errors: M` when applicable) once files are loaded, replacing the plain version string.
- File menu gains an Export submenu between Load Folder and Preferences.
- Options menu gains Show load errors… between Refresh and Clear All.

## [1.0.0] - initial release

- Load individual DICOM files via File > Load File or drag-and-drop.
- Recursively load a folder of DICOM files via File > Load Folder or drag-and-drop.
- Scrollable, expandable tag tree organised by Patient > Study > Series > Instance.
- Color highlighting: configurable malformed-tag color (VR mismatch detection) and named Tag Profiles (arbitrary sets of tags rendered in a chosen color).
- Built-in PHI profile highlights 126 protected health information attributes in orange.
- Italic rendering of private tags (odd group number), configurable per preference.
- Light and dark themes; system font selection for tree rows.
- Preferences persisted to `%USERPROFILE%\.dcomhdr\settings.json`.
- Status bar with file count and live clock.
