/**
 * Graphics Library Tests
 * 
 * Comprehensive test suite for the Graphics library and Shapes module.
 * Tests can run in Node.js (with jsdom) or directly in the browser.
 * 
 * @version 1.0.0
 * @license MIT
 */

// Mock canvas for Node.js testing environment
function createMockCanvas(width = 800, height = 600) {
    const canvas = {
        width: width,
        height: height,
        tagName: 'CANVAS',
        getContext: function(type) {
            if (type !== '2d') return null;
            return createMockContext();
        },
        toDataURL: function() {
            return 'data:image/png;base64,mock';
        }
    };
    return canvas;
}

function createMockContext() {
    const ctx = {
        fillStyle: '#000000',
        strokeStyle: '#000000',
        lineWidth: 1,
        font: '12px Arial',
        globalAlpha: 1,
        globalCompositeOperation: 'source-over',
        lineCap: 'butt',
        lineJoin: 'miter',
        miterLimit: 10,
        shadowColor: 'transparent',
        shadowBlur: 0,
        shadowOffsetX: 0,
        shadowOffsetY: 0,
        textAlign: 'left',
        textBaseline: 'alphabetic',
        save: function() {},
        restore: function() {},
        fillRect: function() {},
        strokeRect: function() {},
        clearRect: function() {},
        beginPath: function() {},
        moveTo: function() {},
        lineTo: function() {},
        arc: function() {},
        fill: function() {},
        stroke: function() {},
        translate: function() {},
        rotate: function() {},
        scale: function() {},
        setTransform: function() {},
        clip: function() {},
        fillText: function() {},
        strokeText: function() {},
        measureText: function(text) {
            return { width: text.length * 8 };
        },
        drawImage: function() {},
        ellipse: function() {},
        quadraticCurveTo: function() {},
        bezierCurveTo: function() {},
        closePath: function() {},
        getImageData: function() {
            return { data: new Uint8ClampedArray(0) };
        },
        putImageData: function() {},
        rect: function() {}
    };
    return ctx;
}

// Test utilities
const TestRunner = {
    passed: 0,
    failed: 0,
    results: [],
    
    assert(condition, message) {
        if (!condition) {
            throw new Error(`Assertion failed: ${message}`);
        }
    },
    
    assertEqual(actual, expected, message) {
        if (actual !== expected) {
            throw new Error(`${message || 'Assertion failed'}: expected ${expected}, got ${actual}`);
        }
    },
    
    assertThrows(fn, message) {
        let threw = false;
        try {
            fn();
        } catch (e) {
            threw = true;
        }
        if (!threw) {
            throw new Error(message || 'Expected function to throw');
        }
    },
    
    test(name, fn) {
        try {
            fn();
            this.passed++;
            this.results.push({ name, status: 'PASS' });
            console.log(`✓ ${name}`);
        } catch (e) {
            this.failed++;
            this.results.push({ name, status: 'FAIL', error: e.message });
            console.error(`✗ ${name}: ${e.message}`);
        }
    },
    
    summary() {
        console.log(`\n--- Test Summary ---`);
        console.log(`Passed: ${this.passed}`);
        console.log(`Failed: ${this.failed}`);
        console.log(`Total: ${this.passed + this.failed}`);
        return this.failed === 0;
    }
};

