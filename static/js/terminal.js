/**
 * Terminal.js - Client-side terminal renderer for WebOS
 * 
 * This module provides terminal emulation rendering on HTML5 Canvas
 * with support for ANSI escape sequences, colors, and cursor positioning.
 */

class TerminalRenderer {
    /**
     * Creates a new terminal renderer
     * @param {HTMLCanvasElement} canvas - The canvas element to render on
     * @param {Object} options - Configuration options
     */
    constructor(canvas, options = {}) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        
        // Configuration
        this.options = {
            fontFamily: options.fontFamily || 'Menlo, Monaco, "Courier New", monospace',
            fontSize: options.fontSize || 14,
            lineHeight: options.lineHeight || 1.2,
            cursorBlink: options.cursorBlink !== false,
            cursorStyle: options.cursorStyle || 'block',
            scrollbackSize: options.scrollbackSize || 1000,
            ...options
        };
        
        // Cell dimensions
        this.cellWidth = this.measureCellWidth();
        this.cellHeight = Math.floor(this.options.fontSize * this.options.lineHeight);
        
        // Terminal state
        this.rows = Math.floor(canvas.height / this.cellHeight) || 24;
        this.cols = Math.floor(canvas.width / this.cellWidth) || 80;
        
        // Screen buffer
        this.screen = this.createBuffer(this.cols, this.rows);
        this.scrollback = [];
        
        // Cursor state
        this.cursor = {
            x: 0,
            y: 0,
            visible: true,
            blinkState: true
        };
        
        // Current attributes
        this.currentAttrs = this.defaultAttributes();
        
        // Color palette (xterm-256 colors + truecolor)
        this.colors = this.createColorPalette();
        
        // Mouse state
        this.mouse = {
            x: 0,
            y: 0,
            buttons: 0,
            mode: MouseMode.NONE
        };
        
        // Tab stops
        this.tabStops = [];
        this.initTabStops();
        
        // Event handlers
        this.setupEventListeners();
        
        // Animation frame for cursor blink
        this.blinkTimer = null;
        this.lastBlinkTime = 0;
        
