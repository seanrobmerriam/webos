/**
 * Connection - WebSocket connection management with automatic reconnection
 * 
 * @module connection
 */

class Connection {
    /**
     * Create a new Connection instance
     * @param {string} url - WebSocket server URL
     * @param {Object} [options] - Connection options
     */
    constructor(url, options = {}) {
        this.url = url;
        this.ws = null;
        this.reconnectDelay = options.reconnectDelay || 1000;
        this.maxRetries = options.maxRetries || 10;
        this.currentRetries = 0;
        this.messageQueue = [];
        this.isConnecting = false;
        
        // Callbacks
        this.onOpen = null;
        this.onMessage = null;
        this.onClose = null;
        this.onError = null;
        this.onStateChange = null;
        
        // Connection state
        this._state = 'DISCONNECTED';
    }
    
    /**
     * Get current connection state
     * @returns {string} Connection state
     */
    get state() {
        return this._state;
    }
    
    /**
     * Set connection state and notify listeners
     * @param {string} newState - New state
     */
    _setState(newState) {
        const oldState = this._state;
        this._state = newState;
        if (this.onStateChange && oldState !== newState) {
            this.onStateChange(newState, oldState);
        }
    }
    
    /**
     * Connect to the WebSocket server
     * @returns {Promise<void>} Connection promise
     */
    connect() {
        return new Promise((resolve, reject) => {
            if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
                resolve();
                return;
            }
            
            this._setState('CONNECTING');
            this.isConnecting = true;
            
            try {
                this.ws = new WebSocket(this.url);
                
                this.ws.onopen = () => {
                    this.isConnecting = false;
                    this.currentRetries = 0;
                    this._setState('CONNECTED');
                    
                    // Flush message queue
                    this._flushQueue();
                    
                    if (this.onOpen) {
                        this.onOpen();
                    }
                    resolve();
                };
                
                this.ws.onmessage = (event) => {
                    try {
                        const message = ProtocolClient.decodeMessage(new Uint8Array(event.data));
                        if (this.onMessage) {
                            this.onMessage(message);
                        }
                    } catch (err) {
                        console.error('Failed to decode message:', err);
                    }
                };
                
                this.ws.onclose = (event) => {
                    this.isConnecting = false;
                    this._setState('DISCONNECTED');
                    
                    if (this.onClose) {
                        this.onClose(event);
                    }
                    
                    // Attempt reconnection if not intentionally closed
                    if (!event.wasClean && this.currentRetries < this.maxRetries) {
                        this._scheduleReconnect();
                    }
                };
                
                this.ws.onerror = (error) => {
                    this.isConnecting = false;
                    this._setState('ERROR');
                    
                    if (this.onError) {
                        this.onError(error);
                    }
                    
                    // Reconnection handled by onclose
                };
            } catch (err) {
                this.isConnecting = false;
                this._setState('ERROR');
                reject(err);
            }
        });
    }
    
    /**
     * Schedule a reconnection attempt with exponential backoff
     */
    _scheduleReconnect() {
        this._setState('RECONNECTING');
        const delay = Math.min(this.reconnectDelay * Math.pow(2, this.currentRetries), 30000);
        
        setTimeout(() => {
            this.currentRetries++;
            if (this.currentRetries <= this.maxRetries) {
                this.connect().catch(() => {});
            } else {
                this._setState('DISCONNECTED');
            }
        }, delay);
    }
    
    /**
     * Send a message through the WebSocket connection
     * @param {number} opcode - Message opcode
     * @param {Uint8Array} payload - Message payload
     * @returns {boolean} Success status
     */
    send(opcode, payload) {
        if (this._state !== 'CONNECTED') {
            // Queue message for later
            this.messageQueue.push({ opcode, payload });
            return false;
        }
        
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            this.messageQueue.push({ opcode, payload });
            return false;
        }
        
        try {
            const message = ProtocolClient.encodeMessage(opcode, payload);
            this.ws.send(message);
            return true;
        } catch (err) {
            console.error('Failed to send message:', err);
            return false;
        }
    }
    
    /**
     * Flush queued messages
     */
    _flushQueue() {
        while (this.messageQueue.length > 0) {
            const { opcode, payload } = this.messageQueue.shift();
            this.send(opcode, payload);
        }
    }
    
    /**
     * Close the WebSocket connection
     * @param {number} [code] - Close code
     * @param {string} [reason] - Close reason
     */
    close(code, reason) {
        if (this.ws) {
            this.ws.close(code, reason);
            this.ws = null;
        }
        this._setState('DISCONNECTED');
    }
    
    /**
     * Register a message handler
     * @param {Function} callback - Message handler function
     * @returns {Function} Unsubscribe function
     */
    onMessage(callback) {
        this.onMessage = callback;
        return () => { this.onMessage = null; };
    }
    
    /**
     * Register a state change handler
     * @param {Function} callback - State change handler
     * @returns {Function} Unsubscribe function
     */
    onStateChange(callback) {
        this.onStateChange = callback;
        return () => { this.onStateChange = null; };
    }
}

// Export for different environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Connection;
}
if (typeof window !== 'undefined') {
    window.Connection = Connection;
}
