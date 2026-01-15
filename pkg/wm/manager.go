package wm

import "sync"

// Manager manages windows and desktops.
type Manager struct {
	mu             sync.RWMutex
	windows        map[string]*Window
	desktops       []*Desktop
	currentDesktop int
	focusedWindow  string
	nextWindowID   int
	screenWidth    int
	screenHeight   int
}

// Config holds configuration for the window manager.
type Config struct {
	ScreenWidth     int
	ScreenHeight    int
	InitialDesktops int
}

// NewManager creates a new window manager with the given configuration.
func NewManager(cfg Config) *Manager {
	desktopCount := cfg.InitialDesktops
	if desktopCount <= 0 {
		desktopCount = 4
	}

	desktops := make([]*Desktop, desktopCount)
	for i := 0; i < desktopCount; i++ {
		desktops[i] = NewDesktop(i, desktopName(i))
	}

	return &Manager{
		windows:        make(map[string]*Window),
		desktops:       desktops,
		currentDesktop: 0,
		screenWidth:    cfg.ScreenWidth,
		screenHeight:   cfg.ScreenHeight,
		nextWindowID:   1,
	}
}

// desktopName returns a default name for a desktop.
func desktopName(index int) string {
	switch index {
	case 0:
		return "Main"
	case 1:
		return "Work"
	case 2:
		return "娱乐" // Entertainment
	case 3:
		return "Other"
	default:
		return "Desktop " + string(rune('1'+index))
	}
}

// CreateWindow creates a new window and returns its ID.
func (m *Manager) CreateWindow(title string, x, y, width, height int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateWindowID(m.nextWindowID)
	m.nextWindowID++

	win := NewWindow(id, title, x, y, width, height)
	m.windows[id] = win

	m.desktops[m.currentDesktop].Windows = append(
		m.desktops[m.currentDesktop].Windows,
		id,
	)

	m.focusedWindow = id

	return id, nil
}

// generateWindowID generates a unique window ID.
func generateWindowID(n int) string {
	return "win_" + string(rune('a'+(n%26))) + string(rune('a'+((n/26)%26))) + string(rune('0'+((n/100)%10)))
}

// CloseWindow closes and removes a window.
func (m *Manager) CloseWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	// Remove from desktop
	desktop := m.desktops[win.Desktop]
	for i, wid := range desktop.Windows {
		if wid == id {
			desktop.Windows = append(desktop.Windows[:i], desktop.Windows[i+1:]...)
			break
		}
	}

	delete(m.windows, id)

	if m.focusedWindow == id {
		m.focusedWindow = ""
		if len(desktop.Windows) > 0 {
			m.focusedWindow = desktop.Windows[len(desktop.Windows)-1]
		}
	}

	return nil
}

// GetWindow returns a window by ID.
func (m *Manager) GetWindow(id string) (*Window, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	win, exists := m.windows[id]
	if !exists {
		return nil, ErrWindowNotFound
	}

	return win, nil
}

// ListWindows returns all window IDs.
func (m *Manager) ListWindows() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.windows))
	for id := range m.windows {
		ids = append(ids, id)
	}
	return ids
}

// ListWindowsOnDesktop returns all window IDs on the specified desktop.
func (m *Manager) ListWindowsOnDesktop(desktopIndex int) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if desktopIndex < 0 || desktopIndex >= len(m.desktops) {
		return nil, ErrInvalidDesktopIndex
	}

	desktop := m.desktops[desktopIndex]
	ids := make([]string, len(desktop.Windows))
	copy(ids, desktop.Windows)
	return ids, nil
}

// FocusWindow sets the focused window.
func (m *Manager) FocusWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.windows[id]; !exists {
		return ErrWindowNotFound
	}

	m.focusedWindow = id
	return nil
}

// GetFocusedWindow returns the focused window ID.
func (m *Manager) GetFocusedWindow() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.focusedWindow
}

// MoveWindow moves a window to a new position.
func (m *Manager) MoveWindow(id string, x, y int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	win.Frame.X = x
	win.Frame.Y = y
	return nil
}

// ResizeWindow resizes a window.
func (m *Manager) ResizeWindow(id string, width, height int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	win.Frame.Width = width
	win.Frame.Height = height
	return nil
}

// MoveResizeWindow moves and resizes a window.
func (m *Manager) MoveResizeWindow(id string, x, y, width, height int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	win.Frame = Frame{X: x, Y: y, Width: width, Height: height}
	return nil
}

// MinimizeWindow minimizes a window.
func (m *Manager) MinimizeWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	if !win.Flags.Minimizable {
		return nil
	}

	win.State = WindowStateMinimized
	win.Visible = false
	return nil
}

// MaximizeWindow maximizes a window.
func (m *Manager) MaximizeWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	if !win.Flags.Maximizable {
		return nil
	}

	if win.State != WindowStateMaximized {
		win.NormalFrame = win.Frame
		win.Frame = Frame{X: 0, Y: 0, Width: m.screenWidth, Height: m.screenHeight}
		win.State = WindowStateMaximized
	}

	return nil
}

// RestoreWindow restores a window from minimized/maximized state.
func (m *Manager) RestoreWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	switch win.State {
	case WindowStateMinimized:
		win.State = WindowStateNormal
		win.Visible = true
	case WindowStateMaximized:
		win.Frame = win.NormalFrame
		win.State = WindowStateNormal
	case WindowStateFullscreen:
		win.Frame = win.NormalFrame
		win.State = WindowStateNormal
	}

	return nil
}

