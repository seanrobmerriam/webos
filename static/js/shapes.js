/**
 * Shapes - Shape Primitives for Graphics Library
 * 
 * Provides a collection of reusable shape classes and utilities
 * for 2D graphics rendering.
 * 
 * @version 1.0.0
 * @license MIT
 */

/**
 * Base Shape class that all shapes inherit from.
 * 
 * @abstract
 */
class Shape {
    /**
     * Creates a new Shape.
     * 
     * @param {Object} options - Shape options
     * @param {string} [options.fill] - Fill color
     * @param {string} [options.stroke] - Stroke color
     * @param {number} [options.strokeWidth=1] - Stroke width
     */
    constructor(options = {}) {
        this.fill = options.fill || null;
        this.stroke = options.stroke || null;
        this.strokeWidth = options.strokeWidth || 1;
        this.alpha = options.alpha !== undefined ? options.alpha : 1;
        this.visible = options.visible !== undefined ? options.visible : true;
    }

    /**
     * Renders the shape using a Graphics context.
     * 
     * @param {Graphics} g - The Graphics context
     * @abstract
     */
    draw(g) {
        throw new Error('draw() must be implemented by subclass');
    }

    /**
     * Gets the bounding box of the shape.
     * 
     * @returns {Object} Bounding box with x, y, width, height
     * @abstract
     */
    getBounds() {
        throw new Error('getBounds() must be implemented by subclass');
    }

    /**
     * Checks if a point is inside the shape.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @returns {boolean} True if point is inside
     * @abstract
     */
    containsPoint(x, y) {
        throw new Error('containsPoint() must be implemented by subclass');
    }
}

/**
 * Rectangle shape.
 */
