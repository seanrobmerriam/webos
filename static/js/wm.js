/**
 * WindowManager - Complete window management system
 * 
 * A comprehensive window manager for browser-based GUI with support for:
 * - Multiple windows with floating, tiling, and grid layouts
 * - Virtual desktops
 * - Window decorations (title bar, borders, buttons)
 * - Drag and resize operations
 * - Minimize/maximize/restore
 * - Window snapping
 * - Keyboard shortcuts
 * - Task switcher (Alt+Tab)
 */

class WindowManager {
    /**
     * Creates a new window manager
     * @param {Object} config - Configuration
     */
    constructor(config = {}) {
        // Canvas and context for rendering
        this._canvas = config.canvas || null;
        this._ctx = config.ctx || null;

        // Screen dimensions
        this._screenWidth = config.screenWidth || window.innerWidth;
        this._screenHeight = config.screenHeight || window.innerHeight;

        // Windows
        this._windows = new Map();
        this._windowOrder = []; // For z-order
        this._focusedWindow = null;

        // Desktop management
        this._desktopManager = new DesktopManager({
            maxDesktops: config.maxDesktops || 4,
            initialCount: config.initialDesktops || 4,
            onDesktopChange: (from, to) => this._onDesktopChange(from, to)
        });

        // Layout management
        this._layoutManager = new LayoutManager();

        // Task switcher state
        this._taskSwitcherActive = false;
        this._taskSwitcherIndex = 0;

        // Drag state
        this._dragState = {
            active: false,
            window: null,
            startX: 0,
            startY: 0,
            initialX: 0,
            initialY: 0,
            handle: null
        };

        // Resize state
        this._resizeState = {
            active: false,
            window: null,
            handle: null,
            startX: 0,
            startY: 0,
            initialWidth: 0,
            initialHeight: 0,
            initialX: 0,
            initialY: 0
        };

        // Snap preview state
        this._snapPreview = {
            active: false,
            position: null,
            frame: null
        };

        // Window z-index counter
        this._nextZIndex = 100;

        // Event bindings
        this._boundOnMouseDown = this._onMouseDown.bind(this);
        this._boundOnMouseMove = this._onMouseMove.bind(this);
        this._boundOnMouseUp = this._onMouseUp.bind(this);
        this._boundOnKeyDown = this._onKeyDown.bind(this);
        this._boundOnKeyUp = this._onKeyUp.bind(this);
        this._boundOnResize = this._onResize.bind(this);
        this._boundOnContextMenu = this._onContextMenu.bind(this);

        // Initialize
        this._setupEventListeners();
    }

    /**
     * Gets the canvas
     * @returns {HTMLCanvasElement|null} Canvas
     */
    get canvas() {
        return this._canvas;
    }

    /**
     * Sets the canvas
     * @param {HTMLCanvasElement} canvas - Canvas element
     */
    set canvas(canvas) {
        this._canvas = canvas;
        this._ctx = canvas ? canvas.getContext('2d') : null;
        this._resizeCanvas();
    }

    /**
     * Gets the context
     * @returns {CanvasRenderingContext2D|null} Context
     */
    get ctx() {
        return this._ctx;
    }

    /**
     * Gets the screen width
     * @returns {number} Screen width
     */
    get screenWidth() {
        return this._screenWidth;
    }

    /**
     * Gets the screen height
     * @returns {number} Screen height
     */
    get screenHeight() {
        return this._screenHeight;
    }

    /**
     * Gets all windows
     * @returns {Window[]} All windows
     */
    get windows() {
        return Array.from(this._windows.values());
    }

    /**
     * Gets visible windows
     * @returns {Window[]} Visible windows
     */
    get visibleWindows() {
        return this.windows.filter(w => w.visible && !w.minimized);
    }

    /**
     * Gets the focused window
     * @returns {Window|null} Focused window
     */
    get focusedWindow() {
        return this._focusedWindow;
    }

    /**
     * Gets the desktop manager
     * @returns {DesktopManager} Desktop manager
     */
    get desktopManager() {
        return this._desktopManager;
    }

    /**
     * Gets the layout manager
     * @returns {LayoutManager} Layout manager
     */
    get layoutManager() {
        return this._layoutManager;
    }

    /**
     * Gets the task switcher state
     * @returns {boolean} Task switcher active
     */
    get taskSwitcherActive() {
        return this._taskSwitcherActive;
    }

