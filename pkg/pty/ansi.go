package pty

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// ANSI escape sequence constants.
const (
	ESC = '\x1b'
	BEL = '\x07'
	BS  = '\x08'
	HT  = '\x09'
	LF  = '\x0a'
	VT  = '\x0b'
	FF  = '\x0c'
	CR  = '\x0d'
)

// Parser processes ANSI escape sequences.
type Parser struct {
	term *Terminal
}

// NewParser creates a new ANSI parser for the given terminal.
func NewParser(term *Terminal) *Parser {
	return &Parser{term: term}
}

// Parse processes input data and updates the terminal state.
func (p *Parser) Parse(data []byte) {
	p.parseBytes(data)
}

// parseBytes handles the raw byte input.
func (p *Parser) parseBytes(data []byte) {
	i := 0
	for i < len(data) {
		b := data[i]
		switch {
		case b == ESC:
			// Check for CSI or other escape sequences
			if i+1 < len(data) {
				next := data[i+1]
				if next == '[' {
					// CSI sequence
					end := p.findSequenceEnd(data[i+2:])
					if end >= 0 {
						p.parseCSI(data[i+2 : i+2+end])
						i += 2 + end + 1 // Skip CSI prefix, sequence, and final char
						continue
					}
				} else if next == ']' {
					// OSC sequence
					end := p.findSequenceEnd(data[i+2:])
					if end >= 0 {
						p.parseOSC(data[i+2 : i+2+end])
						i += 2 + end + 1
						continue
					}
				} else if next == '(' || next == ')' {
					// Character set designation
					p.parseCharset(data[i+1], data[i+2])
					i += 3
					continue
				} else if next == 'M' {
					// Reverse index (RI)
					p.term.ReverseIndex()
					i += 2
					continue
				} else if next == 'E' {
					// Next line (NEL)
					p.term.CarriageReturn()
					p.term.LineFeed()
					i += 2
					continue
				} else if next == '7' {
					// Save cursor (DECSC)
					p.term.SaveCursor()
					i += 2
					continue
				} else if next == '8' {
					// Restore cursor (DECRC)
					p.term.RestoreCursor()
					i += 2
					continue
				} else if next == '=' {
					// Application keypad mode (DECPAM)
					p.term.SetAppKeypad(true)
					i += 2
					continue
				} else if next == '>' {
					// Normal keypad mode (DECPNM)
					p.term.SetAppKeypad(false)
					i += 2
					continue
				}
			}
			i++

		case b == HT:
			// Tab
			p.term.TabForward(1)
			i++

		case b == BS:
			// Backspace
			p.term.Backspace()
			i++

		case b == BEL:
			// Bell - ignore for now
			i++

		case b == CR:
			// Carriage return
			p.term.CarriageReturn()
			i++

		case b == LF || b == VT || b == FF:
			// Line feed
			p.term.LineFeed()
			i++

		default:
			// Regular character
			if b >= 0x20 && b <= 0x7e || b >= 0x80 {
				// Printable character
				r, _ := utf8.DecodeRune(data[i:])
				p.term.WriteChar(r)
				i += utf8.RuneLen(r)
			} else {
				// Control character - ignore
				i++
			}
		}
	}
}

// findSequenceEnd finds the end of an escape sequence.
func (p *Parser) findSequenceEnd(data []byte) int {
	for i, b := range data {
		if (b >= 0x40 && b <= 0x7e) || b == '~' {
			return i
		}
	}
	return -1
}

