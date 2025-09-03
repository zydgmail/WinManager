package encoders

import (
	"image"
	"testing"
)

func TestEncoderService(t *testing.T) {
	service := NewEncoderService()

	// Test basic service functionality
	if service == nil {
		t.Fatal("NewEncoderService returned nil")
	}

	// Test supported codecs
	supportedCodecs := service.GetSupportedCodecs()
	if len(supportedCodecs) == 0 {
		t.Error("No supported codecs found")
	}

	// Test JPEG codec (should always be available)
	if !service.Supports(JPEGCodec) {
		t.Error("JPEG codec should always be supported")
	}

	// Test creating JPEG encoder
	size := image.Point{X: 640, Y: 480}
	encoder, err := service.NewEncoder(JPEGCodec, size, 30)
	if err != nil {
		t.Fatalf("Failed to create JPEG encoder: %v", err)
	}
	defer encoder.Close()

	if encoder.GetCodec() != JPEGCodec {
		t.Error("Encoder codec mismatch")
	}

	videoSize, err := encoder.VideoSize()
	if err != nil {
		t.Fatalf("Failed to get video size: %v", err)
	}

	if videoSize != size {
		t.Errorf("Video size mismatch: expected %v, got %v", size, videoSize)
	}
}

func TestCodecNames(t *testing.T) {
	testCases := []struct {
		codec VideoCodec
		name  string
	}{
		{NoCodec, "none"},
		{JPEGCodec, "jpeg"},
		{JPEGTurboCodec, "jpeg-turbo"},
		{H264Codec, "h264"},
		{VP8Codec, "vp8"},
		{NVENCCodec, "nvenc"},
	}

	for _, tc := range testCases {
		name := GetCodecName(tc.codec)
		if name != tc.name {
			t.Errorf("GetCodecName(%d) = %s, want %s", tc.codec, name, tc.name)
		}

		codec := ParseCodecName(tc.name)
		if codec != tc.codec {
			t.Errorf("ParseCodecName(%s) = %d, want %d", tc.name, codec, tc.codec)
		}
	}

	// Test unknown codec
	unknownName := GetCodecName(VideoCodec(999))
	if unknownName != "unknown" {
		t.Errorf("GetCodecName(999) = %s, want unknown", unknownName)
	}

	unknownCodec := ParseCodecName("invalid")
	if unknownCodec != NoCodec {
		t.Errorf("ParseCodecName(invalid) = %d, want %d", unknownCodec, NoCodec)
	}
}

func TestEncoderRegistration(t *testing.T) {
	// Test registering a custom encoder
	customCodec := VideoCodec(100)
	
	// Create a dummy factory
	factory := func(size image.Point, frameRate int) (Encoder, error) {
		return &JPEGEncoder{size: size, quality: 80}, nil
	}

	// Register the encoder
	RegisterEncoder(customCodec, factory)

	// Check if it's registered
	service := NewEncoderService()
	if !service.Supports(customCodec) {
		t.Error("Custom codec should be supported after registration")
	}

	// Test creating encoder with custom codec
	encoder, err := service.NewEncoder(customCodec, image.Point{X: 320, Y: 240}, 15)
	if err != nil {
		t.Fatalf("Failed to create custom encoder: %v", err)
	}
	defer encoder.Close()

	// Unregister the encoder
	UnregisterEncoder(customCodec)

	// Check if it's unregistered
	if service.Supports(customCodec) {
		t.Error("Custom codec should not be supported after unregistration")
	}
}

func TestInvalidEncoderCreation(t *testing.T) {
	service := NewEncoderService()

	// Test unsupported codec
	_, err := service.NewEncoder(VideoCodec(999), image.Point{X: 640, Y: 480}, 30)
	if err == nil {
		t.Error("Expected error for unsupported codec")
	}

	// Test invalid size
	_, err = service.NewEncoder(JPEGCodec, image.Point{X: 0, Y: 0}, 30)
	if err == nil {
		t.Error("Expected error for invalid size")
	}

	// Test negative size
	_, err = service.NewEncoder(JPEGCodec, image.Point{X: -1, Y: -1}, 30)
	if err == nil {
		t.Error("Expected error for negative size")
	}
}

func TestNilFactoryRegistration(t *testing.T) {
	// Test registering nil factory (should not panic)
	customCodec := VideoCodec(101)
	RegisterEncoder(customCodec, nil)

	service := NewEncoderService()
	if service.Supports(customCodec) {
		t.Error("Nil factory should not be registered")
	}
}

func BenchmarkEncoderCreation(b *testing.B) {
	service := NewEncoderService()
	size := image.Point{X: 640, Y: 480}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder, err := service.NewEncoder(JPEGCodec, size, 30)
		if err != nil {
			b.Fatalf("Failed to create encoder: %v", err)
		}
		encoder.Close()
	}
}

func BenchmarkCodecNameLookup(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetCodecName(JPEGCodec)
		_ = ParseCodecName("jpeg")
	}
}
