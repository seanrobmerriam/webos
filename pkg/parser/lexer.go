// Package parser provides lexical analysis and parsing for the WebOS shell.
// It implements a lexer for tokenizing shell commands and a recursive descent
// parser for building abstract syntax trees (ASTs).
package parser

import (
	"unicode"
)

// TokenType represents the type of a lexical token.
type TokenType int

// Token types for the shell language.
const (
	TokenEOF TokenType = iota
	TokenError
	TokenNewline
	TokenWord
	TokenString      // Single-quoted string
	TokenDoubleQuote // Double-quoted string with variable expansion
	TokenPipe        // |
	TokenAnd         // &&
	TokenOr          // ||
	TokenBackground  // &
	TokenRedirectIn  // <
	TokenRedirectOut // >
	TokenAppend      // >>
	TokenSemicolon   // //
	TokenLeftParen   // (
	TokenRightParen  // )
)

// Token represents a lexical token in the shell language.
type Token struct {
	Type TokenType
	Text string
	Pos  int // Position in the input string
}

// String returns a string representation of the token.
func (t Token) String() string {
	if t.Text != "" {
		return t.Text
	}
	return tokenTypeToString(t.Type)
}

// tokenTypeToString returns the string representation of a token type.
func tokenTypeToString(t TokenType) string {
	switch t {
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "ERROR"
	case TokenNewline:
		return "NEWLINE"
	case TokenWord:
		return "WORD"
	case TokenString:
		return "STRING"
	case TokenDoubleQuote:
		return "DQSTRING"
	case TokenPipe:
		return "|"
	case TokenAnd:
		return "&&"
	case TokenOr:
		return "||"
	case TokenBackground:
		return "&"
	case TokenRedirectIn:
		return "<"
	case TokenRedirectOut:
		return ">"
	case TokenAppend:
		return ">>"
	case TokenSemicolon:
		return ";"
	case TokenLeftParen:
		return "("
	case TokenRightParen:
		return ")"
	default:
		return "UNKNOWN"
	}
}

// Lexer performs lexical analysis on shell input.
type Lexer struct {
	input  string
	pos    int
	width  int
	tokens []Token
}

// NewLexer creates a new Lexer for the given input.
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		tokens: make([]Token, 0, 32),
	}
}

// NextToken returns the next token in the input.
func (l *Lexer) NextToken() Token {
	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos}
	}

	// Skip whitespace
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		if l.input[l.pos] == '\n' {
			l.pos++
			return Token{Type: TokenNewline, Pos: l.pos - 1}
		}
		l.pos++
	}

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Pos: l.pos}
	}

	start := l.pos
	ch := rune(l.input[l.pos])

	// Handle strings
	if ch == '\'' {
		return l.scanSingleQuote(start)
	}
	if ch == '"' {
		return l.scanDoubleQuote(start)
	}

	// Handle operators
	if l.width = utf8RuneLen(ch); l.isOperator(ch) {
		l.pos += l.width
		// Check for two-character operators
		if l.pos < len(l.input) {
			twoChar := l.input[start:l.pos] + string(l.input[l.pos])
			switch twoChar {
			case ">>":
				l.pos++
				return Token{Type: TokenAppend, Text: ">>", Pos: start}
			case "&&":
				l.pos++
				return Token{Type: TokenAnd, Text: "&&", Pos: start}
			case "||":
				l.pos++
				return Token{Type: TokenOr, Text: "||", Pos: start}
			}
		}
		return Token{Type: tokenTypeFromChar(ch), Text: string(ch), Pos: start}
	}

	// Handle words and numbers
	for l.pos < len(l.input) {
		ch := rune(l.input[l.pos])
		if unicode.IsSpace(ch) || l.isOperator(ch) || ch == ';' || ch == '(' || ch == ')' {
			break
		}
		l.pos += utf8RuneLen(ch)
	}

	return Token{Type: TokenWord, Text: l.input[start:l.pos], Pos: start}
}

// scanSingleQuote scans a single-quoted string.
func (l *Lexer) scanSingleQuote(start int) Token {
	l.pos++ // Skip opening quote
	for l.pos < len(l.input) && l.input[l.pos] != '\'' {
		l.pos++
	}
	if l.pos < len(l.input) {
		l.pos++ // Skip closing quote
	}
	return Token{Type: TokenString, Text: l.input[start:l.pos], Pos: start}
}

// scanDoubleQuote scans a double-quoted string with variable expansion.
func (l *Lexer) scanDoubleQuote(start int) Token {
	l.pos++ // Skip opening quote
	for l.pos < len(l.input) && l.input[l.pos] != '"' {
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos += 2 // Skip escaped character
		} else {
			l.pos++
		}
	}
	if l.pos < len(l.input) {
		l.pos++ // Skip closing quote
	}
	return Token{Type: TokenDoubleQuote, Text: l.input[start:l.pos], Pos: start}
}

// isOperator returns true if the character is an operator.
func (l *Lexer) isOperator(ch rune) bool {
	return ch == '|' || ch == '<' || ch == '>' || ch == '&' || ch == ';' || ch == '(' || ch == ')'
}

// tokenTypeFromChar returns the token type for a single character.
func tokenTypeFromChar(ch rune) TokenType {
	switch ch {
	case '|':
		return TokenPipe
	case '&':
		return TokenBackground
	case '<':
		return TokenRedirectIn
	case '>':
		return TokenRedirectOut
	case ';':
		return TokenSemicolon
	case '(':
		return TokenLeftParen
	case ')':
		return TokenRightParen
	default:
		return TokenWord
	}
}

// utf8RuneLen returns the length of a UTF-8 rune in bytes.
func utf8RuneLen(ch rune) int {
	switch {
	case ch < 0x80:
		return 1
	case ch < 0x800:
		return 2
	case ch < 0x10000:
		return 3
	default:
		return 4
	}
}

// Tokens returns all tokens from the input.
func (l *Lexer) Tokens() []Token {
	tokens := make([]Token, 0, 32)
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return tokens
}
