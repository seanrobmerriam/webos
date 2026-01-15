/**
 * Graphics - 2D Graphics Rendering Library
 * 
 * Provides a complete 2D graphics API for the browser canvas,
 * including shape drawing, text rendering, image handling,
 * transformations, clipping, and blending operations.
 * 
 * @version 1.0.0
 * @license MIT
 */

class Graphics {
    /**
     * Creates a new Graphics instance.
     * 
     * @param {HTMLCanvasElement} canvas - The canvas element to render to
     * @throws {Error} If canvas is not provided or is not a canvas element
     */
    constructor(canvas) {
        if (!canvas) {
            throw new Error('Canvas element is required');
        }
        if (canvas.tagName !== 'CANVAS') {
            throw new Error('Element must be a canvas');
        }
        
        this.canvas = canvas;
        this.ctx = canvas.getContext('2d');
        
        // Transformation state
        this._transform = {
            translateX: 0,
            translateY: 0,
            rotation: 0,
            scaleX: 1,
            scaleY: 1
        };
        
        // Stack for saving/restoring state
        this._stateStack = [];
        
        // Default styles
        this._defaultFont = '12px Arial';
        this._defaultLineWidth = 1;
        this._defaultStrokeStyle = '#000000';
        this._defaultFillStyle = '#000000';
    }

    /**
     * Saves the current graphics state to the stack.
     * Includes transformation state, styles, and clipping region.
     * 
     * @returns {Graphics} Returns this for method chaining
     */
    save() {
        const state = {
            transform: { ...this._transform },
            fillStyle: this.ctx.fillStyle,
            strokeStyle: this.ctx.strokeStyle,
            lineWidth: this.ctx.lineWidth,
            font: this.ctx.font,
            globalAlpha: this.ctx.globalAlpha,
            globalCompositeOperation: this.ctx.globalCompositeOperation,
            lineCap: this.ctx.lineCap,
            lineJoin: this.ctx.lineJoin,
            miterLimit: this.ctx.miterLimit,
            shadowColor: this.ctx.shadowColor,
            shadowBlur: this.ctx.shadowBlur,
            shadowOffsetX: this.ctx.shadowOffsetX,
            shadowOffsetY: this.ctx.shadowOffsetY,
            textAlign: this.ctx.textAlign,
            textBaseline: this.ctx.textBaseline
        };
        this._stateStack.push(state);
        this.ctx.save();
        return this;
    }

    /**
     * Restores the previously saved graphics state from the stack.
     * 
     * @returns {Graphics} Returns this for method chaining
     * @throws {Error} If no saved state exists
     */
    restore() {
        if (this._stateStack.length === 0) {
            throw new Error('No saved state to restore');
        }
        const state = this._stateStack.pop();
        this._transform = state.transform;
        this.ctx.restore();
        return this;
    }

    /**
     * Draws a filled rectangle.
     * 
     * @param {number} x - The x-coordinate of the rectangle's upper-left corner
     * @param {number} y - The y-coordinate of the rectangle's upper-left corner
     * @param {number} w - The width of the rectangle
     * @param {number} h - The height of the rectangle
     * @param {string} color - The fill color (CSS color string)
     * @returns {Graphics} Returns this for method chaining
     * @throws {Error} If parameters are invalid
     */
    drawRect(x, y, w, h, color) {
        if (typeof x !== 'number' || typeof y !== 'number' || 
            typeof w !== 'number' || typeof h !== 'number') {
            throw new Error('Invalid rectangle parameters');
        }
        if (w < 0 || h < 0) {
            throw new Error('Width and height must be non-negative');
        }
        
        this.ctx.fillStyle = color || this._defaultFillStyle;
        this.ctx.fillRect(x, y, w, h);
        return this;
    }

    /**
     * Draws a rectangle outline (stroke only).
     * 
     * @param {number} x - The x-coordinate of the rectangle's upper-left corner
     * @param {number} y - The y-coordinate of the rectangle's upper-left corner
     * @param {number} w - The width of the rectangle
     * @param {number} h - The height of the rectangle
     * @param {string} color - The stroke color (CSS color string)
     * @param {number} [lineWidth=1] - The width of the stroke
     * @returns {Graphics} Returns this for method chaining
     */
    drawRectOutline(x, y, w, h, color, lineWidth = 1) {
        this.ctx.strokeStyle = color || this._defaultStrokeStyle;
        this.ctx.lineWidth = lineWidth;
        this.ctx.strokeRect(x, y, w, h);
        return this;
    }

