package utility

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/png"
	"strings"

	"golang.org/x/image/bmp"
)

// Base64PNGToBMP converts a base64 encoded PNG image string to a base64 encoded BMP image string.
func Base64PNGToBMP(base64Str string) (string, error) {
	if base64Str == "" {
		return "", nil
	}

	// Remove data URI prefix if present
	if strings.Contains(base64Str, ",") {
		parts := strings.SplitN(base64Str, ",", 2)
		base64Str = parts[1]
	}

	// Decode Base64
	imgBytes, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return "", err
	}

	// Decode PNG
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return "", err
	}

	// Encode as BMP
	var bmpBuffer bytes.Buffer
	if err := bmp.Encode(&bmpBuffer, img); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(bmpBuffer.Bytes()), nil
}