// parseCSI handles Control Sequence Introducer sequences.
func (p *Parser) parseCSI(seq []byte) {
	if len(seq) == 0 {
		return
	}

	// Get final character
	final := seq[len(seq)-1]
	intermediate := []byte{}
	params := []byte{}

	// Split into intermediate and parameter bytes
	for i := 0; i < len(seq)-1; i++ {
		b := seq[i]
		if b >= 0x20 && b <= 0x2f {
			intermediate = append(intermediate, b)
		} else if b >= 0x30 && b <= 0x3f {
			params = append(params, b)
		}
	}

	// Parse parameters
	paramVals := p.parseParams(params)

	switch final {
	case 'A': // CUU - Cursor up
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorUp(n)

	case 'B': // CUD - Cursor down
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorDown(n)

	case 'C': // CUF - Cursor forward (right)
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorForward(n)

	case 'D': // CUB - Cursor backward (left)
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorBackward(n)

	case 'E': // CNL - Cursor down and to column 1
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorDown(n)
		p.term.CarriageReturn()

	case 'F': // CPL - Cursor up and to column 1
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorUp(n)
		p.term.CarriageReturn()

	case 'G': // CHA - Cursor to column
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorToColumn(n)

	case 'H': // CUP - Cursor position
		row := p.getParam(paramVals, 0, 1)
		col := p.getParam(paramVals, 1, 1)
		p.term.MoveCursorTo(col, row)

	case 'J': // ED - Erase display
		mode := p.getParam(paramVals, 0, 0)
		switch mode {
		case 0:
			p.term.ClearToEndOfScreen()
		case 1:
			p.term.ClearToBeginningOfScreen()
		case 2, 3:
			p.term.ClearScreen()
		}

	case 'K': // EL - Erase line
		mode := p.getParam(paramVals, 0, 0)
		switch mode {
		case 0:
			p.term.ClearToEndOfLine()
		case 1:
			p.term.ClearToBeginningOfLine()
		case 2:
			p.term.ClearLine()
		}

	case 'L': // IL - Insert lines
		n := p.getParam(paramVals, 0, 1)
		p.term.InsertLines(n)

	case 'M': // DL - Delete lines
		n := p.getParam(paramVals, 0, 1)
		p.term.DeleteLines(n)

	case 'P': // DCH - Delete characters
		n := p.getParam(paramVals, 0, 1)
		p.term.DeleteChars(n)

	case 'S': // SU - Scroll up
		n := p.getParam(paramVals, 0, 1)
		p.term.ScrollUp(n)

	case 'T': // SD - Scroll down
		n := p.getParam(paramVals, 0, 1)
		p.term.ScrollDown(n)

	case 'X': // ECH - Erase characters
		n := p.getParam(paramVals, 0, 1)
		p.term.EraseChars(n)

	case 'Z': // CBT - Cursor backward tab
		n := p.getParam(paramVals, 0, 1)
		p.term.TabBackward(n)

	case 'b': // REP - Repeat previous character
		n := p.getParam(paramVals, 0, 1)
		p.term.RepeatChar(n)

	case 'c': // DA - Primary device attributes
		// Terminal reports its capabilities
		_ = intermediate
		_ = paramVals

	case 'd': // VPA - Vertical position absolute
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorToRow(n)

	case 'e': // VPR - Vertical position relative
		n := p.getParam(paramVals, 0, 1)
		p.term.CursorDown(n)

	case 'f': // HVP - Horizontal and vertical position
		row := p.getParam(paramVals, 0, 1)
		col := p.getParam(paramVals, 1, 1)
		p.term.MoveCursorTo(col, row)

	case 'g': // TBC - Tab clear
		mode := p.getParam(paramVals, 0, 0)
		switch mode {
		case 0:
			p.term.ClearTab()
		case 3:
			p.term.ClearAllTabs()
		}

	case 'h': // SM - Set mode
		for _, val := range paramVals {
			p.term.SetMode(val, true)
		}

	case 'l': // RM - Reset mode
		for _, val := range paramVals {
			p.term.SetMode(val, false)
		}

	case 'm': // SGR - Set graphics rendition
		p.term.SetGraphicsRendition(paramVals)

	case 'n': // DSR - Device status report
		mode := p.getParam(paramVals, 0, 0)
		switch mode {
		case 5:
			// Status report - ready
			p.term.ReportStatus()
		case 6:
			// Cursor position report
			p.term.ReportCursorPosition()
		}

	case 'p': // DECSTR - Soft reset
		p.term.SoftReset()

	case 'q': // DECLL - Load LEDs
		_ = paramVals

	case 'r': // DECSTBM - Set scrolling region
		top := p.getParam(paramVals, 0, 1)
		bottom := p.getParam(paramVals, 1, p.term.Rows)
		p.term.SetScrollingRegion(top-1, bottom-1)

	case 's': // SCP - Save cursor position
		p.term.SaveCursor()

	case 'u': // RCP - Restore cursor position
		p.term.RestoreCursor()

	case '~': // Delete (DEL) - not typically used
		_ = intermediate
	}
}