    /**
     * Sets up event listeners
     */
    _setupEventListeners() {
        if (typeof window !== 'undefined') {
            window.addEventListener('resize', this._boundOnResize);
            window.addEventListener('keydown', this._boundOnKeyDown);
            window.addEventListener('keyup', this._boundOnKeyUp);
            window.addEventListener('contextmenu', this._boundOnContextMenu);
        }
    }

    /**
     * Removes event listeners
     */
    removeEventListeners() {
        if (typeof window !== 'undefined') {
            window.removeEventListener('resize', this._boundOnResize);
            window.removeEventListener('keydown', this._boundOnKeyDown);
            window.removeEventListener('keyup', this._boundOnKeyUp);
            window.removeEventListener('contextmenu', this._boundOnContextMenu);
        }
    }

    /**
     * Resizes the canvas to fit the screen
     */
    _resizeCanvas() {
        if (this._canvas) {
            this._canvas.width = this._screenWidth;
            this._canvas.height = this._screenHeight;
        }
    }

    /**
     * Handles resize events
     */
    _onResize() {
        this._screenWidth = window.innerWidth;
        this._screenHeight = window.innerHeight;
        this._resizeCanvas();
        this._emit('resized', { width: this._screenWidth, height: this._screenHeight });
    }

    /**
     * Handles mouse down events
     * @param {MouseEvent} e - Mouse event
     */
    _onMouseDown(e) {
        const rect = this._canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;

        // Find the topmost window under the mouse
        const window = this._getWindowAtPoint(x, y);
        if (window) {
            this._bringToFront(window);
            this._focusWindow(window);

            // Check for resize handle
            const handle = window.getResizeHandle(x, y);
            if (handle && window.resizable) {
                this._startResize(window, handle, x, y);
            } else if (window.isInTitleBar(x, y) && window.movable) {
                this._startDrag(window, x, y);
            }
        }

        this._emit('mouseDown', { x, y, window, canvas: this._canvas });
    }

    /**
     * Handles mouse move events
     * @param {MouseEvent} e - Mouse event
     */
    _onMouseMove(e) {
        const rect = this._canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;

        if (this._dragState.active) {
            this._handleDrag(x, y);
        } else if (this._resizeState.active) {
            this._handleResize(x, y);
        } else {
            // Update cursor
            const window = this._getWindowAtPoint(x, y);
            if (window) {
                const handle = window.getResizeHandle(x, y);
                this._setCursor(window, handle, x, y);
            } else {
                this._canvas.style.cursor = 'default';
            }
        }

        this._emit('mouseMove', { x, y, canvas: this._canvas });
    }

    /**
     * Handles mouse up events
     * @param {MouseEvent} e - Mouse event
     */
    _onMouseUp(e) {
        if (this._dragState.active) {
            this._endDrag();
        }
        if (this._resizeState.active) {
            this._endResize();
        }

        this._emit('mouseUp', { canvas: this._canvas });
    }

    /**
     * Handles key down events
     * @param {KeyboardEvent} e - Keyboard event
     */
    _onKeyDown(e) {
        const modifiers = e.shiftKey || e.ctrlKey || e.altKey || e.metaKey;

        // Alt+Tab - Task switcher
        if (e.altKey && e.key === 'Tab') {
            e.preventDefault();
            if (e.shiftKey) {
                this._taskSwitcherPrevious();
            } else {
                this._taskSwitcherNext();
            }
            return;
        }

        // Escape - Close task switcher
        if (e.key === 'Escape' && this._taskSwitcherActive) {
            this._closeTaskSwitcher();
            return;
        }

        // Window shortcuts (without modifiers)
        if (!modifiers) {
            switch (e.key) {
                case 'F11':
                    e.preventDefault();
                    this._toggleFullscreenFocused();
                    break;
            }
        }

        // Window management shortcuts
        if (e.ctrlKey || e.metaKey) {
            switch (e.key) {
                case 'n':
                case 'N':
                    e.preventDefault();
                    this._emit('shortcut:newWindow');
                    break;
                case 'w':
                case 'W':
                    e.preventDefault();
                    if (this._focusedWindow) {
                        this.closeWindow(this._focusedWindow.id);
                    }
                    break;
                case 'm':
                case 'M':
                    e.preventDefault();
                    if (this._focusedWindow) {
                        this._focusedWindow.minimize();
                    }
                    break;
                case 'ArrowUp':
                    e.preventDefault();
                    if (this._focusedWindow) {
                        this.snapWindow(this._focusedWindow.id, 'top');
                    }
                    break;
                case 'ArrowDown':
                    e.preventDefault();
                    if (this._focusedWindow) {
                        this.snapWindow(this._focusedWindow.id, 'bottom');
                    }
                    break;
                case 'ArrowLeft':
                    e.preventDefault();
                    if (this._focusedWindow) {
                        this.snapWindow(this._focusedWindow.id, 'left');
                    }
                    break;
                case 'ArrowRight':
                    e.preventDefault();
                    if (this._focusedWindow) {
                        this.snapWindow(this._focusedWindow.id, 'right');
                    }
                    break;
            }
        }

        // Desktop switching
        if (e.ctrlKey || e.metaKey) {
            if (e.key >= '1' && e.key <= '4') {
                e.preventDefault();
                this._desktopManager.switchTo(parseInt(e.key) - 1);
            }
        }

        this._emit('keyDown', e);
    }

