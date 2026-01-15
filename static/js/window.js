/**
 * Window - Window class for the window manager
 * 
 * Represents a single window with properties for position, size, state,
 * and methods for manipulation.
 */

// Window state constants
const WindowState = {
    NORMAL: 'normal',
    MINIMIZED: 'minimized',
    MAXIMIZED: 'maximized',
    FULLSCREEN: 'fullscreen'
};

// Window type constants
const WindowType = {
    REGULAR: 'regular',
    DIALOG: 'dialog',
    UTILITY: 'utility',
    TOOLTIP: 'tooltip'
};

/**
 * Window class
 */
class Window {
    /**
     * Creates a new window
     * @param {Object} config - Window configuration
     */
    constructor(config = {}) {
        // Generate unique ID if not provided
        this._id = config.id || Window._generateId();

        // Core properties
        this._title = config.title || 'Untitled';
        this._type = config.type || WindowType.REGULAR;

        // Position and size
        this._x = config.x || 100;
        this._y = config.y || 100;
        this._width = config.width || 640;
        this._height = config.height || 480;

        // Normal frame (for restore after maximize)
        this._normalFrame = {
            x: this._x,
            y: this._y,
            width: this._width,
            height: this._height
        };

        // State
        this._state = WindowState.NORMAL;
        this._minimized = false;
        this._maximized = false;
        this._fullscreen = false;
        this._visible = true;

        // Desktop
        this._desktop = config.desktop || 0;

        // Content
        this._content = config.content || null;
        this._contentURL = config.contentURL || null;

        // Flags
        this._flags = {
            resizable: config.resizable !== false,
            movable: config.movable !== false,
            closable: config.closable !== false,
            minimizable: config.minimizable !== false,
            maximizable: config.maximizable !== false,
            hasTitleBar: config.hasTitleBar !== false,
            hasBorder: config.hasBorder !== false,
            alwaysOnTop: config.alwaysOnTop || false
        };

        // Decoration settings
        this._decorations = {
            titleBarHeight: 32,
            borderWidth: 1,
            buttonSize: 24,
            titleBarColor: '#2d3748',
            titleBarTextColor: '#ffffff'
        };

        // Parent/child relationships
        this._parent = config.parent || null;
        this._children = [];

        // Event handlers
        this._eventHandlers = {};

        // Z-index
        this._zIndex = 0;

        // Element reference (DOM)
        this._element = null;

        // Minimum size
        this._minWidth = config.minWidth || 100;
        this._minHeight = config.minHeight || 60;
    }

    /**
     * Generates a unique window ID
     * @returns {string} Window ID
     */
    static _generateId() {
        return 'win_' + Date.now().toString(36) + '_' + Math.random().toString(36).substr(2, 9);
    }

    // Properties with getters/setters

    get id() {
        return this._id;
    }

    get title() {
        return this._title;
    }

    set title(value) {
        this._title = value;
        this._emit('titleChanged', value);
    }

    get type() {
        return this._type;
    }

    get x() {
        return this._x;
    }

    set x(value) {
        this._x = value;
        this._emit('positionChanged', { x: value, y: this._y });
    }

    get y() {
        return this._y;
    }

    set y(value) {
        this._y = value;
        this._emit('positionChanged', { x: this._x, y: value });
    }

    get width() {
        return this._width;
    }

    set width(value) {
        this._width = Math.max(this._minWidth, value);
        this._emit('sizeChanged', { width: this._width, height: this._height });
    }

    get height() {
        return this._height;
    }

    set height(value) {
        this._height = Math.max(this._minHeight, value);
        this._emit('sizeChanged', { width: this._width, height: this._height });
    }

    get frame() {
        return {
            x: this._x,
            y: this._y,
            width: this._width,
            height: this._height
        };
    }

    set frame(value) {
        if (value.x !== undefined) this._x = value.x;
        if (value.y !== undefined) this._y = value.y;
        if (value.width !== undefined) this._width = Math.max(this._minWidth, value.width);
        if (value.height !== undefined) this._height = Math.max(this._minHeight, value.height);
        this._emit('frameChanged', this.frame);
    }

    get normalFrame() {
        return { ...this._normalFrame };
    }

    set normalFrame(value) {
        this._normalFrame = {
            x: value.x || this._x,
            y: value.y || this._y,
            width: value.width || this._width,
            height: value.height || this._height
        };
    }

    get state() {
        return this._state;
    }

    set state(value) {
        this._state = value;
        this._emit('stateChanged', value);
    }

    get minimized() {
        return this._minimized;
    }

    set minimized(value) {
        this._minimized = value;
        if (value) {
            this._state = WindowState.MINIMIZED;
            this._visible = false;
        } else {
            this._state = WindowState.NORMAL;
            this._visible = true;
        }
        this._emit('minimizedChanged', value);
    }

