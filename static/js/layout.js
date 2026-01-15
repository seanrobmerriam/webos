/**
 * Layout - Window layout algorithms for the window manager
 * 
 * Provides various layout algorithms including floating, tiling, and
 * custom arrangements for window management.
 */

/**
 * Layout base class
 */
class Layout {
    /**
     * Creates a new layout
     * @param {string} name - Layout name
     */
    constructor(name = 'Layout') {
        this.name = name;
        this._gap = 4;
        this._padding = 8;
    }

    /**
     * Gets the gap between windows
     * @returns {number} Gap size in pixels
     */
    get gap() {
        return this._gap;
    }

    /**
     * Sets the gap between windows
     * @param {number} gap - Gap size
     */
    set gap(gap) {
        this._gap = Math.max(0, Math.min(32, gap));
    }

    /**
     * Gets the padding
     * @returns {number} Padding size in pixels
     */
    get padding() {
        return this._padding;
    }

    /**
     * Sets the padding
     * @param {number} padding - Padding size
     */
    set padding(padding) {
        this._padding = Math.max(0, Math.min(32, padding));
    }

    /**
     * Applies the layout to windows
     * @param {Window[]} windows - Windows to layout
     * @param {Object} area - Layout area {x, y, width, height}
     */
    apply(windows, area) {
        throw new Error('apply() must be implemented by subclass');
    }

    /**
     * Adds a window to the layout
     * @param {Window} window - Window to add
     * @param {Object} area - Layout area
     */
    addWindow(window, area) {
        this.apply([window], area);
    }

    /**
     * Serializes the layout state
     * @returns {Object} Serialized state
     */
    serialize() {
        return {
            name: this.name,
            gap: this._gap,
            padding: this._padding
        };
    }
}

/**
 * FloatingLayout - Floating window layout (free positioning)
 */
class FloatingLayout extends Layout {
    constructor() {
        super('Floating');
    }

    apply(windows, area) {
        // Floating layout doesn't auto-arrange
        windows.forEach(win => {
            if (!win.maximized && !win.fullscreen) {
                win.ensureBounds();
            }
        });
    }
}

/**
 * TilingLayout - Automatic tiling layout
 */
class TilingLayout extends Layout {
    /**
     * Creates a new tiling layout
     * @param {string} orientation - 'horizontal' or 'vertical'
     */
    constructor(orientation = 'horizontal') {
        super('Tiling');
        this._orientation = orientation;
        this._masterRatio = 0.6;
        this._masterCount = 1;
    }

    /**
     * Gets the orientation
     * @returns {string} Orientation
     */
    get orientation() {
        return this._orientation;
    }

    /**
     * Sets the orientation
     * @param {string} orientation - 'horizontal' or 'vertical'
     */
    set orientation(orientation) {
        if (orientation !== 'horizontal' && orientation !== 'vertical') {
            throw new Error('Invalid orientation: must be horizontal or vertical');
        }
        this._orientation = orientation;
    }

    /**
     * Gets the master ratio
     * @returns {number} Master ratio (0.1 to 0.9)
     */
    get masterRatio() {
        return this._masterRatio;
    }

    /**
     * Sets the master ratio
     * @param {number} ratio - Master ratio
     */
    set masterRatio(ratio) {
        this._masterRatio = Math.max(0.1, Math.min(0.9, ratio));
    }

    /**
     * Gets the master window count
     * @returns {number} Master count
     */
    get masterCount() {
        return this._masterCount;
    }

    /**
     * Sets the master window count
     * @param {number} count - Master count
     */
    set masterCount(count) {
        this._masterCount = Math.max(1, count);
    }

    apply(windows, area) {
        if (windows.length === 0) return;

        const visibleWindows = windows.filter(w => !w.minimized && w.visible);
        if (visibleWindows.length === 0) return;

        const masterCount = Math.min(this._masterCount, visibleWindows.length);
        const slaveCount = visibleWindows.length - masterCount;

        const workArea = {
            x: area.x + this._padding,
            y: area.y + this._padding,
            width: area.width - (this._padding * 2),
            height: area.height - (this._padding * 2)
        };

        if (this._orientation === 'horizontal') {
            this._applyHorizontal(visibleWindows, workArea, masterCount, slaveCount);
        } else {
            this._applyVertical(visibleWindows, workArea, masterCount, slaveCount);
        }
    }