    /**
     * Handles key up events
     * @param {KeyboardEvent} e - Keyboard event
     */
    _onKeyUp(e) {
        // Close task switcher on Alt release if it was activated
        if (e.key === 'Alt' && this._taskSwitcherActive) {
            this._closeTaskSwitcher();
        }

        this._emit('keyUp', e);
    }

    /**
     * Handles context menu events
     * @param {MouseEvent} e - Mouse event
     */
    _onContextMenu(e) {
        const rect = this._canvas.getBoundingClientRect();
        const x = e.clientX - rect.left;
        const y = e.clientY - rect.top;

        const window = this._getWindowAtPoint(x, y);
        if (window) {
            e.preventDefault();
            this._showWindowMenu(window, x, y);
        }

        this._emit('contextMenu', { x, y, window });
    }

    /**
     * Shows window context menu
     * @param {Window} window - Window
     * @param {number} x - X position
     * @param {number} y - Y position
     */
    _showWindowMenu(window, x, y) {
        const menu = {
            type: 'windowMenu',
            windowId: window.id,
            x,
            y,
            items: [
                { label: 'Restore', action: () => window.restore() },
                { label: 'Move', action: () => this._startDrag(window, x, y) },
                { label: 'Size', action: () => this._startResize(window, 'se', x, y) },
                { label: 'Minimize', action: () => window.minimize(), disabled: !window.minimizable },
                { label: 'Maximize', action: () => window.maximize(), disabled: !window.maximizable },
                { type: 'separator' },
                { label: 'Close', action: () => this.closeWindow(window.id), disabled: !window.closable }
            ]
        };

        this._emit('showMenu', menu);
    }

    /**
     * Sets the cursor based on position
     * @param {Window} window - Window
     * @param {string} handle - Resize handle
     * @param {number} x - X position
     * @param {number} y - Y position
     */
    _setCursor(window, handle, x, y) {
        let cursor = 'default';

        if (handle) {
            switch (handle) {
                case 'n': case 's': cursor = 'ns-resize'; break;
                case 'e': case 'w': cursor = 'ew-resize'; break;
                case 'ne': case 'sw': cursor = 'nesw-resize'; break;
                case 'nw': case 'se': cursor = 'nwse-resize'; break;
            }
        } else if (window.isInTitleBar(x, y)) {
            cursor = 'move';
        }

        this._canvas.style.cursor = cursor;
    }

    /**
     * Gets the window at a point
     * @param {number} x - X coordinate
     * @param {number} y - Y coordinate
     * @returns {Window|null} Window at point
     */
    _getWindowAtPoint(x, y) {
        // Check in reverse z-order (top to bottom)
        for (let i = this._windowOrder.length - 1; i >= 0; i--) {
            const window = this._windows.get(this._windowOrder[i]);
            if (window && window.visible && window.containsPoint(x, y)) {
                return window;
            }
        }
        return null;
    }

    /**
     * Brings a window to the front
     * @param {Window} window - Window to bring front
     */
    _bringToFront(window) {
        const index = this._windowOrder.indexOf(window.id);
        if (index > -1) {
            this._windowOrder.splice(index, 1);
        }
        this._windowOrder.push(window.id);
        window.zIndex = this._nextZIndex++;
    }

    /**
     * Focuses a window
     * @param {Window} window - Window to focus
     */
    _focusWindow(window) {
        if (this._focusedWindow && this._focusedWindow !== window) {
            this._focusedWindow.blur();
        }
        this._focusedWindow = window;
        window.focus();
        this._emit('windowFocused', window);
    }

    /**
     * Starts dragging a window
     * @param {Window} window - Window to drag
     * @param {number} x - Start X
     * @param {number} y - Start Y
     */
    _startDrag(window, x, y) {
        this._dragState = {
            active: true,
            window,
            startX: x,
            startY: y,
            initialX: window.x,
            initialY: window.y
        };

        this._canvas.style.cursor = 'move';
    }

