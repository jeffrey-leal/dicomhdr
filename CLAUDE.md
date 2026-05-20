# dicomhdr

A Fyne-based Windows GUI application for inspecting DICOM files.

## Build

Requires CGO and the mingw64 GCC toolchain. **Must be built from an MSYS2 MinGW64 terminal** — the Claude Code sandbox and plain PowerShell both fail at the CGO compilation or link stage.

Open MSYS2 MinGW64, then:

```bash
export PATH="/c/Program Files/Go/bin:$PATH"
cd /c/Users/jeffr/source/repos/dicomhdr
```

Release build:
```bash
CGO_ENABLED=1 CC=/c/msys64/mingw64/bin/gcc.exe GOAMD64=v3 \
  go build -ldflags="-s -w -H windowsgui -X main.buildDate=$(date +%Y-%m-%d)" -o dicomhdr.exe .
```

- `-s -w` strips debug info (smaller binary, faster load)
- `-H windowsgui` sets the PE subsystem to WINDOWS GUI — suppresses the console window. Use this instead of `-extldflags=-mwindows`; the extldflags form is overridden by Go's own `-mconsole` default and has no effect.
- `GOAMD64=v3` enables AVX2 instructions (optional)
- `-X main.buildDate=$(date +%Y-%m-%d)` injects the build date shown in Help > About

Development build (with debug info):
```bash
CGO_ENABLED=1 CC=/c/msys64/mingw64/bin/gcc.exe \
  go build -ldflags="-X main.buildDate=$(date +%Y-%m-%d)" -o dicomhdr.exe .
```

## Project structure

| File | Purpose |
|---|---|
| `main.go` | App entry point, window layout, menu bar, status bar |
| `dicomtree.go` | `dicomTreeModel` — tree data structure and DICOM dataset ingestion |
| `dicomfile.go` | `isDICOMFile` — magic-byte verification (DICM at offset 128) |
| `preferences.go` | `appTheme`, system font scanner, preferences dialog |

## Key dependencies

- `fyne.io/fyne/v2 v2.7.3` — GUI framework
- `github.com/suyashkumar/dicom v1.1.0` — DICOM parsing
- `github.com/sqweek/dialog` — native Windows file/folder picker (avoids Fyne file-attributes bug on drive switch)

## Notes

- App ID: `com.jeffreyleal.dicomhdr` (required for Fyne Preferences API)
- Theme and font preferences are persisted via `fyne.App.Preferences()`
- File dialogs use the native Win32 Common Dialog, not Fyne's built-in dialog
- DICOM files are verified by magic bytes before parsing; non-DICOM files are skipped silently