    /**
     * Draws a filled circle.
     * 
     * @param {number} x - The x-coordinate of the circle's center
     * @param {number} y - The y-coordinate of the circle's center
     * @param {number} r - The radius of the circle
     * @param {string} color - The fill color (CSS color string)
     * @returns {Graphics} Returns this for method chaining
     * @throws {Error} If parameters are invalid
     */
    drawCircle(x, y, r, color) {
        if (typeof x !== 'number' || typeof y !== 'number' || typeof r !== 'number') {
            throw new Error('Invalid circle parameters');
        }
        if (r < 0) {
            throw new Error('Radius must be non-negative');
        }
        
        this.ctx.fillStyle = color || this._defaultFillStyle;
        this.ctx.beginPath();
        this.ctx.arc(x, y, r, 0, Math.PI * 2);
        this.ctx.fill();
        return this;
    }

    /**
     * Draws a circle outline (stroke only).
     * 
     * @param {number} x - The x-coordinate of the circle's center
     * @param {number} y - The y-coordinate of the circle's center
     * @param {number} r - The radius of the circle
     * @param {string} color - The stroke color (CSS color string)
     * @param {number} [lineWidth=1] - The width of the stroke
     * @returns {Graphics} Returns this for method chaining
     */
    drawCircleOutline(x, y, r, color, lineWidth = 1) {
        this.ctx.strokeStyle = color || this._defaultStrokeStyle;
        this.ctx.lineWidth = lineWidth;
        this.ctx.beginPath();
        this.ctx.arc(x, y, r, 0, Math.PI * 2);
        this.ctx.stroke();
        return this;
    }

    /**
     * Draws a line segment.
     * 
     * @param {number} x1 - The x-coordinate of the start point
     * @param {number} y1 - The y-coordinate of the start point
     * @param {number} x2 - The x-coordinate of the end point
     * @param {number} y2 - The y-coordinate of the end point
     * @param {string} color - The line color (CSS color string)
     * @param {number} [width=1] - The line width
     * @returns {Graphics} Returns this for method chaining
     */
    drawLine(x1, y1, x2, y2, color, width = 1) {
        this.ctx.strokeStyle = color || this._defaultStrokeStyle;
        this.ctx.lineWidth = width;
        this.ctx.beginPath();
        this.ctx.moveTo(x1, y1);
        this.ctx.lineTo(x2, y2);
        this.ctx.stroke();
        return this;
    }

    /**
     * Draws a polyline connecting multiple points.
     * 
     * @param {Array<{x: number, y: number}>} points - Array of points to connect
     * @param {string} color - The line color (CSS color string)
     * @param {number} [width=1] - The line width
     * @param {boolean} [closed=false] - Whether to close the polyline
     * @returns {Graphics} Returns this for method chaining
     */
    drawPolyline(points, color, width = 1, closed = false) {
        if (!points || points.length < 2) {
            throw new Error('At least 2 points required');
        }
        
        this.ctx.strokeStyle = color || this._defaultStrokeStyle;
        this.ctx.lineWidth = width;
        this.ctx.beginPath();
        this.ctx.moveTo(points[0].x, points[0].y);
        for (let i = 1; i < points.length; i++) {
            this.ctx.lineTo(points[i].x, points[i].y);
        }
        if (closed) {
            this.ctx.closePath();
        }
        this.ctx.stroke();
        return this;
    }

    /**
     * Draws a filled polygon.
     * 
     * @param {Array<{x: number, y: number}>} points - Array of polygon vertices
     * @param {string} color - The fill color (CSS color string)
     * @returns {Graphics} Returns this for method chaining
     * @throws {Error} If less than 3 points provided
     */
    drawPolygon(points, color) {
        if (!points || points.length < 3) {
            throw new Error('At least 3 points required for polygon');
        }
        
        this.ctx.fillStyle = color || this._defaultFillStyle;
        this.ctx.beginPath();
        this.ctx.moveTo(points[0].x, points[0].y);
        for (let i = 1; i < points.length; i++) {
            this.ctx.lineTo(points[i].x, points[i].y);
        }
        this.ctx.closePath();
        this.ctx.fill();
        return this;
    }

