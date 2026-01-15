/**
 * Client - Main client entry point for WebOS browser client
 * 
 * @module client
 */

class Client {
    /**
     * Create a new Client instance
     * @param {ClientConfig} config - Client configuration
     */
    constructor(config) {
        this.config = {
            serverUrl: config.serverUrl || 'ws://localhost:8080/ws',
            canvasId: config.canvasId || 'display-canvas',
            autoConnect: config.autoConnect !== false,
            reconnectDelay: config.reconnectDelay || 1000,
            maxRetries: config.maxRetries || 10,
            ...config
        };
        
        // Initialize components
        this.state = new ClientState();
        this.connection = new Connection(this.config.serverUrl, {
            reconnectDelay: this.config.reconnectDelay,
            maxRetries: this.config.maxRetries
        });
        
        const canvas = document.getElementById(this.config.canvasId);
        this.display = new DisplayManager(canvas);
        this.input = new InputManager();
        this.shell = new Shell();
        
        // Message handlers
        this._messageHandlers = new Map();
        
        // Bind methods
        this._handleConnectionStateChange = this._handleConnectionStateChange.bind(this);
        this._handleMessage = this._handleMessage.bind(this);
        this._handleInput = this._handleInput.bind(this);
        this._handleStateChange = this._handleStateChange.bind(this);
        
        // Setup
        this._setupEventListeners();
    }
    
    /**
     * Setup event listeners
     */
    _setupEventListeners() {
        // Connection state changes
        this.connection.onStateChange(this._handleConnectionStateChange);
        
        // Incoming messages
        this.connection.onMessage(this._handleMessage);
        
        // Input events
        this.input.onInput(this._handleInput);
        
        // State changes
        this.state.on('stateChange', this._handleStateChange);
        
        // Display resize
        this.display.on('resize', (data) => {
            this.state.setCanvasSize(data.width, data.height);
        });
    }
    
    /**
     * Connect to the server
     * @returns {Promise<void>} Connection promise
     */
    async connect() {
        try {
            await this.connection.connect();
        } catch (err) {
            this.state.setError(`Connection failed: ${err.message}`);
            throw err;
        }
    }
    
    /**
     * Handle connection state changes
     * @param {string} newState - New state
     * @param {string} oldState - Old state
     */
    _handleConnectionStateChange(newState, oldState) {
        this.state.setConnectionState(newState);
        
        // Update UI
        this._updateConnectionStatus(newState);
        
        // Handle state-specific actions
        switch (newState) {
            case 'CONNECTED':
                this._onConnected();
                break;
            case 'DISCONNECTED':
                this._onDisconnected();
                break;
            case 'ERROR':
                this.state.setError('Connection error');
                break;
        }
    }
    
    /**
     * Handle incoming messages from server
     * @param {Object} message - Decoded message
     */
    _handleMessage(message) {
        const opcodeName = ProtocolClient.getOpcodeName(message.opcode);
        
        // Call registered handler if exists
        const handler = this._messageHandlers.get(opcodeName);
        if (handler) {
            handler(message);
        }
        
        // Handle by opcode type
        switch (message.opcode) {
            case ProtocolClient.Opcodes.DISPLAY:
                this._handleDisplayMessage(message);
                break;
            case ProtocolClient.Opcodes.ERROR:
                this._handleErrorMessage(message);
                break;
            case ProtocolClient.Opcodes.PING:
                this._handlePingMessage(message);
                break;
            default:
                console.debug('Unhandled message:', opcodeName, message);
        }
    }
    
    /**
     * Handle display messages
     * @param {Object} message - Display message
     */
    _handleDisplayMessage(message) {
        // Decode display instructions from payload
        try {
            const codec = ProtocolClient.createCodec(message.payload);
            const instructionCount = codec.readUint32();
            const instructions = [];
            
            for (let i = 0; i < instructionCount; i++) {
                const type = codec.readByte();
                instructions.push({ type });
            }
            
            this.display.render(instructions);
        } catch (err) {
            console.error('Failed to process display message:', err);
        }
    }
    
    /**
     * Handle error messages
     * @param {Object} message - Error message
     */
    _handleErrorMessage(message) {
        try {
            const codec = ProtocolClient.createCodec(message.payload);
            const errorCode = codec.readUint32();
            const errorMessage = codec.readBytes(codec.remaining());
            const text = new TextDecoder().decode(errorMessage);
            
            this.state.setError(`Server error ${errorCode}: ${text}`);
        } catch (err) {
            this.state.setError('Unknown server error');
        }
    }
    
    /**
     * Handle ping messages (respond with pong)
     * @param {Object} message - Ping message
     */
    _handlePingMessage(message) {
        this.send(ProtocolClient.Opcodes.PONG, null);
    }
    
    /**
     * Handle input events from InputManager
     * @param {string} type - Input type (keyboard, mouse)
     * @param {Object} data - Input data
     */
    _handleInput(type, data) {
        // Encode and send input to server
        try {
            const payload = this._encodeInput(type, data);
            this.send(ProtocolClient.Opcodes.INPUT, payload);
        } catch (err) {
            console.error('Failed to send input:', err);
        }
    }
    
    /**
     * Encode input data for transmission
     * @param {string} type - Input type
     * @param {Object} data - Input data
     * @returns {Uint8Array} Encoded payload
     */
    _encodeInput(type, data) {
        const encoder = new TextEncoder();
        const json = JSON.stringify({ type, ...data });
        return encoder.encode(json);
    }
    
