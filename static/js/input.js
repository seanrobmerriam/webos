/**
 * InputManager - Keyboard/mouse event capture and forwarding to server
 * 
 * @module input
 */

class InputManager {
    /**
     * Create a new InputManager instance
     * @param {Object} [options] - Input options
     */
    constructor(options = {}) {
        this.options = {
            throttleKeyboard: options.throttleKeyboard || 16,
            throttleMouse: options.throttleMouse || 16,
            captureAll: options.captureAll !== false,
            ...options
        };
        
        // Keyboard state
        this.keyboardState = {};
        this._keysPressed = new Set();
        
        // Mouse state
        this.mouseState = {
            x: 0,
            y: 0,
            buttons: {},
            wheelDelta: 0
        };
        this._lastMouseX = 0;
        this._lastMouseY = 0;
        
        // Throttling
        this._lastKeyboardSend = 0;
        this._lastMouseSend = 0;
        
        // Callbacks
        this.onInput = null;
        this._targetElement = null;
        
        // Bind methods
        this._handleKeyDown = this._handleKeyDown.bind(this);
        this._handleKeyUp = this._handleKeyUp.bind(this);
        this._handleMouseMove = this._handleMouseMove.bind(this);
        this._handleMouseDown = this._handleMouseDown.bind(this);
        this._handleMouseUp = this._handleMouseUp.bind(this);
        this._handleWheel = this._handleWheel.bind(this);
        this._handleFocus = this._handleFocus.bind(this);
    }
    
    /**
     * Initialize input capture
     * @param {HTMLElement} [element] - Target element (defaults to document)
     */
    init(element = null) {
        this._targetElement = element || document;
        
        // Keyboard events
        this._targetElement.addEventListener('keydown', this._handleKeyDown);
        this._targetElement.addEventListener('keyup', this._handleKeyUp);
        this._targetElement.addEventListener('keypress', () => {});
        
        // Mouse events
        this._targetElement.addEventListener('mousemove', this._handleMouseMove);
        this._targetElement.addEventListener('mousedown', this._handleMouseDown);
        this._targetElement.addEventListener('mouseup', this._handleMouseUp);
        this._targetElement.addEventListener('wheel', this._handleWheel, { passive: true });
        this._targetElement.addEventListener('contextmenu', (e) => e.preventDefault());
        
        // Focus handling
        document.addEventListener('focus', this._handleFocus, true);
    }
    
    /**
     * Cleanup input capture
     */
    destroy() {
        if (this._targetElement) {
            this._targetElement.removeEventListener('keydown', this._handleKeyDown);
            this._targetElement.removeEventListener('keyup', this._handleKeyUp);
            this._targetElement.removeEventListener('keypress', () => {});
            this._targetElement.removeEventListener('mousemove', this._handleMouseMove);
            this._targetElement.removeEventListener('mousedown', this._handleMouseDown);
            this._targetElement.removeEventListener('mouseup', this._handleMouseUp);
            this._targetElement.removeEventListener('wheel', this._handleWheel);
            this._targetElement.removeEventListener('contextmenu', (e) => e.preventDefault());
        }
        document.removeEventListener('focus', this._handleFocus, true);
    }
    
    /**
     * Handle key down event
     * @param {KeyboardEvent} event
     */
    _handleKeyDown(event) {
        if (!this.options.captureAll && !event.target.matches('input, textarea')) {
            return;
        }
        
        event.preventDefault();
        
        const key = this._normalizeKey(event);
        this._keysPressed.add(key);
        this.keyboardState[key] = true;
        
        this._sendInput('keyboard', {
            type: 'keydown',
            key: key,
            code: event.code,
            keyCode: event.keyCode,
            repeat: event.repeat,
            modifiers: this._getModifiers(event)
        });
    }
    
    /**
     * Handle key up event
     * @param {KeyboardEvent} event
     */
    _handleKeyUp(event) {
        if (!this.options.captureAll && !event.target.matches('input, textarea')) {
            return;
        }
        
        event.preventDefault();
        
        const key = this._normalizeKey(event);
        this._keysPressed.delete(key);
        this.keyboardState[key] = false;
        
        this._sendInput('keyboard', {
            type: 'keyup',
            key: key,
            code: event.code,
            keyCode: event.keyCode,
            modifiers: this._getModifiers(event)
        });
    }
    