    get maximized() {
        return this._maximized;
    }

    set maximized(value) {
        this._maximized = value;
        if (value) {
            this._state = WindowState.MAXIMIZED;
        } else {
            this._state = WindowState.NORMAL;
        }
        this._emit('maximizedChanged', value);
    }

    get fullscreen() {
        return this._fullscreen;
    }

    set fullscreen(value) {
        this._fullscreen = value;
        if (value) {
            this._state = WindowState.FULLSCREEN;
        } else {
            this._state = WindowState.NORMAL;
        }
        this._emit('fullscreenChanged', value);
    }

    get visible() {
        return this._visible;
    }

    set visible(value) {
        this._visible = value;
        this._emit('visibleChanged', value);
    }

    get desktop() {
        return this._desktop;
    }

    set desktop(value) {
        this._desktop = value;
        this._emit('desktopChanged', value);
    }

    get content() {
        return this._content;
    }

    set content(value) {
        this._content = value;
        this._emit('contentChanged', value);
    }

    get contentURL() {
        return this._contentURL;
    }

    set contentURL(value) {
        this._contentURL = value;
        this._emit('contentURLChanged', value);
    }

    get zIndex() {
        return this._zIndex;
    }

    set zIndex(value) {
        this._zIndex = value;
        this._emit('zIndexChanged', value);
    }

    get parent() {
        return this._parent;
    }

    set parent(value) {
        this._parent = value;
    }

    get element() {
        return this._element;
    }

    set element(value) {
        this._element = value;
    }

    // Flag getters/setters

    get resizable() {
        return this._flags.resizable;
    }

    get movable() {
        return this._flags.movable;
    }

    get closable() {
        return this._flags.closable;
    }

    get minimizable() {
        return this._flags.minimizable;
    }

    get maximizable() {
        return this._flags.maximizable;
    }

    get hasTitleBar() {
        return this._flags.hasTitleBar;
    }

    get hasBorder() {
        return this._flags.hasBorder;
    }

    get alwaysOnTop() {
        return this._flags.alwaysOnTop;
    }

    // Decoration properties

    get titleBarHeight() {
        return this._decorations.titleBarHeight;
    }

    get contentWidth() {
        const borderWidth = this._decorations.borderWidth;
        return this._width - (borderWidth * 2);
    }

    get contentHeight() {
        const titleBarHeight = this._flags.hasTitleBar ? this._decorations.titleBarHeight : 0;
        const borderWidth = this._decorations.borderWidth;
        return this._height - titleBarHeight - (borderWidth * 2);
    }

    /**
     * Sets the position and size at once
     * @param {number} x - X position
     * @param {number} y - Y position
     * @param {number} width - Width
     * @param {number} height - Height
     */
    setBounds(x, y, width, height) {
        this._x = x;
        this._y = y;
        this._width = Math.max(this._minWidth, width);
        this._height = Math.max(this._minHeight, height);
        this._emit('frameChanged', this.frame);
    }

    /**
     * Moves the window
     * @param {number} x - New X position
     * @param {number} y - New Y position
     */
    move(x, y) {
        if (!this._flags.movable) return;
        this.x = x;
        this.y = y;
    }

    /**
     * Resizes the window
     * @param {number} width - New width
     * @param {number} height - New height
     */
    resize(width, height) {
        if (!this._flags.resizable) return;
        this.width = width;
        this.height = height;
    }

    /**
     * Ensures window has valid bounds
     */
    ensureBounds() {
        this._x = Math.max(0, this._x);
        this._y = Math.max(0, this._y);
        this._width = Math.max(this._minWidth, this._width);
        this._height = Math.max(this._minHeight, this._height);
    }

    /**
     * Checks if a point is within the window
     * @param {number} x - X coordinate
     * @param {number} y - Y coordinate
     * @returns {boolean} True if point is within
     */
    containsPoint(x, y) {
        return x >= this._x && x <= this._x + this._width &&
               y >= this._y && y <= this._y + this._height;
    }

    /**
     * Checks if a point is in the title bar
     * @param {number} x - X coordinate
     * @param {number} y - Y coordinate
     * @returns {boolean} True if in title bar
     */
    isInTitleBar(x, y) {
        if (!this._flags.hasTitleBar) return false;
        return x >= this._x && x <= this._x + this._width &&
               y >= this._y && y <= this._y + this._decorations.titleBarHeight;
    }

