package api

import (
	"context"
	"fmt"
	"time"

	pb "winmanager-agent/protos"
	"winmanager-agent/pkg/screen"
	"winmanager-agent/pkg/input"

	log "github.com/sirupsen/logrus"
)

// GRPCServer implements the gRPC service interface
type GRPCServer struct {
	pb.UnimplementedGuacdServer
}

// Mouse handles mouse input events
func (s *GRPCServer) Mouse(ctx context.Context, req *pb.MouseRequest) (*pb.MouseReply, error) {
	log.WithFields(log.Fields{
		"x":      req.X,
		"y":      req.Y,
		"method": req.Method,
		"ts":     req.Ts,
	}).Debug("Received mouse event")

	// Validate coordinates
	if req.X < 0 || req.Y < 0 {
		return nil, fmt.Errorf("invalid coordinates: x=%d, y=%d", req.X, req.Y)
	}

	// Execute mouse action
	if err := input.HandleMouseEvent(int(req.X), int(req.Y), int(req.Method)); err != nil {
		log.WithError(err).Error("Failed to handle mouse event")
		return nil, fmt.Errorf("mouse event failed: %w", err)
	}

	return &pb.MouseReply{
		Ack: time.Now().Unix(),
	}, nil
}

// Key handles keyboard input events
func (s *GRPCServer) Key(ctx context.Context, req *pb.KeyRequest) (*pb.KeyReply, error) {
	log.WithFields(log.Fields{
		"key":    req.Key,
		"method": req.Method,
		"ts":     req.Ts,
	}).Debug("Received key event")

	// Execute keyboard action
	if err := input.HandleKeyEvent(int(req.Key), int(req.Method)); err != nil {
		log.WithError(err).Error("Failed to handle key event")
		return nil, fmt.Errorf("key event failed: %w", err)
	}

	return &pb.KeyReply{
		Ack: time.Now().Unix(),
	}, nil
}

// Screenshot captures and returns a screenshot
func (s *GRPCServer) Screenshot(ctx context.Context, req *pb.ScreenshotRequest) (*pb.ScreenshotReply, error) {
	log.Debug("Received screenshot request")

	// Capture screenshot
	imageData, err := screen.CaptureScreenshot()
	if err != nil {
		log.WithError(err).Error("Failed to capture screenshot")
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	log.WithField("size", len(imageData)).Debug("Screenshot captured")

	return &pb.ScreenshotReply{
		Image: imageData,
	}, nil
}

// Paste handles clipboard paste operations
func (s *GRPCServer) Paste(ctx context.Context, req *pb.PasteRequest) (*pb.PasteReply, error) {
	log.WithField("data_length", len(req.Data)).Debug("Received paste request")

	// Validate input
	if req.Data == "" {
		return nil, fmt.Errorf("paste data cannot be empty")
	}

	// Execute paste operation
	if err := input.HandlePasteEvent(req.Data); err != nil {
		log.WithError(err).Error("Failed to handle paste event")
		return nil, fmt.Errorf("paste failed: %w", err)
	}

	return &pb.PasteReply{
		Ack: time.Now().Unix(),
	}, nil
}
