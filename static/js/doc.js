/**
 * WebOS JavaScript Client
 * 
 * A vanilla JavaScript client for the web-based operating system.
 * Provides WebSocket communication, canvas rendering, and UI shell functionality.
 * 
 * @package webos/client
 * @version 1.0.0
 * @license MIT
 * 
 * @modules
 * - {@link module:connection Connection} - WebSocket connection management
 * - {@link module:display DisplayManager} - Canvas rendering engine
 * - {@link module:input InputManager} - Keyboard/mouse event capture
 * - {@link module:shell Shell} - UI shell and window management
 * - {@link module:state ClientState} - Client state management
 * 
 * @example
 * // Initialize the client
 * const client = new Client({
 *     serverUrl: 'ws://localhost:8080/ws',
 *     canvasId: 'display-canvas'
 * });
 * 
 * // Connect to the server
 * await client.connect();
 * 
 * @see {@link https://github.com/example/webos Project Repository}
 * @see {@link docs/PROTOCOL_SPEC.md Protocol Specification}
 */

/**
 * @typedef {Object} ClientConfig
 * @property {string} serverUrl - WebSocket server URL
 * @property {string} canvasId - Canvas element ID for rendering
 * @property {boolean} [autoConnect=true] - Auto-connect on initialization
 * @property {number} [reconnectDelay=1000] - Initial reconnect delay in ms
 * @property {number} [maxRetries=10] - Maximum reconnection attempts
 */

/**
 * @typedef {Object} ConnectionState
 * @property {string} DISCONNECTED - Not connected
 * @property {string} CONNECTING - Connection in progress
 * @property {string} CONNECTED - Successfully connected
 * @property {string} RECONNECTING - Attempting to reconnect
 * @property {string} ERROR - Connection error occurred
 */

/**
 * @typedef {Object} WindowConfig
 * @property {string} id - Unique window identifier
 * @property {string} title - Window title
 * @property {number} x - X position
 * @property {number} y - Y position
 * @property {number} width - Window width
 * @property {number} height - Window height
 * @property {boolean} [resizable=true] - Allow resize
 * @property {boolean} [minimizable=true] - Allow minimize
 * @property {boolean} [closable=true] - Allow close
 */

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        description: 'WebOS JavaScript Client',
        version: '1.0.0',
        modules: ['connection', 'display', 'input', 'shell', 'state']
    };
}
