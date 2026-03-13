package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
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
	short := hex.EncodeToString(h[:8])
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.set && h == d.last {
		log.Printf("Dedup: duplicate detected, hash=%s, size=%d bytes", short, len(imgBytes))
		return false
	}
	prevSet := d.set
	prevShort := hex.EncodeToString(d.last[:8])
	d.last = h
	d.set = true
	if prevSet {
		log.Printf("Dedup: new image, hash=%s (prev=%s), size=%d bytes", short, prevShort, len(imgBytes))
	} else {
		log.Printf("Dedup: first image, hash=%s, size=%d bytes", short, len(imgBytes))
	}
	return true
}
