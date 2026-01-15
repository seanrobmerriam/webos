/**
 * Shell - UI shell with window management, desktop background, and window chrome
 * 
 * @module shell
 */

class Shell {
    /**
     * Create a new Shell instance
     * @param {Object} [options] - Shell options
     */
    constructor(options = {}) {
        this.options = {
            containerId: options.containerId || 'window-layer',
            taskbarId: options.taskbarId || 'taskbar-items',
            ...options
        };
        
        // Window management
        this.windows = new Map();
        this._zIndexCounter = 100;
        this._focusedWindow = null;
        this._windowOrder = [];
        
        // Container elements
        this._container = document.getElementById(this.options.containerId);
        this._taskbar = document.getElementById(this.options.taskbarId);
        
        // Desktop background
        this._desktop = document.getElementById('desktop');
        
        // Dragging state
        this._draggedWindow = null;
        this._dragOffset = { x: 0, y: 0 };
        this._originalPosition = null;
        
        // Resizing state
        this._resizedWindow = null;
        this._resizeDirection = null;
        this._resizeStart = { x: 0, y: 0, width: 0, height: 0 };
        
        // Bind methods
        this._handleMouseMove = this._handleMouseMove.bind(this);
        this._handleMouseUp = this._handleMouseUp.bind(this);
        
        // Setup global event listeners for drag/resize
        document.addEventListener('mousemove', this._handleMouseMove);
        document.addEventListener('mouseup', this._handleMouseUp);
    }
    
    /**
     * Create a new window
     * @param {Object} config - Window configuration
     * @returns {Object} Window object
     */
    createWindow(config) {
        const window = {
            id: config.id || this._generateId(),
            title: config.title || 'Window',
            x: config.x || 100,
            y: config.y || 100,
            width: config.width || 400,
            height: config.height || 300,
            minWidth: config.minWidth || 150,
            minHeight: config.minHeight || 80,
            resizable: config.resizable !== false,
            minimizable: config.minimizable !== false,
            closable: config.closable !== false,
            maximized: false,
            minimized: false,
            element: null,
            content: null,
            onClose: config.onClose || null,
            onFocus: config.onFocus || null
        };
        
        this._createWindowElement(window);
        this._addWindowToManager(window);
        
        return window;
    }
    
    /**
     * Create window DOM element
     * @param {Object} window - Window object
     */
    _createWindowElement(window) {
        const element = document.createElement('div');
        element.className = 'window';
        element.id = `window-${window.id}`;
        element.style.left = `${window.x}px`;
        element.style.top = `${window.y}px`;
        element.style.width = `${window.width}px`;
        element.style.height = `${window.height}px`;
        element.style.zIndex = this._zIndexCounter++;
        
        // Title bar
        const titlebar = document.createElement('div');
        titlebar.className = 'window-titlebar';
        
        const title = document.createElement('div');
        title.className = 'window-title';
        title.textContent = window.title;
        
        const controls = document.createElement('div');
        controls.className = 'window-controls';
        
        // Minimize button
        if (window.minimizable) {
            const minimizeBtn = document.createElement('button');
            minimizeBtn.className = 'window-control minimize';
            minimizeBtn.title = 'Minimize';
            minimizeBtn.addEventListener('click', () => this.minimizeWindow(window.id));
            controls.appendChild(minimizeBtn);
        }
        
        // Maximize button
        if (window.resizable) {
            const maximizeBtn = document.createElement('button');
            maximizeBtn.className = 'window-control maximize';
            maximizeBtn.title = 'Maximize';
            maximizeBtn.addEventListener('click', () => this.toggleMaximize(window.id));
            controls.appendChild(maximizeBtn);
        }
        
        // Close button
        if (window.closable) {
            const closeBtn = document.createElement('button');
            closeBtn.className = 'window-control close';
            closeBtn.title = 'Close';
            closeBtn.addEventListener('click', () => this.closeWindow(window.id));
            controls.appendChild(closeBtn);
        }
        
        titlebar.appendChild(title);
        titlebar.appendChild(controls);
        
        // Make title bar draggable
        titlebar.addEventListener('mousedown', (e) => {
            if (!window.maximized) {
                this._startDrag(window.id, e);
            }
        });
        
        // Content area
        const content = document.createElement('div');
        content.className = 'window-content';
        if (config.content) {
            if (typeof config.content === 'string') {
                content.innerHTML = config.content;
            } else {
                content.appendChild(config.content);
            }
        }
        
        element.appendChild(titlebar);
        element.appendChild(content);
        
        // Store references
        window.element = element;
        window.titlebar = titlebar;
        window.content = content;
        
        // Add to container
        if (this._container) {
            this._container.appendChild(element);
        }
        
        // Focus on create
        this.focusWindow(window.id);
    }
    
