/**
 * ClientState - Client state management with connection state tracking
 * 
 * @module state
 */

class ClientState {
    /**
     * Connection states
     */
    static get ConnectionState() {
        return {
            DISCONNECTED: 'DISCONNECTED',
            CONNECTING: 'CONNECTING',
            CONNECTED: 'CONNECTED',
            RECONNECTING: 'RECONNECTING',
            ERROR: 'ERROR'
        };
    }
    
    /**
     * Display modes
     */
    static get DisplayMode() {
        return {
            WINDOWED: 'WINDOWED',
            FULLSCREEN: 'FULLSCREEN',
            MAXIMIZED: 'MAXIMIZED'
        };
    }
    
    /**
     * Create a new ClientState instance
     * @param {Object} [initialState] - Initial state values
     */
    constructor(initialState = {}) {
        // Core connection state
        this._connectionState = initialState.connectionState || ClientState.ConnectionState.DISCONNECTED;
        
        // Authentication state
        this._authenticated = initialState.authenticated || false;
        this._username = initialState.username || null;
        this._authToken = initialState.authToken || null;
        
        // Display state
        this._displayMode = initialState.displayMode || ClientState.DisplayMode.WINDOWED;
        this._canvasWidth = initialState.canvasWidth || 0;
        this._canvasHeight = initialState.canvasHeight || 0;
        
        // Error handling
        this._lastError = initialState.lastError || null;
        this._errorCount = 0;
        
        // Event listeners
        this._listeners = new Map();
    }
    
    // Connection state
    get connectionState() {
        return this._connectionState;
    }
    
    setConnectionState(value) {
        this._setState('connectionState', value);
    }
    
    // Authentication state
    get authenticated() {
        return this._authenticated;
    }
    
    get username() {
        return this._username;
    }
    
    setAuthenticated(value, username = null, token = null) {
        this._authenticated = value;
        this._username = username;
        this._authToken = token;
        this._emit('authChange', { authenticated: value, username });
    }
    
    // Display state
    get displayMode() {
        return this._displayMode;
    }
    
    setDisplayMode(value) {
        this._setState('displayMode', value);
    }
    
    get canvasWidth() {
        return this._canvasWidth;
    }
    
    get canvasHeight() {
        return this._canvasHeight;
    }
    
    setCanvasSize(width, height) {
        const changed = this._canvasWidth !== width || this._canvasHeight !== height;
        this._canvasWidth = width;
        this._canvasHeight = height;
        if (changed) {
            this._emit('canvasResize', { width, height });
        }
    }
    
    // Error handling
    get lastError() {
        return this._lastError;
    }
    
    setError(error) {
        this._lastError = error;
        this._errorCount++;
        this._emit('error', { error, count: this._errorCount });
    }
    
    clearError() {
        this._lastError = null;
    }
    
    get errorCount() {
        return this._errorCount;
    }
    
    // State setter helper
    _setState(key, value) {
        if (this[`_${key}`] !== value) {
            const oldValue = this[`_${key}`];
            this[`_${key}`] = value;
            this._emit('stateChange', { key, oldValue, newValue: value });
        }
    }
    
    // Event system
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
        if (this._listeners.has(event)) {
            this._listeners.get(event).delete(callback);
        }
    }
    
    /**
     * Emit an event
     * @param {string} event - Event name
     * @param {*} data - Event data
     */
    _emit(event, data) {
        if (this._listeners.has(event)) {
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
     * Get a snapshot of the current state
     * @returns {Object} State snapshot
     */
    toJSON() {
        return {
            connectionState: this._connectionState,
            authenticated: this._authenticated,
            username: this._username,
            displayMode: this._displayMode,
            canvasWidth: this._canvasWidth,
            canvasHeight: this._canvasHeight,
            lastError: this._lastError,
            errorCount: this._errorCount
        };
    }
    
    /**
     * Reset state to initial values
     */
    reset() {
        this._connectionState = ClientState.ConnectionState.DISCONNECTED;
        this._authenticated = false;
        this._username = null;
        this._authToken = null;
        this._displayMode = ClientState.DisplayMode.WINDOWED;
        this._lastError = null;
        this._errorCount = 0;
        this._emit('reset', {});
    }
}

// Export for different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ClientState;
}
if (typeof window !== 'undefined') {
    window.ClientState = ClientState;
}
