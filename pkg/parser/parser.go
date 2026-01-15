// Package parser provides lexical analysis and parsing for the WebOS shell.
// This file implements a recursive descent parser for shell commands.
package parser

import "fmt"

// Parser represents a recursive descent parser for shell commands.
type Parser struct {
	lexer *Lexer
	pos   int
	tok   Token
}

// NewParser creates a new Parser for the given input.
func NewParser(input string) *Parser {
	return &Parser{
		lexer: NewLexer(input),
		pos:   0,
		tok:   Token{Type: TokenEOF, Pos: 0},
	}
}

// Parse parses the input and returns the AST.
func (p *Parser) Parse() (*ListNode, error) {
	p.nextToken()
	list := p.parseList()
	if list == nil {
		return nil, fmt.Errorf("parse error: failed to parse command list")
	}
	return list, nil
}

// nextToken advances to the next token.
func (p *Parser) nextToken() {
	p.tok = p.lexer.NextToken()
	p.pos++
}

// expect consumes the current token and returns it if it matches the expected type.
func (p *Parser) expect(t TokenType) (Token, error) {
	if p.tok.Type != t {
		return p.tok, fmt.Errorf("parse error: expected %s, got %s", tokenTypeToString(t), tokenTypeToString(p.tok.Type))
	}
	tok := p.tok
	p.nextToken()
	return tok, nil
}

// parseList parses a list of pipelines.
func (p *Parser) parseList() *ListNode {
	list := &ListNode{
		Elements: make([]*PipelineNode, 0),
		Sep:      make([]TokenType, 0),
	}

	for p.tok.Type != TokenEOF && p.tok.Type != TokenRightParen {
		pipeline := p.parsePipeline()
		if pipeline == nil {
			// Skip error and try to continue
			p.nextToken()
			continue
		}
		list.Elements = append(list.Elements, pipeline)

		// Check for separator
		if p.tok.Type == TokenSemicolon {
			list.Sep = append(list.Sep, TokenSemicolon)
			p.nextToken()
		} else if p.tok.Type == TokenBackground {
			list.Sep = append(list.Sep, TokenBackground)
			p.nextToken()
		} else if p.tok.Type == TokenNewline {
			p.nextToken()
		}
	}

	return list
}

// parsePipeline parses a pipeline of commands.
func (p *Parser) parsePipeline() *PipelineNode {
	pipeline := &PipelineNode{
		Commands: make([]*CommandNode, 0),
	}

	// Parse first command
	cmd := p.parseCommand()
	if cmd == nil {
		return nil
	}
	pipeline.Commands = append(pipeline.Commands, cmd)

	// Check for pipeline operator
	for p.tok.Type == TokenPipe {
		p.nextToken()
		cmd = p.parseCommand()
		if cmd == nil {
			return nil
		}
		pipeline.Commands = append(pipeline.Commands, cmd)
	}

	// Check for && or ||
	if p.tok.Type == TokenAnd {
		pipeline.And = true
		p.nextToken()
	} else if p.tok.Type == TokenOr {
		pipeline.Or = true
		p.nextToken()
	}

	return pipeline
}

// parseCommand parses a simple command with arguments and redirections.
func (p *Parser) parseCommand() *CommandNode {
	cmd := &CommandNode{
		Args:         make([]string, 0),
		Env:          make([]string, 0),
		Redirections: make([]*RedirectNode, 0),
	}

	// Check for subshell or function definition
	if p.tok.Type == TokenLeftParen {
		cmd.Subshell = true
		p.nextToken()
		list := p.parseList()
		if list == nil {
			return nil
		}
		if _, err := p.expect(TokenRightParen); err != nil {
			return nil
		}
		// For subshell, we return a special command
		cmd.Args = append(cmd.Args, list.String())
		return cmd
	}

	// Parse command name
	if p.tok.Type != TokenWord {
		// Might be a control structure
		return p.parseControlStructure()
	}

	cmd.Name = p.tok.Text
	p.nextToken()

	// Parse arguments
	for p.tok.Type == TokenWord {
		cmd.Args = append(cmd.Args, p.tok.Text)
		p.nextToken()
	}

	// Parse redirections
	for p.tok.Type == TokenRedirectIn || p.tok.Type == TokenRedirectOut ||
		p.tok.Type == TokenAppend {
		redir := p.parseRedirection()
		if redir != nil {
			cmd.Redirections = append(cmd.Redirections, redir)
		}
	}

	// Don't consume background here - let parseList handle it as separator
	// Background will be consumed by parseList

	return cmd
}

// parseControlStructure parses control structures like if, while, for, case.
func (p *Parser) parseControlStructure() *CommandNode {
	// For simplicity, return a command node with the control structure
	cmd := &CommandNode{
		Args:         make([]string, 0),
		Redirections: make([]*RedirectNode, 0),
	}

	switch p.tok.Type {
	case TokenWord:
		switch p.tok.Text {
		case "if":
			cmd.Args = append(cmd.Args, "if")
			p.nextToken()
			// Parse condition
			for p.tok.Type != TokenEOF && p.tok.Text != "then" {
				cmd.Args = append(cmd.Args, p.tok.Text)
				p.nextToken()
			}
		case "while", "until":
			cmd.Args = append(cmd.Args, p.tok.Text)
			p.nextToken()
			for p.tok.Type != TokenEOF && p.tok.Text != "do" {
				cmd.Args = append(cmd.Args, p.tok.Text)
				p.nextToken()
			}
		case "for":
			cmd.Args = append(cmd.Args, "for")
			p.nextToken()
			if p.tok.Type == TokenWord {
				cmd.Args = append(cmd.Args, p.tok.Text)
				p.nextToken()
			}
			if p.tok.Text == "in" {
				cmd.Args = append(cmd.Args, "in")
				p.nextToken()
				for p.tok.Type == TokenWord {
					cmd.Args = append(cmd.Args, p.tok.Text)
					p.nextToken()
				}
			}
		case "case":
			cmd.Args = append(cmd.Args, "case")
			p.nextToken()
			if p.tok.Type == TokenWord {
				cmd.Args = append(cmd.Args, p.tok.Text)
				p.nextToken()
			}
		}
	}

	return cmd
}

// parseRedirection parses an I/O redirection.
func (p *Parser) parseRedirection() *RedirectNode {
	redir := &RedirectNode{
		RedirType: TokenRedirectIn, // Initialize with default
	}

	switch p.tok.Type {
	case TokenRedirectIn:
		redir.RedirType = TokenRedirectIn
		p.nextToken()
	case TokenRedirectOut:
		redir.RedirType = TokenRedirectOut
		p.nextToken()
	case TokenAppend:
		redir.RedirType = TokenAppend
		p.nextToken()
	default:
		return nil
	}

	// Get the file path
	if p.tok.Type == TokenWord {
		redir.File = p.tok.Text
		p.nextToken()
	}

	return redir
}

// ParseString is a convenience function to parse a command string.
func ParseString(input string) (*ListNode, error) {
	return NewParser(input).Parse()
}
