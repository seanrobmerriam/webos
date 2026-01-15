package wm

import "errors"

// WindowState represents the current state of a window.
type WindowState int

const (
	// WindowStateNormal indicates the window is in its normal/regular state.
	WindowStateNormal WindowState = iota
	// WindowStateMinimized indicates the window is minimized (hidden from view).
	WindowStateMinimized
	// WindowStateMaximized indicates the window is maximized (full screen).
	WindowStateMaximized
	// WindowStateFullscreen indicates the window is in fullscreen mode.
	WindowStateFullscreen
)

// String returns a string representation of the window state.
func (s WindowState) String() string {
	switch s {
	case WindowStateNormal:
		return "normal"
	case WindowStateMinimized:
		return "minimized"
	case WindowStateMaximized:
		return "maximized"
	case WindowStateFullscreen:
		return "fullscreen"
	default:
		return "unknown"
	}
}

// WindowType represents the type of a window.
type WindowType int

const (
	// WindowTypeRegular is a normal application window.
	WindowTypeRegular WindowType = iota
	// WindowTypeDialog is a modal or modeless dialog window.
	WindowTypeDialog
	// WindowTypeUtility is a utility window like a palette or toolbar.
	WindowTypeUtility
	// WindowTypeTooltip is a tooltip window.
	WindowTypeTooltip
)

// String returns a string representation of the window type.
func (t WindowType) String() string {
	switch t {
	case WindowTypeRegular:
		return "regular"
	case WindowTypeDialog:
		return "dialog"
	case WindowTypeUtility:
		return "utility"
	case WindowTypeTooltip:
		return "tooltip"
	default:
		return "unknown"
	}
}

// WindowFlags contains various window flags.
type WindowFlags struct {
	Resizable   bool // Whether the window can be resized
	Movable     bool // Whether the window can be moved
	Closable    bool // Whether the window can be closed
	Minimizable bool // Whether the window can be minimized
	Maximizable bool // Whether the window can be maximized
	HasTitleBar bool // Whether the window has a title bar
	HasBorder   bool // Whether the window has a border
	AlwaysOnTop bool // Whether the window stays on top of others
}

// DefaultWindowFlags returns the default window flags.
func DefaultWindowFlags() WindowFlags {
	return WindowFlags{
		Resizable:   true,
		Movable:     true,
		Closable:    true,
		Minimizable: true,
		Maximizable: true,
		HasTitleBar: true,
		HasBorder:   true,
		AlwaysOnTop: false,
	}
}

// Frame represents the position and dimensions of a window.
type Frame struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// NormalFrame returns the frame before maximization (for restore).
func (f *Frame) NormalFrame() Frame {
	return Frame{
		X:      f.X,
		Y:      f.Y,
		Width:  f.Width,
		Height: f.Height,
	}
}

// Contains checks if a point is within the frame.
func (f *Frame) Contains(x, y int) bool {
	return x >= f.X && x <= f.X+f.Width &&
		y >= f.Y && y <= f.Y+f.Height
}

// Window represents a window in the window manager.
type Window struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Frame       Frame       `json:"frame"`
	NormalFrame Frame       `json:"normal_frame,omitempty"`
	State       WindowState `json:"state"`
	Type        WindowType  `json:"type"`
	Flags       WindowFlags `json:"flags"`
	Desktop     int         `json:"desktop"`
	Visible     bool        `json:"visible"`
	ParentID    string      `json:"parent_id,omitempty"`
}

// NewWindow creates a new window with the given parameters.
func NewWindow(id, title string, x, y, width, height int) *Window {
	return &Window{
		ID:          id,
		Title:       title,
		Frame:       Frame{X: x, Y: y, Width: width, Height: height},
		State:       WindowStateNormal,
		Type:        WindowTypeRegular,
		Flags:       DefaultWindowFlags(),
		Desktop:     0,
		Visible:     true,
		NormalFrame: Frame{X: x, Y: y, Width: width, Height: height},
	}
}

// Desktop represents a virtual desktop.
type Desktop struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Windows  []string `json:"windows"`
	GridSize int      `json:"grid_size"` // Grid size for tiling (e.g., 8x8)
}

// NewDesktop creates a new desktop with the given ID and name.
func NewDesktop(id int, name string) *Desktop {
	return &Desktop{
		ID:       id,
		Name:     name,
		Windows:  make([]string, 0),
		GridSize: 8,
	}
}

// ErrWindowNotFound is returned when a window is not found.
var ErrWindowNotFound = errors.New("window not found")

// ErrDesktopNotFound is returned when a desktop is not found.
var ErrDesktopNotFound = errors.New("desktop not found")

// ErrInvalidWindowID is returned when a window ID is invalid.
var ErrInvalidWindowID = errors.New("invalid window ID")

// ErrInvalidDesktopIndex is returned when a desktop index is invalid.
var ErrInvalidDesktopIndex = errors.New("invalid desktop index")
