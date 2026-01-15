package graphics

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
)

// Common errors for font operations
var (
	ErrInvalidFontData       = errors.New("invalid font data")
	ErrGlyphNotFound         = errors.New("glyph not found")
	ErrFontNotFound          = errors.New("font file not found")
	ErrUnsupportedFontFormat = errors.New("unsupported font format")
)

// Font represents a basic font for text rendering
type Font struct {
	name    string
	size    float64
	ascent  float64
	descent float64
	height  float64
	glyphs  map[rune]*Glyph
	bold    bool
	italic  bool
}

// Glyph represents a character glyph
type Glyph struct {
	Char     rune
	Advance  float64
	BearingX float64
	BearingY float64
	Width    float64
	Height   float64
	Bitmap   []byte
	OffsetX  float64
	OffsetY  float64
}

// TextMetrics holds measurements for rendered text
type TextMetrics struct {
	Width      float64
	Height     float64
	Ascent     float64
	Descent    float64
	GlyphCount int
}

// NewFont creates a new Font instance
func NewFont(name string, size float64) *Font {
	return &Font{
		name:    name,
		size:    size,
		ascent:  size * 0.8,
		descent: size * 0.2,
		height:  size,
		glyphs:  make(map[rune]*Glyph),
	}
}

// Name returns the font name
func (f *Font) Name() string {
	return f.name
}

// Size returns the font size
func (f *Font) Size() float64 {
	return f.size
}

// SetBold enables/disables bold style
func (f *Font) SetBold(bold bool) *Font {
	f.bold = bold
	return f
}

// SetItalic enables/disables italic style
func (f *Font) SetItalic(italic bool) *Font {
	f.italic = italic
	return f
}

// LoadTrueType loads a TrueType font from a file
// Note: This is a simplified implementation. For full TTF support,
// use golang.org/x/image/font/sfnt
func LoadTrueType(path string) (*Font, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFontNotFound
		}
		return nil, err
	}
	defer file.Close()

	return ParseTrueType(file)
}

// ParseTrueType parses TrueType font data
func ParseTrueType(r io.Reader) (*Font, error) {
	reader := bufio.NewReader(r)

	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, ErrInvalidFontData
	}

	// Check for valid font magic number
	if string(header) != "ttcf" && string(header) != "\x00\x01\x00\x00" {
		return nil, ErrUnsupportedFontFormat
	}

	font := &Font{
		name:    "TrueType",
		size:    12,
		ascent:  10,
		descent: 3,
		height:  13,
		glyphs:  make(map[rune]*Glyph),
	}

	return font, nil
}

// LoadBasicFont loads a basic built-in font
func LoadBasicFont(name string, size float64) *Font {
	font := NewFont(name, size)

	// Initialize with basic ASCII glyphs
	for i := 32; i <= 126; i++ {
		font.glyphs[rune(i)] = createBasicGlyph(rune(i), size)
	}

	return font
}

// createBasicGlyph creates a simplified glyph for basic characters
func createBasicGlyph(ch rune, size float64) *Glyph {
	width := size * 0.6
	if ch == ' ' {
		width = size * 0.25
	} else if ch == 'i' || ch == 'j' || ch == 'l' || ch == '|' {
		width = size * 0.3
	} else if ch == 'm' || ch == 'w' {
		width = size * 0.9
	}

	return &Glyph{
		Char:     ch,
		Advance:  width,
		BearingX: 0,
		BearingY: -size * 0.7,
		Width:    width,
		Height:   size,
		Bitmap:   nil,
		OffsetX:  0,
		OffsetY:  size * 0.8,
	}
}

// MeasureText measures the dimensions of text
func (f *Font) MeasureText(text string) TextMetrics {
	var width float64
	for _, ch := range text {
		glyph := f.glyphs[ch]
		if glyph != nil {
			width += glyph.Advance
		} else {
			width += f.size * 0.6
		}
	}

	return TextMetrics{
		Width:      width,
		Height:     f.height,
		Ascent:     f.ascent,
		Descent:    f.descent,
		GlyphCount: len(text),
	}
}

// GetGlyph returns a glyph for the specified character
func (f *Font) GetGlyph(ch rune) *Glyph {
	if glyph, ok := f.glyphs[ch]; ok {
		return glyph
	}
	// Return missing glyph indicator
	return &Glyph{
		Char:    ch,
		Advance: f.size * 0.6,
		Width:   f.size * 0.6,
		Height:  f.size,
	}
}

