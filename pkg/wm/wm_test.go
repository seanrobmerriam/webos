package wm

import (
	"testing"
)

func TestNewWindow(t *testing.T) {
	win := NewWindow("test-id", "Test Window", 100, 200, 800, 600)

	if win.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", win.ID)
	}
	if win.Title != "Test Window" {
		t.Errorf("expected title 'Test Window', got '%s'", win.Title)
	}
	if win.Frame.X != 100 {
		t.Errorf("expected X 100, got %d", win.Frame.X)
	}
	if win.Frame.Y != 200 {
		t.Errorf("expected Y 200, got %d", win.Frame.Y)
	}
	if win.Frame.Width != 800 {
		t.Errorf("expected Width 800, got %d", win.Frame.Width)
	}
	if win.Frame.Height != 600 {
		t.Errorf("expected Height 600, got %d", win.Frame.Height)
	}
	if win.State != WindowStateNormal {
		t.Errorf("expected state WindowStateNormal, got %v", win.State)
	}
	if win.Type != WindowTypeRegular {
		t.Errorf("expected type WindowTypeRegular, got %v", win.Type)
	}
	if !win.Flags.Resizable {
		t.Error("expected window to be resizable by default")
	}
	if win.Desktop != 0 {
		t.Errorf("expected desktop 0, got %d", win.Desktop)
	}
	if !win.Visible {
		t.Error("expected window to be visible by default")
	}
}

func TestFrameContains(t *testing.T) {
	frame := Frame{X: 100, Y: 100, Width: 200, Height: 150}

	tests := []struct {
		x, y     int
		expected bool
	}{
		{150, 175, true},  // Center
		{100, 100, true},  // Top-left corner
		{300, 250, true},  // Bottom-right corner
		{99, 100, false},  // Left edge
		{301, 100, false}, // Right edge
		{100, 99, false},  // Top edge
		{100, 251, false}, // Bottom edge
	}

	for _, tt := range tests {
		result := frame.Contains(tt.x, tt.y)
		if result != tt.expected {
			t.Errorf("Contains(%d, %d) = %v, expected %v", tt.x, tt.y, result, tt.expected)
		}
	}
}

func TestNewManager(t *testing.T) {
	cfg := Config{
		ScreenWidth:     1920,
		ScreenHeight:    1080,
		InitialDesktops: 2,
	}

	mgr := NewManager(cfg)

	if mgr.GetScreenWidth() != 1920 {
		t.Errorf("expected screen width 1920, got %d", mgr.GetScreenWidth())
	}
	if mgr.GetScreenHeight() != 1080 {
		t.Errorf("expected screen height 1080, got %d", mgr.GetScreenHeight())
	}
	if mgr.GetDesktopCount() != 2 {
		t.Errorf("expected 2 desktops, got %d", mgr.GetDesktopCount())
	}
	if mgr.GetCurrentDesktop() != 0 {
		t.Errorf("expected current desktop 0, got %d", mgr.GetCurrentDesktop())
	}
}

func TestCreateWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, err := mgr.CreateWindow("Test", 100, 100, 800, 600)
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}

	if id == "" {
		t.Error("expected non-empty window ID")
	}

	win, err := mgr.GetWindow(id)
	if err != nil {
		t.Fatalf("GetWindow failed: %v", err)
	}

	if win.Title != "Test" {
		t.Errorf("expected title 'Test', got '%s'", win.Title)
	}
	if win.Frame.X != 100 || win.Frame.Y != 100 {
		t.Errorf("unexpected frame: %+v", win.Frame)
	}
}

func TestCloseWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, err := mgr.CreateWindow("Test", 100, 100, 800, 600)
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}

	err = mgr.CloseWindow(id)
	if err != nil {
		t.Fatalf("CloseWindow failed: %v", err)
	}

	_, err = mgr.GetWindow(id)
	if err != ErrWindowNotFound {
		t.Errorf("expected ErrWindowNotFound, got %v", err)
	}
}

func TestCloseWindowNotFound(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	err := mgr.CloseWindow("nonexistent")
	if err != ErrWindowNotFound {
		t.Errorf("expected ErrWindowNotFound, got %v", err)
	}
}

func TestFocusWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id1, _ := mgr.CreateWindow("Window 1", 100, 100, 800, 600)
	id2, _ := mgr.CreateWindow("Window 2", 200, 200, 600, 400)

	err := mgr.FocusWindow(id1)
	if err != nil {
		t.Fatalf("FocusWindow failed: %v", err)
	}

	if mgr.GetFocusedWindow() != id1 {
		t.Errorf("expected focused window %s, got %s", id1, mgr.GetFocusedWindow())
	}

	err = mgr.FocusWindow(id2)
	if err != nil {
		t.Fatalf("FocusWindow failed: %v", err)
	}

	if mgr.GetFocusedWindow() != id2 {
		t.Errorf("expected focused window %s, got %s", id2, mgr.GetFocusedWindow())
	}
}

func TestMoveWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	err := mgr.MoveWindow(id, 200, 300)
	if err != nil {
		t.Fatalf("MoveWindow failed: %v", err)
	}

	win, _ := mgr.GetWindow(id)
	if win.Frame.X != 200 || win.Frame.Y != 300 {
		t.Errorf("expected position (200, 300), got (%d, %d)", win.Frame.X, win.Frame.Y)
	}
}

func TestResizeWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	err := mgr.ResizeWindow(id, 1024, 768)
	if err != nil {
		t.Fatalf("ResizeWindow failed: %v", err)
	}

	win, _ := mgr.GetWindow(id)
	if win.Frame.Width != 1024 || win.Frame.Height != 768 {
		t.Errorf("expected size (1024, 768), got (%d, %d)", win.Frame.Width, win.Frame.Height)
	}
}

func TestMinimizeWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	err := mgr.MinimizeWindow(id)
	if err != nil {
		t.Fatalf("MinimizeWindow failed: %v", err)
	}

	state, _ := mgr.GetWindowState(id)
	if state != WindowStateMinimized {
		t.Errorf("expected state WindowStateMinimized, got %v", state)
	}
}

func TestMaximizeWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	err := mgr.MaximizeWindow(id)
	if err != nil {
		t.Fatalf("MaximizeWindow failed: %v", err)
	}

	state, _ := mgr.GetWindowState(id)
	if state != WindowStateMaximized {
		t.Errorf("expected state WindowStateMaximized, got %v", state)
	}

	win, _ := mgr.GetWindow(id)
	if win.Frame.X != 0 || win.Frame.Y != 0 {
		t.Errorf("expected position (0, 0), got (%d, %d)", win.Frame.X, win.Frame.Y)
	}
	if win.Frame.Width != 1920 || win.Frame.Height != 1080 {
		t.Errorf("expected size (1920, 1080), got (%d, %d)", win.Frame.Width, win.Frame.Height)
	}
}

func TestRestoreWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)
	mgr.MaximizeWindow(id)

	err := mgr.RestoreWindow(id)
	if err != nil {
		t.Fatalf("RestoreWindow failed: %v", err)
	}

	state, _ := mgr.GetWindowState(id)
	if state != WindowStateNormal {
		t.Errorf("expected state WindowStateNormal, got %v", state)
	}

	win, _ := mgr.GetWindow(id)
	if win.Frame.X != 100 || win.Frame.Y != 100 {
		t.Errorf("expected position (100, 100), got (%d, %d)", win.Frame.X, win.Frame.Y)
	}
	if win.Frame.Width != 800 || win.Frame.Height != 600 {
		t.Errorf("expected size (800, 600), got (%d, %d)", win.Frame.Width, win.Frame.Height)
	}
}

func TestSnapWindow(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	tests := []struct {
		position  SnapPosition
		expectedX int
		expectedY int
		expectedW int
		expectedH int
	}{
		{SnapLeft, 0, 0, 960, 1080},
		{SnapRight, 960, 0, 960, 1080},
		{SnapTop, 0, 0, 1920, 540},
		{SnapBottom, 0, 540, 1920, 540},
		{SnapTopLeft, 0, 0, 960, 540},
		{SnapTopRight, 960, 0, 960, 540},
		{SnapBottomLeft, 0, 540, 960, 540},
		{SnapBottomRight, 960, 540, 960, 540},
	}

	for _, tt := range tests {
		err := mgr.SnapWindow(id, tt.position)
		if err != nil {
			t.Fatalf("SnapWindow(%v) failed: %v", tt.position, err)
		}

		win, _ := mgr.GetWindow(id)
		if win.Frame.X != tt.expectedX || win.Frame.Y != tt.expectedY {
			t.Errorf("Snap(%v): expected position (%d, %d), got (%d, %d)",
				tt.position, tt.expectedX, tt.expectedY, win.Frame.X, win.Frame.Y)
		}
		if win.Frame.Width != tt.expectedW || win.Frame.Height != tt.expectedH {
			t.Errorf("Snap(%v): expected size (%d, %d), got (%d, %d)",
				tt.position, tt.expectedW, tt.expectedH, win.Frame.Width, win.Frame.Height)
		}
	}
}

func TestSwitchDesktop(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080, InitialDesktops: 3})

	err := mgr.SwitchDesktop(2)
	if err != nil {
		t.Fatalf("SwitchDesktop failed: %v", err)
	}

	if mgr.GetCurrentDesktop() != 2 {
		t.Errorf("expected current desktop 2, got %d", mgr.GetCurrentDesktop())
	}
}

func TestSwitchDesktopInvalid(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080, InitialDesktops: 2})

	err := mgr.SwitchDesktop(5)
	if err != ErrInvalidDesktopIndex {
		t.Errorf("expected ErrInvalidDesktopIndex, got %v", err)
	}
}