    /**
     * Handle state changes
     * @param {Object} event - State change event
     */
    _handleStateChange(event) {
        // Update UI based on state changes
        switch (event.key) {
            case 'authenticated':
                this._updateAuthUI(event.newValue);
                break;
            case 'displayMode':
                this._updateDisplayMode(event.newValue);
                break;
        }
    }
    
    /**
     * Update connection status indicator
     * @param {string} state - Connection state
     */
    _updateConnectionStatus(state) {
        const statusEl = document.getElementById('connection-status');
        if (!statusEl) return;
        
        statusEl.className = `status-${state.toLowerCase()}`;
        
        switch (state) {
            case 'CONNECTING':
                statusEl.textContent = 'Connecting...';
                break;
            case 'CONNECTED':
                statusEl.textContent = 'Connected';
                break;
            case 'DISCONNECTED':
                statusEl.textContent = 'Disconnected';
                break;
            case 'RECONNECTING':
                statusEl.textContent = 'Reconnecting...';
                break;
            case 'ERROR':
                statusEl.textContent = 'Connection Error';
                break;
        }
    }
    
    /**
     * Update authentication UI
     * @param {boolean} authenticated - Auth state
     */
    _updateAuthUI(authenticated) {
        const loginModal = document.getElementById('login-modal');
        if (loginModal) {
            loginModal.classList.toggle('hidden', authenticated);
        }
    }
    
    /**
     * Update display mode
     * @param {string} mode - Display mode
     */
    _updateDisplayMode(mode) {
        const canvas = this.display.canvas;
        switch (mode) {
            case 'FULLSCREEN':
                if (canvas.requestFullscreen) {
                    canvas.requestFullscreen();
                }
                break;
            case 'WINDOWED':
                if (document.fullscreenElement) {
                    document.exitFullscreen();
                }
                break;
        }
    }
    
    /**
     * Called when connected
     */
    _onConnected() {
        // Send authentication if we have credentials
        if (this.state.authToken) {
            this._sendAuth();
        }
        
        // Start input capture
        this.input.init();
        
        // Initial render
        this.display.render();
    }
    
    /**
     * Called when disconnected
     */
    _onDisconnected() {
        // Stop input capture
        this.input.destroy();
    }
    
    /**
     * Send authentication data
     */
    _sendAuth() {
        const encoder = new TextEncoder();
        const payload = encoder.encode(JSON.stringify({
            username: this.state.username,
            token: this.state.authToken
        }));
        this.send(ProtocolClient.Opcodes.AUTH, payload);
    }
    
    /**
     * Send a message to the server
     * @param {number} opcode - Message opcode
     * @param {Uint8Array} payload - Message payload
     * @returns {boolean} Send status
     */
    send(opcode, payload) {
        return this.connection.send(opcode, payload);
    }
    
    /**
     * Register a message handler
     * @param {string} opcodeName - Opcode name or number
     * @param {Function} handler - Handler function
     * @returns {Function} Unsubscribe function
     */
    on(opcodeName, handler) {
        const key = typeof opcodeName === 'number' 
            ? ProtocolClient.getOpcodeName(opcodeName) 
            : opcodeName;
        this._messageHandlers.set(key, handler);
        return () => this._messageHandlers.delete(key);
    }
    
    /**
     * Create a window
     * @param {Object} config - Window configuration
     * @returns {Object} Window object
     */
    createWindow(config) {
        return this.shell.createWindow(config);
    }
    
    /**
     * Close a window
     * @param {string} windowId - Window ID
     */
    closeWindow(windowId) {
        this.shell.closeWindow(windowId);
    }
    
    /**
     * Login to the server
     * @param {string} username - Username
     * @param {string} password - Password
     * @returns {Promise<void>} Login promise
     */
    async login(username, password) {
        const encoder = new TextEncoder();
        const payload = encoder.encode(JSON.stringify({ username, password }));
        
        this.send(ProtocolClient.Opcodes.AUTH, payload);
        
        // Wait for auth response (simplified - in real app, use proper auth flow)
        return new Promise((resolve, reject) => {
            const unsubscribe = this.on('AUTH', (message) => {
                unsubscribe();
                // Parse auth response
                const decoder = new TextDecoder();
                try {
                    const response = JSON.parse(decoder.decode(message.payload));
                    if (response.success) {
                        this.state.setAuthenticated(true, username, response.token);
                        resolve();
                    } else {
                        this.state.setError(response.error || 'Authentication failed');
                        reject(new Error(response.error || 'Authentication failed'));
                    }
                } catch (err) {
                    this.state.setError('Invalid auth response');
                    reject(err);
                }
            });
            
            // Timeout after 10 seconds
            setTimeout(() => {
                unsubscribe();
                this.state.setError('Auth timeout');
                reject(new Error('Auth timeout'));
            }, 10000);
        });
    }
    
    /**
     * Logout and disconnect
     */
    logout() {
        this.state.setAuthenticated(false);
        this.connection.close();
    }
    
    /**
     * Get current state snapshot
     * @returns {Object} State snapshot
     */
    getState() {
        return this.state.toJSON();
    }
    
    /**
     * Cleanup and destroy client
     */
    destroy() {
        this.input.destroy();
        this.shell.destroy();
        this.connection.close();
        this.state.reset();
    }
}

// Export for different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Client;
}
if (typeof window !== 'undefined') {
    window.Client = Client;
}
