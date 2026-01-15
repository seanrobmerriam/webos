// Package parser provides tests for the lexer and parser.
package parser

import (
	"testing"
)

func TestLexerTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected []TokenType
	}{
		{"echo hello", []TokenType{TokenWord, TokenWord, TokenEOF}},
		{"echo hello world", []TokenType{TokenWord, TokenWord, TokenWord, TokenEOF}},
		{"echo 'hello world'", []TokenType{TokenWord, TokenString, TokenEOF}},
		{"cat file.txt | grep pattern", []TokenType{TokenWord, TokenWord, TokenPipe, TokenWord, TokenWord, TokenEOF}},
		{"cmd1 && cmd2", []TokenType{TokenWord, TokenAnd, TokenWord, TokenEOF}},
		{"cmd1 || cmd2", []TokenType{TokenWord, TokenOr, TokenWord, TokenEOF}},
		{"cmd &", []TokenType{TokenWord, TokenBackground, TokenEOF}},
		{"cmd; cmd", []TokenType{TokenWord, TokenSemicolon, TokenWord, TokenEOF}},
		{"cmd > output.txt", []TokenType{TokenWord, TokenRedirectOut, TokenWord, TokenEOF}},
		{"cmd < input.txt", []TokenType{TokenWord, TokenRedirectIn, TokenWord, TokenEOF}},
		{"cmd >> output.txt", []TokenType{TokenWord, TokenAppend, TokenWord, TokenEOF}},
		{"(cmd)", []TokenType{TokenLeftParen, TokenWord, TokenRightParen, TokenEOF}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			for i, expected := range tt.expected {
				tok := lexer.NextToken()
				if tok.Type != expected {
					t.Errorf("token %d: expected %s, got %s", i, tokenTypeToString(expected), tokenTypeToString(tok.Type))
				}
			}
		})
	}
}

func TestLexerStringTokens(t *testing.T) {
	tests := []struct {
		input       string
		expectedTok TokenType
		expectedStr string
	}{
		{"'hello'", TokenString, "'hello'"},
		{"'hello world'", TokenString, "'hello world'"},
		{"\"hello\"", TokenDoubleQuote, "\"hello\""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != tt.expectedTok {
				t.Errorf("expected token type %s, got %s", tokenTypeToString(tt.expectedTok), tokenTypeToString(tok.Type))
			}
			if tok.Text != tt.expectedStr {
				t.Errorf("expected text %q, got %q", tt.expectedStr, tok.Text)
			}
		})
	}
}

func TestLexerWhitespace(t *testing.T) {
	input := "  echo   hello  \tworld  \n"
	lexer := NewLexer(input)

	tok := lexer.NextToken()
	if tok.Type != TokenWord || tok.Text != "echo" {
		t.Errorf("expected echo, got %s", tok.Text)
	}

	tok = lexer.NextToken()
	if tok.Type != TokenWord || tok.Text != "hello" {
		t.Errorf("expected hello, got %s", tok.Text)
	}

	tok = lexer.NextToken()
	if tok.Type != TokenWord || tok.Text != "world" {
		t.Errorf("expected world, got %s", tok.Text)
	}
}

func TestLexerAllTokens(t *testing.T) {
	// Test all token types
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"|", TokenPipe},
		{"&", TokenBackground},
		{"<", TokenRedirectIn},
		{">", TokenRedirectOut},
		{";", TokenSemicolon},
		{"(", TokenLeftParen},
		{")", TokenRightParen},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tokenTypeToString(tt.expected), tokenTypeToString(tok.Type))
			}
		})
	}
}

func TestLexerTwoCharOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenType
	}{
		{"&&", TokenAnd},
		{"||", TokenOr},
		{">>", TokenAppend},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tok := lexer.NextToken()
			if tok.Type != tt.expected {
				t.Errorf("expected %s, got %s", tokenTypeToString(tt.expected), tokenTypeToString(tok.Type))
			}
		})
	}
}

func TestParserParseString(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"echo hello", true},
		{"echo hello world", true},
		{"cat file.txt | grep pattern", true},
		{"cmd1 && cmd2", true},
		{"cmd1 || cmd2", true},
		{"cmd &", true},
		{"cmd; cmd", true},
		{"cmd > output.txt", true},
		{"cmd < input.txt", true},
		{"(cmd)", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			list, err := ParseString(tt.input)
			if tt.valid && err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got none")
			}
			if tt.valid && list == nil {
				t.Error("expected non-nil list")
			}
		})
	}
}

