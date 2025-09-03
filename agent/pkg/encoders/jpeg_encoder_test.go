package encoders

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

func TestJPEGEncoder(t *testing.T) {
	size := image.Point{X: 320, Y: 240}
	encoder, err := newJPEGEncoder(size, 30)
	if err != nil {
		t.Fatalf("Failed to create JPEG encoder: %v", err)
	}
	defer encoder.Close()

	// Test encoder properties
	if encoder.GetCodec() != JPEGCodec {
		t.Error("JPEG encoder codec mismatch")
	}

	videoSize, err := encoder.VideoSize()
	if err != nil {
		t.Fatalf("Failed to get video size: %v", err)
	}

	if videoSize != size {
		t.Errorf("Video size mismatch: expected %v, got %v", size, videoSize)
	}

	// Create a test image
	testImage := createTestImage(size.X, size.Y)

	// Test encoding
	encoded, err := encoder.Encode(testImage)
	if err != nil {
		t.Fatalf("Failed to encode image: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data is empty")
	}

	// Check JPEG header
	if len(encoded) < 2 || encoded[0] != 0xFF || encoded[1] != 0xD8 {
		t.Error("Invalid JPEG header")
	}
}

func TestJPEGEncoderWithOptions(t *testing.T) {
	size := image.Point{X: 160, Y: 120}
	opts := JPEGOptions{Quality: 95}

	encoder, err := newJPEGEncoderWithOptions(size, 15, opts)
	if err != nil {
		t.Fatalf("Failed to create JPEG encoder with options: %v", err)
	}
	defer encoder.Close()

	jpegEncoder := encoder.(*JPEGEncoder)
	if jpegEncoder.GetQuality() != 95 {
		t.Errorf("Quality mismatch: expected 95, got %d", jpegEncoder.GetQuality())
	}
}

func TestJPEGEncoderInvalidInputs(t *testing.T) {
	// Test invalid size
	_, err := newJPEGEncoder(image.Point{X: 0, Y: 0}, 30)
	if err == nil {
		t.Error("Expected error for invalid size")
	}

	_, err = newJPEGEncoder(image.Point{X: -1, Y: 100}, 30)
	if err == nil {
		t.Error("Expected error for negative width")
	}

	// Test invalid quality
	size := image.Point{X: 100, Y: 100}
	_, err = newJPEGEncoderWithOptions(size, 30, JPEGOptions{Quality: 0})
	if err == nil {
		t.Error("Expected error for quality 0")
	}

	_, err = newJPEGEncoderWithOptions(size, 30, JPEGOptions{Quality: 101})
	if err == nil {
		t.Error("Expected error for quality 101")
	}
}

func TestJPEGEncoderNilFrame(t *testing.T) {
	size := image.Point{X: 100, Y: 100}
	encoder, err := newJPEGEncoder(size, 30)
	if err != nil {
		t.Fatalf("Failed to create JPEG encoder: %v", err)
	}
	defer encoder.Close()

	// Test encoding nil frame
	_, err = encoder.Encode(nil)
	if err == nil {
		t.Error("Expected error for nil frame")
	}
}

func TestJPEGEncoderQualitySettings(t *testing.T) {
	size := image.Point{X: 100, Y: 100}
	encoder, err := newJPEGEncoder(size, 30)
	if err != nil {
		t.Fatalf("Failed to create JPEG encoder: %v", err)
	}
	defer encoder.Close()

	jpegEncoder := encoder.(*JPEGEncoder)

	// Test setting valid quality
	err = jpegEncoder.SetQuality(50)
	if err != nil {
		t.Fatalf("Failed to set quality: %v", err)
	}

	if jpegEncoder.GetQuality() != 50 {
		t.Errorf("Quality mismatch: expected 50, got %d", jpegEncoder.GetQuality())
	}

	// Test setting invalid quality
	err = jpegEncoder.SetQuality(0)
	if err == nil {
		t.Error("Expected error for quality 0")
	}

	err = jpegEncoder.SetQuality(101)
	if err == nil {
		t.Error("Expected error for quality 101")
	}
}

func TestJPEGEncoderSizeMismatch(t *testing.T) {
	size := image.Point{X: 100, Y: 100}
	encoder, err := newJPEGEncoder(size, 30)
	if err != nil {
		t.Fatalf("Failed to create JPEG encoder: %v", err)
	}
	defer encoder.Close()

	// Create image with different size
	testImage := createTestImage(200, 150)

	// Should still work but log a warning
	encoded, err := encoder.Encode(testImage)
	if err != nil {
		t.Fatalf("Failed to encode image with size mismatch: %v", err)
	}

	if len(encoded) == 0 {
		t.Error("Encoded data is empty")
	}
}

func TestDefaultJPEGOptions(t *testing.T) {
	opts := DefaultJPEGOptions()
	if opts.Quality != 80 {
		t.Errorf("Default quality mismatch: expected 80, got %d", opts.Quality)
	}
}

// Helper function to create a test image
func createTestImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	// Fill with a gradient pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	
	return img
}

func BenchmarkJPEGEncoding(b *testing.B) {
	size := image.Point{X: 640, Y: 480}
	encoder, err := newJPEGEncoder(size, 30)
	if err != nil {
		b.Fatalf("Failed to create JPEG encoder: %v", err)
	}
	defer encoder.Close()

	testImage := createTestImage(size.X, size.Y)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := encoder.Encode(testImage)
		if err != nil {
			b.Fatalf("Failed to encode image: %v", err)
		}
	}
}

func BenchmarkJPEGEncodingDifferentQualities(b *testing.B) {
	size := image.Point{X: 320, Y: 240}
	testImage := createTestImage(size.X, size.Y)
	qualities := []int{10, 50, 80, 95}

	for _, quality := range qualities {
		b.Run(fmt.Sprintf("Quality%d", quality), func(b *testing.B) {
			encoder, err := newJPEGEncoderWithOptions(size, 30, JPEGOptions{Quality: quality})
			if err != nil {
				b.Fatalf("Failed to create JPEG encoder: %v", err)
			}
			defer encoder.Close()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := encoder.Encode(testImage)
				if err != nil {
					b.Fatalf("Failed to encode image: %v", err)
				}
			}
		})
	}
}