class Rect extends Shape {
    /**
     * Creates a new Rectangle.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} width - The width
     * @param {number} height - The height
     * @param {Object} [options] - Shape options
     */
    constructor(x, y, width, height, options = {}) {
        super(options);
        this.x = x;
        this.y = y;
        this.width = width;
        this.height = height;
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        
        if (this.fill) {
            g.drawRect(this.x, this.y, this.width, this.height, this.fill);
        }
        if (this.stroke) {
            g.drawRectOutline(this.x, this.y, this.width, this.height, this.stroke, this.strokeWidth);
        }
        
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        return {
            x: this.x,
            y: this.y,
            width: this.width,
            height: this.height
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        return x >= this.x && x <= this.x + this.width &&
               y >= this.y && y <= this.y + this.height;
    }
}

/**
 * Circle shape.
 */
class Circle extends Shape {
    /**
     * Creates a new Circle.
     * 
     * @param {number} x - The x-coordinate of center
     * @param {number} y - The y-coordinate of center
     * @param {number} radius - The radius
     * @param {Object} [options] - Shape options
     */
    constructor(x, y, radius, options = {}) {
        super(options);
        this.x = x;
        this.y = y;
        this.radius = radius;
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        
        if (this.fill) {
            g.drawCircle(this.x, this.y, this.radius, this.fill);
        }
        if (this.stroke) {
            g.drawCircleOutline(this.x, this.y, this.radius, this.stroke, this.strokeWidth);
        }
        
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        return {
            x: this.x - this.radius,
            y: this.y - this.radius,
            width: this.radius * 2,
            height: this.radius * 2
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        const dx = x - this.x;
        const dy = y - this.y;
        return (dx * dx + dy * dy) <= (this.radius * this.radius);
    }
}

/**
 * Ellipse shape.
 */
class Ellipse extends Shape {
    /**
     * Creates a new Ellipse.
     * 
     * @param {number} x - The x-coordinate of center
     * @param {number} y - The y-coordinate of center
     * @param {number} radiusX - The x-radius
     * @param {number} radiusY - The y-radius
     * @param {number} [rotation=0] - Rotation in radians
     * @param {Object} [options] - Shape options
     */
    constructor(x, y, radiusX, radiusY, rotation = 0, options = {}) {
        super(options);
        this.x = x;
        this.y = y;
        this.radiusX = radiusX;
        this.radiusY = radiusY;
        this.rotation = rotation;
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        
        if (this.fill) {
            g.drawEllipse(this.x, this.y, this.radiusX, this.radiusY, 
                         this.rotation, 0, Math.PI * 2, this.fill);
        }
        if (this.stroke) {
            g.drawEllipse(this.x, this.y, this.radiusX, this.radiusY, 
                         this.rotation, 0, Math.PI * 2, this.stroke, true);
        }
        
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        const maxRadius = Math.max(this.radiusX, this.radiusY);
        return {
            x: this.x - maxRadius,
            y: this.y - maxRadius,
            width: maxRadius * 2,
            height: maxRadius * 2
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        const cos = Math.cos(this.rotation);
        const sin = Math.sin(this.rotation);
        const dx = x - this.x;
        const dy = y - this.y;
        const rx = dx * cos + dy * sin;
        const ry = dy * cos - dx * sin;
        return (rx * rx) / (this.radiusX * this.radiusX) + 
               (ry * ry) / (this.radiusY * this.radiusY) <= 1;
    }
}

/**
 * Line shape.
 */
class Line extends Shape {
    /**
     * Creates a new Line.
     * 
     * @param {number} x1 - The x-coordinate of start
     * @param {number} y1 - The y-coordinate of start
     * @param {number} x2 - The x-coordinate of end
     * @param {number} y2 - The y-coordinate of end
     * @param {Object} [options] - Shape options
     */
    constructor(x1, y1, x2, y2, options = {}) {
        super(options);
        this.x1 = x1;
        this.y1 = y1;
        this.x2 = x2;
        this.y2 = y2;
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        g.drawLine(this.x1, this.y1, this.x2, this.y2, this.stroke, this.strokeWidth);
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        const minX = Math.min(this.x1, this.x2);
        const minY = Math.min(this.y1, this.y2);
        const maxX = Math.max(this.x1, this.x2);
        const maxY = Math.max(this.y1, this.y2);
        return {
            x: minX,
            y: minY,
            width: maxX - minX,
            height: maxY - minY
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        const dx = this.x2 - this.x1;
        const dy = this.y2 - this.y1;
        const lengthSquared = dx * dx + dy * dy;
        
        if (lengthSquared === 0) {
            const dist = Math.sqrt((x - this.x1) ** 2 + (y - this.y1) ** 2);
            return dist <= this.strokeWidth;
        }
        
        const t = Math.max(0, Math.min(1, 
            ((x - this.x1) * dx + (y - this.y1) * dy) / lengthSquared));
        const nearestX = this.x1 + t * dx;
        const nearestY = this.y1 + t * dy;
        const dist = Math.sqrt((x - nearestX) ** 2 + (y - nearestY) ** 2);
        
        return dist <= this.strokeWidth;
    }
}

/**
 * Polygon shape.
 */
class Polygon extends Shape {
    /**
     * Creates a new Polygon.
     * 
     * @param {Array<{x: number, y: number}>} points - Array of vertices
     * @param {Object} [options] - Shape options
     */
    constructor(points, options = {}) {
        super(options);
        this.points = points || [];
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible || this.points.length < 3) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        
        if (this.fill) {
            g.drawPolygon(this.points, this.fill);
        }
        if (this.stroke) {
            g.drawPolyline(this.points, this.stroke, this.strokeWidth, true);
        }
        
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        if (this.points.length === 0) {
            return { x: 0, y: 0, width: 0, height: 0 };
        }
        
        let minX = Infinity, minY = Infinity;
        let maxX = -Infinity, maxY = -Infinity;
        
        for (const point of this.points) {
            minX = Math.min(minX, point.x);
            minY = Math.min(minY, point.y);
            maxX = Math.max(maxX, point.x);
            maxY = Math.max(maxY, point.y);
        }
        
        return {
            x: minX,
            y: minY,
            width: maxX - minX,
            height: maxY - minY
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        if (this.points.length < 3) return false;
        
        let inside = false;
        for (let i = 0, j = this.points.length - 1; 
             i < this.points.length; j = i++) {
            const xi = this.points[i].x, yi = this.points[i].y;
            const xj = this.points[j].x, yj = this.points[j].y;
            
            if (((yi > y) !== (yj > y)) && 
                (x < (xj - xi) * (y - yi) / (yj - yi) + xi)) {
                inside = !inside;
            }
        }
        return inside;
    }

