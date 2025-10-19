//go:build !windows
// +build !windows

package sound

// Non-windows stub: identical API but does nothing.
// Build-tagged for non-Windows targets only so windows build includes only sound_windows.go.

type SoundManager struct {
	disabled bool
	volume   float64
}

func New(sampleRate int) (*SoundManager, error) {
	return &SoundManager{disabled: true, volume: 0}, nil
}

func (sm *SoundManager) SetDisabled(d bool)   { sm.disabled = d }
func (sm *SoundManager) SetVolume(v float64)  { sm.volume = v }
func (sm *SoundManager) PlayEvent(evt string) {}