// Graphics Class Tests
function runGraphicsTests() {
    console.log('\n=== Graphics Class Tests ===\n');
    
    TestRunner.test('Graphics constructor creates instance', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        TestRunner.assert(g !== null, 'Graphics should not be null');
        TestRunner.assert(g.ctx !== null, 'Context should not be null');
    });
    
    TestRunner.test('Graphics constructor throws on null canvas', () => {
        TestRunner.assertThrows(() => new Graphics(null), 'Should throw on null canvas');
    });
    
    TestRunner.test('Graphics constructor throws on non-canvas element', () => {
        TestRunner.assertThrows(() => new Graphics({ tagName: 'DIV' }), 'Should throw on non-canvas');
    });
    
    TestRunner.test('Graphics drawRect draws filled rectangle', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawRect(10, 10, 100, 50, '#ff0000');
    });
    
    TestRunner.test('Graphics drawRectOutline draws rectangle outline', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawRectOutline(10, 10, 100, 50, '#000000', 2);
    });
    
    TestRunner.test('Graphics drawCircle draws filled circle', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawCircle(100, 100, 50, '#00ff00');
    });
    
    TestRunner.test('Graphics drawCircleOutline draws circle outline', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawCircleOutline(100, 100, 50, '#000000', 2);
    });
    
    TestRunner.test('Graphics drawLine draws line segment', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawLine(0, 0, 100, 100, '#000000', 2);
    });
    
    TestRunner.test('Graphics drawPolyline draws connected lines', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        const points = [{ x: 0, y: 0 }, { x: 50, y: 50 }, { x: 100, y: 0 }];
        g.drawPolyline(points, '#000000', 2, false);
    });
    
    TestRunner.test('Graphics drawPolygon draws filled polygon', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        const points = [{ x: 50, y: 0 }, { x: 100, y: 100 }, { x: 0, y: 100 }];
        g.drawPolygon(points, '#0000ff');
    });
    
    TestRunner.test('Graphics drawText renders text', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawText('Hello', 10, 10, '14px Arial', '#000000');
    });
    
    TestRunner.test('Graphics drawText with options', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawText('Centered', 100, 50, '16px Arial', '#000000', {
            textAlign: 'center',
            textBaseline: 'middle'
        });
    });
    
    TestRunner.test('Graphics measureText returns metrics', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        const metrics = g.measureText('Test', '12px Arial');
        TestRunner.assert(metrics.width > 0, 'Should return valid width');
    });
    
    TestRunner.test('Graphics translate applies translation', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.translate(50, 100);
        const transform = g.getTransform();
        TestRunner.assertEqual(transform.translateX, 50, 'X translation');
        TestRunner.assertEqual(transform.translateY, 100, 'Y translation');
    });
    
    TestRunner.test('Graphics rotate applies rotation', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.rotate(Math.PI / 2);
        const transform = g.getTransform();
        TestRunner.assertEqual(transform.rotation, Math.PI / 2, 'Rotation angle');
    });
    
    TestRunner.test('Graphics scale applies scaling', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.scale(2, 3);
        const transform = g.getTransform();
        TestRunner.assertEqual(transform.scaleX, 2, 'X scale');
        TestRunner.assertEqual(transform.scaleY, 3, 'Y scale');
    });
    
    TestRunner.test('Graphics resetTransform clears transforms', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.translate(50, 50);
        g.rotate(Math.PI / 2);
        g.scale(2, 2);
        g.resetTransform();
        const transform = g.getTransform();
        TestRunner.assertEqual(transform.translateX, 0, 'X translation reset');
        TestRunner.assertEqual(transform.translateY, 0, 'Y translation reset');
        TestRunner.assertEqual(transform.rotation, 0, 'Rotation reset');
        TestRunner.assertEqual(transform.scaleX, 1, 'X scale reset');
        TestRunner.assertEqual(transform.scaleY, 1, 'Y scale reset');
    });
    
    TestRunner.test('Graphics clip sets clipping region', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.clip(0, 0, 100, 100);
    });
    
    TestRunner.test('Graphics clipPolygon sets polygon clip', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        const points = [{ x: 50, y: 0 }, { x: 100, y: 100 }, { x: 0, y: 100 }];
        g.clipPolygon(points);
    });
    
    TestRunner.test('Graphics save and restore state', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.translate(50, 50);
        g.setGlobalAlpha(0.5);
        g.save();
        g.translate(100, 100);
        g.setGlobalAlpha(0.8);
        g.restore();
        const transform = g.getTransform();
        TestRunner.assertEqual(transform.translateX, 50, 'Restored X translation');
    });
    
    TestRunner.test('Graphics setGlobalAlpha sets opacity', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.setGlobalAlpha(0.5);
    });
    
    TestRunner.test('Graphics setGlobalCompositeOperation sets blend mode', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.setGlobalCompositeOperation('multiply');
    });
    
    TestRunner.test('Graphics setShadow and clearShadow', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.setShadow('#000000', 5, 2, 2);
        g.clearShadow();
    });
    
    TestRunner.test('Graphics drawEllipse draws ellipse', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawEllipse(100, 100, 50, 30, 0, 0, Math.PI * 2, '#ff0000');
    });
    
    TestRunner.test('Graphics drawRoundedRect draws rounded rectangle', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.drawRoundedRect(10, 10, 100, 50, 10, '#00ff00');
    });
    
    TestRunner.test('Graphics clearRect clears region', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.clearRect(0, 0, 100, 100);
    });
    
    TestRunner.test('Graphics clear clears canvas', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.clear();
    });
    
    TestRunner.test('Graphics fill fills canvas', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.fill('#ffffff');
    });
    
    TestRunner.test('Graphics getCanvas returns canvas', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        TestRunner.assert(g.getCanvas() === canvas, 'Should return same canvas');
    });
    
    TestRunner.test('Graphics getContext returns context', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        TestRunner.assert(g.getContext() !== null, 'Should return context');
    });
    
    TestRunner.test('Graphics getDimensions returns size', () => {
        const canvas = createMockCanvas(800, 600);
        const g = new Graphics(canvas);
        const dims = g.getDimensions();
        TestRunner.assertEqual(dims.width, 800, 'Width');
        TestRunner.assertEqual(dims.height, 600, 'Height');
    });
    
    TestRunner.test('Graphics resize changes size', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        g.resize(1024, 768);
        const dims = g.getDimensions();
        TestRunner.assertEqual(dims.width, 1024, 'New width');
        TestRunner.assertEqual(dims.height, 768, 'New height');
    });
    
    TestRunner.test('Graphics toDataURL returns data URL', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        const dataURL = g.toDataURL();
        TestRunner.assert(dataURL.startsWith('data:'), 'Should return data URL');
    });
}