func TestParserSimpleCommand(t *testing.T) {
	input := "echo hello"
	list, err := ParseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(list.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(list.Elements))
	}

	pipeline := list.Elements[0]
	if len(pipeline.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(pipeline.Commands))
	}

	cmd := pipeline.Commands[0]
	if cmd.Name != "echo" {
		t.Errorf("expected command 'echo', got '%s'", cmd.Name)
	}

	if len(cmd.Args) != 1 || cmd.Args[0] != "hello" {
		t.Errorf("expected args ['hello'], got %v", cmd.Args)
	}
}

func TestParserPipeline(t *testing.T) {
	input := "cat file.txt | grep pattern | wc -l"
	list, err := ParseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(list.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(list.Elements))
	}

	pipeline := list.Elements[0]
	if len(pipeline.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(pipeline.Commands))
	}

	expectedCmds := []string{"cat", "grep", "wc"}
	for i, expected := range expectedCmds {
		if pipeline.Commands[i].Name != expected {
			t.Errorf("command %d: expected '%s', got '%s'", i, expected, pipeline.Commands[i].Name)
		}
	}
}

func TestParserListWithSemicolon(t *testing.T) {
	input := "cmd1; cmd2; cmd3"
	list, err := ParseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(list.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(list.Elements))
	}

	if len(list.Sep) != 2 {
		t.Fatalf("expected 2 separators, got %d", len(list.Sep))
	}

	for i, sep := range list.Sep {
		if sep != TokenSemicolon {
			t.Errorf("separator %d: expected %s, got %s", i, tokenTypeToString(TokenSemicolon), tokenTypeToString(sep))
		}
	}
}

func TestParserBackground(t *testing.T) {
	input := "cmd &"
	list, err := ParseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(list.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(list.Elements))
	}

	if len(list.Sep) != 1 || list.Sep[0] != TokenBackground {
		t.Errorf("expected background separator")
	}
}

func TestParserRedirection(t *testing.T) {
	input := "cmd < input.txt > output.txt"
	list, err := ParseString(input)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(list.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(list.Elements))
	}

	cmd := list.Elements[0].Commands[0]
	if len(cmd.Redirections) != 2 {
		t.Fatalf("expected 2 redirections, got %d", len(cmd.Redirections))
	}

	if cmd.Redirections[0].RedirType != TokenRedirectIn {
		t.Errorf("expected input redirection first")
	}
	if cmd.Redirections[0].File != "input.txt" {
		t.Errorf("expected input.txt, got %s", cmd.Redirections[0].File)
	}

	if cmd.Redirections[1].RedirType != TokenRedirectOut {
		t.Errorf("expected output redirection second")
	}
	if cmd.Redirections[1].File != "output.txt" {
		t.Errorf("expected output.txt, got %s", cmd.Redirections[1].File)
	}
}

func TestParserEmptyInput(t *testing.T) {
	list, err := ParseString("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if list == nil {
		t.Error("expected non-nil list")
	}
}

func TestParserWhitespaceOnly(t *testing.T) {
	list, err := ParseString("   \t\n  ")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if list == nil {
		t.Error("expected non-nil list")
	}
}

func TestTokenString(t *testing.T) {
	tests := []struct {
		token    Token
		expected string
	}{
		{Token{Type: TokenEOF, Text: ""}, "EOF"},
		{Token{Type: TokenWord, Text: "hello"}, "hello"},
		{Token{Type: TokenPipe, Text: "|"}, "|"},
		{Token{Type: TokenError, Text: ""}, "ERROR"},
	}

	for _, tt := range tests {
		result := tt.token.String()
		if result != tt.expected && tt.token.Text == "" {
			// If Text is empty, check against type-based string
			if result != tt.expected {
				t.Errorf("token.String(): expected %q, got %q", tt.expected, result)
			}
		}
	}
}

func TestASTCommandNodeString(t *testing.T) {
	cmd := &CommandNode{
		Name: "echo",
		Args: []string{"hello", "world"},
	}

	str := cmd.String()
	if str != "echo hello world" {
		t.Errorf("expected 'echo hello world', got '%s'", str)
	}
}

func TestASTPipelineNodeString(t *testing.T) {
	pipeline := &PipelineNode{
		Commands: []*CommandNode{
			{Name: "cat", Args: []string{"file.txt"}},
			{Name: "grep", Args: []string{"pattern"}},
		},
	}

	str := pipeline.String()
	expected := "cat file.txt | grep pattern"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}

func TestASTListNodeString(t *testing.T) {
	list := &ListNode{
		Elements: []*PipelineNode{
			{Commands: []*CommandNode{{Name: "cmd1"}}},
			{Commands: []*CommandNode{{Name: "cmd2"}}},
		},
		Sep: []TokenType{TokenSemicolon},
	}

	str := list.String()
	expected := "cmd1; cmd2"
	if str != expected {
		t.Errorf("expected '%s', got '%s'", expected, str)
	}
}
