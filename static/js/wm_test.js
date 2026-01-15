/**
 * Window Manager Tests
 * 
 * Tests for the window manager JavaScript components.
 */

// Mock canvas for testing
function createMockCanvas(width = 1920, height = 1080) {
    const canvas = {
        width,
        height,
        style: {},
        getBoundingClientRect: () => ({ left: 0, top: 0, width, height }),
        getContext: () => createMockContext(),
        addEventListener: () => {},
        removeEventListener: () => {}
    };
    return canvas;
}

function createMockContext() {
    return {
        fillRect: () => {},
        fillText: () => {},
        strokeRect: () => {},
        beginPath: () => {},
        arc: () => {},
        fill: () => {},
        stroke: () => {},
        save: () => {},
        restore: () => {},
        scale: () => {},
        translate: () => {},
        rotate: () => {},
        clearRect: () => {}
    };
}

// Test utilities
const assert = {
    equal: (actual, expected, message = '') => {
        if (actual !== expected) {
            throw new Error(`${message}: expected ${expected}, got ${actual}`);
        }
    },
    true: (value, message = '') => {
        if (!value) {
            throw new Error(`${message}: expected true, got ${value}`);
        }
    },
    false: (value, message = '') => {
        if (value) {
            throw new Error(`${message}: expected false, got ${value}`);
        }
    },
    throws: (fn, errorType, message = '') => {
        try {
            fn();
            throw new Error(`${message}: expected to throw`);
        } catch (e) {
            if (errorType && !(e instanceof errorType)) {
                throw new Error(`${message}: expected ${errorType.name}, got ${e.constructor.name}`);
            }
        }
    },
    arrayEqual: (actual, expected, message = '') => {
        if (actual.length !== expected.length) {
            throw new Error(`${message}: arrays have different lengths`);
        }
        for (let i = 0; i < actual.length; i++) {
            if (actual[i] !== expected[i]) {
                throw new Error(`${message}: arrays differ at index ${i}`);
            }
        }
    }
};

// Test runner
class TestRunner {
    constructor() {
        this.tests = [];
        this.passed = 0;
        this.failed = 0;
        this.results = [];
    }

    add(name, fn) {
        this.tests.push({ name, fn });
    }

    async run() {
        console.log('Running Window Manager Tests...\n');

        for (const test of this.tests) {
            try {
                await test.fn();
                console.log(`✓ ${test.name}`);
                this.passed++;
                this.results.push({ name: test.name, passed: true });
            } catch (e) {
                console.log(`✗ ${test.name}: ${e.message}`);
                this.failed++;
                this.results.push({ name: test.name, passed: false, error: e.message });
            }
        }

        console.log(`\n${this.passed} passed, ${this.failed} failed`);
        return this.results;
    }
}