    /**
     * Add window to manager
     * @param {Object} window - Window object
     */
    _addWindowToManager(window) {
        this.windows.set(window.id, window);
        this._windowOrder.push(window.id);
        this._addTaskbarItem(window);
    }
    
    /**
     * Add window to taskbar
     * @param {Object} window - Window object
     */
    _addTaskbarItem(window) {
        if (!this._taskbar) return;
        
        const item = document.createElement('div');
        item.className = 'taskbar-item';
        item.id = `taskbar-${window.id}`;
        item.textContent = window.title;
        item.addEventListener('click', () => {
            if (window.minimized) {
                this.restoreWindow(window.id);
            } else if (this._focusedWindow === window.id) {
                this.minimizeWindow(window.id);
            } else {
                this.focusWindow(window.id);
            }
        });
        
        this._taskbar.appendChild(item);
    }
    
    /**
     * Close a window
     * @param {string} windowId - Window ID
     */
    closeWindow(windowId) {
        const window = this.windows.get(windowId);
        if (!window) return;
        
        if (window.onClose) {
            window.onClose();
        }
        
        // Remove from DOM
        if (window.element && window.element.parentNode) {
            window.element.parentNode.removeChild(window.element);
        }
        
        // Remove from taskbar
        const taskbarItem = document.getElementById(`taskbar-${windowId}`);
        if (taskbarItem && taskbarItem.parentNode) {
            taskbarItem.parentNode.removeChild(taskbarItem);
        }
        
        // Remove from manager
        this.windows.delete(windowId);
        this._windowOrder = this._windowOrder.filter(id => id !== windowId);
        
        // Update focus
        if (this._focusedWindow === windowId) {
            this._focusedWindow = null;
            if (this._windowOrder.length > 0) {
                const lastId = this._windowOrder[this._windowOrder.length - 1];
                this.focusWindow(lastId);
            }
        }
    }
    
    /**
     * Focus a window
     * @param {string} windowId - Window ID
     */
    focusWindow(windowId) {
        const window = this.windows.get(windowId);
        if (!window || window.minimized) return;
        
        // Update z-index
        window.element.style.zIndex = this._zIndexCounter++;
        
        // Update focused state
        this._unfocusAll();
        window.element.classList.add('focused');
        this._focusedWindow = windowId;
        
        // Update taskbar
        this._updateTaskbar();
        
        if (window.onFocus) {
            window.onFocus();
        }
    }
    
    /**
     * Minimize a window
     * @param {string} windowId - Window ID
     */
    minimizeWindow(windowId) {
        const window = this.windows.get(windowId);
        if (!window || !window.minimizable) return;
        
        window.minimized = true;
        window.element.classList.add('minimized');
        this._updateTaskbar();
        
        // Focus next window
        if (this._focusedWindow === windowId) {
            this._focusedWindow = null;
            const nextWindow = this._windowOrder.find(id => {
                const w = this.windows.get(id);
                return w && !w.minimized;
            });
            if (nextWindow) {
                this.focusWindow(nextWindow);
            }
        }
    }
    
    /**
     * Restore a minimized window
     * @param {string} windowId - Window ID
     */
    restoreWindow(windowId) {
        const window = this.windows.get(windowId);
        if (!window) return;
        
        window.minimized = false;
        window.element.classList.remove('minimized');
        this.focusWindow(windowId);
    }
    
    /**
     * Toggle window maximize state
     * @param {string} windowId - Window ID
     */
    toggleMaximize(windowId) {
        const window = this.windows.get(windowId);
        if (!window || !window.resizable) return;
        
        if (window.maximized) {
            // Restore
            window.maximized = false;
            window.element.classList.remove('maximized');
            window.element.style.left = `${window._restoreState.x}px`;
            window.element.style.top = `${window._restoreState.y}px`;
            window.element.style.width = `${window._restoreState.width}px`;
            window.element.style.height = `${window._restoreState.height}px`;
        } else {
            // Store current state
            window._restoreState = {
                x: window.x,
                y: window.y,
                width: window.width,
                height: window.height
            };
            
            // Maximize
            window.maximized = true;
            window.element.classList.add('maximized');
            window.element.style.left = '0px';
            window.element.style.top = '0px';
            window.element.style.width = '100%';
            window.element.style.height = 'calc(100% - 40px)';
        }
    }
    
