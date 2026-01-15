package pty

import (
	"container/ring"
	"sync"
)

// TerminalState represents the current state of the terminal.
type TerminalState int

const (
	// StateNormal is the normal input state.
	StateNormal TerminalState = iota
	// StateEscape is the escape state (after ESC).
	StateEscape
	// StateCSI is the Control Sequence Introducer state.
	StateCSI
	// StateOSC is the Operating System Command state.
	StateOSC
	// StateDCS is the Device Control String state.
	StateDCS
)

// MouseMode defines the mouse tracking mode.
type MouseMode int

const (
	// MouseNone disables mouse tracking.
	MouseNone MouseMode = iota
	// MouseX10 implements X10 mouse reporting.
	MouseX10
	// MouseNormal implements normal mouse tracking.
	MouseNormal
	// MouseHighlight implements mouse highlighting.
	MouseHighlight
	// MouseButtonMotion implements button-event mouse tracking.
	MouseButtonMotion
	// MouseAllMotion implements all-motion mouse tracking.
	MouseAllMotion
)

// Cursor represents the current cursor position and state.
type Cursor struct {
	X, Y     int
	Visible  bool
	Blinking bool
	Style    CursorStyle
}

// CursorStyle defines the cursor appearance.
type CursorStyle int

const (
	// CursorBlock is a block cursor.
	CursorBlock CursorStyle = iota
	// CursorBlockBlinking is a blinking block cursor.
	CursorBlockBlinking
	// CursorUnderline is an underline cursor.
	CursorUnderline
	// CursorUnderlineBlinking is a blinking underline cursor.
	CursorUnderlineBlinking
	// CursorBar is a vertical bar cursor.
	CursorBar
	// CursorBarBlinking is a blinking vertical bar cursor.
	CursorBarBlinking
)

// Terminal represents a virtual terminal with screen buffer and state.
type Terminal struct {
	mu         sync.RWMutex
	Rows       int
	Cols       int
	Screen     *ScreenBuffer
	Scrollback *ring.Ring
	Cursor     Cursor
	Attributes CellAttributes
	State      TerminalState
	Title      string

	// Modes
	AppKeypad      bool
	AutoWrap       bool
	InsertMode     bool
	OriginMode     bool
	LineWrap       bool
	BracketedPaste bool

	// Mouse tracking
	MouseMode    MouseMode
	MouseX       int
	MouseY       int
	MouseButtons int

	// Tab stops
	tabs map[int]bool

	// Saved state
	savedCursor Cursor
	savedAttrs  CellAttributes

	// Scrolling region
	scrollTop    int
	scrollBottom int

	// Output buffer for reading
	outputBuffer []byte
	outputCond   *sync.Cond
}

// NewTerminal creates a new terminal with the specified dimensions.
func NewTerminal(cols, rows int) (*Terminal, error) {
	// Reject negative dimensions first
	if cols < 0 || rows < 0 {
		return nil, ErrInvalidDimensions
	}

	// Use defaults for zero values
	if cols == 0 {
		cols = DefaultCols
	}
	if rows == 0 {
		rows = DefaultRows
	}

	screen, err := NewScreenBuffer(cols, rows, DefaultScrollback)
	if err != nil {
		return nil, err
	}

	term := &Terminal{
		Cols:         cols,
		Rows:         rows,
		Screen:       screen,
		Attributes:   DefaultAttributes(),
		Cursor:       Cursor{X: 0, Y: 0, Visible: true},
		AppKeypad:    false,
		AutoWrap:     true,
		InsertMode:   false,
		OriginMode:   false,
		LineWrap:     true,
		MouseMode:    MouseNone,
		tabs:         make(map[int]bool),
		scrollTop:    0,
		scrollBottom: rows - 1,
		outputBuffer: make([]byte, 0, 4096),
		outputCond:   sync.NewCond(&sync.Mutex{}),
	}

	// Set default tab stops
	for i := 8; i < cols; i += 8 {
		term.tabs[i] = true
	}

	return term, nil
}

// Resize changes the terminal dimensions.
func (t *Terminal) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return ErrInvalidDimensions
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.Screen.Resize(cols, rows); err != nil {
		return err
	}

	t.Cols = cols
	t.Rows = rows
	t.scrollBottom = rows - 1

	return nil
}

// Write writes data to the terminal screen.
func (t *Terminal) Write(data []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, b := range data {
		t.writeByte(b)
	}
}

// writeByte processes a single byte.
func (t *Terminal) writeByte(b byte) {
	switch {
	case b == '\n':
		t.lineFeed()
	case b == '\r':
		t.carriageReturn()
	case b == '\t':
		t.tabForward(1)
	case b == '\b':
		t.backspace()
	case b == ESC:
		t.State = StateEscape
	case b >= 0x20 && b <= 0x7e || b >= 0x80:
		t.writeChar(rune(b))
	default:
		// Control characters
	}
}