    /**
     * Handle mouse move event
     * @param {MouseEvent} event
     */
    _handleMouseMove(event) {
        const now = Date.now();
        if (now - this._lastMouseSend < this.options.throttleMouse) {
            return;
        }
        
        const rect = this._targetElement.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;
        
        this._lastMouseX = x;
        this._lastMouseY = y;
        this.mouseState.x = x;
        this.mouseState.y = y;
        
        this._sendInput('mouse', {
            type: 'mousemove',
            x: x,
            y: y,
            buttons: this._getMouseButtons(event)
        });
        
        this._lastMouseSend = now;
    }
    
    /**
     * Handle mouse down event
     * @param {MouseEvent} event
     */
    _handleMouseDown(event) {
        event.preventDefault();
        
        const rect = this._targetElement.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;
        
        this.mouseState.buttons[event.button] = true;
        
        this._sendInput('mouse', {
            type: 'mousedown',
            x: x,
            y: y,
            button: event.button,
            buttons: this._getMouseButtons(event)
        });
    }
    
    /**
     * Handle mouse up event
     * @param {MouseEvent} event
     */
    _handleMouseUp(event) {
        event.preventDefault();
        
        const rect = this._targetElement.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;
        
        this.mouseState.buttons[event.button] = false;
        
        this._sendInput('mouse', {
            type: 'mouseup',
            x: x,
            y: y,
            button: event.button,
            buttons: this._getMouseButtons(event)
        });
    }
    
    /**
     * Handle mouse wheel event
     * @param {WheelEvent} event
     */
    _handleWheel(event) {
        event.preventDefault();
        
        const rect = this._targetElement.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;
        
        this._sendInput('mouse', {
            type: 'wheel',
            x: x,
            y: y,
            deltaX: event.deltaX,
            deltaY: event.deltaY,
            deltaMode: event.deltaMode
        });
    }
    
    /**
     * Handle focus change
     * @param {FocusEvent} event
     */
    _handleFocus(event) {
        if (event.type === 'focus') {
            // Clear keyboard state on focus change
            this._keysPressed.clear();
            this.keyboardState = {};
        }
    }
    
    /**
     * Normalize key name
     * @param {KeyboardEvent} event
     * @returns {string} Normalized key name
     */
    _normalizeKey(event) {
        if (event.key) {
            return event.key.length === 1 ? event.key.toLowerCase() : event.key;
        }
        return String.fromCharCode(event.keyCode);
    }
    
    /**
     * Get active modifiers
     * @param {KeyboardEvent} event
     * @returns {Object} Modifier states
     */
    _getModifiers(event) {
        return {
            shift: event.shiftKey,
            ctrl: event.ctrlKey,
            alt: event.altKey,
            meta: event.metaKey
        };
    }
    
    /**
     * Get mouse button states
     * @param {MouseEvent} event
     * @returns {Object} Button states
     */
    _getMouseButtons(event) {
        return {
            left: (event.buttons & 1) !== 0,
            middle: (event.buttons & 4) !== 0,
            right: (event.buttons & 2) !== 0
        };
    }
    
    /**
     * Send input to callback
     * @param {string} type - Input type (keyboard, mouse)
     * @param {Object} data - Input data
     */
    _sendInput(type, data) {
        if (this.onInput) {
            this.onInput(type, data);
        }
    }
    
    /**
     * Get current keyboard state
     * @returns {Object} Key states
     */
    getKeyboardState() {
        return { ...this.keyboardState };
    }
    
    /**
     * Get current mouse state
     * @returns {Object} Mouse state
     */
    getMouseState() {
        return { ...this.mouseState };
    }
    
    /**
     * Check if a key is pressed
     * @param {string} key - Key to check
     * @returns {boolean} Key pressed state
     */
    isKeyPressed(key) {
        return this._keysPressed.has(key);
    }
    
    /**
     * Check if a mouse button is pressed
     * @param {number} button - Button to check (0=left, 1=middle, 2=right)
     * @returns {boolean} Button pressed state
     */
    isMouseButtonPressed(button) {
        return !!this.mouseState.buttons[button];
    }
    
    /**
     * Register an input handler
     * @param {Function} callback - Input handler
     * @returns {Function} Unsubscribe function
     */
    onInput(callback) {
        this.onInput = callback;
        return () => { this.onInput = null; };
    }
}

// Export for different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = InputManager;
}
if (typeof window !== 'undefined') {
    window.InputManager = InputManager;
}