    /**
     * Draws text on the canvas.
     * 
     * @param {string} text - The text to draw
     * @param {number} x - The x-coordinate of the text position
     * @param {number} y - The y-coordinate of the text position
     * @param {string} [font] - CSS font specification (e.g., '14px Arial')
     * @param {string} [color] - The text color (CSS color string)
     * @param {Object} [options] - Additional rendering options
     * @param {string} [options.textAlign='left'] - Text alignment ('left', 'center', 'right')
     * @param {string} [options.textBaseline='top'] - Text baseline ('top', 'middle', 'bottom')
     * @returns {Graphics} Returns this for method chaining
     */
    drawText(text, x, y, font, color, options = {}) {
        if (font) {
            this.ctx.font = font;
        } else {
            this.ctx.font = this._defaultFont;
        }
        
        this.ctx.fillStyle = color || this._defaultFillStyle;
        
        if (options.textAlign) {
            this.ctx.textAlign = options.textAlign;
        }
        if (options.textBaseline) {
            this.ctx.textBaseline = options.textBaseline;
        }
        
        this.ctx.fillText(String(text), x, y);
        return this;
    }

    /**
     * Draws outlined text on the canvas.
     * 
     * @param {string} text - The text to draw
     * @param {number} x - The x-coordinate of the text position
     * @param {number} y - The y-coordinate of the text position
     * @param {string} [font] - CSS font specification
     * @param {string} [fillColor] - The fill color
     * @param {string} [strokeColor] - The stroke color
     * @param {number} [strokeWidth=2] - The stroke width
     * @returns {Graphics} Returns this for method chaining
     */
    drawTextOutline(text, x, y, font, fillColor, strokeColor, strokeWidth = 2) {
        if (font) {
            this.ctx.font = font;
        }
        
        if (fillColor) {
            this.ctx.fillStyle = fillColor;
        }
        if (strokeColor) {
            this.ctx.strokeStyle = strokeColor;
            this.ctx.lineWidth = strokeWidth;
        }
        
        this.ctx.fillText(String(text), x, y);
        if (strokeColor) {
            this.ctx.strokeText(String(text), x, y);
        }
        return this;
    }

    /**
     * Measures the width of text.
     * 
     * @param {string} text - The text to measure
     * @param {string} [font] - CSS font specification
     * @returns {Object} TextMetrics object with width property
     */
    measureText(text, font) {
        if (font) {
            this.ctx.font = font;
        }
        return this.ctx.measureText(String(text));
    }

    /**
     * Draws an image on the canvas.
     * 
     * @param {HTMLImageElement|HTMLCanvasElement} image - The image to draw
     * @param {number} x - The x-coordinate of the destination
     * @param {number} y - The y-coordinate of the destination
     * @param {number} [w] - The width of the destination (defaults to image width)
     * @param {number} [h] - The height of the destination (defaults to image height)
     * @returns {Graphics} Returns this for method chaining
     * @throws {Error} If image is not loaded or valid
     */
    drawImage(image, x, y, w, h) {
        if (!image) {
            throw new Error('Image is required');
        }
        
        if (w !== undefined && h !== undefined) {
            this.ctx.drawImage(image, x, y, w, h);
        } else if (w !== undefined) {
            const aspectRatio = image.height / image.width;
            this.ctx.drawImage(image, x, y, w, w * aspectRatio);
        } else if (h !== undefined) {
            const aspectRatio = image.width / image.height;
            this.ctx.drawImage(image, x, y, h * aspectRatio, h);
        } else {
            this.ctx.drawImage(image, x, y);
        }
        return this;
    }

    /**
     * Draws a portion of an image (image slicing).
     * 
     * @param {HTMLImageElement|HTMLCanvasElement} image - The source image
     * @param {number} sx - Source x-coordinate
     * @param {number} sy - Source y-coordinate
     * @param {number} sw - Source width
     * @param {number} sh - Source height
     * @param {number} dx - Destination x-coordinate
     * @param {number} dy - Destination y-coordinate
     * @param {number} dw - Destination width
     * @param {number} dh - Destination height
     * @returns {Graphics} Returns this for method chaining
     */
    drawImageSlice(image, sx, sy, sw, sh, dx, dy, dw, dh) {
        this.ctx.drawImage(image, sx, sy, sw, sh, dx, dy, dw, dh);
        return this;
    }

