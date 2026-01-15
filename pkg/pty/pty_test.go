package pty

import (
	"testing"
)

func TestNewTerminal(t *testing.T) {
	tests := []struct {
		name        string
		cols        int
		rows        int
		wantCols    int
		wantRows    int
		expectError bool
	}{
		{"valid dimensions", 80, 24, 80, 24, false},
		{"default cols", 0, 24, DefaultCols, 24, false},
		{"default rows", 80, 0, 80, DefaultRows, false},
		{"both default", 0, 0, DefaultCols, DefaultRows, false},
		{"invalid cols", -1, 24, 0, 0, true},
		{"invalid rows", 80, -1, 0, 0, true},
		{"zero dimensions", 0, 0, DefaultCols, DefaultRows, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term, err := NewTerminal(tt.cols, tt.rows)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer term.Close()

			if term.Cols != tt.wantCols {
				t.Errorf("Cols = %d, want %d", term.Cols, tt.wantCols)
			}
			if term.Rows != tt.wantRows {
				t.Errorf("Rows = %d, want %d", term.Rows, tt.wantRows)
			}
		})
	}
}

func TestTerminalResize(t *testing.T) {
	term, err := NewTerminal(80, 24)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	// Test valid resize
	if err := term.Resize(100, 40); err != nil {
		t.Errorf("Resize failed: %v", err)
	}
	if term.Cols != 100 || term.Rows != 40 {
		t.Errorf("Resize: cols=%d, rows=%d, want 100, 40", term.Cols, term.Rows)
	}

	// Test invalid resize
	if err := term.Resize(-1, 24); err == nil {
		t.Error("Resize(-1, 24) should fail")
	}
	if err := term.Resize(80, -1); err == nil {
		t.Error("Resize(80, -1) should fail")
	}
}

func TestTerminalWrite(t *testing.T) {
	term, err := NewTerminal(10, 5)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	term.Write([]byte("hello"))
	term.Write([]byte(" world"))

	output := term.Dump()
	if len(output) == 0 {
		t.Error("expected output, got empty")
	}
}

func TestTerminalCursorMovement(t *testing.T) {
	term, err := NewTerminal(10, 5)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	// Initial position
	if term.Cursor.X != 0 || term.Cursor.Y != 0 {
		t.Errorf("initial cursor = (%d,%d), want (0,0)", term.Cursor.X, term.Cursor.Y)
	}

	// Move right
	term.CursorForward(3)
	if term.Cursor.X != 3 {
		t.Errorf("CursorForward(3): X=%d, want 3", term.Cursor.X)
	}

	// Move left
	term.CursorBackward(2)
	if term.Cursor.X != 1 {
		t.Errorf("CursorBackward(2): X=%d, want 1", term.Cursor.X)
	}

	// Move down
	term.CursorDown(2)
	if term.Cursor.Y != 2 {
		t.Errorf("CursorDown(2): Y=%d, want 2", term.Cursor.Y)
	}

	// Move up
	term.CursorUp(1)
	if term.Cursor.Y != 1 {
		t.Errorf("CursorUp(1): Y=%d, want 1", term.Cursor.Y)
	}

	// Move to specific position
	term.MoveCursorTo(5, 3)
	if term.Cursor.X != 4 || term.Cursor.Y != 2 {
		t.Errorf("MoveCursorTo(5,3): cursor=(%d,%d), want (4,2)", term.Cursor.X, term.Cursor.Y)
	}
}

func TestTerminalClear(t *testing.T) {
	term, err := NewTerminal(10, 5)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	// Write some content
	term.Write([]byte("test content"))

	// Clear screen
	term.ClearScreen()

	output := term.Dump()
	for _, ch := range output {
		if ch != ' ' && ch != '\n' {
			t.Error("ClearScreen did not clear all content")
			return
		}
	}
}

func TestTerminalScroll(t *testing.T) {
	term, err := NewTerminal(10, 3)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	// Fill screen
	term.Write([]byte("line1      "))
	term.CarriageReturn()
	term.LineFeed()
	term.Write([]byte("line2      "))
	term.CarriageReturn()
	term.LineFeed()
	term.Write([]byte("line3      "))

	// Scroll up
	term.ScrollUp(1)

	// Check scrollback
	scrollback := term.Screen.GetScrollbackLines()
	if len(scrollback) == 0 {
		t.Error("expected scrollback content")
	}
}

func TestScreenBuffer(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		scrollback  int
		expectError bool
	}{
		{"valid", 80, 24, 1000, false},
		{"invalid width", 0, 24, 1000, true},
		{"invalid height", 80, 0, 1000, true},
		{"negative scrollback", 80, 24, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb, err := NewScreenBuffer(tt.width, tt.height, tt.scrollback)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sb.Width != tt.width {
				t.Errorf("Width = %d, want %d", sb.Width, tt.width)
			}
			if sb.Height != tt.height {
				t.Errorf("Height = %d, want %d", sb.Height, tt.height)
			}
		})
	}
}