// Window Tests
function testWindow() {
    const runner = new TestRunner();

    runner.add('Window creation', () => {
        const win = new Window({ title: 'Test', x: 100, y: 200, width: 640, height: 480 });
        assert.equal(win.title, 'Test', 'title');
        assert.equal(win.x, 100, 'x');
        assert.equal(win.y, 200, 'y');
        assert.equal(win.width, 640, 'width');
        assert.equal(win.height, 480, 'height');
        assert.equal(win.state, WindowState.NORMAL, 'state');
        assert.true(win.visible, 'visible');
        assert.true(win.resizable, 'resizable');
        assert.true(win.closable, 'closable');
    });

    runner.add('Window setBounds', () => {
        const win = new Window();
        win.setBounds(50, 60, 320, 240);
        assert.equal(win.x, 50, 'x');
        assert.equal(win.y, 60, 'y');
        assert.equal(win.width, 320, 'width');
        assert.equal(win.height, 240, 'height');
    });

    runner.add('Window containsPoint', () => {
        const win = new Window({ x: 100, y: 100, width: 200, height: 150 });
        assert.true(win.containsPoint(150, 175), 'center');
        assert.true(win.containsPoint(100, 100), 'top-left');
        assert.true(win.containsPoint(300, 250), 'bottom-right');
        assert.false(win.containsPoint(99, 100), 'left edge');
        assert.false(win.containsPoint(301, 100), 'right edge');
    });

    runner.add('Window isInTitleBar', () => {
        const win = new Window({ x: 100, y: 100, width: 200, height: 150 });
        assert.true(win.isInTitleBar(150, 115), 'in title bar');
        assert.false(win.isInTitleBar(150, 150), 'below title bar');
    });

    runner.add('Window minimize/restore', () => {
        const win = new Window();
        win.minimize();
        assert.true(win.minimized, 'minimized');
        assert.equal(win.state, WindowState.MINIMIZED, 'state');
        assert.false(win.visible, 'not visible');

        win.restore();
        assert.false(win.minimized, 'not minimized');
        assert.equal(win.state, WindowState.NORMAL, 'state');
        assert.true(win.visible, 'visible');
    });

    runner.add('Window maximize/restore', () => {
        const win = new Window({ x: 100, y: 100, width: 640, height: 480 });
        win.maximize();
        assert.true(win.maximized, 'maximized');
        assert.equal(win.state, WindowState.MAXIMIZED, 'state');

        win.restore();
        assert.false(win.maximized, 'not maximized');
        assert.equal(win.state, WindowState.NORMAL, 'state');
    });

    runner.add('Window resize handle detection', () => {
        const win = new Window({ x: 100, y: 100, width: 200, height: 150 });

        assert.equal(win.getResizeHandle(100, 100, 8), 'nw', 'northwest');
        assert.equal(win.getResizeHandle(300, 100, 8), 'ne', 'northeast');
        assert.equal(win.getResizeHandle(100, 250, 8), 'sw', 'southwest');
        assert.equal(win.getResizeHandle(300, 250, 8), 'se', 'southeast');
        assert.equal(win.getResizeHandle(100, 175, 8), 'w', 'west');
        assert.equal(win.getResizeHandle(300, 175, 8), 'e', 'east');
        assert.equal(win.getResizeHandle(200, 100, 8), 'n', 'north');
        assert.equal(win.getResizeHandle(200, 250, 8), 's', 'south');
        assert.equal(win.getResizeHandle(200, 175, 8), null, 'inside');
    });

    runner.add('Window serialization', () => {
        const win = new Window({ title: 'Test', x: 100, y: 200, width: 640, height: 480 });
        const data = win.serialize();

        assert.equal(data.title, 'Test', 'title');
        assert.equal(data.frame.x, 100, 'x');
        assert.equal(data.frame.y, 200, 'y');
        assert.equal(data.frame.width, 640, 'width');
        assert.equal(data.frame.height, 480, 'height');
        assert.equal(data.state, WindowState.NORMAL, 'state');
    });

    runner.add('Window deserialization', () => {
        const data = {
            id: 'test-win',
            title: 'Deserialized',
            type: 'regular',
            frame: { x: 50, y: 60, width: 320, height: 240 },
            state: 'normal',
            minimized: false,
            maximized: false,
            fullscreen: false,
            visible: true,
            desktop: 0
        };

        const win = Window.deserialize(data);
        assert.equal(win.title, 'Deserialized', 'title');
        assert.equal(win.x, 50, 'x');
        assert.equal(win.y, 60, 'y');
        assert.equal(win.width, 320, 'width');
        assert.equal(win.height, 240, 'height');
    });

    runner.add('Window events', () => {
        const win = new Window();
        let eventFired = false;
        let receivedValue = null;

        win.on('titleChanged', (value) => {
            eventFired = true;
            receivedValue = value;
        });

        win.title = 'New Title';

        assert.true(eventFired, 'event fired');
        assert.equal(receivedValue, 'New Title', 'received value');
    });

    return runner.run();
}

