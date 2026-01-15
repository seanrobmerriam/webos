package pty

import (
	"errors"
	"strings"
)

// DefaultRows is the default number of rows in a terminal.
const DefaultRows = 24

// DefaultCols is the default number of columns in a terminal.
const DefaultCols = 80

// DefaultScrollback is the default scrollback buffer size.
const DefaultScrollback = 1000

// ErrInvalidDimensions is returned when terminal dimensions are invalid.
var ErrInvalidDimensions = errors.New("invalid terminal dimensions")

// Color represents a terminal color.
type Color struct {
	Type  ColorType
	Value uint8
	RGB   [3]uint8
}

// ColorType defines the type of color.
type ColorType int

const (
	// ColorDefault represents the default terminal color.
	ColorDefault ColorType = iota
	// ColorStandard represents one of the 8 standard colors.
	ColorStandard
	// ColorBright represents one of the 8 bright colors.
	ColorBright
	// Color256 represents a 256-color palette index.
	Color256
	// ColorRGB represents a truecolor RGB value.
	ColorRGB
)

// Standard colors matching ANSI codes.
var (
	DefaultColor  = Color{Type: ColorDefault}
	Black         = Color{Type: ColorStandard, Value: 0}
	Red           = Color{Type: ColorStandard, Value: 1}
	Green         = Color{Type: ColorStandard, Value: 2}
	Yellow        = Color{Type: ColorStandard, Value: 3}
	Blue          = Color{Type: ColorStandard, Value: 4}
	Magenta       = Color{Type: ColorStandard, Value: 5}
	Cyan          = Color{Type: ColorStandard, Value: 6}
	White         = Color{Type: ColorStandard, Value: 7}
	BrightBlack   = Color{Type: ColorBright, Value: 0}
	BrightRed     = Color{Type: ColorBright, Value: 1}
	BrightGreen   = Color{Type: ColorBright, Value: 2}
	BrightYellow  = Color{Type: ColorBright, Value: 3}
	BrightBlue    = Color{Type: ColorBright, Value: 4}
	BrightMagenta = Color{Type: ColorBright, Value: 5}
	BrightCyan    = Color{Type: ColorBright, Value: 6}
	BrightWhite   = Color{Type: ColorBright, Value: 7}
)

// CellAttributes represents text formatting attributes.
type CellAttributes struct {
	Bold            bool
	Faint           bool
	Italic          bool
	Underline       bool
	Blink           bool
	Reverse         bool
	Conceal         bool
	CrossedOut      bool
	DoubleUnderline bool
	Foreground      Color
	Background      Color
}

// DefaultAttributes returns the default cell attributes.
func DefaultAttributes() CellAttributes {
	return CellAttributes{
		Foreground: DefaultColor,
		Background: DefaultColor,
	}
}

// Cell represents a single character cell in the terminal.
type Cell struct {
	Char       rune
	Attributes CellAttributes
}

// IsEmpty returns true if the cell contains only a space.
func (c Cell) IsEmpty() bool {
	return c.Char == ' ' || c.Char == 0
}

// ScreenBuffer represents a rectangular region of terminal cells.
type ScreenBuffer struct {
	Rows          [][]Cell
	Width         int
	Height        int
	scrollback    [][]Cell
	maxScrollback int
	cursorY       int
	cursorX       int
}

// NewScreenBuffer creates a new screen buffer with the specified dimensions.
func NewScreenBuffer(width, height, scrollback int) (*ScreenBuffer, error) {
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidDimensions
	}

	if scrollback < 0 {
		scrollback = DefaultScrollback
	}

	rows := make([][]Cell, height)
	for i := range rows {
		rows[i] = make([]Cell, width)
		for j := range rows[i] {
			rows[i][j] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}

	return &ScreenBuffer{
		Rows:          rows,
		Width:         width,
		Height:        height,
		maxScrollback: scrollback,
		scrollback:    make([][]Cell, 0, scrollback),
	}, nil
}