    /**
     * Start dragging a window
     * @param {string} windowId - Window ID
     * @param {MouseEvent} event - Mouse event
     */
    _startDrag(windowId, event) {
        const window = this.windows.get(windowId);
        if (!window) return;
        
        this._draggedWindow = window;
        this._dragOffset = {
            x: event.clientX - window.x,
            y: event.clientY - window.y
        };
        
        this.focusWindow(windowId);
    }
    
    /**
     * Handle mouse move for drag/resize
     * @param {MouseEvent} event
     */
    _handleMouseMove(event) {
        if (this._draggedWindow) {
            event.preventDefault();
            const window = this._draggedWindow;
            
            window.x = event.clientX - this._dragOffset.x;
            window.y = event.clientY - this._dragOffset.y;
            
            window.element.style.left = `${window.x}px`;
            window.element.style.top = `${window.y}px`;
        }
        
        if (this._resizedWindow) {
            event.preventDefault();
            const window = this._resizedWindow;
            const deltaX = event.clientX - this._resizeStart.x;
            const deltaY = event.clientY - this._resizeStart.y;
            
            if (this._resizeDirection.includes('e')) {
                window.width = Math.max(window.minWidth, this._resizeStart.width + deltaX);
            }
            if (this._resizeDirection.includes('w')) {
                const newWidth = Math.max(window.minWidth, this._resizeStart.width - deltaX);
                window.x = window.x + (this._resizeStart.width - newWidth);
                window.width = newWidth;
            }
            if (this._resizeDirection.includes('s')) {
                window.height = Math.max(window.minHeight, this._resizeStart.height + deltaY);
            }
            if (this._resizeDirection.includes('n')) {
                const newHeight = Math.max(window.minHeight, this._resizeStart.height - deltaY);
                window.y = window.y + (this._resizeStart.height - newHeight);
                window.height = newHeight;
            }
            
            window.element.style.width = `${window.width}px`;
            window.element.style.height = `${window.height}px`;
        }
    }
    
    /**
     * Handle mouse up to end drag/resize
     */
    _handleMouseUp() {
        this._draggedWindow = null;
        this._resizedWindow = null;
    }
    
    /**
     * Unfocus all windows
     */
    _unfocusAll() {
        for (const window of this.windows.values()) {
            window.element.classList.remove('focused');
        }
    }
    
    /**
     * Update taskbar to reflect current state
     */
    _updateTaskbar() {
        for (const [id, window] of this.windows) {
            const item = document.getElementById(`taskbar-${id}`);
            if (item) {
                item.classList.toggle('active', 
                    this._focusedWindow === id && !window.minimized);
                item.classList.toggle('minimized', window.minimized);
            }
        }
    }
    
    /**
     * Generate unique window ID
     * @returns {string} Window ID
     */
    _generateId() {
        return `window-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Get all windows
     * @returns {Array} Window objects
     */
    getWindows() {
        return Array.from(this.windows.values());
    }
    
    /**
     * Get focused window
     * @returns {string|null} Focused window ID
     */
    getFocusedWindow() {
        return this._focusedWindow;
    }
    
    /**
     * Set window content
     * @param {string} windowId - Window ID
     * @param {string|HTMLElement} content - New content
     */
    setWindowContent(windowId, content) {
        const window = this.windows.get(windowId);
        if (!window) return;
        
        if (typeof content === 'string') {
            window.content.innerHTML = content;
        } else {
            window.content.innerHTML = '';
            window.content.appendChild(content);
        }
    }
    
    /**
     * Set window title
     * @param {string} windowId - Window ID
     * @param {string} title - New title
     */
    setWindowTitle(windowId, title) {
        const window = this.windows.get(windowId);
        if (!window) return;
        
        window.title = title;
        const titleElement = window.element.querySelector('.window-title');
        if (titleElement) {
            titleElement.textContent = title;
        }
        
        // Update taskbar
        const taskbarItem = document.getElementById(`taskbar-${windowId}`);
        if (taskbarItem) {
            taskbarItem.textContent = title;
        }
    }
    
    /**
     * Cleanup shell
     */
    destroy() {
        document.removeEventListener('mousemove', this._handleMouseMove);
        document.removeEventListener('mouseup', this._handleMouseUp);
        
        for (const window of this.windows.values()) {
            if (window.element && window.element.parentNode) {
                window.element.parentNode.removeChild(window.element);
            }
        }
        this.windows.clear();
    }
}

// Export for different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Shell;
}
if (typeof window !== 'undefined') {
    window.Shell = Shell;
}
