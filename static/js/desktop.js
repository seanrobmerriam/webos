/**
 * Desktop - Virtual desktop management for the window manager
 * 
 * Handles multiple virtual desktops, window placement per desktop,
 * and desktop switching animations.
 */

class Desktop {
    /**
     * Creates a new desktop
     * @param {number} id - Desktop identifier
     * @param {string} name - Desktop name
     */
    constructor(id = 0, name = 'Main') {
        this.id = id;
        this.name = name;
        this.windows = new Set();
        this._gridSize = 8;
        this._wallpaper = null;
    }

    /**
     * Gets the grid size for tiling
     * @returns {number} Grid size
     */
    get gridSize() {
        return this._gridSize;
    }

    /**
     * Sets the grid size for tiling
     * @param {number} size - Grid size
     */
    set gridSize(size) {
        this._gridSize = Math.max(2, Math.min(16, size));
    }

    /**
     * Gets the wallpaper
     * @returns {string|null} Wallpaper URL or null
     */
    get wallpaper() {
        return this._wallpaper;
    }

    /**
     * Sets the wallpaper
     * @param {string} url - Wallpaper URL
     */
    set wallpaper(url) {
        this._wallpaper = url;
    }

    /**
     * Adds a window to this desktop
     * @param {Window} window - Window to add
     */
    addWindow(window) {
        if (window) {
            this.windows.add(window);
        }
    }

    /**
     * Removes a window from this desktop
     * @param {Window} window - Window to remove
     */
    removeWindow(window) {
        if (window) {
            this.windows.delete(window);
        }
    }

    /**
     * Checks if a window is on this desktop
     * @param {Window} window - Window to check
     * @returns {boolean} True if window is on this desktop
     */
    hasWindow(window) {
        return this.windows.has(window);
    }

    /**
     * Gets all visible windows on this desktop
     * @returns {Window[]} Visible windows
     */
    getVisibleWindows() {
        return Array.from(this.windows).filter(w => w.visible && !w.minimized);
    }

    /**
     * Gets all windows on this desktop
     * @returns {Window[]} All windows
     */
    getWindows() {
        return Array.from(this.windows);
    }

    /**
     * Gets the number of windows
     * @returns {number} Window count
     */
    get windowCount() {
        return this.windows.size;
    }

    /**
     * Clears all windows from this desktop
     */
    clear() {
        this.windows.clear();
    }

    /**
     * Serializes the desktop state
     * @returns {Object} Serialized state
     */
    serialize() {
        return {
            id: this.id,
            name: this.name,
            windowIds: Array.from(this.windows).map(w => w.id),
            gridSize: this._gridSize,
            wallpaper: this._wallpaper
        };
    }

    /**
     * Creates a desktop from serialized state
     * @param {Object} data - Serialized state
     * @returns {Desktop} New desktop instance
     */
    static deserialize(data) {
        const desktop = new Desktop(data.id, data.name);
        desktop.gridSize = data.gridSize || 8;
        desktop._wallpaper = data.wallpaper || null;
        return desktop;
    }
}

/**
 * DesktopManager - Manages multiple virtual desktops
 */
class DesktopManager {
    /**
     * Creates a new desktop manager
     * @param {Object} config - Configuration
     */
    constructor(config = {}) {
        this._desktops = [];
        this._currentIndex = 0;
        this._maxDesktops = config.maxDesktops || 4;
        this._onDesktopChange = config.onDesktopChange || null;
        this._animationDuration = config.animationDuration || 200;

        // Create initial desktops
        this._createInitialDesktops(config.initialCount || 4);
    }

    /**
     * Creates initial desktops
     * @param {number} count - Number of desktops to create
     */
    _createInitialDesktops(count) {
        const names = ['Main', 'Work', '娱乐', 'Other'];
        for (let i = 0; i < Math.min(count, this._maxDesktops); i++) {
            const name = i < names.length ? names[i] : `Desktop ${i + 1}`;
            this._desktops.push(new Desktop(i, name));
        }
    }

    /**
     * Gets the current desktop index
     * @returns {number} Current index
     */
    get currentIndex() {
        return this._currentIndex;
    }

    /**
     * Gets the current desktop
     * @returns {Desktop} Current desktop
     */
    get currentDesktop() {
        return this._desktops[this._currentIndex];
    }

    /**
     * Gets all desktops
     * @returns {Desktop[]} All desktops
     */
    get desktops() {
        return [...this._desktops];
    }

    /**
     * Gets the number of desktops
     * @returns {number} Desktop count
     */
    get count() {
        return this._desktops.length;
    }

    /**
     * Gets the maximum number of desktops
     * @returns {number} Max desktops
     */
    get maxDesktops() {
        return this._maxDesktops;
    }

    /**
     * Sets the callback for desktop changes
     * @param {Function} callback - Callback function
     */
    setOnDesktopChange(callback) {
        this._onDesktopChange = callback;
    }

    /**
     * Switches to a desktop by index
     * @param {number} index - Desktop index
     * @param {boolean} animate - Whether to animate the transition
     * @returns {Promise<void>} Promise that resolves when animation completes
     */
    async switchTo(index, animate = true) {
        if (index < 0 || index >= this._desktops.length) {
            throw new Error(`Invalid desktop index: ${index}`);
        }

        const oldIndex = this._currentIndex;
        this._currentIndex = index;

        if (animate && this._onDesktopChange) {
            await this._onDesktopChange(oldIndex, index);
        }

        // Emit event for UI updates
        if (typeof EventBus !== 'undefined') {
            EventBus.emit('desktop:switched', {
                from: oldIndex,
                to: index,
                desktop: this.currentDesktop
            });
        }
    }

