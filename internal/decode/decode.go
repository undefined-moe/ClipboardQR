package decode

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"

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
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", err
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", err
	}

	result, err := reader.Decode(bmp, nil)
	if err != nil {
		// NotFoundException is normal — means no QR code in image
		if _, ok := err.(gozxing.NotFoundException); ok {
			return "", nil
		}
		return "", err
	}

	return result.GetText(), nil
}
