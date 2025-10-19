//go:build windows
// +build windows

package sound

import (
	"sync"
	"syscall"
	"time"
)

// Windows implementation — uses kernel32.Beep.
// No external deps. Beep is synchronous so we call it from goroutines.

type SoundManager struct {
	mu       sync.Mutex
	proc     *syscall.LazyProc
	disabled bool
	volume   float64
}

func New(sampleRate int) (*SoundManager, error) {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel.NewProc("Beep")
	return &SoundManager{
		proc:   proc,
		volume: 1.0,
	}, nil
}

func (sm *SoundManager) SetDisabled(d bool) {
	sm.mu.Lock()
	sm.disabled = d
	sm.mu.Unlock()
}

func (sm *SoundManager) SetVolume(v float64) {
	sm.mu.Lock()
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	sm.volume = v
	sm.mu.Unlock()
}

// PlayEvent plays short tones for events. Non-blocking.
func (sm *SoundManager) PlayEvent(evt string) {
	sm.mu.Lock()
	disabled := sm.disabled
	proc := sm.proc
	sm.mu.Unlock()
	if disabled || proc == nil {
		return
	}

	// run in a goroutine so UI ticks are never blocked
	go func() {
		switch evt {
		case "typeclick":
			callBeep(proc, 1200, 12) // short click (per character)
		case "response":
			callBeep(proc, 880, 60)
			time.Sleep(12 * time.Millisecond)
			callBeep(proc, 990, 40)
		case "startup":
			callBeep(proc, 220, 160)
			time.Sleep(28 * time.Millisecond)
			callBeep(proc, 330, 110)
		case "error":
			callBeep(proc, 440, 100)
			time.Sleep(10 * time.Millisecond)
			callBeep(proc, 370, 140)
		case "notification":
			callBeep(proc, 660, 150)
		default:
			callBeep(proc, 880, 50)
		}
	}()
}

func callBeep(proc *syscall.LazyProc, freq int, durMs int) {
	// freq range enforced by Windows Beep: 37..32767
	if freq < 37 {
		freq = 37
	}
	if freq > 32767 {
		freq = 32767
	}
	if durMs < 5 {
		durMs = 5
	}
	// ignore return values
	proc.Call(uintptr(freq), uintptr(durMs))
}
