package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DetailLevel represents the level of detail for image analysis
type DetailLevel string

const (
	// DetailAuto uses the default detail level
	DetailAuto DetailLevel = "auto"
	// DetailLow uses a low detail level
	DetailLow DetailLevel = "low"
	// DetailHigh uses a high detail level
	DetailHigh DetailLevel = "high"
)

// Image represents an image with various operations
type Image struct {
	img image.Image
}

// ImageData is a struct for JSON serialization
type ImageData struct {
	Data string `json:"data"`
}

// FromPIL creates an Image from a standard Go image
func FromPIL(img image.Image) *Image {
	return &Image{img: img}
}

// FromURI creates an Image from a data URI
func FromURI(uri string) (*Image, error) {
	uriPattern := regexp.MustCompile(`^data:image/(?:png|jpeg);base64,(.*)$`)
	matches := uriPattern.FindStringSubmatch(uri)
	if len(matches) != 2 {
		return nil, fmt.Errorf("invalid URI format. it should be a base64 encoded image URI")
	}

	// Extract base64 data
	base64Data := matches[1]
	return FromBase64(base64Data)
}

// FromURL creates an Image from a URL
func FromURL(url string) (*Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return &Image{img: img}, nil
}

// FromBase64 creates an Image from a base64 string
func FromBase64(base64Str string) (*Image, error) {
	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return &Image{img: img}, nil
}

// ToBase64 converts the image to a base64 string
func (i *Image) ToBase64() (string, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, i.img)
	if err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// FromFile creates an Image from a file path
func FromFile(filePath string) (*Image, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return &Image{img: img}, nil
}

// DataURI returns the data URI representation of the image
func (i *Image) DataURI() (string, error) {
	base64Str, err := i.ToBase64()
	if err != nil {
		return "", err
	}
	return convertBase64ToDataURI(base64Str), nil
}

// ToOpenAIFormat converts the image to the format expected by OpenAI
func (i *Image) ToOpenAIFormat(detail DetailLevel) (map[string]interface{}, error) {
	dataURI, err := i.DataURI()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"type": "image_url",
		"image_url": map[string]interface{}{
			"url":    dataURI,
			"detail": string(detail),
		},
	}, nil
}

// getMIMETypeFromData determines the MIME type based on image data
func getMIMETypeFromData(data []byte) string {
	if len(data) < 12 {
		return "image/jpeg" // Default for small data
	}

	// Check signatures for common formats
	if bytes.HasPrefix(data, []byte("\xFF\xD8\xFF")) {
		return "image/jpeg"
	} else if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return "image/png"
	} else if bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a")) {
		return "image/gif"
	} else if bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")) {
		return "image/webp"
	}

	return "image/jpeg" // Default for unknown formats
}

// convertBase64ToDataURI converts a base64 string to a data URI
func convertBase64ToDataURI(base64Image string) string {
	data, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return fmt.Sprintf("data:image/png;base64,%s", base64Image)
	}

	mimeType := getMIMETypeFromData(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Image)
}

// SaveToFile saves the image to a file with format determined by extension
func (i *Image) SaveToFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg":
		return jpeg.Encode(file, i.img, nil)
	case ".png":
		return png.Encode(file, i.img)
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}
}

// HTTPHandler returns the image as an HTTP response
func (i *Image) HTTPHandler(w http.ResponseWriter, format string) error {
	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
		return jpeg.Encode(w, i.img, nil)
	case "png":
		w.Header().Set("Content-Type", "image/png")
		return png.Encode(w, i.img)
	default:
		return fmt.Errorf("unsupported image format: %s", format)
	}
}

// Clone creates a copy of the image
func (i *Image) Clone() *Image {
	bounds := i.img.Bounds()
	newImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			newImg.Set(x, y, i.img.At(x, y))
		}
	}
	return &Image{img: newImg}
}

// FromReaderWithFormat decodes an image from an io.Reader with a specified format
func FromReaderWithFormat(r io.Reader, format string) (*Image, error) {
	var img image.Image
	var err error

	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		img, err = jpeg.Decode(r)
	case "png":
		img, err = png.Decode(r)
	default:
		// Try to decode automatically
		img, _, err = image.Decode(r)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return &Image{img: img}, nil
}