    /**
     * Handles drag operation
     * @param {number} x - Current X
     * @param {number} y - Current Y
     */
    _handleDrag(x, y) {
        const state = this._dragState;
        const dx = x - state.startX;
        const dy = y - state.startY;

        const newX = state.initialX + dx;
        const newY = state.initialY + dy;

        // Check for snap positions
        const snapResult = this._checkSnap(newX, newY, state.window.width, state.window.height);
        if (snapResult) {
            state.window.move(snapResult.x, snapResult.y);
            this._showSnapPreview(snapResult);
        } else {
            state.window.move(newX, newY);
            this._hideSnapPreview();
        }
    }

    /**
     * Ends drag operation
     */
    _endDrag() {
        if (this._snapPreview.active) {
            this._applySnap(this._snapPreview.position);
        }
        this._hideSnapPreview();
        this._dragState = { active: false, window: null };
        this._canvas.style.cursor = 'default';
    }

    /**
     * Starts resizing a window
     * @param {Window} window - Window to resize
     * @param {string} handle - Resize handle
     * @param {number} x - Start X
     * @param {number} y - Start Y
     */
    _startResize(window, handle, x, y) {
        this._resizeState = {
            active: true,
            window,
            handle,
            startX: x,
            startY: y,
            initialWidth: window.width,
            initialHeight: window.height,
            initialX: window.x,
            initialY: window.y
        };

        this._canvas.style.cursor = this._getResizeCursor(handle);
    }

    /**
     * Handles resize operation
     * @param {number} x - Current X
     * @param {number} y - Current Y
     */
    _handleResize(x, y) {
        const state = this._resizeState;
        const dx = x - state.startX;
        const dy = y - state.startY;

        let newX = state.initialX;
        let newY = state.initialY;
        let newWidth = state.initialWidth;
        let newHeight = state.initialHeight;

        const handle = state.handle;

        // Apply resize based on handle
        if (handle.includes('e')) {
            newWidth = Math.max(state.window.minWidth, state.initialWidth + dx);
        }
        if (handle.includes('w')) {
            const maxWidth = state.initialX + state.initialWidth - state.window.minWidth;
            const possibleWidth = Math.max(state.window.minWidth, state.initialWidth - dx);
            if (possibleWidth <= maxWidth) {
                newX = state.initialX + dx;
                newWidth = possibleWidth;
            }
        }
        if (handle.includes('s')) {
            newHeight = Math.max(state.window.minHeight, state.initialHeight + dy);
        }
        if (handle.includes('n')) {
            const maxHeight = state.initialY + state.initialHeight - state.window.minHeight;
            const possibleHeight = Math.max(state.window.minHeight, state.initialHeight - dy);
            if (possibleHeight <= maxHeight) {
                newY = state.initialY + dy;
                newHeight = possibleHeight;
            }
        }

        state.window.setBounds(newX, newY, newWidth, newHeight);
    }

    /**
     * Ends resize operation
     */
    _endResize() {
        this._resizeState = { active: false, window: null, handle: null };
        this._canvas.style.cursor = 'default';
    }

    /**
     * Gets the cursor for a resize handle
     * @param {string} handle - Handle name
     * @returns {string} CSS cursor
     */
    _getResizeCursor(handle) {
        switch (handle) {
            case 'n': case 's': return 'ns-resize';
            case 'e': case 'w': return 'ew-resize';
            case 'ne': case 'sw': return 'nesw-resize';
            case 'nw': case 'se': return 'nwse-resize';
            default: return 'default';
        }
    }

    /**
     * Checks if window should snap
     * @param {number} x - X position
     * @param {number} y - Y position
     * @param {number} width - Window width
     * @param {number} height - Window height
     * @returns {Object|null} Snap result
     */
    _checkSnap(x, y, width, height) {
        const snapThreshold = 10;
        const halfWidth = this._screenWidth / 2;
        const halfHeight = this._screenHeight / 2;

        let snapPosition = null;
        let snapX = x;
        let snapY = y;

        // Check left/right snap
        if (Math.abs(x) < snapThreshold) {
            snapPosition = 'left';
            snapX = 0;
        } else if (Math.abs(x + width - this._screenWidth) < snapThreshold) {
            snapPosition = 'right';
            snapX = this._screenWidth - width;
        }

        // Check top/bottom snap
        if (Math.abs(y) < snapThreshold) {
            snapPosition = snapPosition ? 'top' + snapPosition.charAt(0).toUpperCase() + snapPosition.slice(1) : 'top';
            snapY = 0;
        } else if (Math.abs(y + height - this._screenHeight) < snapThreshold) {
            snapPosition = snapPosition ? 'bottom' + snapPosition.charAt(0).toUpperCase() + snapPosition.slice(1) : 'bottom';
            snapY = this._screenHeight - height;
        }

        if (snapPosition) {
            return { position: snapPosition, x: snapX, y: snapY, width, height };
        }

        return null;
    }

