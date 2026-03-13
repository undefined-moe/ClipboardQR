//go:build darwin

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#include <stdlib.h>

int64_t clipboard_change_count() {
    return (int64_t)[[NSPasteboard generalPasteboard] changeCount];
}

// clipboard_read_image reads PNG or TIFF from the pasteboard.
// Returns NULL if no image is available.
// Caller must free the returned bytes with free().
void* clipboard_read_image(int* outLen) {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];

    // Try PNG first
    NSData *data = [pb dataForType:NSPasteboardTypePNG];

    // Fall back to TIFF (macOS screenshots and most app copies are TIFF)
    if (!data) {
        data = [pb dataForType:NSPasteboardTypeTIFF];
    }

    if (!data || [data length] == 0) {
        *outLen = 0;
        return NULL;
    }

    *outLen = (int)[data length];
    void* buf = malloc(*outLen);
    if (!buf) {
        *outLen = 0;
        return NULL;
    }
    memcpy(buf, [data bytes], *outLen);
    return buf;
}
*/
import "C"

import (
	"context"
	"time"
	"unsafe"
)

const darwinPollInterval = 500 * time.Millisecond

type darwinWatcher struct{}

// NewWatcher creates a macOS clipboard watcher using NSPasteboard polling.
// It handles both PNG and TIFF clipboard images via CGo.
func NewWatcher() (ClipboardWatcher, error) {
	return &darwinWatcher{}, nil
}

func (w *darwinWatcher) Watch(ctx context.Context) (<-chan []byte, error) {
	out := make(chan []byte, 1)
	go func() {
		defer close(out)
		var lastCount C.int64_t = -1
		ticker := time.NewTicker(darwinPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count := C.clipboard_change_count()
				if count == lastCount {
					continue
				}
				lastCount = count

				var outLen C.int
				ptr := C.clipboard_read_image(&outLen)
				if ptr == nil || outLen == 0 {
					continue
				}
				// Copy bytes to Go slice and free C memory
				n := int(outLen)
				imgBytes := make([]byte, n)
				copy(imgBytes, (*[1 << 28]byte)(unsafe.Pointer(ptr))[:n:n])
				C.free(ptr)

				if len(imgBytes) > 0 {
					select {
					case out <- imgBytes:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return out, nil
}
