// Package volume provides a cross-platform helper for setting system volume.
// On Linux it uses `pactl` (PulseAudio) or falls back to `amixer`.
// On macOS it uses `osascript`.
// On Windows it delegates to github.com/itchyny/volume-go.
package volume

import (
	"fmt"

	volumego "github.com/itchyny/volume-go"
)

// Set adjusts the system master volume to the given percentage (0–100).
func Set(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("volume percent must be between 0 and 100, got %d", percent)
	}
	return volumego.SetVolume(percent)
}

// Get returns the current system master volume as a percentage (0–100).
func Get() (int, error) {
	return volumego.GetVolume()
}