    /**
     * Shows snap preview
     * @param {Object} snapResult - Snap result
     */
    _showSnapPreview(snapResult) {
        this._snapPreview = {
            active: true,
            position: snapResult.position,
            frame: snapResult
        };
        this._emit('snapPreview', snapResult);
    }

    /**
     * Hides snap preview
     */
    _hideSnapPreview() {
        if (this._snapPreview.active) {
            this._snapPreview = { active: false, position: null, frame: null };
            this._emit('snapPreviewHidden');
        }
    }

    /**
     * Applies snap
     * @param {string} position - Snap position
     */
    _applySnap(position) {
        if (this._focusedWindow && this._snapPreview.active) {
            this.snapWindow(this._focusedWindow.id, position);
        }
    }

    /**
     * Toggles fullscreen for focused window
     */
    _toggleFullscreenFocused() {
        if (this._focusedWindow) {
            this._focusedWindow.fullscreen = !this._focusedWindow.fullscreen;
        }
    }

    /**
     * Opens the task switcher
     */
    _openTaskSwitcher() {
        if (this._taskSwitcherActive) return;

        this._taskSwitcherActive = true;
        this._taskSwitcherIndex = 0;
        this._emit('taskSwitcherOpened');
    }

    /**
     * Closes the task switcher
     */
    _closeTaskSwitcher() {
        if (!this._taskSwitcherActive) return;

        this._taskSwitcherActive = false;
        this._emit('taskSwitcherClosed');
    }

    /**
     * Navigates to next window in task switcher
     */
    _taskSwitcherNext() {
        const visibleWindows = this.visibleWindows;
        if (visibleWindows.length === 0) return;

        if (!this._taskSwitcherActive) {
            this._openTaskSwitcher();
        }

        this._taskSwitcherIndex = (this._taskSwitcherIndex + 1) % visibleWindows.length;
        this._emit('taskSwitcherChanged', {
            index: this._taskSwitcherIndex,
            window: visibleWindows[this._taskSwitcherIndex]
        });
    }

    /**
     * Navigates to previous window in task switcher
     */
    _taskSwitcherPrevious() {
        const visibleWindows = this.visibleWindows;
        if (visibleWindows.length === 0) return;

        if (!this._taskSwitcherActive) {
            this._openTaskSwitcher();
        }

        this._taskSwitcherIndex = (this._taskSwitcherIndex - 1 + visibleWindows.length) % visibleWindows.length;
        this._emit('taskSwitcherChanged', {
            index: this._taskSwitcherIndex,
            window: visibleWindows[this._taskSwitcherIndex]
        });
    }

    /**
     * Handles desktop change
     * @param {number} from - From desktop
     * @param {number} to - To desktop
     */
    _onDesktopChange(from, to) {
        // Hide windows from old desktop
        const fromWindows = this._desktopManager.getDesktop(from).getWindows();
        fromWindows.forEach(win => {
            if (this._windows.has(win.id)) {
                this._windows.get(win.id).hide();
            }
        });

        // Show windows on new desktop
        const toWindows = this._desktopManager.getDesktop(to).getWindows();
        toWindows.forEach(win => {
            if (this._windows.has(win.id)) {
                this._windows.get(win.id).show();
            }
        });

        this._emit('desktopChanged', { from, to });
    }

    /**
     * Creates a new window
     * @param {Object} config - Window configuration
     * @returns {Window} Created window
     */
    createWindow(config = {}) {
        const window = new Window({
            ...config,
            desktop: config.desktop !== undefined ? config.desktop : this._desktopManager.currentIndex
        });

        this._windows.set(window.id, window);
        this._windowOrder.push(window.id);
        this._desktopManager.currentDesktop.addWindow(window);

        // Set initial position if not specified
        if (config.x === undefined || config.y === undefined) {
            this._cascadeWindowPosition(window);
        }

        // Focus the new window
        this._bringToFront(window);
        this._focusWindow(window);

        // Emit event
        this._emit('windowCreated', window);

        return window;
    }