// Resize changes the dimensions of the screen buffer.
func (sb *ScreenBuffer) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return ErrInvalidDimensions
	}

	oldRows := sb.Rows
	oldHeight := sb.Height

	newRows := make([][]Cell, height)
	for i := range newRows {
		newRows[i] = make([]Cell, width)
		for j := range newRows[i] {
			newRows[i][j] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}

	// Copy existing content, preserving as much as possible
	copyHeight := min(oldHeight, height)
	copyWidth := min(sb.Width, width)
	for y := 0; y < copyHeight; y++ {
		for x := 0; x < copyWidth; x++ {
			newRows[y][x] = oldRows[y][x]
		}
	}

	sb.Rows = newRows
	sb.Width = width
	sb.Height = height

	return nil
}

// SetScrollbackLimit sets the maximum number of lines in the scrollback buffer.
func (sb *ScreenBuffer) SetScrollbackLimit(limit int) {
	if limit < 0 {
		limit = 0
	}
	sb.maxScrollback = limit
	if len(sb.scrollback) > limit {
		sb.scrollback = sb.scrollback[len(sb.scrollback)-limit:]
	}
}

// GetScrollbackLines returns the scrollback buffer lines.
func (sb *ScreenBuffer) GetScrollbackLines() [][]Cell {
	result := make([][]Cell, len(sb.scrollback))
	copy(result, sb.scrollback)
	return result
}

// ScrollUp scrolls the screen up by n lines, moving content to scrollback.
func (sb *ScreenBuffer) ScrollUp(n int) {
	if n <= 0 {
		return
	}
	if n >= sb.Height {
		n = sb.Height
	}

	// Move affected lines to scrollback
	for i := 0; i < n; i++ {
		sb.scrollback = append(sb.scrollback, sb.Rows[i])
		if len(sb.scrollback) > sb.maxScrollback {
			sb.scrollback = sb.scrollback[len(sb.scrollback)-sb.maxScrollback:]
		}
	}

	// Shift rows up
	copy(sb.Rows, sb.Rows[n:])

	// Clear new bottom rows
	for y := sb.Height - n; y < sb.Height; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Rows[y][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}
}

// ScrollDown scrolls the screen down by n lines.
func (sb *ScreenBuffer) ScrollDown(n int) {
	if n <= 0 {
		return
	}
	if n >= sb.Height {
		n = sb.Height
	}

	// Shift rows down
	copy(sb.Rows[n:], sb.Rows[:sb.Height-n])

	// Clear new top rows
	for y := 0; y < n; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Rows[y][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}
}

// Clear clears the entire screen.
func (sb *ScreenBuffer) Clear() {
	for y := 0; y < sb.Height; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Rows[y][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}
}

