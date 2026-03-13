package detect

import (
	"sync"
	"testing"
)

func TestDeduplicator_IsNew(t *testing.T) {
	t.Run("first_call_always_new", func(t *testing.T) {
		d := &Deduplicator{}
		if !d.IsNew([]byte("hello")) {
			t.Error("first call should return true")
		}
	})

	t.Run("same_bytes_not_new", func(t *testing.T) {
		d := &Deduplicator{}
		data := []byte("same image data")
		d.IsNew(data) // first call
		if d.IsNew(data) {
			t.Error("same bytes should return false on second call")
		}
	})

	t.Run("different_bytes_is_new", func(t *testing.T) {
		d := &Deduplicator{}
		d.IsNew([]byte("first"))
		if !d.IsNew([]byte("second")) {
			t.Error("different bytes should return true")
		}
	})

	t.Run("returns_to_false_on_repeat", func(t *testing.T) {
		d := &Deduplicator{}
		d.IsNew([]byte("a"))
		d.IsNew([]byte("b"))
		// Going back to "a" — different from last ("b"), so should be new
		if !d.IsNew([]byte("a")) {
			t.Error("repeating an old value that differs from last should be new")
		}
	})

	t.Run("concurrent_safe", func(t *testing.T) {
		d := &Deduplicator{}
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				d.IsNew([]byte{byte(i)})
			}(i)
		}
		wg.Wait()
		// No race — just verifying no panic or deadlock
	})
}