    /**
     * Calculates cascaded position for a new window
     * @param {Window} window - Window
     */
    _cascadeWindowPosition(window) {
        const offset = (this._windowOrder.length - 1) * 30;
        window.x = Math.min(100 + offset, this._screenWidth - 200);
        window.y = Math.min(100 + offset, this._screenHeight - 200);
    }

    /**
     * Closes a window
     * @param {string} windowId - Window ID
     * @returns {boolean} True if window was closed
     */
    closeWindow(windowId) {
        const window = this._windows.get(windowId);
        if (!window) return false;

        // Emit closing event
        this._emit('windowClosing', window);

        // Remove from desktop
        this._desktopManager.currentDesktop.removeWindow(window);

        // Remove from window order
        const orderIndex = this._windowOrder.indexOf(windowId);
        if (orderIndex > -1) {
            this._windowOrder.splice(orderIndex, 1);
        }

        // Delete window
        this._windows.delete(windowId);

        // Update focus if needed
        if (this._focusedWindow === window) {
            const visibleWindows = this.visibleWindows;
            if (visibleWindows.length > 0) {
                this._focusWindow(visibleWindows[visibleWindows.length - 1]);
            } else {
                this._focusedWindow = null;
            }
        }

        // Emit closed event
        this._emit('windowClosed', { id: windowId });

        return true;
    }

    /**
     * Focuses a window by ID
     * @param {string} windowId - Window ID
     */
    focusWindow(windowId) {
        const window = this._windows.get(windowId);
        if (!window) return;

        // Check if window is on current desktop
        if (window.desktop !== this._desktopManager.currentIndex) {
            this._desktopManager.switchTo(window.desktop);
        }

        this._bringToFront(window);
        this._focusWindow(window);
    }

    /**
     * Minimizes a window
     * @param {string} windowId - Window ID
     */
    minimizeWindow(windowId) {
        const window = this._windows.get(windowId);
        if (window && window.minimizable) {
            window.minimize();
            this._emit('windowMinimized', window);
        }
    }

    /**
     * Maximizes a window
     * @param {string} windowId - Window ID
     */
    maximizeWindow(windowId) {
        const window = this._windows.get(windowId);
        if (window && window.maximizable) {
            window.maximize();
            this._emit('windowMaximized', window);
        }
    }

    /**
     * Restores a window
     * @param {string} windowId - Window ID
     */
    restoreWindow(windowId) {
        const window = this._windows.get(windowId);
        if (window) {
            window.restore();
            this._bringToFront(window);
            this._focusWindow(window);
            this._emit('windowRestored', window);
        }
    }

    /**
     * Snaps a window to a position
     * @param {string} windowId - Window ID
     * @param {string} position - Snap position (left, right, top, bottom, topLeft, etc.)
     */
    snapWindow(windowId, position) {
        const window = this._windows.get(windowId);
        if (!window) return;

        let x, y, width, height;
        const halfWidth = Math.floor(this._screenWidth / 2);
        const halfHeight = Math.floor(this._screenHeight / 2);

        switch (position) {
            case 'left':
                x = 0; y = 0; width = halfWidth; height = this._screenHeight;
                break;
            case 'right':
                x = halfWidth; y = 0; width = halfWidth; height = this._screenHeight;
                break;
            case 'top':
                x = 0; y = 0; width = this._screenWidth; height = halfHeight;
                break;
            case 'bottom':
                x = 0; y = halfHeight; width = this._screenWidth; height = halfHeight;
                break;
            case 'topLeft':
                x = 0; y = 0; width = halfWidth; height = halfHeight;
                break;
            case 'topRight':
                x = halfWidth; y = 0; width = halfWidth; height = halfHeight;
                break;
            case 'bottomLeft':
                x = 0; y = halfHeight; width = halfWidth; height = halfHeight;
                break;
            case 'bottomRight':
                x = halfWidth; y = halfHeight; width = halfWidth; height = halfHeight;
                break;
            default:
                return;
        }

        window.setBounds(x, y, width, height);
        this._emit('windowSnapped', { window, position });
    }

    /**
     * Moves a window to a different desktop
     * @param {string} windowId - Window ID
     * @param {number} desktopIndex - Target desktop index
     */
    moveWindowToDesktop(windowId, desktopIndex) {
        const window = this._windows.get(windowId);
        if (!window) return;

        this._desktopManager.moveWindowToDesktop(window, desktopIndex);
        this._emit('windowMovedToDesktop', { window, desktop: desktopIndex });
    }