        // Start rendering
        this.startBlink();
    }
    
    /**
     * Measure the width of a cell character
     * @returns {number} Cell width in pixels
     */
    measureCellWidth() {
        const testChar = 'W';
        const metrics = this.ctx.measureText(testChar);
        return Math.ceil(metrics.width);
    }
    
    /**
     * Creates the color palette
     * @returns {Object} Color palette
     */
    createColorPalette() {
        // Standard 8 colors
        const standard = [
            '#000000', // Black
            '#cd3131', // Red
            '#0dbc2e', // Green
            '#e5e510', // Yellow
            '#2472c8', // Blue
            '#bc3fbc', // Magenta
            '#11a8cd', // Cyan
            '#e5e5e5'  // White
        ];
        
        // Bright 8 colors
        const bright = [
            '#666666', // Bright Black (Gray)
            '#f14c4c', // Bright Red
            '#23d18b', // Bright Green
            '#f5f543', // Bright Yellow
            '#3b8eea', // Bright Blue
            '#d670d6', // Bright Magenta
            '#29b7da', // Bright Cyan
            '#ffffff'  // Bright White
        ];
        
        // 256 color palette (simplified)
        const palette256 = [];
        for (let i = 0; i < 256; i++) {
            if (i < 16) {
                palette256[i] = i < 8 ? standard[i] : bright[i - 8];
            } else if (i < 232) {
                // 6x6x6 cube
                const cube = i - 16;
                const r = Math.floor(cube / 36) * 51;
                const g = Math.floor((cube % 36) / 6) * 51;
                const b = (cube % 6) * 51;
                palette256[i] = `rgb(${r},${g},${b})`;
            } else {
                // Grayscale
                const gray = (i - 232) * 10 + 8;
                palette256[i] = `rgb(${gray},${gray},${gray})`;
            }
        }
        
        return { standard, bright, palette256 };
    }
    
    /**
     * Gets the default cell attributes
     * @returns {Object} Default attributes
     */
    defaultAttributes() {
        return {
            bold: false,
            faint: false,
            italic: false,
            underline: false,
            blink: false,
            reverse: false,
            conceal: false,
            crossedOut: false,
            foreground: { type: 'default' },
            background: { type: 'default' }
        };
    }
    
    /**
     * Creates a screen buffer
     * @param {number} cols - Number of columns
     * @param {number} rows - Number of rows
     * @returns {Array} Screen buffer
     */
    createBuffer(cols, rows) {
        const buffer = [];
        for (let y = 0; y < rows; y++) {
            buffer[y] = [];
            for (let x = 0; x < cols; x++) {
                buffer[y][x] = {
                    char: ' ',
                    attrs: this.defaultAttributes()
                };
            }
        }
        return buffer;
    }
    
    /**
     * Initialize tab stops
     */
    initTabStops() {
        this.tabStops = [];
        for (let i = 8; i < this.cols; i += 8) {
            this.tabStops.push(i);
        }
    }
    
    /**
     * Setup event listeners
     */
    setupEventListeners() {
        this.canvas.addEventListener('keydown', (e) => this.handleKeyDown(e));
        this.canvas.addEventListener('keypress', (e) => this.handleKeyPress(e));
        this.canvas.addEventListener('click', (e) => this.handleClick(e));
        this.canvas.addEventListener('mousemove', (e) => this.handleMouseMove(e));
        this.canvas.addEventListener('mousedown', (e) => this.handleMouseDown(e));
        this.canvas.addEventListener('mouseup', (e) => this.handleMouseUp(e));
        this.canvas.addEventListener('contextmenu', (e) => e.preventDefault());
    }
    
    /**
     * Handle key down events
     * @param {KeyboardEvent} e - Keyboard event
     */
    handleKeyDown(e) {
        let handled = false;
        
        // Handle special keys
        switch (e.key) {
            case 'Enter':
                this.emit('\r');
                handled = true;
                break;
            case 'Backspace':
                this.emit('\x7f');
                handled = true;
                break;
            case 'Tab':
                this.emit('\t');
                handled = true;
                break;
            case 'Escape':
                this.emit('\x1b');
                handled = true;
                break;
            case 'ArrowUp':
                this.emit('\x1b[A');
                handled = true;
                break;
            case 'ArrowDown':
                this.emit('\x1b[B');
                handled = true;
                break;
            case 'ArrowRight':
                this.emit('\x1b[C');
                handled = true;
                break;
            case 'ArrowLeft':
                this.emit('\x1b[D');
                handled = true;
                break;
            case 'Home':
                this.emit('\x1b[H');
                handled = true;
                break;
            case 'End':
                this.emit('\x1b[F');
                handled = true;
                break;
            case 'PageUp':
                this.emit('\x1b[5~');
                handled = true;
                break;
            case 'PageDown':
                this.emit('\x1b[6~');
                handled = true;
                break;
            case 'F1':
                this.emit('\x1bOP');
                handled = true;
                break;
            case 'F2':
                this.emit('\x1bOQ');
                handled = true;
                break;
            case 'F3':
                this.emit('\x1bOR');
                handled = true;
                break;
            case 'F4':
                this.emit('\x1bOS');
                handled = true;
                break;
            case 'F5':
                this.emit('\x1b[15~');
                handled = true;
                break;
            case 'F6':
                this.emit('\x1b[17~');
                handled = true;
                break;
            case 'F7':
                this.emit('\x1b[18~');
                handled = true;
                break;
            case 'F8':
                this.emit('\x1b[19~');
                handled = true;
                break;
            case 'F9':
                this.emit('\x1b[20~');
                handled = true;
                break;
            case 'F10':
                this.emit('\x1b[21~');
                handled = true;
                break;
            case 'F11':
                this.emit('\x1b[23~');
                handled = true;
                break;
            case 'F12':
                this.emit('\x1b[24~');
                handled = true;
                break;
        }
        
        if (handled) {
            e.preventDefault();
            e.stopPropagation();
        }
    }
    
    /**
     * Handle key press events
     * @param {KeyboardEvent} e - Keyboard event
     */
    handleKeyPress(e) {
        if (e.charCode > 0) {
            this.emit(String.fromCharCode(e.charCode));
        }
    }
    
    /**
     * Handle click events
     * @param {MouseEvent} e - Mouse event
     */
    handleClick(e) {
        const rect = this.canvas.getBoundingClientRect();
        const x = Math.floor((e.clientX - rect.left) / this.cellWidth);
        const y = Math.floor((e.clientY - rect.top) / this.cellHeight);
        
        this.emitMouseEvent(x, y, 0, 'click');
    }
    
    /**
     * Handle mouse move events
     * @param {MouseEvent} e - Mouse event
     */
    handleMouseMove(e) {
        const rect = this.canvas.getBoundingClientRect();
        const x = Math.floor((e.clientX - rect.left) / this.cellWidth);
        const y = Math.floor((e.clientY - rect.top) / this.cellHeight);
        
        this.mouse.x = x;
        this.mouse.y = y;
        
        if (this.mouse.mode !== MouseMode.NONE) {
            this.emitMouseEvent(x, y, e.buttons, 'move');
        }
    }
    
    /**
     * Handle mouse down events
     * @param {MouseEvent} e - Mouse event
     */
    handleMouseDown(e) {
        this.mouse.buttons = e.buttons;
        
        const rect = this.canvas.getBoundingClientRect();
        const x = Math.floor((e.clientX - rect.left) / this.cellWidth);
        const y = Math.floor((e.clientY - rect.top) / this.cellHeight);
        
        this.emitMouseEvent(x, y, e.buttons, 'down');
    }
    
    /**
     * Handle mouse up events
     * @param {MouseEvent} e - Mouse event
     */
    handleMouseUp(e) {
        this.mouse.buttons = 0;
        
        const rect = this.canvas.getBoundingClientRect();
        const x = Math.floor((e.clientX - rect.left) / this.cellWidth);
        const y = Math.floor((e.clientY - rect.top) / this.cellHeight);
        
        this.emitMouseEvent(x, y, 0, 'up');
    }
    
    /**
     * Emit a mouse event
     * @param {number} x - Cell X position
     * @param {number} y - Cell Y position
     * @param {number} buttons - Button state
     * @param {string} type - Event type
     */
    emitMouseEvent(x, y, buttons, type) {
        if (this.mouse.mode === MouseMode.NONE) return;
        
        let mask = 0;
        if (buttons & 1) mask |= 0x01;      // Left button
        if (buttons & 4) mask |= 0x02;      // Middle button
        if (buttons & 2) mask |= 0x04;      // Right button
        
        if (type === 'move' && (mask & 0x20)) {
            mask &= ~0x20; // Clear motion flag
        }
        
        const encoding = this.mouse.mode === MouseMode.X10 ? '1005' : '1006';
        this.emit(`\x1b[${encoding};${mask};${x + 1};${y + 1}M`);
    }
    
    /**
     * Emit data to the server
     * @param {string} data - Data to emit
     */
    emit(data) {
        if (this.options.onData) {
            this.options.onData(data);
        }
    }
    
    /**
     * Resize the terminal
     * @param {number} cols - New number of columns
     * @param {number} rows - New number of rows
     */
    resize(cols, rows) {
        if (cols === this.cols && rows === this.rows) return;
        
        // Create new buffer
        const newScreen = this.createBuffer(cols, rows);
        
        // Copy existing content
        const copyCols = Math.min(cols, this.cols);
        const copyRows = Math.min(rows, this.rows);
        
        for (let y = 0; y < copyRows; y++) {
            for (let x = 0; x < copyCols; x++) {
                newScreen[y][x] = this.screen[y][x];
            }
        }
        
        this.screen = newScreen;
        this.cols = cols;
        this.rows = rows;
        this.cellWidth = this.measureCellWidth();
        this.initTabStops();
        
        // Resize canvas
        this.canvas.width = cols * this.cellWidth;
        this.canvas.height = rows * this.cellHeight;
        
        this.emit(`\x1b[8;${rows};${cols}t`);
    }
    
    /**
     * Process received data from the server
     * @param {string} data - Data to process
     */
    processData(data) {
        for (let i = 0; i < data.length; i++) {
            this.processByte(data.charCodeAt(i), data[i]);
        }
        this.render();
    }
    
    /**
     * Process a single byte
     * @param {number} byteCode - Byte code
     * @param {string} char - Character
     */
    processByte(byteCode, char) {
        if (byteCode === 0x1b) { // ESC
            this.processEscapeSequence();
        } else if (byteCode === 0x0d) { // CR
            this.cursor.x = 0;
        } else if (byteCode === 0x0a) { // LF
            this.cursor.y++;
            if (this.cursor.y >= this.rows) {
                this.scrollUp(1);
            }
        } else if (byteCode === 0x08) { // BS
            if (this.cursor.x > 0) this.cursor.x--;
        } else if (byteCode === 0x09) { // TAB
            this.tabForward();
        } else if (byteCode >= 0x20 || byteCode === 0xa0) {
            this.writeChar(char);
        }
    }
    
    /**
     * Process escape sequence
     */
    processEscapeSequence() {
        let seq = '';
        let i = 1;
        
        while (i < this.pendingData.length) {
            const ch = this.pendingData[i];
            if ((ch >= 'A' && ch <= 'Z') || ch === '~' || ch === '@') {
                seq += ch;
                break;
            }
            seq += ch;
            i++;
        }
        
        if (seq.startsWith('[')) {
            this.processCSI(seq.substring(1));
        } else if (seq.startsWith(']')) {
            this.processOSC(seq.substring(1));
        } else if (seq === 'M') {
            this.reverseIndex();
        }
        
        this.pendingData = this.pendingData.substring(i + 1);
    }
    
    /**
     * Process CSI (Control Sequence Introducer) sequences
     * @param {string} seq - Sequence
     */
    processCSI(seq) {
        // Parse parameters
        const match = seq.match(/^([0-9;]*)([@A-Za-z]$)/);
        if (!match) return;
        
        const params = match[1].split(';').map(s => parseInt(s, 10) || 0);
        const final = match[2];
        
        switch (final) {
            case 'A': // CUU - Cursor up
                this.cursor.y -= params[0] || 1;
                break;
            case 'B': // CUD - Cursor down
                this.cursor.y += params[0] || 1;
                break;
            case 'C': // CUF - Cursor forward
                this.cursor.x += params[0] || 1;
                break;
            case 'D': // CUB - Cursor backward
                this.cursor.x -= params[0] || 1;
                break;
            case 'H': // CUP - Cursor position
                this.cursor.x = (params[1] || 1) - 1;
                this.cursor.y = (params[0] || 1) - 1;
                break;
            case 'J': // ED - Erase display
                this.eraseDisplay(params[0]);
                break;
            case 'K': // EL - Erase line
                this.eraseLine(params[0]);
                break;
            case 'm': // SGR - Set graphics rendition
                this.setGraphicsRendition(params);
                break;
            case 'h': // SM - Set mode
                this.setMode(params[0], true);
                break;
            case 'l': // RM - Reset mode
                this.setMode(params[0], false);
                break;
            case 'r': // DECSTBM - Set scrolling region
                this.setScrollingRegion(params[0] - 1, params[1] - 1);
                break;
        }
        
        // Clamp cursor position
        this.cursor.x = Math.max(0, Math.min(this.cols - 1, this.cursor.x));
        this.cursor.y = Math.max(0, Math.min(this.rows - 1, this.cursor.y));
    }
    
    /**
     * Process OSC (Operating System Command) sequences
     * @param {string} seq - Sequence
     */
    processOSC(seq) {
        const parts = seq.split(';');
        const command = parseInt(parts[0], 10);
        const data = parts.slice(1).join(';');
        
        switch (command) {
            case 0: // Set window title
            case 1: // Set icon name
            case 2: // Set window title and icon name
                if (this.options.onTitleChange) {
                    this.options.onTitleChange(data);
                }
                break;
            case 4: // Set color palette
                this.setColorPalette(data);
                break;
        }
    }
    
    /**
     * Write a character to the screen
     * @param {string} char - Character to write
     */
    writeChar(char) {
        if (this.cursor.x >= this.cols) {
            if (this.options.autoWrap) {
                this.cursor.x = 0;
                this.cursor.y++;
                if (this.cursor.y >= this.rows) {
                    this.scrollUp(1);
                }
            } else {
                this.cursor.x = this.cols - 1;
            }
        }
        
        this.screen[this.cursor.y][this.cursor.x] = {
            char: char,
            attrs: { ...this.currentAttrs }
        };
        
        this.cursor.x++;
    }
    
    /**
     * Tab forward
     */
    tabForward() {
        while (this.cursor.x < this.cols) {
            this.cursor.x++;
            if (this.tabStops.includes(this.cursor.x)) break;
        }
    }
    
    /**
     * Reverse index (scroll down, move cursor up if at top)
     */
    reverseIndex() {
        if (this.cursor.y === 0) {
            this.scrollDown(1);
        } else {
            this.cursor.y--;
        }
    }
    
    /**
     * Scroll up (move content to scrollback)
     * @param {number} lines - Number of lines to scroll
     */
    scrollUp(lines) {
        lines = lines || 1;
        
        // Move lines to scrollback
        for (let i = 0; i < lines && i < this.rows; i++) {
            this.scrollback.push([...this.screen[i]]);
            if (this.scrollback.length > this.options.scrollbackSize) {
                this.scrollback.shift();
            }
        }
        
        // Shift screen up
        for (let y = 0; y < this.rows - lines; y++) {
            this.screen[y] = this.screen[y + lines];
        }
        
        // Clear bottom lines
        for (let y = this.rows - lines; y < this.rows; y++) {
            this.screen[y] = this.createBufferRow();
        }
    }
    
    /**
     * Scroll down
     * @param {number} lines - Number of lines to scroll
     */
    scrollDown(lines) {
        lines = lines || 1;
        
        // Shift screen down
        for (let y = this.rows - 1; y >= lines; y--) {
            this.screen[y] = this.screen[y - lines];
        }
        
        // Clear top lines
        for (let y = 0; y < lines; y++) {
            this.screen[y] = this.createBufferRow();
        }
    }
    
    /**
     * Create an empty buffer row
     * @returns {Array} Empty row
     */
    createBufferRow() {
        const row = [];
        for (let x = 0; x < this.cols; x++) {
            row[x] = {
                char: ' ',
                attrs: this.defaultAttributes()
            };
        }
        return row;
    }
    
    /**
     * Erase display
     * @param {number} mode - Erase mode
     */
    eraseDisplay(mode) {
        mode = mode || 0;
        
        if (mode === 0) {
            // Erase from cursor to end of screen
            for (let x = this.cursor.x; x < this.cols; x++) {
                this.screen[this.cursor.y][x] = { char: ' ', attrs: this.defaultAttributes() };
            }
            for (let y = this.cursor.y + 1; y < this.rows; y++) {
                this.screen[y] = this.createBufferRow();
            }
        } else if (mode === 1) {
            // Erase from beginning of screen to cursor
            for (let y = 0; y < this.cursor.y; y++) {
                this.screen[y] = this.createBufferRow();
            }
            for (let x = 0; x <= this.cursor.x; x++) {
                this.screen[this.cursor.y][x] = { char: ' ', attrs: this.defaultAttributes() };
            }
        } else if (mode === 2) {
            // Erase entire screen
            for (let y = 0; y < this.rows; y++) {
                this.screen[y] = this.createBufferRow();
            }
        }
    }
    
    /**
     * Erase line
     * @param {number} mode - Erase mode
     */
    eraseLine(mode) {
        mode = mode || 0;
        
        if (mode === 0) {
            // Erase from cursor to end of line
            for (let x = this.cursor.x; x < this.cols; x++) {
                this.screen[this.cursor.y][x] = { char: ' ', attrs: this.defaultAttributes() };
            }
        } else if (mode === 1) {
            // Erase from beginning of line to cursor
            for (let x = 0; x <= this.cursor.x; x++) {
                this.screen[this.cursor.y][x] = { char: ' ', attrs: this.defaultAttributes() };
            }
        } else if (mode === 2) {
            // Erase entire line
            this.screen[this.cursor.y] = this.createBufferRow();
        }
    }
    
    /**
     * Set graphics rendition (colors and attributes)
     * @param {Array} params - SGR parameters
     */
    setGraphicsRendition(params) {
        for (const param of params) {
            switch (param) {
                case 0:
                    this.currentAttrs = this.defaultAttributes();
                    break;
                case 1:
                    this.currentAttrs.bold = true;
                    break;
                case 3:
                    this.currentAttrs.italic = true;
                    break;
                case 4:
                    this.currentAttrs.underline = true;
                    break;
                case 5:
                    this.currentAttrs.blink = true;
                    break;
                case 7:
                    this.currentAttrs.reverse = true;
                    break;
                case 22:
                    this.currentAttrs.bold = false;
                    break;
                case 23:
                    this.currentAttrs.italic = false;
                    break;
                case 24:
                    this.currentAttrs.underline = false;
                    break;
                case 25:
                    this.currentAttrs.blink = false;
                    break;
                case 27:
                    this.currentAttrs.reverse = false;
                    break;
                case 30: // Black foreground
                case 31: // Red foreground
                case 32: // Green foreground
                case 33: // Yellow foreground
                case 34: // Blue foreground
                case 35: // Magenta foreground
                case 36: // Cyan foreground
                case 37: // White foreground
                    this.currentAttrs.foreground = { type: 'standard', value: param - 30 };
                    break;
                case 38:
                    // Extended foreground color
                    if (params[1] === 5) {
                        this.currentAttrs.foreground = { type: '256', value: params[2] };
                    }
                    break;
                case 39:
                    this.currentAttrs.foreground = { type: 'default' };
                    break;
                case 40: // Black background
                case 41: // Red background
                case 42: // Green background
                case 43: // Yellow background
                case 44: // Blue background
                case 45: // Magenta background
                case 46: // Cyan background
                case 47: // White background
                    this.currentAttrs.background = { type: 'standard', value: param - 40 };
                    break;
                case 48:
                    // Extended background color
                    if (params[1] === 5) {
                        this.currentAttrs.background = { type: '256', value: params[2] };
                    }
                    break;
                case 49:
                    this.currentAttrs.background = { type: 'default' };
                    break;
                case 90: // Bright black foreground
                case 91: // Bright red foreground
                case 92: // Bright green foreground
                case 93: // Bright yellow foreground
                case 94: // Bright blue foreground
                case 95: // Bright magenta foreground
                case 96: // Bright cyan foreground
                case 97: // Bright white foreground
                    this.currentAttrs.foreground = { type: 'bright', value: param - 90 };
                    break;
                case 100: // Bright black background
                case 101: // Bright red background
                case 102: // Bright green background
                case 103: // Bright yellow background
                case 104: // Bright blue background
                case 105: // Bright magenta background
                case 106: // Bright cyan background
                case 107: // Bright white background
                    this.currentAttrs.background = { type: 'bright', value: param - 100 };
                    break;
            }
        }
    }
    
    /**
     * Set terminal mode
     * @param {number} mode - Mode number
     * @param {boolean} enabled - Whether to enable the mode
     */
    setMode(mode, enabled) {
        switch (mode) {
            case 25: // Cursor visible
                this.cursor.visible = enabled;
                break;
            case 1049: // Alternate screen buffer
                this.options.alternateScreen = enabled;
                break;
        }
    }
    
    /**
     * Set scrolling region
     * @param {number} top - Top row (0-indexed)
     * @param {number} bottom - Bottom row (0-indexed)
     */
    setScrollingRegion(top, bottom) {
        this.scrollRegion = { top, bottom };
    }
    
    /**
     * Set color palette
     * @param {string} data - Color palette data
     */
    setColorPalette(data) {
        const parts = data.split(';');
        const index = parseInt(parts[0], 10);
        const color = parts[1];
        
        if (index !== undefined && color) {
            if (color.startsWith('rgb:')) {
                const rgb = color.substring(4).split('/').map(v => parseInt(v, 16));
                this.colors.palette256[index] = `rgb(${rgb[0]},${rgb[1]},${rgb[2]})`;
            }
        }
    }
    
    /**
     * Get foreground color
     * @param {Object} attrs - Cell attributes
     * @returns {string} Color string
     */
    getForegroundColor(attrs) {
        if (attrs.reverse) {
            return this.getBackgroundColor(attrs);
        }
        
        const fg = attrs.foreground;
        if (fg.type === 'default') return '#ffffff';
        if (fg.type === 'standard') return this.colors.standard[fg.value];
        if (fg.type === 'bright') return this.colors.bright[fg.value];
        if (fg.type === '256') return this.colors.palette256[fg.value];
        if (fg.type === 'rgb') return `rgb(${fg.r},${fg.g},${fg.b})`;
        
        return '#ffffff';
    }
    
    /**
     * Get background color
     * @param {Object} attrs - Cell attributes
     * @returns {string} Color string
     */
    getBackgroundColor(attrs) {
        if (attrs.reverse) {
            return this.getForegroundColor(attrs);
        }
        
        const bg = attrs.background;
        if (bg.type === 'default') return '#000000';
        if (bg.type === 'standard') return this.colors.standard[bg.value];
        if (bg.type === 'bright') return this.colors.bright[bg.value];
        if (bg.type === '256') return this.colors.palette256[bg.value];
        if (bg.type === 'rgb') return `rgb(${bg.r},${bg.g},${bg.b})`;
        
        return '#000000';
    }
    
    /**
     * Start cursor blinking
     */
    startBlink() {
        const blink = () => {
            this.lastBlinkTime = Date.now();
            if (this.options.cursorBlink) {
                this.cursor.blinkState = !this.cursor.blinkState;
                this.render();
            }
            this.blinkTimer = requestAnimationFrame(blink);
        };
        blink();
    }
    
    /**
     * Stop cursor blinking
     */
    stopBlink() {
        if (this.blinkTimer) {
            cancelAnimationFrame(this.blinkTimer);
            this.blinkTimer = null;
        }
    }
    
    /**
     * Render the terminal
     */
    render() {
        // Clear canvas
        this.ctx.fillStyle = this.getBackgroundColor(this.defaultAttributes());
        this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
        
        // Set font
        this.ctx.font = `${this.options.fontSize}px ${this.options.fontFamily}`;
        this.ctx.textBaseline = 'top';
        
        // Render each cell
        for (let y = 0; y < this.rows; y++) {
            for (let x = 0; x < this.cols; x++) {
                this.renderCell(x, y);
            }
        }
        
        // Render cursor
        this.renderCursor();
    }
    
    /**
     * Render a single cell
     * @param {number} x - X position
     * @param {number} y - Y position
     */
    renderCell(x, y) {
        const cell = this.screen[y][x];
        if (!cell) return;
        
        const fgColor = this.getForegroundColor(cell.attrs);
        const bgColor = this.getBackgroundColor(cell.attrs);
        
        // Set colors
        this.ctx.fillStyle = bgColor;
        this.ctx.fillRect(x * this.cellWidth, y * this.cellHeight, this.cellWidth, this.cellHeight);
        
        this.ctx.fillStyle = fgColor;
        
        // Apply bold
        if (cell.attrs.bold) {
            this.ctx.font = `bold ${this.options.fontSize}px ${this.options.fontFamily}`;
        } else {
            this.ctx.font = `${this.options.fontSize}px ${this.options.fontFamily}`;
        }
        
        // Render character
        const char = cell.char || ' ';
        this.ctx.fillText(char, x * this.cellWidth, y * this.cellHeight + 2);
    }
    
    /**
     * Render cursor
     */
    renderCursor() {
        if (!this.cursor.visible || !this.cursor.blinkState) return;
        
        const x = this.cursor.x * this.cellWidth;
        const y = this.cursor.y * this.cellHeight;
        
        // Draw cursor based on style
        if (this.options.cursorStyle === 'block') {
            const cell = this.screen[this.cursor.y][this.cursor.x];
            const fgColor = this.getForegroundColor(cell.attrs);
            const bgColor = this.getBackgroundColor(cell.attrs);
            
            // Inverted colors for cursor
            this.ctx.fillStyle = fgColor;
            this.ctx.fillRect(x, y, this.cellWidth, this.cellHeight);
            
            this.ctx.fillStyle = bgColor;
            this.ctx.fillText(cell.char || ' ', x, y + 2);
        } else if (this.options.cursorStyle === 'underline') {
            this.ctx.fillStyle = '#ffffff';
            this.ctx.fillRect(x, y + this.cellHeight - 3, this.cellWidth, 2);
        } else if (this.options.cursorStyle === 'bar') {
            this.ctx.fillStyle = '#ffffff';
            this.ctx.fillRect(x, y, 2, this.cellHeight);
        }
    }
    
    /**
     * Get scrollback content
     * @returns {Array} Scrollback lines
     */
    getScrollback() {
        return [...this.scrollback];
    }
    
    /**
     * Clear scrollback
     */
    clearScrollback() {
        this.scrollback = [];
    }
    
    /**
     * Focus the terminal
     */
    focus() {
        this.canvas.focus();
    }
    
    /**
     * Destroy the terminal renderer
     */
    destroy() {
        this.stopBlink();
        this.canvas.removeEventListener('keydown', this.handleKeyDown);
        this.canvas.removeEventListener('keypress', this.handleKeyPress);
        this.canvas.removeEventListener('click', this.handleClick);
        this.canvas.removeEventListener('mousemove', this.handleMouseMove);
        this.canvas.removeEventListener('mousedown', this.handleMouseDown);
        this.canvas.removeEventListener('mouseup', this.handleMouseUp);
    }
}

// Mouse mode constants
const MouseMode = {
    NONE: 0,
    X10: 1,
    NORMAL: 2,
    HIGHLIGHT: 3,
    BUTTON_MOTION: 4,
    ALL_MOTION: 5
};

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { TerminalRenderer, MouseMode };
}