    _applyHorizontal(windows, area, masterCount, slaveCount) {
        const gap = this._gap;
        const masterWidth = Math.floor(area.width * this._masterRatio);
        
        // Calculate master area dimensions
        const masterAreaWidth = masterWidth - (gap * (masterCount - 1));
        const masterWindowWidth = Math.floor(masterAreaWidth / masterCount);
        
        // Position master windows
        for (let i = 0; i < masterCount; i++) {
            const win = windows[i];
            const x = area.x + (i * (masterWindowWidth + gap));
            win.setBounds(x, area.y, masterWindowWidth, area.height);
        }

        if (slaveCount > 0) {
            // Calculate slave area dimensions
            const slaveAreaWidth = area.width - masterWidth - gap;
            const slaveWindowWidth = Math.floor(slaveAreaWidth / slaveCount);
            
            // Position slave windows
            for (let i = 0; i < slaveCount; i++) {
                const win = windows[masterCount + i];
                const x = area.x + masterWidth + gap + (i * (slaveWindowWidth + gap));
                win.setBounds(x, area.y, slaveWindowWidth, area.height);
            }
        }
    }

    _applyVertical(windows, area, masterCount, slaveCount) {
        const gap = this._gap;
        const masterHeight = Math.floor(area.height * this._masterRatio);
        
        // Calculate master area dimensions
        const masterAreaHeight = masterHeight - (gap * (masterCount - 1));
        const masterWindowHeight = Math.floor(masterAreaHeight / masterCount);
        
        // Position master windows
        for (let i = 0; i < masterCount; i++) {
            const win = windows[i];
            const y = area.y + (i * (masterWindowHeight + gap));
            win.setBounds(area.x, y, area.width, masterWindowHeight);
        }

        if (slaveCount > 0) {
            // Calculate slave area dimensions
            const slaveAreaHeight = area.height - masterHeight - gap;
            const slaveWindowHeight = Math.floor(slaveAreaHeight / slaveCount);
            
            // Position slave windows
            for (let i = 0; i < slaveCount; i++) {
                const win = windows[masterCount + i];
                const y = area.y + masterHeight + gap + (i * (slaveWindowHeight + gap));
                win.setBounds(area.x, y, area.width, slaveWindowHeight);
            }
        }
    }

    serialize() {
        const data = super.serialize();
        data.orientation = this._orientation;
        data.masterRatio = this._masterRatio;
        data.masterCount = this._masterCount;
        return data;
    }
}

/**
 * GridLayout - Grid-based layout
 */
class GridLayout extends Layout {
    constructor() {
        super('Grid');
        this._columns = 2;
        this._rows = 2;
    }

    /**
     * Gets the number of columns
     * @returns {number} Columns
     */
    get columns() {
        return this._columns;
    }

    /**
     * Sets the number of columns
     * @param {number} cols - Columns
     */
    set columns(cols) {
        this._columns = Math.max(1, Math.min(8, cols));
    }

    /**
     * Gets the number of rows
     * @returns {number} Rows
     */
    get rows() {
        return this._rows;
    }

    /**
     * Sets the number of rows
     * @param {number} rows - Rows
     */
    set rows(rows) {
        this._rows = Math.max(1, Math.min(8, rows));
    }

    apply(windows, area) {
        const visibleWindows = windows.filter(w => !w.minimized && w.visible);
        if (visibleWindows.length === 0) return;

        const gap = this._gap;
        const workArea = {
            x: area.x + this._padding,
            y: area.y + this._padding,
            width: area.width - (this._padding * 2),
            height: area.height - (this._padding * 2)
        };

        const cols = Math.min(this._columns, visibleWindows.length);
        const rows = Math.ceil(visibleWindows.length / cols);

        const cellWidth = (workArea.width - (gap * (cols - 1))) / cols;
        const cellHeight = (workArea.height - (gap * (rows - 1))) / rows;

        visibleWindows.forEach((win, i) => {
            const col = i % cols;
            const row = Math.floor(i / cols);
            const x = workArea.x + (col * (cellWidth + gap));
            const y = workArea.y + (row * (cellHeight + gap));
            win.setBounds(x, y, cellWidth, cellHeight);
        });
    }

    serialize() {
        const data = super.serialize();
        data.columns = this._columns;
        data.rows = this._rows;
        return data;
    }
}

/**
 * MonocleLayout - All windows maximized, cycling through focus
 */
class MonocleLayout extends Layout {
    constructor() {
        super('Monocle');
    }

