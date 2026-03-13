package decode

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	goqr "github.com/skip2/go-qrcode"
)

func makeQRPNG(t *testing.T, content string) []byte {
	t.Helper()
	b, err := goqr.Encode(content, goqr.Medium, 256)
	if err != nil {
		t.Fatalf("failed to generate QR: %v", err)
	}
	return b
}

func makeSolidPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
	return buf.Bytes()
}

func TestDecodeQR(t *testing.T) {
	tests := []struct {
		name      string
		input     func() []byte
		wantText  string
		wantErr   bool
		wantEmpty bool // ("", nil)
	}{
		{
			name:     "valid_qr_text",
			input:    func() []byte { return makeQRPNG(t, "hello world") },
			wantText: "hello world",
		},
		{
			name:     "valid_qr_url",
			input:    func() []byte { return makeQRPNG(t, "https://example.com") },
			wantText: "https://example.com",
		},
		{
			name:      "non_qr_image",
			input:     func() []byte { return makeSolidPNG(t) },
			wantEmpty: true,
		},
		{
			name:    "empty_bytes",
			input:   func() []byte { return []byte{} },
			wantErr: true,
		},
		{
			name:    "corrupt_bytes",
			input:   func() []byte { return []byte{0xFF, 0xFE, 0x00, 0x01} },
			wantErr: true,
		},
		{
			name:     "utf8_content",
			input:    func() []byte { return makeQRPNG(t, "你好世界") },
			wantText: "你好世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeQR(tt.input())
			if tt.wantErr {
				if err == nil {
					t.Errorf("DecodeQR() expected error, got nil (text=%q)", got)
				}
				return
			}
			if err != nil {
				t.Errorf("DecodeQR() unexpected error: %v", err)
				return
			}
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("DecodeQR() expected empty string, got %q", got)
				}
				return
			}
			if got != tt.wantText {
				t.Errorf("DecodeQR() = %q, want %q", got, tt.wantText)
			}
		})
	}
}