    /**
     * Adds a point to the polygon.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @returns {Polygon} Returns this for chaining
     */
    addPoint(x, y) {
        this.points.push({ x, y });
        return this;
    }
}

/**
 * Rounded Rectangle shape.
 */
class RoundedRect extends Shape {
    /**
     * Creates a new RoundedRect.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} width - The width
     * @param {number} height - The height
     * @param {number} [radius=5] - The corner radius
     * @param {Object} [options] - Shape options
     */
    constructor(x, y, width, height, radius = 5, options = {}) {
        super(options);
        this.x = x;
        this.y = y;
        this.width = width;
        this.height = height;
        this.radius = radius;
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        
        if (this.fill) {
            g.drawRoundedRect(this.x, this.y, this.width, this.height, this.radius, this.fill);
        }
        if (this.stroke) {
            g.drawRoundedRect(this.x, this.y, this.width, this.height, this.radius, this.stroke, true);
        }
        
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        return {
            x: this.x,
            y: this.y,
            width: this.width,
            height: this.height
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        return x >= this.x && x <= this.x + this.width &&
               y >= this.y && y <= this.y + this.height;
    }
}

/**
 * Text shape.
 */
class Text extends Shape {
    /**
     * Creates a new Text shape.
     * 
     * @param {string} text - The text content
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {Object} [options] - Shape options
     * @param {string} [options.font] - CSS font specification
     * @param {string} [options.textAlign='left'] - Text alignment
     * @param {string} [options.textBaseline='top'] - Text baseline
     */
    constructor(text, x, y, options = {}) {
        super(options);
        this.text = text;
        this.x = x;
        this.y = y;
        this.font = options.font || '12px Arial';
        this.textAlign = options.textAlign || 'left';
        this.textBaseline = options.textBaseline || 'top';
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        g.drawText(this.text, this.x, this.y, this.font, this.fill, {
            textAlign: this.textAlign,
            textBaseline: this.textBaseline
        });
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        const metrics = g.measureText(this.text, this.font);
        return {
            x: this.x,
            y: this.y,
            width: metrics.width,
            height: 12 // Approximate height
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        const bounds = this.getBounds();
        return x >= bounds.x && x <= bounds.x + bounds.width &&
               y >= bounds.y && y <= bounds.y + bounds.height;
    }
}

/**
 * Image shape.
 */
class ImageShape extends Shape {
    /**
     * Creates a new ImageShape.
     * 
     * @param {HTMLImageElement|HTMLCanvasElement} image - The image
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} [width] - The width (optional)
     * @param {number} [height] - The height (optional)
     * @param {Object} [options] - Shape options
     */
    constructor(image, x, y, width, height, options = {}) {
        super(options);
        this.image = image;
        this.x = x;
        this.y = y;
        this.width = width;
        this.height = height;
    }

    /**
     * @inheritdoc
     */
    draw(g) {
        if (!this.visible || !this.image) return;
        
        g.save();
        g.setGlobalAlpha(this.alpha);
        g.drawImage(this.image, this.x, this.y, this.width, this.height);
        g.restore();
    }

    /**
     * @inheritdoc
     */
    getBounds() {
        const imgWidth = this.width || this.image.width;
        const imgHeight = this.height || this.image.height;
        return {
            x: this.x,
            y: this.y,
            width: imgWidth,
            height: imgHeight
        };
    }

