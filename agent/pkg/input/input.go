package input

import (
	"fmt"

	"github.com/go-vgo/robotgo"
	log "github.com/sirupsen/logrus"
)

// MouseMethod represents different mouse actions
type MouseMethod int

const (
	MouseMove MouseMethod = iota
	MouseLeftClick
	MouseMiddleClick
	MouseRightClick = 4
)

// KeyMethod represents different key actions
type KeyMethod int

const (
	KeyPress KeyMethod = iota
	KeyRelease
)

// HandleMouseEvent processes mouse input events
func HandleMouseEvent(x, y int, method int) error {
	log.WithFields(log.Fields{
		"x":      x,
		"y":      y,
		"method": method,
	}).Debug("Processing mouse event")

	// Move mouse to position
	robotgo.Move(x, y)

	// Perform action based on method
	switch MouseMethod(method) {
	case MouseMove:
		// Just move, no click
		break
	case MouseLeftClick:
		robotgo.Click("left")
	case MouseMiddleClick:
		robotgo.Click("middle")
	case MouseRightClick:
		robotgo.Click("right")
	default:
		return fmt.Errorf("unknown mouse method: %d", method)
	}

	return nil
}

// HandleKeyEvent processes keyboard input events
func HandleKeyEvent(key int, method int) error {
	log.WithFields(log.Fields{
		"key":    key,
		"method": method,
	}).Debug("Processing key event")

	// Convert key code to robotgo key string
	keyStr := convertKeyCode(key)
	if keyStr == "" {
		return fmt.Errorf("unsupported key code: %d", key)
	}

	// Perform action based on method
	switch KeyMethod(method) {
	case KeyPress:
		robotgo.KeyDown(keyStr)
	case KeyRelease:
		robotgo.KeyUp(keyStr)
	default:
		return fmt.Errorf("unknown key method: %d", method)
	}

	return nil
}

// HandlePasteEvent processes clipboard paste operations
func HandlePasteEvent(data string) error {
	log.WithField("data_length", len(data)).Debug("Processing paste event")

	// Set clipboard content
	robotgo.WriteAll(data)

	// Simulate Ctrl+V
	robotgo.KeyDown("ctrl")
	robotgo.KeyTap("v")
	robotgo.KeyUp("ctrl")

	return nil
}

// convertKeyCode converts numeric key codes to robotgo key strings
func convertKeyCode(keyCode int) string {
	// Handle ASCII printable characters
	if keyCode >= 32 && keyCode <= 126 {
		return string(rune(keyCode))
	}

	// Handle special keys
	specialKeys := map[int]string{
		8:   "backspace",
		9:   "tab",
		13:  "enter",
		16:  "shift",
		17:  "ctrl",
		18:  "alt",
		20:  "caps_lock",
		27:  "escape",
		32:  "space",
		33:  "page_up",
		34:  "page_down",
		35:  "end",
		36:  "home",
		37:  "left",
		38:  "up",
		39:  "right",
		40:  "down",
		45:  "insert",
		46:  "delete",
		112: "f1",
		113: "f2",
		114: "f3",
		115: "f4",
		116: "f5",
		117: "f6",
		118: "f7",
		119: "f8",
		120: "f9",
		121: "f10",
		122: "f11",
		123: "f12",
	}

	if key, exists := specialKeys[keyCode]; exists {
		return key
	}

	// Return empty string for unsupported keys
	log.WithField("key_code", keyCode).Warn("Unsupported key code")
	return ""
}

// TypeText types the given text using robotgo
func TypeText(text string) error {
	log.WithField("text_length", len(text)).Debug("Typing text")
	robotgo.TypeStr(text)
	return nil
}

// SendKeyCombo sends a key combination
func SendKeyCombo(keys ...string) error {
	log.WithField("keys", keys).Debug("Sending key combination")

	if len(keys) == 0 {
		return fmt.Errorf("no keys provided")
	}

	// Press all keys down
	for _, key := range keys {
		robotgo.KeyDown(key)
	}

	// Release all keys in reverse order
	for i := len(keys) - 1; i >= 0; i-- {
		robotgo.KeyUp(keys[i])
	}

	return nil
}
