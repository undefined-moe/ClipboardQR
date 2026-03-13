package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

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
		}
		cancel()
	}()

	watchLoop(ctx, ch, dedup, notifier)

	log.Println("ClipboardQR: stopped.")
}

func watchLoop(ctx context.Context, ch <-chan []byte, dedup *detect.Deduplicator, notifier notify.Notifier) {
	for {
		select {
		case <-ctx.Done():
			return
		case imgBytes, ok := <-ch:
			if !ok {
				return
			}
			processImage(ctx, imgBytes, dedup, notifier)
		}
	}
}

func processImage(ctx context.Context, imgBytes []byte, dedup *detect.Deduplicator, notifier notify.Notifier) {
	if !dedup.IsNew(imgBytes) {
		log.Println("ClipboardQR: skipping duplicate image")
		return
	}

	text, err := decode.DecodeQR(imgBytes)
	if err != nil {
		log.Printf("ClipboardQR: decode error: %v", err)
		return
	}
	if text == "" {
		log.Println("ClipboardQR: no QR code found in image")
		return
	}

	isURL := detect.IsURL(text)
	log.Printf("ClipboardQR: QR detected (isURL=%v): %s", isURL, text)

	if err := notifier.Notify(ctx, text, isURL); err != nil {
		log.Printf("ClipboardQR: notify error: %v", err)
	}
}