// ClearToEndOfScreen clears from cursor to end of screen.
func (sb *ScreenBuffer) ClearToEndOfScreen(cursorX, cursorY int) {
	// Clear from cursor to end of line
	for x := cursorX; x < sb.Width; x++ {
		sb.Rows[cursorY][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
	}
	// Clear remaining lines
	for y := cursorY + 1; y < sb.Height; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Rows[y][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}
}

// ClearToBeginningOfScreen clears from beginning of screen to cursor.
func (sb *ScreenBuffer) ClearToBeginningOfScreen(cursorX, cursorY int) {
	// Clear from beginning of line to cursor
	for x := 0; x <= cursorX; x++ {
		sb.Rows[cursorY][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
	}
	// Clear previous lines
	for y := 0; y < cursorY; y++ {
		for x := 0; x < sb.Width; x++ {
			sb.Rows[y][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
		}
	}
}

// ClearLine clears the current line.
func (sb *ScreenBuffer) ClearLine(y int) {
	for x := 0; x < sb.Width; x++ {
		sb.Rows[y][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
	}
}

// ClearToEndOfLine clears from cursor to end of line.
func (sb *ScreenBuffer) ClearToEndOfLine(x, y int) {
	for i := x; i < sb.Width; i++ {
		sb.Rows[y][i] = Cell{Char: ' ', Attributes: DefaultAttributes()}
	}
}

// ClearToBeginningOfLine clears from beginning of line to cursor.
func (sb *ScreenBuffer) ClearToBeginningOfLine(x, y int) {
	for i := 0; i <= x; i++ {
		sb.Rows[y][i] = Cell{Char: ' ', Attributes: DefaultAttributes()}
	}
}

// InsertLines inserts n blank lines at the cursor row.
func (sb *ScreenBuffer) InsertLines(n, cursorY int) {
	if n <= 0 {
		return
	}

	end := sb.Height - n
	if cursorY > end {
		cursorY = end
	}

	// Move lines down
	for y := end - 1; y >= cursorY; y-- {
		copy(sb.Rows[y+n], sb.Rows[y])
	}

	// Clear inserted lines
	for y := cursorY; y < cursorY+n; y++ {
		sb.ClearLine(y)
	}
}

// DeleteLines deletes n lines at the cursor row.
func (sb *ScreenBuffer) DeleteLines(n, cursorY int) {
	if n <= 0 {
		return
	}

	end := sb.Height - n
	if cursorY > end {
		cursorY = end
	}

	// Move lines up
	for y := cursorY; y < end; y++ {
		copy(sb.Rows[y], sb.Rows[y+n])
	}

	// Clear bottom lines
	for y := end; y < sb.Height; y++ {
		sb.ClearLine(y)
	}
}

// InsertChars inserts n blank characters at the cursor position.
func (sb *ScreenBuffer) InsertChars(n, cursorX, cursorY int, attrs CellAttributes) {
	if n <= 0 {
		return
	}

	end := sb.Width - n
	if cursorX > end {
		cursorX = end
	}

	// Shift characters right
	for x := end; x > cursorX; x-- {
		sb.Rows[cursorY][x+n-1] = sb.Rows[cursorY][x-1]
	}

	// Clear inserted space
	for x := cursorX; x < cursorX+n && x < sb.Width; x++ {
		sb.Rows[cursorY][x] = Cell{Char: ' ', Attributes: attrs}
	}
}

// DeleteChars deletes n characters at the cursor position.
func (sb *ScreenBuffer) DeleteChars(n, cursorX, cursorY int) {
	if n <= 0 {
		return
	}

	end := sb.Width - n
	if cursorX > end {
		cursorX = end
	}

	// Shift characters left
	for x := cursorX; x < end; x++ {
		sb.Rows[cursorY][x] = sb.Rows[cursorY][x+n]
	}

	// Clear remaining characters
	for x := end; x < sb.Width; x++ {
		sb.Rows[cursorY][x] = Cell{Char: ' ', Attributes: DefaultAttributes()}
	}
}

// EraseChars erases n characters (replaces with spaces).
func (sb *ScreenBuffer) EraseChars(n, cursorX, cursorY int, attrs CellAttributes) {
	if n <= 0 {
		return
	}

	end := cursorX + n
	if end > sb.Width {
		end = sb.Width
	}

	for x := cursorX; x < end; x++ {
		sb.Rows[cursorY][x] = Cell{Char: ' ', Attributes: attrs}
	}
}

// GetCell returns the cell at the specified position.
func (sb *ScreenBuffer) GetCell(x, y int) *Cell {
	if x < 0 || x >= sb.Width || y < 0 || y >= sb.Height {
		return nil
	}
	return &sb.Rows[y][x]
}

// SetCell sets the cell at the specified position.
func (sb *ScreenBuffer) SetCell(x, y int, cell Cell) {
	if x < 0 || x >= sb.Width || y < 0 || y >= sb.Height {
		return
	}
	sb.Rows[y][x] = cell
}

// Dump returns a string representation of the screen buffer.
func (sb *ScreenBuffer) Dump() string {
	var builder strings.Builder
	for y := 0; y < sb.Height; y++ {
		for x := 0; x < sb.Width; x++ {
			ch := sb.Rows[y][x].Char
			if ch == 0 {
				ch = ' '
			}
			builder.WriteRune(ch)
		}
		builder.WriteRune('\n')
	}
	return builder.String()
}