// Desktop Tests
function testDesktop() {
    const runner = new TestRunner();

    runner.add('Desktop creation', () => {
        const desktop = new Desktop(0, 'Main');
        assert.equal(desktop.id, 0, 'id');
        assert.equal(desktop.name, 'Main', 'name');
        assert.equal(desktop.windowCount, 0, 'window count');
    });

    runner.add('Desktop add/remove windows', () => {
        const desktop = new Desktop();
        const win1 = new Window({ id: 'win1' });
        const win2 = new Window({ id: 'win2' });

        desktop.addWindow(win1);
        desktop.addWindow(win2);

        assert.equal(desktop.windowCount, 2, 'window count');

        assert.true(desktop.hasWindow(win1), 'has win1');
        assert.true(desktop.hasWindow(win2), 'has win2');

        desktop.removeWindow(win1);
        assert.equal(desktop.windowCount, 1, 'after remove');
        assert.false(desktop.hasWindow(win1), 'no win1');
    });

    runner.add('Desktop getVisibleWindows', () => {
        const desktop = new Desktop();
        const win1 = new Window({ id: 'win1' });
        const win2 = new Window({ id: 'win2' });
        const win3 = new Window({ id: 'win3', visible: false });

        win2.minimize = true;

        desktop.addWindow(win1);
        desktop.addWindow(win2);
        desktop.addWindow(win3);

        const visible = desktop.getVisibleWindows();
        assert.equal(visible.length, 1, 'one visible');
        assert.equal(visible[0].id, 'win1', 'is win1');
    });

    runner.add('Desktop serialization', () => {
        const desktop = new Desktop(1, 'Work');
        desktop.gridSize = 12;
        desktop.wallpaper = '/wallpapers/test.jpg';

        const data = desktop.serialize();
        assert.equal(data.id, 1, 'id');
        assert.equal(data.name, 'Work', 'name');
        assert.equal(data.gridSize, 12, 'gridSize');
        assert.equal(data.wallpaper, '/wallpapers/test.jpg', 'wallpaper');
    });

    return runner.run();
}

// DesktopManager Tests
function testDesktopManager() {
    const runner = new TestRunner();

    runner.add('DesktopManager creation', () => {
        const manager = new DesktopManager({ initialCount: 3 });
        assert.equal(manager.count, 3, 'desktop count');
        assert.equal(manager.currentIndex, 0, 'current index');
    });

    runner.add('DesktopManager switchDesktop', async () => {
        const manager = new DesktopManager({ initialCount: 3 });

        await manager.switchTo(2);
        assert.equal(manager.currentIndex, 2, 'current index');

        assert.equal(manager.currentDesktop.id, 2, 'current desktop id');
    });

    runner.add('DesktopManager next/previous', async () => {
        const manager = new DesktopManager({ initialCount: 4 });

        await manager.next();
        assert.equal(manager.currentIndex, 1, 'after next');

        await manager.previous();
        assert.equal(manager.currentIndex, 0, 'after previous');
    });

    runner.add('DesktopManager createDesktop', () => {
        const manager = new DesktopManager({ maxDesktops: 4, initialCount: 2 });
        const desktop = manager.createDesktop('Custom');

        assert.equal(manager.count, 3, 'desktop count');
        assert.equal(desktop.name, 'Custom', 'desktop name');
    });

    runner.add('DesktopManager moveWindowToDesktop', () => {
        const manager = new DesktopManager({ initialCount: 3 });
        const window = new Window({ id: 'test-win', desktop: 0 });

        manager.moveWindowToDesktop(window, 2);
        assert.equal(window.desktop, 2, 'window desktop');
    });

    return runner.run();
}

