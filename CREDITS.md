# Credits

## Developer

**Jeffrey Leal**
Email: jeffrey.leal@gmail.com
GitHub: https://github.com/jeffrey-leal

---

## AI Assistance

This application was designed and developed with the assistance of
**Claude Sonnet 4.6** by [Anthropic](https://www.anthropic.com), accessed
through [Claude Code](https://claude.ai/code).

Contributions made with AI assistance include:

- Application architecture and Go source code
- Fyne UI layout, widget composition, and custom rendering
- DICOM tag tree model and dataset ingestion logic
- Tag profile system design and PHI attribute definitions
- Search, filter, export (CSV / JSON / plain text), and keyboard shortcut implementation
- Tag tooltip system with DICOM 2024b dictionary integration
- Progress bar, error log, and status bar behaviour
- Custom theme, system font scanner, and preferences dialog
- About dialog and credits display
- Build scripts and MSYS2 / CGo toolchain configuration
- User manual, changelog, and project documentation

---

## DICOM Standard Reference

Protocol implementation and data dictionary usage follow the DICOM Standard
published by NEMA:

**DICOM PS3 (2024b)**
https://dicom.nema.org/medical/dicom/current

---

## Build Toolchain

| Tool | Version | Purpose |
|---|---|---|
| [Go](https://go.dev) | 1.26.2 | Programming language and build toolchain |
| [CGo](https://pkg.go.dev/cmd/cgo) | (bundled with Go) | C interop layer required by Fyne on Windows |
| [MSYS2 MinGW64 GCC](https://www.msys2.org) | mingw64 | C compiler for CGo on Windows |

---

## Open-Source Libraries

### Direct Dependencies

| Library | Version | Author / Maintainer | License | Purpose |
|---|---|---|---|---|
| [fyne.io/fyne/v2](https://fyne.io) | v2.7.3 | Fyne.io contributors | BSD 3-Clause | GUI framework — windows, widgets, theming, layout |
| [suyashkumar/dicom](https://github.com/suyashkumar/dicom) | v1.1.0 | Suyash Kumar | MIT | DICOM file parsing, data dictionary, and element model |
| [sqweek/dialog](https://github.com/sqweek/dialog) | v0.0.0-20260123140253 | sqweek | ISC | Native Win32 file and folder picker dialogs |

### Notable Indirect Dependencies

The following libraries are pulled in transitively by Fyne and contribute to
core rendering and text handling within the application:

| Library | Author / Maintainer | License | Purpose |
|---|---|---|---|
| [go-text/typesetting](https://github.com/go-text/typesetting) | go-text contributors | MIT | OpenType font shaping and text layout |
| [go-text/render](https://github.com/go-text/render) | go-text contributors | MIT | Vector text rendering engine |
| [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) | Go Authors | BSD 3-Clause | Extended image format support |
| [golang.org/x/text](https://pkg.go.dev/golang.org/x/text) | Go Authors | BSD 3-Clause | Unicode and character encoding support |

A full list of all transitive dependencies and their versions is recorded in
`go.sum`.