    /**
     * Switches to the next desktop
     * @param {boolean} wrap - Whether to wrap around
     * @returns {Promise<void>} Promise that resolves when animation completes
     */
    async next(wrap = true) {
        const nextIndex = this._currentIndex + 1;
        if (nextIndex >= this._desktops.length) {
            if (wrap) {
                return this.switchTo(0);
            }
            throw new Error('No next desktop');
        }
        return this.switchTo(nextIndex);
    }

    /**
     * Switches to the previous desktop
     * @param {boolean} wrap - Whether to wrap around
     * @returns {Promise<void>} Promise that resolves when animation completes
     */
    async previous(wrap = true) {
        const prevIndex = this._currentIndex - 1;
        if (prevIndex < 0) {
            if (wrap) {
                return this.switchTo(this._desktops.length - 1);
            }
            throw new Error('No previous desktop');
        }
        return this.switchTo(prevIndex);
    }

    /**
     * Creates a new desktop
     * @param {string} name - Desktop name (optional)
     * @returns {Desktop} The new desktop
     */
    createDesktop(name = null) {
        if (this._desktops.length >= this._maxDesktops) {
            throw new Error('Maximum number of desktops reached');
        }

        const id = this._desktops.length;
        const desktopName = name || `Desktop ${id + 1}`;
        const desktop = new Desktop(id, desktopName);
        this._desktops.push(desktop);

        // Emit event
        if (typeof EventBus !== 'undefined') {
            EventBus.emit('desktop:created', { desktop });
        }

        return desktop;
    }

    /**
     * Deletes a desktop (cannot delete the last desktop)
     * @param {number} index - Desktop index to delete
     * @returns {boolean} True if deleted
     */
    deleteDesktop(index) {
        if (this._desktops.length <= 1) {
            throw new Error('Cannot delete the last desktop');
        }

        if (index < 0 || index >= this._desktops.length) {
            throw new Error(`Invalid desktop index: ${index}`);
        }

        // Don't allow deleting current desktop
        if (index === this._currentIndex) {
            throw new Error('Cannot delete the current desktop');
        }

        const desktop = this._desktops[index];
        this._desktops.splice(index, 1);

        // Update desktop IDs
        for (let i = index; i < this._desktops.length; i++) {
            this._desktops[i].id = i;
        }

        // Emit event
        if (typeof EventBus !== 'undefined') {
            EventBus.emit('desktop:deleted', { desktop, index });
        }

        return true;
    }

    /**
     * Gets a desktop by index
     * @param {number} index - Desktop index
     * @returns {Desktop} Desktop
     */
    getDesktop(index) {
        if (index < 0 || index >= this._desktops.length) {
            throw new Error(`Invalid desktop index: ${index}`);
        }
        return this._desktops[index];
    }

    /**
     * Renames a desktop
     * @param {number} index - Desktop index
     * @param {string} name - New name
     * @returns {Desktop} The desktop
     */
    renameDesktop(index, name) {
        const desktop = this.getDesktop(index);
        desktop.name = name;

        // Emit event
        if (typeof EventBus !== 'undefined') {
            EventBus.emit('desktop:renamed', { desktop, index, name });
        }

        return desktop;
    }

    /**
     * Moves a window to a different desktop
     * @param {Window} window - Window to move
     * @param {number} targetIndex - Target desktop index
     */
    moveWindowToDesktop(window, targetIndex) {
        const sourceDesktop = this.currentDesktop;
        const targetDesktop = this.getDesktop(targetIndex);

        // Remove from current desktop
        sourceDesktop.removeWindow(window);
        // Add to target desktop
        targetDesktop.addWindow(window);

        // Update window's desktop reference
        window.desktop = targetIndex;

        // If window was visible, hide it and emit event
        if (window.visible && window.desktop !== this._currentIndex) {
            window.hide();
        }

        // Emit event
        if (typeof EventBus !== 'undefined') {
            EventBus.emit('desktop:windowMoved', {
                window,
                from: sourceDesktop.id,
                to: targetDesktop.id
            });
        }
    }

    /**
     * Gets windows for the current desktop
     * @returns {Window[]} Windows on current desktop
     */
    getCurrentWindows() {
        return this.currentDesktop.getWindows();
    }

    /**
     * Gets visible windows for the current desktop
     * @returns {Window[]} Visible windows
     */
    getCurrentVisibleWindows() {
        return this.currentDesktop.getVisibleWindows();
    }

    /**
     * Serializes the desktop manager state
     * @returns {Object} Serialized state
     */
    serialize() {
        return {
            currentIndex: this._currentIndex,
            maxDesktops: this._maxDesktops,
            desktops: this._desktops.map(d => d.serialize())
        };
    }

    /**
     * Creates a desktop manager from serialized state
     * @param {Object} data - Serialized state
     * @param {Object} config - Configuration
     * @returns {DesktopManager} New desktop manager
     */
    static deserialize(data, config = {}) {
        const manager = new DesktopManager({
            maxDesktops: data.maxDesktops || config.maxDesktops || 4,
            initialCount: 0
        });

        manager._desktops = data.desktops.map(d => Desktop.deserialize(d));
        manager._currentIndex = data.currentIndex || 0;

        return manager;
    }
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { Desktop, DesktopManager };
}