// Layout Tests
function testLayout() {
    const runner = new TestRunner();

    runner.add('FloatingLayout apply', () => {
        const layout = new FloatingLayout();
        const windows = [
            new Window({ id: 'win1', x: 100, y: 100, width: 200, height: 150 }),
            new Window({ id: 'win2', x: 200, y: 200, width: 300, height: 200 })
        ];

        layout.apply(windows, { x: 0, y: 0, width: 1920, height: 1080 });

        // Floating layout doesn't change positions
        assert.equal(windows[0].x, 100, 'win1 x');
        assert.equal(windows[1].x, 200, 'win2 x');
    });

    runner.add('TilingLayout horizontal apply', () => {
        const layout = new TilingLayout('horizontal');
        layout.gap = 10;
        layout.padding = 20;

        const windows = [
            new Window({ id: 'win1' }),
            new Window({ id: 'win2' })
        ];

        layout.apply(windows, { x: 0, y: 0, width: 1000, height: 600 });

        // First window should be master (left side)
        assert.true(windows[0].x < windows[1].x, 'win1 left of win2');
    });

    runner.add('TilingLayout vertical apply', () => {
        const layout = new TilingLayout('vertical');

        const windows = [
            new Window({ id: 'win1' }),
            new Window({ id: 'win2' })
        ];

        layout.apply(windows, { x: 0, y: 0, width: 800, height: 1000 });

        // First window should be master (top)
        assert.true(windows[0].y < windows[1].y, 'win1 above win2');
    });

    runner.add('GridLayout apply', () => {
        const layout = new GridLayout();
        layout.columns = 2;

        const windows = [
            new Window({ id: 'win1' }),
            new Window({ id: 'win2' }),
            new Window({ id: 'win3' }),
            new Window({ id: 'win4' })
        ];

        layout.apply(windows, { x: 0, y: 0, width: 800, height: 600 });

        // All windows should be tiled in a grid
        assert.equal(windows.length, 4, 'all windows processed');
    });

    runner.add('LayoutManager cycleLayout', () => {
        const manager = new LayoutManager();

        const firstLayout = manager.currentLayoutName;
        const secondLayout = manager.cycleLayout();
        const thirdLayout = manager.cycleLayout();

        assert.notEqual(firstLayout, secondLayout, 'layouts differ');
        assert.notEqual(secondLayout, thirdLayout, 'layouts differ');
    });

    return runner.run();
}

