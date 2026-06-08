//go:build windows

package main

import "syscall"

// primaryScreenSizePx returns the primary monitor's size in physical pixels and
// the system DPI (96 = 100% scaling). ok is false when the metrics cannot be
// read. Fyne is DPI-aware on Windows, so GetSystemMetrics reports physical
// pixels; callers divide by dpi/96 to convert to Fyne's logical units.
func primaryScreenSizePx() (widthPx, heightPx, dpi int, ok bool) {
	user32 := syscall.NewLazyDLL("user32.dll")
	getSystemMetrics := user32.NewProc("GetSystemMetrics")
	cx, _, _ := getSystemMetrics.Call(uintptr(0)) // SM_CXSCREEN
	cy, _, _ := getSystemMetrics.Call(uintptr(1)) // SM_CYSCREEN
	if cx == 0 || cy == 0 {
		return 0, 0, 96, false
	}

	dpi = 96 // sensible default if GetDpiForSystem is unavailable (pre-Win10 1607)
	if p := user32.NewProc("GetDpiForSystem"); p.Find() == nil {
		if r, _, _ := p.Call(); r > 0 {
			dpi = int(r)
		}
	}
	return int(cx), int(cy), dpi, true
}