    /**
     * Applies the current layout to all windows
     */
    applyLayout() {
        const workArea = {
            x: 0,
            y: 0,
            width: this._screenWidth,
            height: this._screenHeight
        };

        const currentDesktopWindows = this._desktopManager.currentDesktop.getWindows();
        const visibleWindows = currentDesktopWindows.filter(w => this._windows.has(w.id));
        const windowObjects = visibleWindows.map(w => this._windows.get(w.id));

        this._layoutManager.applyLayout(windowObjects, workArea);
        this._emit('layoutApplied', { layout: this._layoutManager.currentLayoutName });
    }

    /**
     * Cycles through layout modes
     */
    cycleLayout() {
        const layoutName = this._layoutManager.cycleLayout();
        this.applyLayout();
        this._emit('layoutChanged', layoutName);
    }

    /**
     * Renders all windows
     */
    render() {
        if (!this._ctx) return;

        // Clear canvas
        this._ctx.fillStyle = '#1a202c';
        this._ctx.fillRect(0, 0, this._screenWidth, this._screenHeight);

        // Render windows in z-order
        for (const windowId of this._windowOrder) {
            const window = this._windows.get(windowId);
            if (window && window.visible) {
                this._renderWindow(window);
            }
        }

        // Render snap preview
        if (this._snapPreview.active && this._snapPreview.frame) {
            this._renderSnapPreview();
        }

        // Render task switcher
        if (this._taskSwitcherActive) {
            this._renderTaskSwitcher();
        }
    }

    /**
     * Renders a single window
     * @param {Window} window - Window to render
     */
    _renderWindow(window) {
        const ctx = this._ctx;
        const dpr = window.devicePixelRatio || 1;

        ctx.save();

        // Apply transform for high DPI
        ctx.scale(dpr, dpr);

        // Draw window background
        ctx.fillStyle = '#2d3748';
        ctx.fillRect(window.x, window.y, window.width, window.height);

        // Draw border
        if (window.hasBorder) {
            ctx.strokeStyle = window === this._focusedWindow ? '#4299e1' : '#4a5568';
            ctx.lineWidth = window === this._focusedWindow ? 2 : 1;
            ctx.strokeRect(window.x, window.y, window.width, window.height);
        }

        // Draw title bar
        if (window.hasTitleBar) {
            const titleBarHeight = window.titleBarHeight;

            // Title bar background
            ctx.fillStyle = window === this._focusedWindow ? '#2b6cb0' : '#4a5568';
            ctx.fillRect(window.x, window.y, window.width, titleBarHeight);

            // Title bar text
            ctx.fillStyle = window.titleBarTextColor || '#ffffff';
            ctx.font = '14px system-ui, sans-serif';
            ctx.textBaseline = 'middle';
            ctx.fillText(window.title, window.x + 10, window.y + titleBarHeight / 2);

            // Window buttons
            const buttonSize = window._decorations.buttonSize;
            const buttonY = window.y + (titleBarHeight - buttonSize) / 2;

            // Close button
            this._renderWindowButton(ctx, window.x + window.width - 10 - buttonSize, buttonY, buttonSize, '#fc8181', '#c53030');

            // Maximize button
            this._renderWindowButton(ctx, window.x + window.width - 20 - buttonSize * 2, buttonY, buttonSize, '#f6e05e', '#d69e2e');

            // Minimize button
            this._renderWindowButton(ctx, window.x + window.width - 30 - buttonSize * 3, buttonY, buttonSize, '#68d391', '#38a169');
        }

        // Draw resize handles
        if (window.resizable && window === this._focusedWindow) {
            ctx.strokeStyle = 'rgba(255, 255, 255, 0.2)';
            ctx.lineWidth = 1;
            ctx.strokeRect(window.x + 4, window.y + 4, window.width - 8, window.height - 8);
        }

        ctx.restore();
    }