    /**
     * Checks if a point is in the resize handle
     * @param {number} x - X coordinate
     * @param {number} y - Y coordinate
     * @param {number} handleSize - Handle size (default 8)
     * @returns {string|null} Resize direction or null
     */
    getResizeHandle(x, y, handleSize = 8) {
        if (!this._flags.resizable) return null;

        const rightEdge = this._x + this._width;
        const bottomEdge = this._y + this._height;
        const leftEdge = this._x;
        const topEdge = this._y;

        // Check corners first
        if (x >= leftEdge && x <= leftEdge + handleSize && y >= topEdge && y <= topEdge + handleSize) {
            return 'nw';
        }
        if (x >= rightEdge - handleSize && x <= rightEdge && y >= topEdge && y <= topEdge + handleSize) {
            return 'ne';
        }
        if (x >= leftEdge && x <= leftEdge + handleSize && y >= bottomEdge - handleSize && y <= bottomEdge) {
            return 'sw';
        }
        if (x >= rightEdge - handleSize && x <= rightEdge && y >= bottomEdge - handleSize && y <= bottomEdge) {
            return 'se';
        }

        // Check edges
        if (x >= leftEdge - handleSize && x <= leftEdge) return 'w';
        if (x >= rightEdge && x <= rightEdge + handleSize) return 'e';
        if (y >= topEdge - handleSize && y <= topEdge) return 'n';
        if (y >= bottomEdge && y <= bottomEdge + handleSize) return 's';

        return null;
    }

    /**
     * Shows the window
     */
    show() {
        this.visible = true;
        if (this._minimized) {
            this.minimized = false;
        }
    }

    /**
     * Hides the window
     */
    hide() {
        this.visible = false;
    }

    /**
     * Minimizes the window
     */
    minimize() {
        if (!this._flags.minimizable) return;
        this.minimized = true;
    }

    /**
     * Maximizes the window
     */
    maximize() {
        if (!this._flags.maximizable) return;
        // Save normal frame before maximizing
        this._normalFrame = {
            x: this._x,
            y: this._y,
            width: this._width,
            height: this._height
        };
        this.maximized = true;
    }

    /**
     * Restores the window from minimized/maximized state
     */
    restore() {
        if (this._maximized) {
            this.maximized = false;
            this.frame = this._normalFrame;
        } else if (this._minimized) {
            this.minimized = false;
        } else if (this._fullscreen) {
            this.fullscreen = false;
            this.frame = this._normalFrame;
        }
    }

    /**
     * Toggles maximized state
     */
    toggleMaximized() {
        if (this._maximized) {
            this.restore();
        } else {
            this.maximize();
        }
    }

    /**
     * Closes the window
     */
    close() {
        this._emit('closing');
        this._emit('closed');
    }

    /**
     * Focuses the window
     */
    focus() {
        this._emit('focused');
    }

    /**
     * Blurs the window (removes focus)
     */
    blur() {
        this._emit('blurred');
    }

    /**
     * Adds an event listener
     * @param {string} event - Event name
     * @param {Function} handler - Handler function
     */
    on(event, handler) {
        if (!this._eventHandlers[event]) {
            this._eventHandlers[event] = [];
        }
        this._eventHandlers[event].push(handler);
    }

    /**
     * Removes an event listener
     * @param {string} event - Event name
     * @param {Function} handler - Handler function
     */
    off(event, handler) {
        if (!this._eventHandlers[event]) return;
        const index = this._eventHandlers[event].indexOf(handler);
        if (index > -1) {
            this._eventHandlers[event].splice(index, 1);
        }
    }

    /**
     * Emits an event
     * @param {string} event - Event name
     * @param {*} data - Event data
     */
    _emit(event, data) {
        if (this._eventHandlers[event]) {
            this._eventHandlers[event].forEach(handler => handler(data, this));
        }
    }

    /**
     * Serializes the window state
     * @returns {Object} Serialized state
     */
    serialize() {
        return {
            id: this._id,
            title: this._title,
            type: this._type,
            frame: this.frame,
            normalFrame: this._normalFrame,
            state: this._state,
            minimized: this._minimized,
            maximized: this._maximized,
            fullscreen: this._fullscreen,
            visible: this._visible,
            desktop: this._desktop,
            contentURL: this._contentURL,
            flags: this._flags,
            zIndex: this._zIndex
        };
    }

    /**
     * Creates a window from serialized state
     * @param {Object} data - Serialized state
     * @returns {Window} New window instance
     */
    static deserialize(data) {
        const win = new Window({
            id: data.id,
            title: data.title,
            type: data.type,
            desktop: data.desktop,
            contentURL: data.contentURL
        });

        win._frame = data.frame;
        win._normalFrame = data.normalFrame || data.frame;
        win._state = data.state || WindowState.NORMAL;
        win._minimized = data.minimized || false;
        win._maximized = data.maximized || false;
        win._fullscreen = data.fullscreen || false;
        win._visible = data.visible !== false;
        win._zIndex = data.zIndex || 0;

        if (data.flags) {
            win._flags = { ...win._flags, ...data.flags };
        }

        return win;
    }
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { Window, WindowState, WindowType };
}