// WindowManager Tests
function testWindowManager() {
    const runner = new TestRunner();

    runner.add('WindowManager creation', () => {
        const canvas = createMockCanvas();
        const wm = new WindowManager({ canvas });

        assert.equal(wm.screenWidth, 1920, 'screen width');
        assert.equal(wm.screenHeight, 1080, 'screen height');
        assert.equal(wm.windows.length, 0, 'window count');
    });

    runner.add('WindowManager createWindow', () => {
        const canvas = createMockCanvas();
        const wm = new WindowManager({ canvas });

        const window = wm.createWindow({ title: 'Test Window', x: 100, y: 100, width: 640, height: 480 });

        assert.equal(wm.windows.length, 1, 'window count');
        assert.equal(window.title, 'Test Window', 'title');
        assert.equal(wm.focusedWindow, window, 'focused');
    });

    runner.add('WindowManager closeWindow', () => {
        const canvas = createMockCanvas();
        const wm = new WindowManager({ canvas });

        const window = wm.createWindow({ title: 'Test' });
        const result = wm.closeWindow(window.id);

        assert.true(result, 'close returned true');
        assert.equal(wm.windows.length, 0, 'no windows');
        assert.equal(wm.getWindow(window.id), null, 'getWindow returns null');
    });

    runner.add('WindowManager minimizeWindow', () => {
        const canvas = createMockCanvas();
        const wm = new WindowManager({ canvas });

        const window = wm.createWindow();
        wm.minimizeWindow(window.id);

        assert.true(window.minimized, 'window minimized');
    });

    runner.add('WindowManager maximizeWindow', () => {
        const canvas = createMockCanvas(1920, 1080);
        const wm = new WindowManager({ canvas });

        const window = wm.createWindow({ x: 100, y: 100, width: 640, height: 480 });
        wm.maximizeWindow(window.id);

        assert.true(window.maximized, 'window maximized');
        assert.equal(window.x, 0, 'x is 0');
        assert.equal(window.y, 0, 'y is 0');
        assert.equal(window.width, 1920, 'width is screen width');
    });

    runner.add('WindowManager snapWindow', () => {
        const canvas = createMockCanvas(1920, 1080);
        const wm = new WindowManager({ canvas });

        const window = wm.createWindow({ width: 640, height: 480 });
        wm.snapWindow(window.id, 'left');

        assert.equal(window.x, 0, 'x is 0');
        assert.equal(window.y, 0, 'y is 0');
        assert.equal(window.width, 960, 'width is half');
        assert.equal(window.height, 1080, 'height is full');
    });

    runner.add('WindowManager focusWindow', () => {
        const canvas = createMockCanvas();
        const wm = new WindowManager({ canvas });

        const win1 = wm.createWindow({ id: 'win1' });
        const win2 = wm.createWindow({ id: 'win2' });

        assert.equal(wm.focusedWindow, win2, 'win2 focused');

        wm.focusWindow('win1');
        assert.equal(wm.focusedWindow, win1, 'win1 focused');
    });

    runner.add('WindowManager serialization', () => {
        const canvas = createMockCanvas(1920, 1080);
        const wm = new WindowManager({ canvas });

        wm.createWindow({ title: 'Window 1', x: 100, y: 100, width: 640, height: 480 });
        wm.createWindow({ title: 'Window 2', x: 200, y: 200, width: 500, height: 400 });

        const data = wm.serialize();

        assert.equal(data.windows.length, 2, '2 windows');
        assert.equal(data.windowOrder.length, 2, '2 in order');
        assert.equal(data.focusedWindow, 'win_0_1', 'focused window');
    });

    runner.add('WindowManager deserialization', () => {
        const canvas = createMockCanvas(1920, 1080);
        const wm = new WindowManager({ canvas });

        const data = {
            windows: [
                { id: 'win1', title: 'Window 1', frame: { x: 100, y: 100, width: 640, height: 480 }, state: 'normal', minimized: false, maximized: false, fullscreen: false, visible: true, desktop: 0, flags: {}, zIndex: 100 },
                { id: 'win2', title: 'Window 2', frame: { x: 200, y: 200, width: 500, height: 400 }, state: 'normal', minimized: false, maximized: false, fullscreen: false, visible: true, desktop: 0, flags: {}, zIndex: 101 }
            ],
            windowOrder: ['win1', 'win2'],
            focusedWindow: 'win2',
            desktopManager: { currentIndex: 0, maxDesktops: 4, desktops: [] },
            layoutManager: { currentLayout: 'floating', layouts: {} },
            screenWidth: 1920,
            screenHeight: 1080
        };

        const restored = WindowManager.deserialize(data, { canvas });

        assert.equal(restored.windows.length, 2, '2 windows');
        assert.equal(restored.getWindow('win1').title, 'Window 1', 'win1 title');
        assert.equal(restored.getWindow('win2').title, 'Window 2', 'win2 title');
    });

    runner.add('WindowManager events', () => {
        const canvas = createMockCanvas();
        const wm = new WindowManager({ canvas });

        let eventFired = false;
        let receivedWindow = null;

        wm.on('windowCreated', (window) => {
            eventFired = true;
            receivedWindow = window;
        });

        wm.createWindow({ title: 'Test' });

        assert.true(eventFired, 'event fired');
        assert.equal(receivedWindow.title, 'Test', 'received window');
    });

    return runner.run();
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        createMockCanvas,
        createMockContext,
        assert,
        TestRunner,
        testWindow,
        testDesktop,
        testDesktopManager,
        testLayout,
        testWindowManager
    };
}

// Run tests if in browser
if (typeof window !== 'undefined') {
    window.addEventListener('load', async () => {
        console.log('Running Window Manager Tests...\n');

        await testWindow();
        console.log('');
        await testDesktop();
        console.log('');
        await testDesktopManager();
        console.log('');
        await testLayout();
        console.log('');
        await testWindowManager();
    });
}