    /**
     * Renders a window button
     * @param {CanvasRenderingContext2D} ctx - Context
     * @param {number} x - X position
     * @param {number} y - Y position
     * @param {number} size - Button size
     * @param {string} fillColor - Fill color
     * @param {string} strokeColor - Stroke color
     */
    _renderWindowButton(ctx, x, y, size, fillColor, strokeColor) {
        ctx.fillStyle = fillColor;
        ctx.strokeStyle = strokeColor;
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(x + size / 2, y + size / 2, size / 2 - 1, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
    }

    /**
     * Renders snap preview
     */
    _renderSnapPreview() {
        const ctx = this._ctx;
        const frame = this._snapPreview.frame;

        ctx.save();
        ctx.fillStyle = 'rgba(66, 153, 225, 0.3)';
        ctx.strokeStyle = '#4299e1';
        ctx.lineWidth = 2;
        ctx.fillRect(frame.x, frame.y, frame.width, frame.height);
        ctx.strokeRect(frame.x, frame.y, frame.width, frame.height);
        ctx.restore();
    }

    /**
     * Renders the task switcher overlay
     */
    _renderTaskSwitcher() {
        const ctx = this._ctx;
        const visibleWindows = this.visibleWindows;

        // Semi-transparent overlay
        ctx.fillStyle = 'rgba(0, 0, 0, 0.7)';
        ctx.fillRect(0, 0, this._screenWidth, this._screenHeight);

        // Window previews
        const previewWidth = 200;
        const previewHeight = 150;
        const gap = 20;
        const totalWidth = visibleWindows.length * (previewWidth + gap) - gap;
        const startX = (this._screenWidth - totalWidth) / 2;
        const startY = (this._screenHeight - previewHeight) / 2;

        visibleWindows.forEach((window, index) => {
            const x = startX + index * (previewWidth + gap);
            const isSelected = index === this._taskSwitcherIndex;

            // Preview background
            ctx.fillStyle = isSelected ? '#4299e1' : '#4a5568';
            ctx.fillRect(x, startY, previewWidth, previewHeight);

            // Preview title
            ctx.fillStyle = '#ffffff';
            ctx.font = '12px system-ui, sans-serif';
            ctx.fillText(window.title, x + 10, startY + 20);

            // Selected indicator
            if (isSelected) {
                ctx.strokeStyle = '#ffffff';
                ctx.lineWidth = 3;
                ctx.strokeRect(x - 3, startY - 3, previewWidth + 6, previewHeight + 6);
            }
        });

        // Instructions
        ctx.fillStyle = '#ffffff';
        ctx.font = '14px system-ui, sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText('Alt+Tab: Switch | Alt+Shift+Tab: Previous | Esc: Close', this._screenWidth / 2, this._screenHeight - 50);
        ctx.textAlign = 'left';
    }

    /**
     * Gets a window by ID
     * @param {string} windowId - Window ID
     * @returns {Window|null} Window
     */
    getWindow(windowId) {
        return this._windows.get(windowId) || null;
    }

    /**
     * Gets all window IDs
     * @returns {string[]} Window IDs
     */
    getWindowIds() {
        return Array.from(this._windows.keys());
    }

    /**
     * Serializes the window manager state
     * @returns {Object} Serialized state
     */
    serialize() {
        return {
            windows: this.windows.map(w => w.serialize()),
            windowOrder: [...this._windowOrder],
            focusedWindow: this._focusedWindow ? this._focusedWindow.id : null,
            desktopManager: this._desktopManager.serialize(),
            layoutManager: this._layoutManager.serialize(),
            screenWidth: this._screenWidth,
            screenHeight: this._screenHeight
        };
    }

    /**
     * Creates a window manager from serialized state
     * @param {Object} data - Serialized state
     * @param {Object} config - Configuration
     * @returns {WindowManager} New window manager
     */
    static deserialize(data, config = {}) {
        const wm = new WindowManager(config);

        // Restore screen size
        if (data.screenWidth) wm._screenWidth = data.screenWidth;
        if (data.screenHeight) wm._screenHeight = data.screenHeight;

        // Restore desktops
        if (data.desktopManager) {
            wm._desktopManager = DesktopManager.deserialize(data.desktopManager);
        }

        // Restore layouts
        if (data.layoutManager) {
            wm._layoutManager = LayoutManager.deserialize(data.layoutManager);
        }

        // Restore windows
        if (data.windows) {
            data.windows.forEach(w => {
                const window = Window.deserialize(w);
                wm._windows.set(window.id, window);
                wm._windowOrder.push(window.id);
                const desktop = wm._desktopManager.getDesktop(window.desktop);
                if (desktop) {
                    desktop.addWindow(window);
                }
            });
        }

        // Restore window order
        if (data.windowOrder) {
            wm._windowOrder = [...data.windowOrder];
        }

        // Restore focus
        if (data.focusedWindow) {
            const window = wm._windows.get(data.focusedWindow);
            if (window) {
                wm._focusWindow(window);
            }
        }

        return wm;
    }

    /**
     * Adds an event listener
     * @param {string} event - Event name
     * @param {Function} handler - Handler function
     */
    on(event, handler) {
        if (!this._eventHandlers) {
            this._eventHandlers = {};
        }
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
        if (!this._eventHandlers || !this._eventHandlers[event]) return;
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
        if (!this._eventHandlers || !this._eventHandlers[event]) return;
        this._eventHandlers[event].forEach(handler => handler(data));
    }
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { WindowManager };
}
