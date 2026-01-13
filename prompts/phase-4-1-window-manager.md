# PHASE 4.1: Window Manager

**Phase Context**: Phase 4 implements the user interface. This sub-phase creates the multi-window management system.

**Sub-Phase Objective**: Implement window creation, management, focus handling, decorations, and layout algorithms.

**Prerequisites**: 
- Phase 1.5 (Client Foundation) must be complete

**Integration Point**: Window manager integrates with display rendering and input handling.

---

## IMPLEMENTATION REQUIREMENTS

### Overview

You are implementing a complete window manager for browser-based GUI with floating, tiling, and virtual desktop support.

---

### Directory Structure

```
webos/
├── static/
│   └── js/
│       ├── wm.js               # Window manager
│       ├── window.js           # Window class
│       ├── layout.js           # Layout algorithms
│       ├── desktop.js          # Desktop management
│       └── wm_test.js          # Tests
└── pkg/
    └── wm/
        ├── doc.go              # Package documentation
        ├── manager.go          # Window manager backend
        ├── state.go            # Window state management
        └── wm_test.go          # Tests
```

---

### Core JavaScript Classes

```javascript
class WindowManager {
    constructor(display) {
        this.windows = new Map();
        this.focused = null;
        this.desktops = [new Desktop()];
        this.currentDesktop = 0;
    }
    
    createWindow(config) { /* Create and register window */ }
    closeWindow(id) { /* Close and cleanup */ }
    focusWindow(id) { /* Set focus */ }
    tileWindows() { /* Auto-arrange */ }
    minimizeWindow(id) { /* Minimize to taskbar */ }
    maximizeWindow(id) { /* Maximize/restore */ }
    snapWindow(id, position) { /* Snap to edge/corner */ }
    moveWindowToDesktop(id, desktopIndex) { /* Virtual desktop */ }
}

class Window {
    constructor(id, x, y, width, height, title) {
        this.id = id;
        this.frame = { x, y, width, height };
        this.content = null;
        this.minimized = false;
        this.maximized = false;
        this.decorations = { titleBar: true, border: true };
    }
    
    render(ctx) { /* Draw window and decorations */ }
    handleEvent(event) { /* Process input */ }
}
```

---

### Features

- Floating windows with drag/resize
- Window decorations (title bar, borders, buttons)
- Minimize/maximize/restore
- Window snapping
- Virtual desktops
- Keyboard shortcuts
- Window menu
- Task switcher (Alt+Tab)

---

## Deliverables

- `static/js/wm.js` - Complete window manager
- Window creation and management
- Multiple layout modes
- Virtual desktop support
- Keyboard navigation