// Shapes Class Tests
function runShapesTests() {
    console.log('\n=== Shapes Class Tests ===\n');
    
    TestRunner.test('Rect constructor creates rectangle', () => {
        const rect = new Rect(10, 20, 100, 50);
        TestRunner.assertEqual(rect.x, 10, 'X position');
        TestRunner.assertEqual(rect.y, 20, 'Y position');
        TestRunner.assertEqual(rect.width, 100, 'Width');
        TestRunner.assertEqual(rect.height, 50, 'Height');
    });
    
    TestRunner.test('Rect with options', () => {
        const rect = new Rect(10, 20, 100, 50, { fill: '#ff0000', stroke: '#000000', strokeWidth: 2 });
        TestRunner.assertEqual(rect.fill, '#ff0000', 'Fill color');
        TestRunner.assertEqual(rect.stroke, '#000000', 'Stroke color');
        TestRunner.assertEqual(rect.strokeWidth, 2, 'Stroke width');
    });
    
    TestRunner.test('Rect getBounds returns bounds', () => {
        const rect = new Rect(10, 20, 100, 50);
        const bounds = rect.getBounds();
        TestRunner.assertEqual(bounds.x, 10, 'Bounds X');
        TestRunner.assertEqual(bounds.y, 20, 'Bounds Y');
        TestRunner.assertEqual(bounds.width, 100, 'Bounds width');
        TestRunner.assertEqual(bounds.height, 50, 'Bounds height');
    });
    
    TestRunner.test('Rect containsPoint checks point', () => {
        const rect = new Rect(0, 0, 100, 50);
        TestRunner.assert(rect.containsPoint(50, 25), 'Point inside');
        TestRunner.assert(!rect.containsPoint(150, 25), 'Point outside X');
        TestRunner.assert(!rect.containsPoint(50, 100), 'Point outside Y');
    });
    
    TestRunner.test('Circle constructor creates circle', () => {
        const circle = new Circle(100, 100, 50);
        TestRunner.assertEqual(circle.x, 100, 'X center');
        TestRunner.assertEqual(circle.y, 100, 'Y center');
        TestRunner.assertEqual(circle.radius, 50, 'Radius');
    });
    
    TestRunner.test('Circle getBounds returns bounds', () => {
        const circle = new Circle(100, 100, 50);
        const bounds = circle.getBounds();
        TestRunner.assertEqual(bounds.x, 50, 'Bounds X');
        TestRunner.assertEqual(bounds.y, 50, 'Bounds Y');
        TestRunner.assertEqual(bounds.width, 100, 'Bounds width');
        TestRunner.assertEqual(bounds.height, 100, 'Bounds height');
    });
    
    TestRunner.test('Circle containsPoint checks point', () => {
        const circle = new Circle(100, 100, 50);
        TestRunner.assert(circle.containsPoint(100, 100), 'Center point');
        TestRunner.assert(circle.containsPoint(150, 100), 'Edge point');
        TestRunner.assert(!circle.containsPoint(200, 100), 'Outside point');
    });
    
    TestRunner.test('Line constructor creates line', () => {
        const line = new Line(0, 0, 100, 100);
        TestRunner.assertEqual(line.x1, 0, 'Start X');
        TestRunner.assertEqual(line.y1, 0, 'Start Y');
        TestRunner.assertEqual(line.x2, 100, 'End X');
        TestRunner.assertEqual(line.y2, 100, 'End Y');
    });
    
    TestRunner.test('Polygon constructor creates polygon', () => {
        const points = [{ x: 0, y: 0 }, { x: 100, y: 0 }, { x: 50, y: 100 }];
        const poly = new Polygon(points);
        TestRunner.assertEqual(poly.points.length, 3, 'Point count');
    });
    
    TestRunner.test('Polygon addPoint adds vertex', () => {
        const poly = new Polygon([]);
        poly.addPoint(0, 0);
        poly.addPoint(100, 0);
        TestRunner.assertEqual(poly.points.length, 2, 'Added points');
    });
    
    TestRunner.test('Polygon containsPoint checks point', () => {
        const triangle = new Polygon([
            { x: 50, y: 0 },
            { x: 100, y: 100 },
            { x: 0, y: 100 }
        ]);
        TestRunner.assert(triangle.containsPoint(50, 50), 'Inside triangle');
        TestRunner.assert(!triangle.containsPoint(0, 0), 'Outside triangle');
    });
    
    TestRunner.test('RoundedRect constructor creates rounded rect', () => {
        const rect = new RoundedRect(10, 10, 100, 50, 10);
        TestRunner.assertEqual(rect.radius, 10, 'Corner radius');
    });
    
    TestRunner.test('Text constructor creates text shape', () => {
        const text = new Text('Hello', 10, 20, { font: '14px Arial' });
        TestRunner.assertEqual(text.text, 'Hello', 'Text content');
        TestRunner.assertEqual(text.font, '14px Arial', 'Font');
    });
    
    TestRunner.test('Shapes factory creates shapes', () => {
        const rect = Shapes.rect(0, 0, 100, 50);
        TestRunner.assert(rect instanceof Rect, 'Factory creates Rect');
        
        const circle = Shapes.circle(100, 100, 50);
        TestRunner.assert(circle instanceof Circle, 'Factory creates Circle');
        
        const line = Shapes.line(0, 0, 100, 100);
        TestRunner.assert(line instanceof Line, 'Factory creates Line');
    });
    
    TestRunner.test('Shapes regularPolygon creates regular polygon', () => {
        const hex = Shapes.regularPolygon(100, 100, 6, 50);
        TestRunner.assertEqual(hex.points.length, 6, 'Hexagon has 6 points');
    });
    
    TestRunner.test('Shapes star creates star shape', () => {
        const star = Shapes.star(100, 100, 5, 50, 25);
        TestRunner.assertEqual(star.points.length, 10, '5-point star has 10 points');
    });
    
    TestRunner.test('Shapes triangle creates triangle', () => {
        const tri = Shapes.triangle(50, 0, 100, 100, 0, 100);
        TestRunner.assertEqual(tri.points.length, 3, 'Triangle has 3 points');
    });
}

