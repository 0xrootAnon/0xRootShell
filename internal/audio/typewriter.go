package audio

import (
	"os"
	"sync"
)

var once sync.Once

// InitDefault is no-op for bell fallback
func InitDefault() {}

// Beep prints ASCII bell (non-blocking).
func Beep() {
	once.Do(func() {
		// try to ensure stdout is not buffered blocking — noop for now
	})
	// best-effort, ignore errors
	_, _ = os.Stdout.Write([]byte("\a"))
}

// Shutdown is no-op
func Shutdown() {}
