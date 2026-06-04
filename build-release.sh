#!/usr/bin/env bash
set -euo pipefail
export PATH="/c/Program Files/Go/bin:$(go env GOPATH)/bin:$PATH"

# Embed the Windows application icon into the executable's PE resources.
# rsrc compiles the .ico into a .syso file; the Go linker automatically links
# any .syso in the package directory into the binary. The _windows_amd64 suffix
# scopes it to Windows amd64 builds via Go's filename build constraints, so
# 'DICOM App Windows.ico' becomes the icon shown for dicomhdr.exe in Explorer
# and the taskbar. Regenerated every build so icon changes always take effect.
if ! command -v rsrc >/dev/null 2>&1; then
  echo "Installing rsrc (icon resource compiler)…"
  go install github.com/akavel/rsrc@latest
fi
rsrc -ico "DICOM App Windows.ico" -o rsrc_windows_amd64.syso
echo "Embedded application icon -> rsrc_windows_amd64.syso"

CGO_ENABLED=1 CC=/c/msys64/mingw64/bin/gcc.exe GOAMD64=v3 \
  go build -ldflags="-s -w -H windowsgui -X main.buildDate=$(date +%Y-%m-%d)" -o dicomhdr.exe .
echo "Built dicomhdr.exe (release)"
