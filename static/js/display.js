/**
 * DisplayManager - Canvas rendering engine with layers and double buffering
 * 
 * @module display
 */

class DisplayManager {
    /**
     * Create a new DisplayManager instance
     * @param {HTMLCanvasElement} canvas - Canvas element for rendering
     * @param {Object} [options] - Display options
     */
    constructor(canvas, options = {}) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d', { alpha: false });
        this.options = {
            useDoubleBuffering: options.useDoubleBuffering !== false,
            clearOnRender: options.clearOnRender !== false,
            ...options
        };
        
        // Layers for rendering
        this.layers = [];
        this._layerMap = new Map();
        
        // Double buffering
        this._offscreenCanvas = null;
        this._offscreenCtx = null;
        
        // Rendering state
        this._needsRedraw = true;
        this._isRendering = false;
        
        // Event listeners
        this._listeners = new Map();
        
        // Initialize
        this._init();
    }
    
    /**
     * Initialize the display manager
     */
    _init() {
        this._resize();
        
        // Setup double buffering
        if (this.options.useDoubleBuffering) {
            this._createOffscreenBuffer();
        }
        
        // Listen for window resize
        window.addEventListener('resize', () => this._resize());
    }
    
    /**
     * Create offscreen buffer for double buffering
     */
    _createOffscreenBuffer() {
        this._offscreenCanvas = document.createElement('canvas');
        this._offscreenCanvas.width = this.canvas.width;
        this._offscreenCanvas.height = this.canvas.height;
        this._offscreenCtx = this._offscreenCanvas.getContext('2d', { alpha: false });
    }
    
    /**
     * Resize canvas to match window size
     */
    _resize() {
        const dpr = window.devicePixelRatio || 1;
        const width = window.innerWidth;
        const height = window.innerHeight;
        
        this.canvas.width = width * dpr;
        this.canvas.height = height * dpr;
        this.canvas.style.width = `${width}px`;
        this.canvas.style.height = `${height}px`;
        
        this.ctx.scale(dpr, dpr);
        
        // Update offscreen buffer
        if (this._offscreenCanvas) {
            this._offscreenCanvas.width = this.canvas.width;
            this._offscreenCanvas.height = this.canvas.height;
            this._offscreenCtx.scale(dpr, dpr);
        }
        
        this._needsRedraw = true;
        this._emit('resize', { width, height, dpr });
    }
    
    /**
     * Add a layer to the rendering stack
     * @param {string} name - Layer name
     * @param {number} zIndex - Z-index for layering
     * @returns {Object} Layer object
     */
    addLayer(name, zIndex = 0) {
        const layer = {
            name,
            zIndex,
            visible: true,
            dirty: true,
            elements: []
        };
        
        this.layers.push(layer);
        this._layerMap.set(name, layer);
        this._sortLayers();
        this._needsRedraw = true;
        
        return layer;
    }
    
    /**
     * Remove a layer
     * @param {string} name - Layer name
     */
    removeLayer(name) {
        const layer = this._layerMap.get(name);
        if (layer) {
            this.layers = this.layers.filter(l => l !== layer);
            this._layerMap.delete(name);
            this._needsRedraw = true;
        }
    }
    
    /**
     * Get a layer by name
     * @param {string} name - Layer name
     * @returns {Object|null} Layer object or null
     */
    getLayer(name) {
        return this._layerMap.get(name) || null;
    }
    
    /**
     * Set layer visibility
     * @param {string} name - Layer name
     * @param {boolean} visible - Visibility state
     */
    setLayerVisibility(name, visible) {
        const layer = this._layerMap.get(name);
        if (layer && layer.visible !== visible) {
            layer.visible = visible;
            layer.dirty = true;
            this._needsRedraw = true;
        }
    }
    
    /**
     * Mark a layer as needing redraw
     * @param {string} name - Layer name
     */
    markLayerDirty(name) {
        const layer = this._layerMap.get(name);
        if (layer) {
            layer.dirty = true;
            this._needsRedraw = true;
        }
    }
    
    /**
     * Sort layers by z-index
     */
    _sortLayers() {
        this.layers.sort((a, b) => a.zIndex - b.zIndex);
    }
    
    /**
     * Clear the display
     */
    clear() {
        const ctx = this._getContext();
        ctx.fillStyle = '#1a1a2e';
        ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
    }
    
    /**
     * Render all layers
     * @param {Array} [instructions] - Rendering instructions from server
     */
    render(instructions) {
        if (this._isRendering) return;
        this._isRendering = true;
        
        try {
            const ctx = this._getContext();
            
            // Clear if needed
            if (this.options.clearOnRender || this._needsRedraw) {
                this.clear();
            }
            
            // Process server instructions if provided
            if (instructions && instructions.length > 0) {
                this._processInstructions(instructions);
            }
            
            // Render visible layers
            for (const layer of this.layers) {
                if (layer.visible) {
                    this._renderLayer(ctx, layer);
                }
            }
            
            // Copy to main canvas if using double buffering
            if (this._offscreenCanvas) {
                this.ctx.drawImage(
                    this._offscreenCanvas,
                    0, 0,
                    this._offscreenCanvas.width,
                    this._offscreenCanvas.height
                );
            }
            
            this._needsRedraw = false;
        } finally {
            this._isRendering = false;
        }
    }
    
    /**
     * Get the appropriate rendering context
     * @returns {CanvasRenderingContext2D}
     */
    _getContext() {
        return this._offscreenCtx || this.ctx;
    }
    
    /**
     * Process rendering instructions from server
     * @param {Array} instructions - Rendering instructions
     */
    _processInstructions(instructions) {
        for (const instruction of instructions) {
            switch (instruction.type) {
                case 'clear':
                    this._getContext().fillStyle = instruction.color || '#1a1a2e';
                    this._getContext().fillRect(
                        instruction.x || 0,
                        instruction.y || 0,
                        instruction.width || this.canvas.width,
                        instruction.height || this.canvas.height
                    );
                    break;
                    
                case 'rect':
                    this._getContext().fillStyle = instruction.color;
                    this._getContext().fillRect(
                        instruction.x,
                        instruction.y,
                        instruction.width,
                        instruction.height
                    );
                    break;
                    
                case 'text':
                    this._getContext().font = instruction.font || '14px sans-serif';
                    this._getContext().fillStyle = instruction.color || '#eaeaea';
                    this._getContext().fillText(
                        instruction.text,
                        instruction.x,
                        instruction.y
                    );
                    break;
                    
                case 'image':
                    // Handle image rendering
                    break;
                    
                default:
                    console.warn('Unknown instruction type:', instruction.type);
            }
        }
    }
    
    /**
     * Render a single layer
     * @param {CanvasRenderingContext2D} ctx - Rendering context
     * @param {Object} layer - Layer to render
     */
    _renderLayer(ctx, layer) {
        // Override in subclass for custom rendering
    }
    
    /**
     * Request a redraw
     */
    requestRedraw() {
        this._needsRedraw = true;
        this.render();
    }
    
    /**
     * Get canvas dimensions
     * @returns {Object} { width, height }
     */
    getSize() {
        return {
            width: this.canvas.width,
            height: this.canvas.height
        };
    }
    
    /**
     * Convert screen coordinates to canvas coordinates
     * @param {number} x - Screen X
     * @param {number} y - Screen Y
     * @returns {Object} { x, y } Canvas coordinates
     */
    screenToCanvas(x, y) {
        const rect = this.canvas.getBoundingClientRect();
        return {
            x: x - rect.left,
            y: y - rect.top
        };
    }
    
    /**
     * Emit an event
     * @param {string} event - Event name
     * @param {*} data - Event data
     */
    _emit(event, data) {
        if (this._listeners && this._listeners.has(event)) {
            for (const callback of this._listeners.get(event)) {
                try {
                    callback(data);
                } catch (err) {
                    console.error(`Error in ${event} listener:`, err);
                }
            }
        }
    }
    
    /**
     * Add an event listener
     * @param {string} event - Event name
     * @param {Function} callback - Callback function
     * @returns {Function} Unsubscribe function
     */
    on(event, callback) {
        if (!this._listeners.has(event)) {
            this._listeners.set(event, new Set());
        }
        this._listeners.get(event).add(callback);
        return () => this.off(event, callback);
    }
    
    /**
     * Remove an event listener
     * @param {string} event - Event name
     * @param {Function} callback - Callback function
     */
    off(event, callback) {
        if (this._listeners && this._listeners.has(event)) {
            this._listeners.get(event).delete(callback);
        }
    }
}

// Export for different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = DisplayManager;
}
if (typeof window !== 'undefined') {
    window.DisplayManager = DisplayManager;
}