// parseOSC handles Operating System Command sequences.
func (p *Parser) parseOSC(seq []byte) {
	// OSC sequences are typically: ESC ] <id> ; <data> BEL or ESC ] <id> ; <data> ST
	// Common ones:
	// 0 - Set window title
	// 1 - Set icon name
	// 2 - Set window title and icon name
	// 4 - Set color palette
	// 10 - Set foreground color
	// 11 - Set background color
	// 12 - Set cursor color
	// 104 - Reset color

	if len(seq) < 2 {
		return
	}

	// Find the semicolon
	semicolon := -1
	for i := 1; i < len(seq); i++ {
		if seq[i] == ';' {
			semicolon = i
			break
		}
	}

	if semicolon < 0 {
		return
	}

	id := 0
	idStr := string(seq[1:semicolon])
	if idVal, err := strconv.Atoi(idStr); err == nil {
		id = idVal
	}

	data := string(seq[semicolon+1:])

	switch id {
	case 0, 1, 2:
		// Window title
		p.term.SetWindowTitle(data)
	case 4:
		// Color palette
		p.parseColorPalette(data)
	case 10, 11, 12, 17, 19:
		// Colors
		p.parseColorSetting(id, data)
	}
}

// parseColorPalette handles color palette OSC sequences.
func (p *Parser) parseColorPalette(data string) {
	// Format: <index> ; rgb:r/g/b or <index> ; ?
	parts := strings.Split(data, ";")
	for i, part := range parts {
		if i == 0 {
			continue // Skip ID
		}
		// Simplified parsing
		_ = part
	}
}

// parseColorSetting handles individual color setting OSC sequences.
func (p *Parser) parseColorSetting(id int, data string) {
	// Format: rgb:r/g/b or ?
	parts := strings.Split(data, ";")
	if len(parts) == 0 {
		return
	}

	colorSpec := parts[0]
	if colorSpec == "?" {
		return // Query - not supported
	}

	// Parse rgb:r/g/b format
	if strings.HasPrefix(colorSpec, "rgb:") {
		rgb := strings.Split(colorSpec[4:], "/")
		if len(rgb) == 3 {
			var r, g, b uint8
			if _, err := strconv.ParseUint(rgb[0], 16, 8); err == nil {
				// Simplified
			}
			_ = r
			_ = g
			_ = b
		}
	}
}

// parseCharset handles character set designation.
func (p *Parser) parseCharset(designator byte, charset byte) {
	_ = designator
	_ = charset
	// Character set switching - not fully implemented
}

// parseParams parses a parameter string into integer values.
func (p *Parser) parseParams(params []byte) []int {
	if len(params) == 0 {
		return []int{}
	}

	result := []int{}
	current := []byte{}

	for _, b := range params {
		if b == ';' {
			if len(current) > 0 {
				if val, err := strconv.Atoi(string(current)); err == nil {
					result = append(result, val)
				}
				current = []byte{}
			} else {
				result = append(result, 0)
			}
		} else if b >= '0' && b <= '9' {
			current = append(current, b)
		}
	}

	// Don't forget the last parameter
	if len(current) > 0 {
		if val, err := strconv.Atoi(string(current)); err == nil {
			result = append(result, val)
		}
	}

	return result
}

// getParam gets a parameter value with a default.
func (p *Parser) getParam(params []int, index, defaultVal int) int {
	if index < 0 || index >= len(params) {
		return defaultVal
	}
	return params[index]
}

// ParseString is a convenience method for parsing string input.
func (p *Parser) ParseString(s string) {
	p.Parse([]byte(s))
}