// ToggleMaximized toggles the maximized state of a window.
func (m *Manager) ToggleMaximized(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	if win.State == WindowStateMaximized {
		return m.RestoreWindow(id)
	}
	return m.MaximizeWindow(id)
}

// SnapWindow snaps a window to a position (left, right, top, bottom, corners).
func (m *Manager) SnapWindow(id string, position SnapPosition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	halfWidth := m.screenWidth / 2
	halfHeight := m.screenHeight / 2

	switch position {
	case SnapLeft:
		win.Frame = Frame{X: 0, Y: 0, Width: halfWidth, Height: m.screenHeight}
	case SnapRight:
		win.Frame = Frame{X: halfWidth, Y: 0, Width: halfWidth, Height: m.screenHeight}
	case SnapTop:
		win.Frame = Frame{X: 0, Y: 0, Width: m.screenWidth, Height: halfHeight}
	case SnapBottom:
		win.Frame = Frame{X: 0, Y: halfHeight, Width: m.screenWidth, Height: halfHeight}
	case SnapTopLeft:
		win.Frame = Frame{X: 0, Y: 0, Width: halfWidth, Height: halfHeight}
	case SnapTopRight:
		win.Frame = Frame{X: halfWidth, Y: 0, Width: halfWidth, Height: halfHeight}
	case SnapBottomLeft:
		win.Frame = Frame{X: 0, Y: halfHeight, Width: halfWidth, Height: halfHeight}
	case SnapBottomRight:
		win.Frame = Frame{X: halfWidth, Y: halfHeight, Width: halfWidth, Height: halfHeight}
	}

	win.State = WindowStateNormal
	return nil
}

// SnapPosition represents where a window should be snapped.
type SnapPosition int

const (
	// SnapLeft snaps to the left half of the screen.
	SnapLeft SnapPosition = iota
	// SnapRight snaps to the right half of the screen.
	SnapRight
	// SnapTop snaps to the top half of the screen.
	SnapTop
	// SnapBottom snaps to the bottom half of the screen.
	SnapBottom
	// SnapTopLeft snaps to the top-left quadrant.
	SnapTopLeft
	// SnapTopRight snaps to the top-right quadrant.
	SnapTopRight
	// SnapBottomLeft snaps to the bottom-left quadrant.
	SnapBottomLeft
	// SnapBottomRight snaps to the bottom-right quadrant.
	SnapBottomRight
)

// CreateDesktop creates a new desktop and returns its index.
func (m *Manager) CreateDesktop(name string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	desktop := NewDesktop(len(m.desktops), name)
	m.desktops = append(m.desktops, desktop)
	return desktop.ID, nil
}

// SwitchDesktop switches to the specified desktop.
func (m *Manager) SwitchDesktop(index int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if index < 0 || index >= len(m.desktops) {
		return ErrInvalidDesktopIndex
	}

	m.currentDesktop = index
	return nil
}

// GetCurrentDesktop returns the current desktop index.
func (m *Manager) GetCurrentDesktop() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentDesktop
}

// GetDesktopCount returns the number of desktops.
func (m *Manager) GetDesktopCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.desktops)
}

// MoveWindowToDesktop moves a window to a different desktop.
func (m *Manager) MoveWindowToDesktop(windowID string, desktopIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[windowID]
	if !exists {
		return ErrWindowNotFound
	}

	if desktopIndex < 0 || desktopIndex >= len(m.desktops) {
		return ErrInvalidDesktopIndex
	}

	// Remove from current desktop
	oldDesktop := m.desktops[win.Desktop]
	for i, id := range oldDesktop.Windows {
		if id == windowID {
			oldDesktop.Windows = append(oldDesktop.Windows[:i], oldDesktop.Windows[i+1:]...)
			break
		}
	}

	// Add to new desktop
	win.Desktop = desktopIndex
	m.desktops[desktopIndex].Windows = append(m.desktops[desktopIndex].Windows, windowID)

	return nil
}

// SetWindowTitle sets the title of a window.
func (m *Manager) SetWindowTitle(id string, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	win.Title = title
	return nil
}

// GetWindowState returns the state of a window.
func (m *Manager) GetWindowState(id string) (WindowState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	win, exists := m.windows[id]
	if !exists {
		return 0, ErrWindowNotFound
	}

	return win.State, nil
}

// ShowWindow shows a window.
func (m *Manager) ShowWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	win.Visible = true
	if win.State == WindowStateMinimized {
		win.State = WindowStateNormal
	}

	return nil
}

// HideWindow hides a window.
func (m *Manager) HideWindow(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	win, exists := m.windows[id]
	if !exists {
		return ErrWindowNotFound
	}

	win.Visible = false
	return nil
}

// SetScreenSize sets the screen dimensions.
func (m *Manager) SetScreenSize(width, height int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.screenWidth = width
	m.screenHeight = height
}

// GetScreenSize returns the screen dimensions.
func (m *Manager) GetScreenSize() (int, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.screenWidth, m.screenHeight
}

// GetScreenWidth returns the screen width.
func (m *Manager) GetScreenWidth() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.screenWidth
}

// GetScreenHeight returns the screen height.
func (m *Manager) GetScreenHeight() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.screenHeight
}