// Integration Tests
function runIntegrationTests() {
    console.log('\n=== Integration Tests ===\n');
    
    TestRunner.test('Graphics and Shapes work together', () => {
        const canvas = createMockCanvas(800, 600);
        const g = new Graphics(canvas);
        
        const rect = new Rect(50, 50, 200, 100, { fill: '#3498db' });
        const circle = new Circle(400, 300, 75, { fill: '#e74c3c' });
        
        rect.draw(g);
        circle.draw(g);
    });
    
    TestRunner.test('Multiple transformations applied correctly', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        
        g.save();
        g.translate(100, 100);
        g.rotate(Math.PI / 4);
        g.scale(2, 2);
        g.drawRect(0, 0, 50, 50, '#ff0000');
        g.restore();
        
        const transform = g.getTransform();
        TestRunner.assertEqual(transform.translateX, 0, 'Translation reset after restore');
    });
    
    TestRunner.test('Shape visibility toggle', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        
        const rect = new Rect(10, 10, 100, 50, { fill: '#ff0000', visible: false });
        rect.draw(g); // Should not draw
    });
    
    TestRunner.test('Shape alpha transparency', () => {
        const canvas = createMockCanvas();
        const g = new Graphics(canvas);
        
        const rect = new Rect(10, 10, 100, 50, { fill: '#ff0000', alpha: 0.5 });
        rect.draw(g);
    });
}

// Run all tests
function runAllTests() {
    console.log('========================================');
    console.log('Graphics Library Test Suite');
    console.log('========================================');
    
    runGraphicsTests();
    runShapesTests();
    runIntegrationTests();
    
    const success = TestRunner.summary();
    return success;
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = {
        createMockCanvas,
        createMockContext,
        runAllTests,
        runGraphicsTests,
        runShapesTests,
        runIntegrationTests,
        TestRunner
    };
}