    /**
     * Clears a rectangular region.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} w - The width
     * @param {number} h - The height
     * @returns {Graphics} Returns this for method chaining
     */
    clearRect(x, y, w, h) {
        this.ctx.clearRect(x, y, w, h);
        return this;
    }

    /**
     * Clears the entire canvas.
     * 
     * @returns {Graphics} Returns this for method chaining
     */
    clear() {
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        return this;
    }

    /**
     * Fills the entire canvas with a color.
     * 
     * @param {string} color - The fill color
     * @returns {Graphics} Returns this for method chaining
     */
    fill(color) {
        this.ctx.fillStyle = color;
        this.ctx.fillRect(0, 0, this.canvas.width, this.canvas.height);
        return this;
    }

    /**
     * Applies a translation transformation.
     * 
     * @param {number} x - The x translation amount
     * @param {number} y - The y translation amount
     * @returns {Graphics} Returns this for method chaining
     */
    translate(x, y) {
        this._transform.translateX += x;
        this._transform.translateY += y;
        this.ctx.translate(x, y);
        return this;
    }

    /**
     * Applies a rotation transformation.
     * 
     * @param {number} angle - The rotation angle in radians
     * @returns {Graphics} Returns this for method chaining
     */
    rotate(angle) {
        this._transform.rotation += angle;
        this.ctx.rotate(angle);
        return this;
    }

    /**
     * Applies a scale transformation.
     * 
     * @param {number} sx - The x-scale factor
     * @param {number} sy - The y-scale factor
     * @returns {Graphics} Returns this for method chaining
     */
    scale(sx, sy) {
        this._transform.scaleX *= sx;
        this._transform.scaleY *= sy;
        this.ctx.scale(sx, sy);
        return this;
    }

    /**
     * Resets all transformations to identity.
     * 
     * @returns {Graphics} Returns this for method chaining
     */
    resetTransform() {
        this._transform = {
            translateX: 0,
            translateY: 0,
            rotation: 0,
            scaleX: 1,
            scaleY: 1
        };
        this.ctx.setTransform(1, 0, 0, 1, 0, 0);
        return this;
    }

    /**
     * Gets the current transformation state.
     * 
     * @returns {Object} Current transformation state
     */
    getTransform() {
        return { ...this._transform };
    }

    /**
     * Sets the clipping region to a rectangle.
     * 
     * @param {number} x - The x-coordinate of the clip rectangle
     * @param {number} y - The y-coordinate of the clip rectangle
     * @param {number} w - The width of the clip rectangle
     * @param {number} h - The height of the clip rectangle
     * @returns {Graphics} Returns this for method chaining
     */
    clip(x, y, w, h) {
        this.ctx.beginPath();
        this.ctx.rect(x, y, w, h);
        this.ctx.clip();
        return this;
    }

    /**
     * Sets a custom clipping path.
     * 
     * @param {Array<{x: number, y: number}>} points - Array of points defining the clip path
     * @returns {Graphics} Returns this for method chaining
     */
    clipPolygon(points) {
        if (!points || points.length < 3) {
            throw new Error('At least 3 points required for polygon clip');
        }
        
        this.ctx.beginPath();
        this.ctx.moveTo(points[0].x, points[0].y);
        for (let i = 1; i < points.length; i++) {
            this.ctx.lineTo(points[i].x, points[i].y);
        }
        this.ctx.closePath();
        this.ctx.clip();
        return this;
    }

    /**
     * Resets the clipping region to the entire canvas.
     * 
     * @returns {Graphics} Returns this for method chaining
     */
    resetClip() {
        this.ctx.restore();
        return this;
    }

    /**
     * Sets the global alpha (opacity).
     * 
     * @param {number} alpha - The alpha value (0-1)
     * @returns {Graphics} Returns this for method chaining
     * @throws {Error} If alpha is not in range 0-1
     */
    setGlobalAlpha(alpha) {
        if (alpha < 0 || alpha > 1) {
            throw new Error('Alpha must be between 0 and 1');
        }
        this.ctx.globalAlpha = alpha;
        return this;
    }

