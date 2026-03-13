package detect

import (
	"crypto/sha256"
	"sync"
)

// Deduplicator tracks the last processed image hash to avoid re-processing
// the same clipboard image multiple times.
type Deduplicator struct {
	mu   sync.Mutex
	last [sha256.Size]byte
	set  bool
}

// IsNew returns true if imgBytes is different from the last processed image.
// The first call always returns true.
// Thread-safe.
func (d *Deduplicator) IsNew(imgBytes []byte) bool {
	h := sha256.Sum256(imgBytes)
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.set && h == d.last {
		return false
	}
	d.last = h
	d.set = true
	return true
}