func TestMoveWindowToDesktop(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080, InitialDesktops: 3})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	err := mgr.MoveWindowToDesktop(id, 2)
	if err != nil {
		t.Fatalf("MoveWindowToDesktop failed: %v", err)
	}

	win, _ := mgr.GetWindow(id)
	if win.Desktop != 2 {
		t.Errorf("expected window desktop 2, got %d", win.Desktop)
	}

	windows, _ := mgr.ListWindowsOnDesktop(2)
	if len(windows) != 1 || windows[0] != id {
		t.Errorf("expected window %s on desktop 2, got %v", id, windows)
	}
}

func TestSetWindowTitle(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Old Title", 100, 100, 800, 600)

	err := mgr.SetWindowTitle(id, "New Title")
	if err != nil {
		t.Fatalf("SetWindowTitle failed: %v", err)
	}

	win, _ := mgr.GetWindow(id)
	if win.Title != "New Title" {
		t.Errorf("expected title 'New Title', got '%s'", win.Title)
	}
}

func TestListWindows(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id1, _ := mgr.CreateWindow("Window 1", 100, 100, 800, 600)
	id2, _ := mgr.CreateWindow("Window 2", 200, 200, 600, 400)

	ids := mgr.ListWindows()
	if len(ids) != 2 {
		t.Errorf("expected 2 windows, got %d", len(ids))
	}

	found1, found2 := false, false
	for _, id := range ids {
		if id == id1 {
			found1 = true
		}
		if id == id2 {
			found2 = true
		}
	}
	if !found1 || !found2 {
		t.Errorf("expected both window IDs, found1=%v, found2=%v", found1, found2)
	}
}

func TestCreateDesktop(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080, InitialDesktops: 1})

	id, err := mgr.CreateDesktop("My Desktop")
	if err != nil {
		t.Fatalf("CreateDesktop failed: %v", err)
	}

	if mgr.GetDesktopCount() != 2 {
		t.Errorf("expected 2 desktops, got %d", mgr.GetDesktopCount())
	}
	if id != 1 {
		t.Errorf("expected desktop ID 1, got %d", id)
	}
}

func TestWindowStateString(t *testing.T) {
	tests := []struct {
		state    WindowState
		expected string
	}{
		{WindowStateNormal, "normal"},
		{WindowStateMinimized, "minimized"},
		{WindowStateMaximized, "maximized"},
		{WindowStateFullscreen, "fullscreen"},
		{WindowState(100), "unknown"},
	}

	for _, tt := range tests {
		result := tt.state.String()
		if result != tt.expected {
			t.Errorf("WindowState(%d).String() = %s, expected %s", tt.state, result, tt.expected)
		}
	}
}

func TestWindowTypeString(t *testing.T) {
	tests := []struct {
		wtype    WindowType
		expected string
	}{
		{WindowTypeRegular, "regular"},
		{WindowTypeDialog, "dialog"},
		{WindowTypeUtility, "utility"},
		{WindowTypeTooltip, "tooltip"},
		{WindowType(100), "unknown"},
	}

	for _, tt := range tests {
		result := tt.wtype.String()
		if result != tt.expected {
			t.Errorf("WindowType(%d).String() = %s, expected %s", tt.wtype, result, tt.expected)
		}
	}
}

func TestDefaultWindowFlags(t *testing.T) {
	flags := DefaultWindowFlags()

	if !flags.Resizable {
		t.Error("expected Resizable to be true")
	}
	if !flags.Movable {
		t.Error("expected Movable to be true")
	}
	if !flags.Closable {
		t.Error("expected Closable to be true")
	}
	if !flags.Minimizable {
		t.Error("expected Minimizable to be true")
	}
	if !flags.Maximizable {
		t.Error("expected Maximizable to be true")
	}
	if !flags.HasTitleBar {
		t.Error("expected HasTitleBar to be true")
	}
	if !flags.HasBorder {
		t.Error("expected HasBorder to be true")
	}
	if flags.AlwaysOnTop {
		t.Error("expected AlwaysOnTop to be false")
	}
}

func TestToggleMaximized(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	id, _ := mgr.CreateWindow("Test", 100, 100, 800, 600)

	// First toggle should maximize
	err := mgr.ToggleMaximized(id)
	if err != nil {
		t.Fatalf("ToggleMaximized failed: %v", err)
	}

	state, _ := mgr.GetWindowState(id)
	if state != WindowStateMaximized {
		t.Errorf("expected state WindowStateMaximized, got %v", state)
	}

	// Second toggle should restore
	err = mgr.ToggleMaximized(id)
	if err != nil {
		t.Fatalf("ToggleMaximized failed: %v", err)
	}

	state, _ = mgr.GetWindowState(id)
	if state != WindowStateNormal {
		t.Errorf("expected state WindowStateNormal, got %v", state)
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := NewManager(Config{ScreenWidth: 1920, ScreenHeight: 1080})

	// Run concurrent operations
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			_, err := mgr.CreateWindow("Test", 100, 100, 800, 600)
			if err != nil {
				t.Errorf("CreateWindow failed: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	ids := mgr.ListWindows()
	if len(ids) != 10 {
		t.Errorf("expected 10 windows, got %d", len(ids))
	}
}