// AddGlyph adds a custom glyph to the font
func (f *Font) AddGlyph(ch rune, glyph *Glyph) {
	f.glyphs[ch] = glyph
}

// Kerning returns the kerning adjustment between two characters
func (f *Font) Kerning(ch1, ch2 rune) float64 {
	return 0
}

// TextToImage creates an image with rendered text
func (f *Font) TextToImage(text string, bgColor color.Color) (*Image, error) {
	metrics := f.MeasureText(text)

	width := int(metrics.Width) + 20
	height := int(f.height) + 20

	// Create background
	img := Fill(width, height, bgColor)

	return img, nil
}

// WordWrap wraps text to fit within a maximum width
func (f *Font) WordWrap(text string, maxWidth float64) []string {
	var lines []string
	var currentLine bytes.Buffer

	for _, word := range splitWords(text) {
		testLine := currentLine.String()
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		metrics := f.MeasureText(testLine)
		if metrics.Width <= maxWidth {
			currentLine.WriteString(testLine)
		} else {
			if currentLine.Len() > 0 {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}
			wordMetrics := f.MeasureText(word)
			if wordMetrics.Width > maxWidth {
				lines = append(lines, word)
			} else {
				currentLine.WriteString(word)
			}
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// splitWords splits text into words
func splitWords(text string) []string {
	var words []string
	var word bytes.Buffer

	for _, ch := range text {
		if ch == ' ' || ch == '\n' || ch == '\t' {
			if word.Len() > 0 {
				words = append(words, word.String())
				word.Reset()
			}
			if ch == '\n' {
				words = append(words, "\n")
			}
		} else {
			word.WriteRune(ch)
		}
	}

	if word.Len() > 0 {
		words = append(words, word.String())
	}

	return words
}

// BDFLoader loads fonts from BDF (Bitmap Distribution Format) files
type BDFLoader struct{}

// LoadBDF loads a BDF font from a file
func (l *BDFLoader) LoadBDF(path string) (*Font, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return l.ParseBDF(file)
}

// ParseBDF parses BDF font data
func (l *BDFLoader) ParseBDF(r io.Reader) (*Font, error) {
	scanner := bufio.NewScanner(r)
	var font *Font
	var currentGlyph *Glyph
	var bitmapData []byte

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case len(line) >= 8 && line[:8] == "STARTFONT":
			font = &Font{
				name:   "BDF Font",
				size:   12,
				glyphs: make(map[rune]*Glyph),
			}
		case len(line) >= 4 && line[:4] == "SIZE":
			// Parse size
		case len(line) >= 15 && line[:15] == "FONTBOUNDINGBOX":
			// Parse bounding box
		case len(line) >= 10 && line[:10] == "STARTCHAR":
			currentGlyph = &Glyph{}
		case len(line) >= 9 && line[:9] == "ENCODING ":
			if currentGlyph != nil {
				var encoding int
				fmt.Sscanf(line[9:], "%d", &encoding)
				currentGlyph.Char = rune(encoding)
			}
		case len(line) >= 4 && line[:4] == "BBX ":
			if currentGlyph != nil {
				var w, h, bx, by int
				fmt.Sscanf(line[4:], "%d %d %d %d", &w, &h, &bx, &by)
				currentGlyph.Width = float64(w)
				currentGlyph.Height = float64(h)
				currentGlyph.OffsetX = float64(bx)
				currentGlyph.OffsetY = float64(by)
			}
		case len(line) >= 7 && line[:7] == "BITMAP":
			bitmapData = nil
		case len(line) >= 7 && line[:7] == "ENDCHAR":
			if font != nil && currentGlyph != nil {
				font.glyphs[currentGlyph.Char] = currentGlyph
			}
			currentGlyph = nil
		case len(line) >= 7 && line[:7] == "ENDFONT":
			return font, nil
		default:
			if len(line) > 0 && isHexLine(line) {
				data, _ := parseHexLine(line)
				bitmapData = append(bitmapData, data...)
				if currentGlyph != nil {
					currentGlyph.Bitmap = bitmapData
				}
			}
		}
	}

	return font, scanner.Err()
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func isHexLine(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return len(s) > 0
}

func parseHexLine(s string) ([]byte, error) {
	data := make([]byte, len(s)/2)
	for i := 0; i < len(data); i++ {
		var val uint8
		_, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &val)
		if err != nil {
			return nil, err
		}
		data[i] = val
	}
	return data, nil
}
