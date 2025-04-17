package utils

import (
	"context"
	"fmt"
	"image"
	"net/http"
	"time"
)

// AsyncFromURL fetches an image from a URL asynchronously
// It returns a channel that will receive the image or an error
func AsyncFromURL(url string) chan struct {
	Img *Image
	Err error
} {
	resultChan := make(chan struct {
		Img *Image
		Err error
	})

	go func() {
		// Create a context with timeout to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create an HTTP request with the context
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			resultChan <- struct {
				Img *Image
				Err error
			}{nil, fmt.Errorf("failed to create request: %w", err)}
			return
		}

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			resultChan <- struct {
				Img *Image
				Err error
			}{nil, fmt.Errorf("failed to download image: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			resultChan <- struct {
				Img *Image
				Err error
			}{nil, fmt.Errorf("failed to download image, status: %d", resp.StatusCode)}
			return
		}

		// Decode the image
		img, _, err := decodeImageWithContext(ctx, resp.Body)
		if err != nil {
			resultChan <- struct {
				Img *Image
				Err error
			}{nil, fmt.Errorf("failed to decode image: %w", err)}
			return
		}

		// Return the image
		resultChan <- struct {
			Img *Image
			Err error
		}{FromPIL(img), nil}
	}()

	return resultChan
}

// decodeImageWithContext decodes an image with context awareness for timeouts
func decodeImageWithContext(ctx context.Context, r ReadCloserWithContext) (image.Image, string, error) {
	// Check if the context is already done
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	default:
		// Continue processing
	}

	// Start a goroutine to decode the image
	type decodeResult struct {
		img    image.Image
		format string
		err    error
	}

	resultChan := make(chan decodeResult)
	go func() {
		img, format, err := image.Decode(r)
		resultChan <- decodeResult{img, format, err}
	}()

	// Wait for either the image to be decoded or the context to be canceled
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, "", fmt.Errorf("failed to decode image: %w", result.err)
		}
		return result.img, result.format, nil
	case <-ctx.Done():
		// Try to cancel the reader if it supports it
		if closer, ok := r.(cancellableReader); ok {
			closer.Cancel()
		}
		return nil, "", ctx.Err()
	}
}

// ReadCloserWithContext is an interface for readers that can be closed and may support cancellation
type ReadCloserWithContext interface {
	Read(p []byte) (n int, err error)
	Close() error
}

// cancellableReader is an optional interface for readers that can be cancelled
type cancellableReader interface {
	Cancel() error
}
