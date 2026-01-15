/*
Package pty provides VT100/xterm-compatible terminal emulation.

This package implements pseudo-terminal (PTY) functionality along with
terminal state management, ANSI escape sequence parsing, and screen
buffer handling for browser-based shell access.

# Features

  - PTY master/slave pair implementation
  - VT100/xterm-compatible ANSI escape sequence parsing
  - Screen buffer with configurable scrollback
  - Cursor positioning and attributes
  - Color support (16 standard colors + truecolor)
  - Keyboard input handling
  - Mouse support (xterm protocol)

# Usage

Create a new terminal with default dimensions:

	term := pty.NewTerminal(80, 24)

Write data to the terminal (from shell process):

	term.Write([]byte("hello\n"))

Read data from the terminal (for display):

	data := term.Read()

Parse ANSI escape sequences from input:

	parser := ansi.NewParser()
	parser.Parse(input, term)

# Color Support

The terminal supports:
  - 8 standard foreground/background colors
  - 8 bright foreground/background colors
  - 256 color mode (xterm-256color)
  - 24-bit truecolor (RGB)

# Mouse Protocol

The terminal supports xterm mouse tracking protocols:
  - X10 mouse reporting
  - Normal mouse tracking (1002)
  - UTF-8 mode mouse (1005)
*/
package pty