    /**
     * @inheritdoc
     */
    containsPoint(x, y) {
        const bounds = this.getBounds();
        return x >= bounds.x && x <= bounds.x + bounds.width &&
               y >= bounds.y && y <= bounds.y + bounds.height;
    }
}

/**
 * Shape factory for creating common shapes.
 */
const Shapes = {
    /**
     * Creates a rectangle.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} width - The width
     * @param {number} height - The height
     * @param {Object} [options] - Shape options
     * @returns {Rect}
     */
    rect(x, y, width, height, options) {
        return new Rect(x, y, width, height, options);
    },

    /**
     * Creates a circle.
     * 
     * @param {number} x - The x-coordinate of center
     * @param {number} y - The y-coordinate of center
     * @param {number} radius - The radius
     * @param {Object} [options] - Shape options
     * @returns {Circle}
     */
    circle(x, y, radius, options) {
        return new Circle(x, y, radius, options);
    },

    /**
     * Creates an ellipse.
     * 
     * @param {number} x - The x-coordinate of center
     * @param {number} y - The y-coordinate of center
     * @param {number} radiusX - The x-radius
     * @param {number} radiusY - The y-radius
     * @param {number} [rotation=0] - Rotation in radians
     * @param {Object} [options] - Shape options
     * @returns {Ellipse}
     */
    ellipse(x, y, radiusX, radiusY, rotation, options) {
        return new Ellipse(x, y, radiusX, radiusY, rotation, options);
    },

    /**
     * Creates a line.
     * 
     * @param {number} x1 - The x-coordinate of start
     * @param {number} y1 - The y-coordinate of start
     * @param {number} x2 - The x-coordinate of end
     * @param {number} y2 - The y-coordinate of end
     * @param {Object} [options] - Shape options
     * @returns {Line}
     */
    line(x1, y1, x2, y2, options) {
        return new Line(x1, y1, x2, y2, options);
    },

    /**
     * Creates a polygon.
     * 
     * @param {Array<{x: number, y: number}>} points - Array of vertices
     * @param {Object} [options] - Shape options
     * @returns {Polygon}
     */
    polygon(points, options) {
        return new Polygon(points, options);
    },

    /**
     * Creates a rounded rectangle.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} width - The width
     * @param {number} height - The height
     * @param {number} [radius=5] - The corner radius
     * @param {Object} [options] - Shape options
     * @returns {RoundedRect}
     */
    roundedRect(x, y, width, height, radius, options) {
        return new RoundedRect(x, y, width, height, radius, options);
    },

    /**
     * Creates a text shape.
     * 
     * @param {string} text - The text content
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {Object} [options] - Shape options
     * @returns {Text}
     */
    text(text, x, y, options) {
        return new Text(text, x, y, options);
    },

    /**
     * Creates an image shape.
     * 
     * @param {HTMLImageElement|HTMLCanvasElement} image - The image
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} [width] - The width
     * @param {number} [height] - The height
     * @param {Object} [options] - Shape options
     * @returns {ImageShape}
     */
    image(image, x, y, width, height, options) {
        return new ImageShape(image, x, y, width, height, options);
    },

    /**
     * Creates a regular polygon.
     * 
     * @param {number} x - The x-coordinate of center
     * @param {number} y - The y-coordinate of center
     * @param {number} sides - Number of sides
     * @param {number} radius - The radius
     * @param {Object} [options] - Shape options
     * @returns {Polygon}
     */
    regularPolygon(x, y, sides, radius, options) {
        const points = [];
        const angleStep = (Math.PI * 2) / sides;
        for (let i = 0; i < sides; i++) {
            const angle = i * angleStep - Math.PI / 2;
            points.push({
                x: x + Math.cos(angle) * radius,
                y: y + Math.sin(angle) * radius
            });
        }
        return new Polygon(points, options);
    },

    /**
     * Creates a star shape.
     * 
     * @param {number} cx - The x-coordinate of center
     * @param {number} cy - The y-coordinate of center
     * @param {number} points - Number of points
     * @param {number} outerRadius - The outer radius
     * @param {number} innerRadius - The inner radius
     * @param {Object} [options] - Shape options
     * @returns {Polygon}
     */
    star(cx, cy, points, outerRadius, innerRadius, options) {
        const starPoints = [];
        const step = Math.PI / points;
        for (let i = 0; i < points * 2; i++) {
            const radius = i % 2 === 0 ? outerRadius : innerRadius;
            const angle = i * step - Math.PI / 2;
            starPoints.push({
                x: cx + Math.cos(angle) * radius,
                y: cy + Math.sin(angle) * radius
            });
        }
        return new Polygon(starPoints, options);
    },

    /**
     * Creates a triangle.
     * 
     * @param {number} x1 - First vertex x
     * @param {number} y1 - First vertex y
     * @param {number} x2 - Second vertex x
     * @param {number} y2 - Second vertex y
     * @param {number} x3 - Third vertex x
     * @param {number} y3 - Third vertex y
     * @param {Object} [options] - Shape options
     * @returns {Polygon}
     */
    triangle(x1, y1, x2, y2, x3, y3, options) {
        return new Polygon([
            { x: x1, y: y1 },
            { x: x2, y: y2 },
            { x: x3, y: y3 }
        ], options);
    },

    /**
     * Creates an arc shape.
     * 
     * @param {number} x - The x-coordinate of center
     * @param {number} y - The y-coordinate of center
     * @param {number} radius - The radius
     * @param {number} startAngle - Start angle in radians
     * @param {number} endAngle - End angle in radians
     * @param {Object} [options] - Shape options
     * @returns {Polygon}
     */
    arc(x, y, radius, startAngle, endAngle, options) {
        const points = [{ x, y }];
        const steps = Math.ceil((endAngle - startAngle) / 0.1);
        for (let i = 0; i <= steps; i++) {
            const angle = startAngle + (i / steps) * (endAngle - startAngle);
            points.push({
                x: x + Math.cos(angle) * radius,
                y: y + Math.sin(angle) * radius
            });
        }
        return new Polygon(points, options);
    }
};

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { Shape, Rect, Circle, Ellipse, Line, Polygon, RoundedRect, Text, ImageShape, Shapes };
}