// writeChar writes a character to the screen at the cursor position.
func (t *Terminal) writeChar(ch rune) {
	if ch == '\n' {
		t.lineFeed()
		return
	}

	if t.InsertMode {
		t.Screen.InsertChars(1, t.Cursor.X, t.Cursor.Y, t.Attributes)
	}

	t.Screen.SetCell(t.Cursor.X, t.Cursor.Y, Cell{Char: ch, Attributes: t.Attributes})
	t.Cursor.X++

	if t.Cursor.X >= t.Cols {
		if t.AutoWrap {
			t.Cursor.X = 0
			t.lineFeed()
		} else {
			t.Cursor.X = t.Cols - 1
		}
	}
}

// WriteChar writes a character with the current attributes.
func (t *Terminal) WriteChar(ch rune) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.writeChar(ch)
}

// Read reads data from the terminal output buffer.
func (t *Terminal) Read(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.outputBuffer) == 0 {
		return 0, nil
	}

	n := copy(data, t.outputBuffer)
	t.outputBuffer = t.outputBuffer[n:]
	return n, nil
}

// ReadAll reads all available data from the terminal.
func (t *Terminal) ReadAll() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()

	data := t.outputBuffer
	t.outputBuffer = make([]byte, 0, 4096)
	return data
}

// WriteToScreen writes data directly to the screen buffer.
func (t *Terminal) WriteToScreen(data []byte) {
	for _, b := range data {
		if b >= 0x20 || b == '\n' || b == '\r' || b == '\t' {
			t.writeByte(b)
		}
	}
}

// Cursor movement methods.

func (t *Terminal) CursorUp(n int) {
	if n <= 0 {
		return
	}
	t.Cursor.Y -= n
	if t.Cursor.Y < 0 {
		t.Cursor.Y = 0
	}
}

func (t *Terminal) CursorDown(n int) {
	if n <= 0 {
		return
	}
	t.Cursor.Y += n
	if t.Cursor.Y >= t.Rows {
		t.Cursor.Y = t.Rows - 1
	}
}

func (t *Terminal) CursorForward(n int) {
	if n <= 0 {
		return
	}
	t.Cursor.X += n
	if t.Cursor.X >= t.Cols {
		t.Cursor.X = t.Cols - 1
	}
}

func (t *Terminal) CursorBackward(n int) {
	if n <= 0 {
		return
	}
	t.Cursor.X -= n
	if t.Cursor.X < 0 {
		t.Cursor.X = 0
	}
}

func (t *Terminal) CursorToColumn(col int) {
	if col <= 0 {
		col = 1
	}
	if col >= t.Cols {
		col = t.Cols
	}
	t.Cursor.X = col - 1
}

func (t *Terminal) CursorToRow(row int) {
	if row <= 0 {
		row = 1
	}
	if row > t.Rows {
		row = t.Rows
	}
	t.Cursor.Y = row - 1
}

func (t *Terminal) MoveCursorTo(col, row int) {
	if col <= 0 {
		col = 1
	}
	if row <= 0 {
		row = 1
	}
	if col > t.Cols {
		col = t.Cols
	}
	if row > t.Rows {
		row = t.Rows
	}
	t.Cursor.X = col - 1
	t.Cursor.Y = row - 1
}

func (t *Terminal) carriageReturn() {
	t.Cursor.X = 0
}

func (t *Terminal) lineFeed() {
	t.Cursor.Y++
	if t.Cursor.Y > t.scrollBottom {
		t.Screen.ScrollUp(1)
		t.Cursor.Y = t.scrollBottom
	}
}

func (t *Terminal) backspace() {
	if t.Cursor.X > 0 {
		t.Cursor.X--
	}
}

func (t *Terminal) Backspace() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.backspace()
}

func (t *Terminal) CarriageReturn() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.carriageReturn()
}

func (t *Terminal) LineFeed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lineFeed()
}

func (t *Terminal) tabForward(n int) {
	if n <= 0 {
		return
	}
	for i := 0; i < n; i++ {
		for t.Cursor.X < t.Cols && !t.tabs[t.Cursor.X] {
			t.Cursor.X++
		}
		t.Cursor.X++
		if t.Cursor.X >= t.Cols {
			t.Cursor.X = t.Cols - 1
			break
		}
	}
}

func (t *Terminal) TabForward(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tabForward(n)
}