    /**
     * Sets the global composite operation (blending mode).
     * 
     * @param {string} operation - The composite operation
     * @returns {Graphics} Returns this for method chaining
     */
    setGlobalCompositeOperation(operation) {
        this.ctx.globalCompositeOperation = operation;
        return this;
    }

    /**
     * Sets the shadow properties for drawing operations.
     * 
     * @param {string} color - The shadow color
     * @param {number} blur - The shadow blur amount
     * @param {number} offsetX - The shadow x offset
     * @param {number} offsetY - The shadow y offset
     * @returns {Graphics} Returns this for method chaining
     */
    setShadow(color, blur, offsetX, offsetY) {
        this.ctx.shadowColor = color;
        this.ctx.shadowBlur = blur;
        this.ctx.shadowOffsetX = offsetX;
        this.ctx.shadowOffsetY = offsetY;
        return this;
    }

    /**
     * Clears the shadow properties.
     * 
     * @returns {Graphics} Returns this for method chaining
     */
    clearShadow() {
        this.ctx.shadowColor = 'transparent';
        this.ctx.shadowBlur = 0;
        this.ctx.shadowOffsetX = 0;
        this.ctx.shadowOffsetY = 0;
        return this;
    }

    /**
     * Sets the line cap style.
     * 
     * @param {string} cap - The line cap ('butt', 'round', 'square')
     * @returns {Graphics} Returns this for method chaining
     */
    setLineCap(cap) {
        this.ctx.lineCap = cap;
        return this;
    }

    /**
     * Sets the line join style.
     * 
     * @param {string} join - The line join ('miter', 'round', 'bevel')
     * @returns {Graphics} Returns this for method chaining
     */
    setLineJoin(join) {
        this.ctx.lineJoin = join;
        return this;
    }

    /**
     * Draws an ellipse.
     * 
     * @param {number} x - The x-coordinate of the center
     * @param {number} y - The y-coordinate of the center
     * @param {number} radiusX - The x-radius
     * @param {number} radiusY - The y-radius
     * @param {number} rotation - The rotation angle in radians
     * @param {number} startAngle - The start angle in radians
     * @param {number} endAngle - The end angle in radians
     * @param {string} [color] - The fill color
     * @param {boolean} [outline=false] - Whether to draw outline only
     * @returns {Graphics} Returns this for method chaining
     */
    drawEllipse(x, y, radiusX, radiusY, rotation, startAngle, endAngle, color, outline = false) {
        this.ctx.beginPath();
        this.ctx.ellipse(x, y, radiusX, radiusY, rotation, startAngle, endAngle);
        if (outline) {
            this.ctx.strokeStyle = color || this._defaultStrokeStyle;
            this.ctx.stroke();
        } else {
            this.ctx.fillStyle = color || this._defaultFillStyle;
            this.ctx.fill();
        }
        return this;
    }

    /**
     * Draws a rounded rectangle.
     * 
     * @param {number} x - The x-coordinate of the upper-left corner
     * @param {number} y - The y-coordinate of the upper-left corner
     * @param {number} w - The width
     * @param {number} h - The height
     * @param {number} radius - The corner radius
     * @param {string} [color] - The fill color
     * @param {boolean} [outline=false] - Whether to draw outline only
     * @returns {Graphics} Returns this for method chaining
     */
    drawRoundedRect(x, y, w, h, radius, color, outline = false) {
        if (w < 0 || h < 0) {
            throw new Error('Width and height must be non-negative');
        }
        
        this.ctx.beginPath();
        this.ctx.moveTo(x + radius, y);
        this.ctx.lineTo(x + w - radius, y);
        this.ctx.quadraticCurveTo(x + w, y, x + w, y + radius);
        this.ctx.lineTo(x + w, y + h - radius);
        this.ctx.quadraticCurveTo(x + w, y + h, x + w - radius, y + h);
        this.ctx.lineTo(x + radius, y + h);
        this.ctx.quadraticCurveTo(x, y + h, x, y + h - radius);
        this.ctx.lineTo(x, y + radius);
        this.ctx.quadraticCurveTo(x, y, x + radius, y);
        this.ctx.closePath();
        
        if (outline) {
            this.ctx.strokeStyle = color || this._defaultStrokeStyle;
            this.ctx.stroke();
        } else {
            this.ctx.fillStyle = color || this._defaultFillStyle;
            this.ctx.fill();
        }
        return this;
    }

