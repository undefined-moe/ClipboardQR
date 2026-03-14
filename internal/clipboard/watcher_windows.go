//go:build windows

package clipboard

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image/png"
	"log"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/image/bmp"
)

const pollInterval = 500 * time.Millisecond

type windowsWatcher struct{}

// NewWatcher creates a Windows clipboard watcher.
func NewWatcher() (ClipboardWatcher, error) {
	return &windowsWatcher{}, nil
}

func (w *windowsWatcher) Watch(ctx context.Context) (<-chan []byte, error) {
	log.Printf("Clipboard: starting Windows watcher (polling, interval=%v, PNG format ID=%d)", pollInterval, pngFormatID)
	out := make(chan []byte, 1)

	go func() {
		defer close(out)
		seq, _, _ := procGetClipboardSequenceNumber.Call()
		log.Printf("Clipboard: initial sequence=%d", seq)

		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		eventCount := 0
		for {
			select {
			case <-ctx.Done():
				log.Printf("Clipboard: context cancelled, total image events: %d", eventCount)
				return
			case <-ticker.C:
				cur, _, _ := procGetClipboardSequenceNumber.Call()
				if cur == seq {
					continue
				}
				log.Printf("Clipboard: sequence changed %d -> %d", seq, cur)
				seq = cur

				imgBytes, format, err := readClipboardImage()
				if err != nil {
					log.Printf("Clipboard: no image: %v", err)
					continue
				}

				eventCount++
				log.Printf("Clipboard: event #%d: %d bytes via %s", eventCount, len(imgBytes), format)

				select {
				case out <- imgBytes:
					log.Printf("Clipboard: event #%d forwarded", eventCount)
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

func readClipboardImage() ([]byte, string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var r uintptr
	for i := 0; i < 10; i++ {
		r, _, _ = procOpenClipboard.Call(0)
		if r != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r == 0 {
		return nil, "", fmt.Errorf("OpenClipboard failed after retries")
	}
	defer procCloseClipboard.Call()

	hasPNG := pngFormatID != 0 && isFormatAvailable(pngFormatID)
	hasDIB := isFormatAvailable(cfDIB)
	hasDIBV5 := isFormatAvailable(cfDIBV5)
	log.Printf("Clipboard: formats available - PNG=%v CF_DIB=%v CF_DIBV5=%v", hasPNG, hasDIB, hasDIBV5)

	if hasPNG {
		if data, err := readRawFormat(pngFormatID); err == nil && len(data) > 0 {
			return data, "PNG", nil
		}
	}

	if hasDIB || hasDIBV5 {
		if data, err := readDIBFormat(); err == nil && len(data) > 0 {
			return data, "CF_DIB", nil
		}
	}

	return nil, "", fmt.Errorf("no image format available (PNG=%v CF_DIB=%v CF_DIBV5=%v)", hasPNG, hasDIB, hasDIBV5)
}

func readRawFormat(format uintptr) ([]byte, error) {
	hMem, _, _ := procGetClipboardData.Call(format)
	if hMem == 0 {
		return nil, fmt.Errorf("GetClipboardData(%d) returned 0", format)
	}
	return copyGlobalMem(hMem, 0)
}

func readDIBFormat() ([]byte, error) {
	hMem, _, _ := procGetClipboardData.Call(cfDIB)
	if hMem == 0 {
		return nil, fmt.Errorf("CF_DIB not available")
	}
	dibData, err := copyGlobalMem(hMem, 40)
	if err != nil {
		return nil, err
	}
	return dibToPNG(dibData)
}

func copyGlobalMem(hMem uintptr, minSize uintptr) ([]byte, error) {
	p, _, _ := procGlobalLock.Call(hMem)
	if p == 0 {
		return nil, fmt.Errorf("GlobalLock failed")
	}
	defer procGlobalUnlock.Call(hMem)

	size, _, _ := procGlobalSize.Call(hMem)
	if size == 0 || size < minSize {
		return nil, fmt.Errorf("global mem too small: %d bytes (need >=%d)", size, minSize)
	}

	data := make([]byte, size)
	procRtlMoveMemory.Call(uintptr(unsafe.Pointer(&data[0])), p, size)
	return data, nil
}

func dibToPNG(dib []byte) ([]byte, error) {
	const fileHeaderLen = 14

	biSize := binary.LittleEndian.Uint32(dib[0:4])
	biBitCount := binary.LittleEndian.Uint16(dib[14:16])
	biCompression := binary.LittleEndian.Uint32(dib[16:20])
	biClrUsed := binary.LittleEndian.Uint32(dib[32:36])

	colorTableSize := uint32(0)
	if biBitCount <= 8 {
		colors := biClrUsed
		if colors == 0 {
			colors = 1 << uint(biBitCount)
		}
		colorTableSize = colors * 4
	} else if biCompression == 3 && biSize == 40 {
		colorTableSize = 12
	}

	bfOffBits := uint32(fileHeaderLen) + biSize + colorTableSize

	log.Printf("Clipboard: DIB biSize=%d %dx%d bits=%d compression=%d colorTable=%d offBits=%d",
		biSize,
		int32(binary.LittleEndian.Uint32(dib[4:8])),
		int32(binary.LittleEndian.Uint32(dib[8:12])),
		biBitCount, biCompression, colorTableSize, bfOffBits)

	var bmpBuf bytes.Buffer
	binary.Write(&bmpBuf, binary.LittleEndian, uint16(0x4D42))
	binary.Write(&bmpBuf, binary.LittleEndian, uint32(fileHeaderLen)+uint32(len(dib)))
	binary.Write(&bmpBuf, binary.LittleEndian, uint32(0))
	binary.Write(&bmpBuf, binary.LittleEndian, bfOffBits)
	bmpBuf.Write(dib)

	img, err := bmp.Decode(&bmpBuf)
	if err != nil {
		return nil, fmt.Errorf("bmp.Decode: %w", err)
	}

	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		return nil, fmt.Errorf("png.Encode: %w", err)
	}

	return pngBuf.Bytes(), nil
}

func isFormatAvailable(format uintptr) bool {
	r, _, _ := procIsClipboardFormatAvailable.Call(format)
	return r != 0
}

const (
	cfDIB   = 8
	cfDIBV5 = 17
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
	procRegisterClipboardFormatA   = user32.NewProc("RegisterClipboardFormatA")

	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGlobalLock     = kernel32.NewProc("GlobalLock")
	procGlobalUnlock   = kernel32.NewProc("GlobalUnlock")
	procGlobalSize     = kernel32.NewProc("GlobalSize")
	procRtlMoveMemory = kernel32.NewProc("RtlMoveMemory")

	pngFormatID uintptr
)

func init() {
	namePtr, _ := syscall.BytePtrFromString("PNG")
	pngFormatID, _, _ = procRegisterClipboardFormatA.Call(uintptr(unsafe.Pointer(namePtr)))
}