    apply(windows, area) {
        const visibleWindows = windows.filter(w => !w.minimized && w.visible);
        if (visibleWindows.length === 0) return;

        // Only the focused window is visible in monocle mode
        const workArea = {
            x: area.x + this._padding,
            y: area.y + this._padding,
            width: area.width - (this._padding * 2),
            height: area.height - (this._padding * 2)
        };

        visibleWindows.forEach((win, i) => {
            if (i === visibleWindows.length - 1) {
                // Last window (top of stack) gets full area
                win.setBounds(workArea.x, workArea.y, workArea.width, workArea.height);
            } else {
                // Other windows are minimized/hidden in monocle view
                // Keep their normal bounds but they're not visible
            }
        });
    }
}

/**
 * LayoutManager - Manages layout modes and switching
 */
class LayoutManager {
    /**
     * Creates a new layout manager
     */
    constructor() {
        this._layouts = {
            floating: new FloatingLayout(),
            tilingH: new TilingLayout('horizontal'),
            tilingV: new TilingLayout('vertical'),
            grid: new GridLayout(),
            monocle: new MonocleLayout()
        };
        this._currentLayout = 'floating';
    }

    /**
     * Gets the current layout
     * @returns {Layout} Current layout
     */
    get currentLayout() {
        return this._layouts[this._currentLayout];
    }

    /**
     * Gets the current layout name
     * @returns {string} Layout name
     */
    get currentLayoutName() {
        return this._currentLayout;
    }

    /**
     * Gets all available layouts
     * @returns {string[]} Layout names
     */
    get layoutNames() {
        return Object.keys(this._layouts);
    }

    /**
     * Gets all layouts
     * @returns {Object} Layouts map
     */
    get layouts() {
        return { ...this._layouts };
    }

    /**
     * Sets the current layout
     * @param {string} name - Layout name
     */
    setLayout(name) {
        if (!this._layouts[name]) {
            throw new Error(`Unknown layout: ${name}`);
        }
        this._currentLayout = name;

        // Emit event
        if (typeof EventBus !== 'undefined') {
            EventBus.emit('layout:changed', {
                layout: name,
                layoutObj: this._layouts[name]
            });
        }
    }

    /**
     * Cycles through layouts
     * @returns {string} New layout name
     */
    cycleLayout() {
        const names = this.layoutNames;
        const currentIndex = names.indexOf(this._currentLayout);
        const nextIndex = (currentIndex + 1) % names.length;
        this.setLayout(names[nextIndex]);
        return this._currentLayout;
    }

    /**
     * Gets a layout by name
     * @param {string} name - Layout name
     * @returns {Layout} Layout
     */
    getLayout(name) {
        return this._layouts[name];
    }

    /**
     * Configures a layout
     * @param {string} name - Layout name
     * @param {Object} config - Configuration
     */
    configureLayout(name, config) {
        const layout = this._layouts[name];
        if (!layout) {
            throw new Error(`Unknown layout: ${name}`);
        }

        if (config.gap !== undefined) layout.gap = config.gap;
        if (config.padding !== undefined) layout.padding = config.padding;

        if (layout instanceof TilingLayout) {
            if (config.orientation !== undefined) layout.orientation = config.orientation;
            if (config.masterRatio !== undefined) layout.masterRatio = config.masterRatio;
            if (config.masterCount !== undefined) layout.masterCount = config.masterCount;
        }

        if (layout instanceof GridLayout) {
            if (config.columns !== undefined) layout.columns = config.columns;
            if (config.rows !== undefined) layout.rows = config.rows;
        }
    }

    /**
     * Applies the current layout to windows
     * @param {Window[]} windows - Windows to layout
     * @param {Object} area - Layout area
     */
    applyLayout(windows, area) {
        this.currentLayout.apply(windows, area);
    }

    /**
     * Serializes the layout manager state
     * @returns {Object} Serialized state
     */
    serialize() {
        return {
            currentLayout: this._currentLayout,
            layouts: Object.fromEntries(
                Object.entries(this._layouts).map(([name, layout]) => [name, layout.serialize()])
            )
        };
    }

    /**
     * Creates a layout manager from serialized state
     * @param {Object} data - Serialized state
     * @returns {LayoutManager} New layout manager
     */
    static deserialize(data) {
        const manager = new LayoutManager();

        if (data.currentLayout) {
            manager._currentLayout = data.currentLayout;
        }

        if (data.layouts) {
            for (const [name, layoutData] of Object.entries(data.layouts)) {
                if (manager._layouts[name]) {
                    manager.configureLayout(name, layoutData);
                }
            }
        }

        return manager;
    }
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        Layout,
        FloatingLayout,
        TilingLayout,
        GridLayout,
        MonocleLayout,
        LayoutManager
    };
}
