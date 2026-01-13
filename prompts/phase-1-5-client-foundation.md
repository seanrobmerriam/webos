# PHASE 1.5: Client Foundation (JavaScript)

**Phase Context**: Phase 1 builds the communication foundation. This sub-phase implements the browser-based client interface.

**Sub-Phase Objective**: Implement WebSocket client, protocol message handling, canvas rendering foundation, event capture, and basic UI shell.

**Prerequisites**: 
- Phase 1.1 (Protocol) must be complete
- Phase 1.2 (WebSocket) recommended

**Integration Point**: Client connects to WebSocket server, exchanges protocol messages, and renders display output.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing vanilla JavaScript client with WebSocket communication, canvas rendering, and basic UI shell.

---

### Directory Structure

```
webos/
├── static/
│   ├── index.html              # Main HTML page
│   ├── js/
│   │   ├── doc.js              # Package documentation
│   │   ├── protocol.js         # Protocol client (from Phase 1.1)
│   │   ├── connection.js       # WebSocket connection management
│   │   ├── display.js          # Canvas rendering engine
│   │   ├── input.js            # Keyboard/mouse event capture
│   │   ├── shell.js            # UI shell and window chrome
│   │   ├── state.js            # Client state management
│   │   └── client.js           # Main client entry point
│   └── css/
│       └── style.css           # Client styling
└── cmd/
    └── client-demo/
        └── main.go             # Simple HTTP server for client testing
```

---

### Core JavaScript Classes

```javascript
// Connection - WebSocket connection management
class Connection {
    constructor(url) {
        this.url = url;
        this.ws = null;
        this.reconnectDelay = 1000;
        this.maxRetries = 10;
    }
    
    connect() { /* ... */ }
    send(opcode, payload) { /* ... */ }
    onMessage(callback) { /* ... */ }
}

// DisplayManager - Canvas rendering
class DisplayManager {
    constructor(canvas) {
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        this.layers = [];
    }
    
    render(instructions) { /* ... */ }
    clear() { /* ... */ }
}

// InputManager - Event capture and forwarding
class InputManager {
    constructor() {
        this.keyboardState = {};
        this.mouseState = { x: 0, y: 0, buttons: {} };
    }
    
    captureKeyboard() { /* ... */ }
    captureMouse() { /* ... */ }
    sendInput(type, data) { /* ... */ }
}

// Shell - UI shell with window management
class Shell {
    constructor() {
        this.windows = [];
        this.focusedWindow = null;
        this.desktop = null;
    }
    
    createWindow(config) { /* ... */ }
    closeWindow(id) { /* ... */ }
    render() { /* ... */ }
}
```

---

### Implementation Steps

1. **Connection**: WebSocket client with reconnection logic
2. **Protocol Integration**: Message encoding/decoding
3. **Display Manager**: Canvas rendering foundation
4. **Input Manager**: Keyboard/mouse capture and forwarding
5. **UI Shell**: Desktop background, window chrome
6. **State Management**: Client-side state machine

---

### Testing Requirements

- WebSocket reconnection
- Protocol message handling
- Canvas rendering correctness
- Event capture accuracy
- Memory usage

---

### Phase 1 Completion

After Phase 1.5, all Phase 1 milestones should be met:
- Client connects securely via WebSocket
- Protocol messages flow bidirectionally
- Authentication and authorization functional
- Basic rendering works in browser
- 80%+ test coverage

---

## Deliverables

- `static/js/` - Complete JavaScript client
- `static/index.html` - Main HTML page
- `static/css/style.css` - Client styling
- Connection to server demonstration
- Canvas rendering demonstration
- Event capture demonstration
