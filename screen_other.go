//go:build !windows

package main

// primaryScreenSizePx has no portable implementation off Windows; callers fall
// back to the default window size.
func primaryScreenSizePx() (widthPx, heightPx, dpi int, ok bool) {
	return 0, 0, 96, false
}