func (t *Terminal) TabBackward(n int) {
	if n <= 0 {
		return
	}
	t.Cursor.X -= n * 8
	if t.Cursor.X < 0 {
		t.Cursor.X = 0
	}
}

func (t *Terminal) ClearTab() {
	delete(t.tabs, t.Cursor.X)
}

func (t *Terminal) ClearAllTabs() {
	t.tabs = make(map[int]bool)
}

// Screen clearing methods.

func (t *Terminal) ClearScreen() {
	t.Screen.Clear()
	t.Cursor.X = 0
	t.Cursor.Y = 0
}

func (t *Terminal) ClearToEndOfScreen() {
	t.Screen.ClearToEndOfScreen(t.Cursor.X, t.Cursor.Y)
}

func (t *Terminal) ClearToBeginningOfScreen() {
	t.Screen.ClearToBeginningOfScreen(t.Cursor.X, t.Cursor.Y)
}

func (t *Terminal) ClearLine() {
	t.Screen.ClearLine(t.Cursor.Y)
}

func (t *Terminal) ClearToEndOfLine() {
	t.Screen.ClearToEndOfLine(t.Cursor.X, t.Cursor.Y)
}

func (t *Terminal) ClearToBeginningOfLine() {
	t.Screen.ClearToBeginningOfLine(t.Cursor.X, t.Cursor.Y)
}

// Scrolling methods.

func (t *Terminal) ScrollUp(n int) {
	if n <= 0 {
		return
	}
	t.Screen.ScrollUp(n)
}

func (t *Terminal) ScrollDown(n int) {
	if n <= 0 {
		return
	}
	t.Screen.ScrollDown(n)
}

func (t *Terminal) SetScrollingRegion(top, bottom int) {
	if top < 0 {
		top = 0
	}
	if bottom >= t.Rows {
		bottom = t.Rows - 1
	}
	if top > bottom {
		top, bottom = bottom, top
	}
	t.scrollTop = top
	t.scrollBottom = bottom
}

func (t *Terminal) ReverseIndex() {
	if t.Cursor.Y <= t.scrollTop {
		t.Screen.ScrollDown(1)
	} else {
		t.Cursor.Y--
	}
}

// Line editing methods.

func (t *Terminal) InsertLines(n int) {
	if n <= 0 {
		return
	}
	t.Screen.InsertLines(n, t.Cursor.Y)
}

func (t *Terminal) DeleteLines(n int) {
	if n <= 0 {
		return
	}
	t.Screen.DeleteLines(n, t.Cursor.Y)
}

func (t *Terminal) InsertChars(n int) {
	if n <= 0 {
		return
	}
	t.Screen.InsertChars(n, t.Cursor.X, t.Cursor.Y, t.Attributes)
}

func (t *Terminal) DeleteChars(n int) {
	if n <= 0 {
		return
	}
	t.Screen.DeleteChars(n, t.Cursor.X, t.Cursor.Y)
}

func (t *Terminal) EraseChars(n int) {
	if n <= 0 {
		return
	}
	t.Screen.EraseChars(n, t.Cursor.X, t.Cursor.Y, t.Attributes)
}

func (t *Terminal) RepeatChar(n int) {
	if n <= 0 {
		return
	}
	cell := t.Screen.GetCell(t.Cursor.X-1, t.Cursor.Y)
	if cell != nil {
		for i := 0; i < n && t.Cursor.X < t.Cols; i++ {
			t.writeChar(cell.Char)
		}
	}
}

// Cursor state methods.

func (t *Terminal) SaveCursor() {
	t.savedCursor = t.Cursor
	t.savedAttrs = t.Attributes
}

func (t *Terminal) RestoreCursor() {
	t.Cursor = t.savedCursor
	t.Attributes = t.savedAttrs
}

func (t *Terminal) SetCursorVisible(visible bool) {
	t.Cursor.Visible = visible
}

// Mode setting methods.

func (t *Terminal) SetMode(mode int, enabled bool) {
	switch mode {
	case 1: // Application cursor keys (DECCKM)
		t.AppKeypad = enabled
	case 2: // DECANM - ANSI mode
	case 3: // 132 columns (DECCOLM)
		_ = mode
	case 4: // Smooth scroll (DECSCLM)
	case 5: // Reverse video (DECSCNM)
	case 6: // Origin mode (DECOM)
		t.OriginMode = enabled
	case 7: // Auto-wrap (DECAWM)
		t.AutoWrap = enabled
	case 20: // LF/NL mode
	case 25: // Cursor visible (DECTCEM)
		t.Cursor.Visible = enabled
	case 40: // 80/132 columns
	case 1047: // Save/restore screen
	case 1048: // Alternate screen buffer
	case 1049: // Save/restore screen + cursor
		_ = mode
	}
}

