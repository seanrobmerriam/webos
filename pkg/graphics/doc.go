/*
Package graphics provides server-side image processing and font handling
capabilities for the WebOS project.

This package implements image manipulation, font loading, and rendering
functionality that can be used on the server side to pre-process graphics
or generate images for the client.

# Image Processing

The package provides comprehensive image processing capabilities:

	// Load an image from file
	img, err := graphics.LoadImage("path/to/image.png")
	if err != nil {
		log.Fatal(err)
	}

	// Resize the image
	resized := img.Resize(200, 200)

	// Crop the image
	cropped := img.Crop(10, 10, 100, 100)

	// Convert to JPEG
	jpegData, err := resized.ToJPEG(85)
	if err != nil {
		log.Fatal(err)
	}

	// Save to file
	os.WriteFile("output.jpg", jpegData, 0644)

# Text Rendering

Font handling with text measurement and rendering:

	font := graphics.LoadBasicFont("Arial", 12)

	// Measure text
	metrics := font.MeasureText("Hello, World!")
	fmt.Printf("Text width: %.2f\n", metrics.Width)

	// Word wrap
	lines := font.WordWrap("Long text that needs wrapping", 100)

# Image Operations

Various image composition operations:

	// Create a filled image
	bg := graphics.Fill(800, 600, color.White)

	// Blend images with transparency
	graphics.Blend(base, overlay, 0, 0, 0.5)

	// Overlay images
	graphics.Overlay(base, overlay, 100, 100)

# Drawing Operations

The Image type implements the draw.Image interface for custom drawing:

	// Get the underlying image.Image
	rawImg := img.GetImage()

	// Draw on the image using the standard library
	draw.Draw(rawImg, rect, otherImage, point, draw.Over)

# Export for Module Systems

The graphics package can be imported in Node.js using the jsdom library:

	const { JSDOM } = require('jsdom');
	const { window } = new JSDOM('');
	const { Graphics } = require('./graphics.js');
	const g = new Graphics(window.document.createElement('canvas'));
*/
package graphics
