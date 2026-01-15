/*
Package parser provides parsing functionality for the WebOS shell.

This package implements a recursive descent parser for shell command language,
supporting:
  - Simple commands
  - Pipelines
  - Lists (sequences of pipelines)
  - Control operators (&&, ||, ;, &)
  - Redirections
  - Quoted and unquoted strings
*/
package parser

import "fmt"

// NodeType represents the type of a syntax tree node.
type NodeType int

const (
	// Node types
	NodeList NodeType = iota
	NodePipeline
	NodeCommand
	NodeRedirect
	NodeWord
)

// Node represents a node in the abstract syntax tree.
type Node interface {
	Type() NodeType
	String() string
}

// ListNode represents a list of pipelines separated by operators.
type ListNode struct {
	Elements []*PipelineNode // Pipeline elements
	Sep      []TokenType     // Separators between elements
}

// tokenTypeString returns a string representation of a TokenType.
func tokenTypeString(t TokenType) string {
	switch t {
	case TokenSemicolon:
		return "; "
	case TokenBackground:
		return "& "
	case TokenAnd:
		return "&& "
	case TokenOr:
		return "|| "
	default:
		return " "
	}
}

// Type returns the node type.
func (n *ListNode) Type() NodeType { return NodeList }

// String returns a string representation of the list.
func (n *ListNode) String() string {
	result := ""
	for i, p := range n.Elements {
		if i > 0 && i <= len(n.Sep) {
			result += tokenTypeString(n.Sep[i-1])
		}
		result += p.String()
	}
	return result
}

// PipelineNode represents a pipeline of commands.
type PipelineNode struct {
	Commands []*CommandNode // Commands in the pipeline
	Inverted bool           // ! operator (negate exit status)
	And      bool           // && operator (short-circuit AND)
	Or       bool           // || operator (short-circuit OR)
}

// Type returns the node type.
func (n *PipelineNode) Type() NodeType { return NodePipeline }

// String returns a string representation of the pipeline.
func (n *PipelineNode) String() string {
	result := ""
	if n.Inverted {
		result += "! "
	}
	for i, cmd := range n.Commands {
		if i > 0 {
			result += " | "
		}
		result += cmd.String()
	}
	return result
}

// CommandNode represents a single command.
type CommandNode struct {
	Name         string          // Command name
	Args         []string        // Command arguments
	Redirections []*RedirectNode // I/O redirections
	Env          []string        // Environment variables
	Subshell     bool            // Whether this is a subshell
}

// Type returns the node type.
func (n *CommandNode) Type() NodeType { return NodeCommand }

// String returns a string representation of the command.
func (n *CommandNode) String() string {
	result := n.Name
	for _, arg := range n.Args {
		result += " " + arg
	}
	for _, r := range n.Redirections {
		result += " " + r.String()
	}
	return result
}

// RedirectType represents the type of redirection.
type RedirectType int

const (
	// Redirect types
	RedirInput RedirectType = iota
	RedirOutput
	RedirAppend
	RedirErr
	RedirErrOut
)

// Redirect represents an I/O redirection.
type Redirect struct {
	Type RedirectType
	File string
}

// RedirectNode represents an I/O redirection.
type RedirectNode struct {
	From      int       // Source file descriptor
	To        int       // Destination file descriptor
	File      string    // Target file (if any)
	RedirType TokenType // Redirect type
}

// Type returns the node type.
func (n *RedirectNode) Type() NodeType { return NodeRedirect }

// String returns a string representation of the redirect.
func (n *RedirectNode) String() string {
	switch n.RedirType {
	case TokenRedirectIn:
		return fmt.Sprintf("< %s", n.File)
	case TokenRedirectOut:
		return fmt.Sprintf("> %s", n.File)
	case TokenAppend:
		return fmt.Sprintf(">> %s", n.File)
	default:
		return "?"
	}
}

// WordNode represents a word in the command.
type WordNode struct {
	Value string
}

// Type returns the node type.
func (n *WordNode) Type() NodeType { return NodeWord }

// String returns a string representation of the word.
func (n *WordNode) String() string {
	return n.Value
}