func (t *Terminal) SoftReset() {
	t.AppKeypad = false
	t.AutoWrap = true
	t.InsertMode = false
	t.OriginMode = false
	t.LineWrap = true
	t.Attributes = DefaultAttributes()
	t.Cursor.X = 0
	t.Cursor.Y = 0
	t.scrollTop = 0
	t.scrollBottom = t.Rows - 1
}

func (t *Terminal) SetAppKeypad(enabled bool) {
	t.AppKeypad = enabled
}

// Graphics rendition (SGR) methods.

func (t *Terminal) SetGraphicsRendition(params []int) {
	for _, p := range params {
		switch {
		case p == 0:
			t.Attributes = DefaultAttributes()
		case p == 1:
			t.Attributes.Bold = true
		case p == 2:
			t.Attributes.Faint = true
		case p == 3:
			t.Attributes.Italic = true
		case p == 4:
			t.Attributes.Underline = true
		case p == 5:
			t.Attributes.Blink = true
		case p == 7:
			t.Attributes.Reverse = true
		case p == 8:
			t.Attributes.Conceal = true
		case p == 9:
			t.Attributes.CrossedOut = true
		case p == 21:
			t.Attributes.Bold = false
		case p == 22:
			t.Attributes.Bold = false
			t.Attributes.Faint = false
		case p == 23:
			t.Attributes.Italic = false
		case p == 24:
			t.Attributes.Underline = false
			t.Attributes.DoubleUnderline = false
		case p == 25:
			t.Attributes.Blink = false
		case p == 27:
			t.Attributes.Reverse = false
		case p == 28:
			t.Attributes.Conceal = false
		case p == 29:
			t.Attributes.CrossedOut = false
		case p >= 30 && p <= 37:
			t.Attributes.Foreground = Color{Type: ColorStandard, Value: uint8(p - 30)}
		case p == 38:
			// Extended foreground color
			if len(params) > 1 && params[1] == 5 {
				if len(params) > 2 {
					t.Attributes.Foreground = Color{Type: Color256, Value: uint8(params[2])}
				}
			} else if len(params) > 4 {
				t.Attributes.Foreground = Color{Type: ColorRGB, RGB: [3]uint8{
					uint8(params[2]), uint8(params[3]), uint8(params[4]),
				}}
			}
		case p == 39:
			t.Attributes.Foreground = DefaultColor
		case p >= 40 && p <= 47:
			t.Attributes.Background = Color{Type: ColorStandard, Value: uint8(p - 40)}
		case p == 48:
			// Extended background color
			if len(params) > 1 && params[1] == 5 {
				if len(params) > 2 {
					t.Attributes.Background = Color{Type: Color256, Value: uint8(params[2])}
				}
			} else if len(params) > 4 {
				t.Attributes.Background = Color{Type: ColorRGB, RGB: [3]uint8{
					uint8(params[2]), uint8(params[3]), uint8(params[4]),
				}}
			}
		case p == 49:
			t.Attributes.Background = DefaultColor
		case p >= 90 && p <= 97:
			t.Attributes.Foreground = Color{Type: ColorBright, Value: uint8(p - 90)}
		case p >= 100 && p <= 107:
			t.Attributes.Background = Color{Type: ColorBright, Value: uint8(p - 100)}
		}
	}
}

// Window title methods.

func (t *Terminal) SetWindowTitle(title string) {
	t.Title = title
}

// Status report methods.

func (t *Terminal) ReportStatus() {
	// Send "OK" status report
	status := string(ESC) + "[0n"
	t.mu.Lock()
	t.outputBuffer = append(t.outputBuffer, status...)
	t.mu.Unlock()
}

func (t *Terminal) ReportCursorPosition() {
	// Send cursor position report: ESC [ row ; col R
	row := t.Cursor.Y + 1
	col := t.Cursor.X + 1
	report := string(ESC) + "[" + itoa(row) + ";" + itoa(col) + "R"
	t.mu.Lock()
	t.outputBuffer = append(t.outputBuffer, report...)
	t.mu.Unlock()
}

// GetOutputBuffer returns a copy of the output buffer.
func (t *Terminal) GetOutputBuffer() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()
	data := make([]byte, len(t.outputBuffer))
	copy(data, t.outputBuffer)
	return data
}

// ClearOutputBuffer clears the output buffer.
func (t *Terminal) ClearOutputBuffer() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.outputBuffer = t.outputBuffer[:0]
}

// Dump returns a string representation of the terminal screen.
func (t *Terminal) Dump() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Screen.Dump()
}

// Close closes the terminal and releases resources.
func (t *Terminal) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.outputCond.Broadcast()
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