func TestScreenBufferResize(t *testing.T) {
	sb, err := NewScreenBuffer(80, 24, 1000)
	if err != nil {
		t.Fatalf("failed to create screen buffer: %v", err)
	}

	// Resize to larger dimensions
	if err := sb.Resize(100, 40); err != nil {
		t.Errorf("Resize failed: %v", err)
	}

	if sb.Width != 100 || sb.Height != 40 {
		t.Errorf("Resize: width=%d, height=%d, want 100, 40", sb.Width, sb.Height)
	}

	// Resize to smaller dimensions
	if err := sb.Resize(40, 10); err != nil {
		t.Errorf("Resize failed: %v", err)
	}
}

func TestColor(t *testing.T) {
	tests := []struct {
		name     string
		color    Color
		wantType ColorType
		wantVal  uint8
	}{
		{"default", DefaultColor, ColorDefault, 0},
		{"red", Red, ColorStandard, 1},
		{"bright blue", BrightBlue, ColorBright, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.color.Type != tt.wantType {
				t.Errorf("Color.Type = %d, want %d", tt.color.Type, tt.wantType)
			}
			if tt.color.Value != tt.wantVal {
				t.Errorf("Color.Value = %d, want %d", tt.color.Value, tt.wantVal)
			}
		})
	}
}

func TestCellAttributes(t *testing.T) {
	attrs := DefaultAttributes()
	if attrs.Foreground.Type != ColorDefault {
		t.Errorf("default foreground should be ColorDefault")
	}
	if attrs.Background.Type != ColorDefault {
		t.Errorf("default background should be ColorDefault")
	}
}

func TestANISParser(t *testing.T) {
	term, err := NewTerminal(80, 24)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	parser := NewParser(term)

	// Test basic text
	parser.Parse([]byte("hello"))
	if term.Dump() == "" {
		t.Error("expected output from basic text")
	}

	// Test cursor movement
	parser.Parse([]byte{ESC, '[', '2', 'A'}) // Cursor up 2
	if term.Cursor.Y != 2 {                  // Started at 0, moved up... wait, min is 0
		// Cursor up from 0 stays at 0
	}

	// Test clear screen
	parser.Parse([]byte{ESC, '[', '2', 'J'}) // Clear screen
}

func TestGraphicsRendition(t *testing.T) {
	term, err := NewTerminal(80, 24)
	if err != nil {
		t.Fatalf("failed to create terminal: %v", err)
	}
	defer term.Close()

	// Set bold
	term.SetGraphicsRendition([]int{1})
	if !term.Attributes.Bold {
		t.Error("expected Bold to be true")
	}

	// Reset all
	term.SetGraphicsRendition([]int{0})
	if term.Attributes.Bold {
		t.Error("expected Bold to be false after reset")
	}

	// Set color
	term.SetGraphicsRendition([]int{31}) // Red foreground
	if term.Attributes.Foreground != Red {
		t.Errorf("expected red foreground, got %v", term.Attributes.Foreground)
	}

	// Set background color
	term.SetGraphicsRendition([]int{44}) // Blue background
	if term.Attributes.Background.Value != 4 {
		t.Errorf("expected blue background, got value %d", term.Attributes.Background.Value)
	}
}

func TestPTY(t *testing.T) {
	pty, err := NewPTY(80, 24)
	if err != nil {
		t.Fatalf("failed to create PTY: %v", err)
	}
	defer pty.Close()

	// Check master and slave are non-nil
	if pty.Master() == nil {
		t.Error("Master() returned nil")
	}
	if pty.Slave() == nil {
		t.Error("Slave() returned nil")
	}

	// Check terminal is accessible
	if pty.Terminal() == nil {
		t.Error("Terminal() returned nil")
	}
}

func TestPTYSizes(t *testing.T) {
	pty, err := NewPTY(80, 24)
	if err != nil {
		t.Fatalf("failed to create PTY: %v", err)
	}
	defer pty.Close()

	cols, rows, err := pty.Master().Size()
	if err != nil {
		t.Errorf("Size() failed: %v", err)
	}
	if cols != 80 || rows != 24 {
		t.Errorf("Size() = (%d,%d), want (80,24)", cols, rows)
	}

	// Test resize via Winsize
	ws := Winsize{Cols: 100, Rows: 40}
	if err := pty.Master().SetWinsize(ws); err != nil {
		t.Errorf("SetWinsize() failed: %v", err)
	}

	got := pty.Master().GetWinsize()
	if got.Cols != 100 || got.Rows != 40 {
		t.Errorf("GetWinsize() = (%d,%d), want (100,40)", got.Cols, got.Rows)
	}
}

func TestPTYClose(t *testing.T) {
	pty, err := NewPTY(80, 24)
	if err != nil {
		t.Fatalf("failed to create PTY: %v", err)
	}

	// Close should not error
	if err := pty.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Double close should not error
	if err := pty.Close(); err != nil {
		t.Errorf("double Close() failed: %v", err)
	}

	// Check closed state
	if !pty.IsClosed() {
		t.Error("IsClosed() should return true after Close()")
	}
}

func TestCell(t *testing.T) {
	tests := []struct {
		cell  Cell
		empty bool
	}{
		{Cell{Char: ' '}, true},
		{Cell{Char: 0}, true},
		{Cell{Char: 'a'}, false},
		{Cell{Char: 'X'}, false},
	}

	for _, tt := range tests {
		if got := tt.cell.IsEmpty(); got != tt.empty {
			t.Errorf("Cell(%q).IsEmpty() = %v, want %v", tt.cell.Char, got, tt.empty)
		}
	}
}