    /**
     * Draws a quadratic curve.
     * 
     * @param {number} cpX - The x-coordinate of the control point
     * @param {number} cpY - The y-coordinate of the control point
     * @param {number} x - The x-coordinate of the end point
     * @param {number} y - The y-coordinate of the end point
     * @param {string} [color] - The stroke color
     * @param {number} [lineWidth=1] - The line width
     * @returns {Graphics} Returns this for method chaining
     */
    drawQuadraticCurve(cpX, cpY, x, y, color, lineWidth = 1) {
        this.ctx.strokeStyle = color || this._defaultStrokeStyle;
        this.ctx.lineWidth = lineWidth;
        this.ctx.beginPath();
        this.ctx.moveTo(this._lastX || 0, this._lastY || 0);
        this.ctx.quadraticCurveTo(cpX, cpY, x, y);
        this.ctx.stroke();
        this._lastX = x;
        this._lastY = y;
        return this;
    }

    /**
     * Draws a bezier curve.
     * 
     * @param {number} cp1X - The x-coordinate of the first control point
     * @param {number} cp1Y - The y-coordinate of the first control point
     * @param {number} cp2X - The x-coordinate of the second control point
     * @param {number} cp2Y - The y-coordinate of the second control point
     * @param {number} x - The x-coordinate of the end point
     * @param {number} y - The y-coordinate of the end point
     * @param {string} [color] - The stroke color
     * @param {number} [lineWidth=1] - The line width
     * @returns {Graphics} Returns this for method chaining
     */
    drawBezierCurve(cp1X, cp1Y, cp2X, cp2Y, x, y, color, lineWidth = 1) {
        this.ctx.strokeStyle = color || this._defaultStrokeStyle;
        this.ctx.lineWidth = lineWidth;
        this.ctx.beginPath();
        this.ctx.moveTo(this._lastX || 0, this._lastY || 0);
        this.ctx.bezierCurveTo(cp1X, cp1Y, cp2X, cp2Y, x, y);
        this.ctx.stroke();
        this._lastX = x;
        this._lastY = y;
        return this;
    }

    /**
     * Gets the canvas element.
     * 
     * @returns {HTMLCanvasElement} The canvas element
     */
    getCanvas() {
        return this.canvas;
    }

    /**
     * Gets the 2D rendering context.
     * 
     * @returns {CanvasRenderingContext2D} The 2D context
     */
    getContext() {
        return this.ctx;
    }

    /**
     * Gets the canvas dimensions.
     * 
     * @returns {Object} Object with width and height properties
     */
    getDimensions() {
        return {
            width: this.canvas.width,
            height: this.canvas.height
        };
    }

    /**
     * Resizes the canvas.
     * 
     * @param {number} width - The new width
     * @param {number} height - The new height
     * @returns {Graphics} Returns this for method chaining
     */
    resize(width, height) {
        this.canvas.width = width;
        this.canvas.height = height;
        return this;
    }

    /**
     * Exports the canvas content as a data URL.
     * 
     * @param {string} [type='image/png'] - The image format
     * @param {number} [quality=0.92] - The quality for JPEG/WebP (0-1)
     * @returns {string} The data URL
     */
    toDataURL(type = 'image/png', quality = 0.92) {
        return this.canvas.toDataURL(type, quality);
    }

    /**
     * Gets pixel data from the canvas.
     * 
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} w - The width
     * @param {number} h - The height
     * @returns {ImageData} The image data
     */
    getImageData(x, y, w, h) {
        return this.ctx.getImageData(x, y, w, h);
    }

    /**
     * Puts pixel data onto the canvas.
     * 
     * @param {ImageData} imageData - The image data to put
     * @param {number} x - The x-coordinate
     * @param {number} y - The y-coordinate
     * @param {number} [dx] - Destination x offset
     * @param {number} [dy] - Destination y offset
     * @returns {Graphics} Returns this for method chaining
     */
    putImageData(imageData, x, y, dx, dy) {
        if (dx !== undefined && dy !== undefined) {
            this.ctx.putImageData(imageData, x, y, dx, dy);
        } else {
            this.ctx.putImageData(imageData, x, y);
        }
        return this;
    }
}

// Export for module systems
if (typeof module !== 'undefined' && module.exports) {
    module.exports = Graphics;
}
