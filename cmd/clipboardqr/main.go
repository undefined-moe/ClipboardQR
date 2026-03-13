package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clipboardqr/internal/clipboard"
	"clipboardqr/internal/decode"
	"clipboardqr/internal/detect"
	"clipboardqr/internal/notify"
	"clipboardqr/internal/tray"
)

var verbose = flag.Bool("v", false, "enable verbose logging")

func main() {
	flag.Parse()

	if !*verbose {
		log.SetOutput(io.Discard)
	}

	tray.Run(onReady, nil)
}

func onReady() {
	ctx, cancel := context.WithCancel(context.Background())

	watcher, err := clipboard.NewWatcher()
	if err != nil {
		log.Printf("ClipboardQR: failed to create watcher: %v", err)
		cancel()
		return
	}

	notifier, err := notify.NewNotifier()
	if err != nil {
		log.Printf("ClipboardQR: failed to create notifier: %v", err)
		cancel()
		return
	}
	defer notifier.Close()

	dedup := &detect.Deduplicator{}

	ch, err := watcher.Watch(ctx)
	if err != nil {
		log.Printf("ClipboardQR: failed to start watcher: %v", err)
		cancel()
		return
	}

	log.Println("ClipboardQR: watching clipboard for QR codes...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-tray.QuitCh():
			log.Println("ClipboardQR: quit requested via tray")
		case <-sigCh:
			log.Println("ClipboardQR: received shutdown signal")
			tray.Quit()
		}
		cancel()
	}()

	watchLoop(ctx, ch, dedup, notifier)

	log.Println("ClipboardQR: stopped.")
}

func watchLoop(ctx context.Context, ch <-chan []byte, dedup *detect.Deduplicator, notifier notify.Notifier) {
	imgCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("WatchLoop: context cancelled, processed %d images total", imgCount)
			return
		case imgBytes, ok := <-ch:
			if !ok {
				log.Printf("WatchLoop: channel closed, processed %d images total", imgCount)
				return
			}
			imgCount++
			log.Printf("WatchLoop: received image #%d, size=%d bytes", imgCount, len(imgBytes))
			processImage(ctx, imgBytes, dedup, notifier)
		}
	}
}

func processImage(ctx context.Context, imgBytes []byte, dedup *detect.Deduplicator, notifier notify.Notifier) {
	start := time.Now()
	log.Printf("Process: start, image size=%d bytes", len(imgBytes))

	if !dedup.IsNew(imgBytes) {
		log.Printf("Process: skipping duplicate image (dedup took %v)", time.Since(start))
		return
	}
	log.Printf("Process: dedup passed (new image), took %v", time.Since(start))

	decodeStart := time.Now()
	text, err := decode.DecodeQR(imgBytes)
	if err != nil {
		log.Printf("Process: decode error after %v: %v", time.Since(decodeStart), err)
		return
	}
	log.Printf("Process: decode completed in %v, result=%q", time.Since(decodeStart), text)

	if text == "" {
		log.Printf("Process: no QR code found, total time %v", time.Since(start))
		return
	}

	isURL := detect.IsURL(text)
	log.Printf("Process: QR detected (isURL=%v, text=%q), notifying...", isURL, text)

	if err := notifier.Notify(ctx, text, isURL); err != nil {
		log.Printf("Process: notify error: %v", err)
	} else {
		log.Printf("Process: notify succeeded, total time %v", time.Since(start))
	}
}
