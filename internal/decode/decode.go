package decode

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	_ "golang.org/x/image/tiff"
)

// reader is reused across calls — it's thread-safe.
var reader = qrcode.NewQRCodeReader()

// DecodeQR decodes a QR code from PNG, TIFF, or JPEG image bytes.
// Returns ("", nil) if no QR code is found in the image.
// Returns ("", error) if the image cannot be decoded.
func DecodeQR(imgBytes []byte) (string, error) {
	log.Printf("Decode: starting, input size: %d bytes", len(imgBytes))

	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		log.Printf("Decode: image.Decode failed: %v (first 16 bytes: %x)", err, head(imgBytes, 16))
		return "", err
	}
	bounds := img.Bounds()
	log.Printf("Decode: image decoded, format=%q, dimensions=%dx%d", format, bounds.Dx(), bounds.Dy())

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		log.Printf("Decode: NewBinaryBitmapFromImage failed: %v", err)
		return "", err
	}
	log.Println("Decode: binary bitmap created successfully")

	result, err := reader.Decode(bmp, nil)
	if err != nil {
		// NotFoundException is normal — means no QR code in image
		if _, ok := err.(gozxing.NotFoundException); ok {
			log.Println("Decode: no QR code found in image (NotFoundException)")
			return "", nil
		}
		log.Printf("Decode: QR decode error: %v", err)
		return "", err
	}

	text := result.GetText()
	log.Printf("Decode: QR code found, text length=%d, text=%q", len(text), text)
	return text, nil
}

func head(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}
