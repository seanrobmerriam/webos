/*
Package graphics provides server-side image processing and font handling
capabilities for the WebOS project.

This package implements image manipulation, font loading, and rendering
functionality that can be used on the server side to pre-process graphics
or generate images for the client.
*/
package graphics

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

// Common errors
var (
	ErrInvalidImageData  = errors.New("invalid image data")
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrFileNotFound      = errors.New("file not found")
)

// Image represents a graphics image with metadata
type Image struct {
	img    image.Image
	width  int
	height int
	format string
}

// NewImage creates a new Image from an image.Image
func NewImage(img image.Image) *Image {
	bounds := img.Bounds()
	return &Image{
		img:    img,
		width:  bounds.Dx(),
		height: bounds.Dy(),
	}
}

// LoadImage loads an image from a file
func LoadImage(path string) (*Image, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	defer file.Close()

	return DecodeImage(file)
}

// DecodeImage decodes an image from an io.Reader
func DecodeImage(r io.Reader) (*Image, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, ErrInvalidImageData
	}

	return &Image{
		img:    img,
		width:  img.Bounds().Dx(),
		height: img.Bounds().Dy(),
		format: format,
	}, nil
}

// DecodeImageData decodes image data from a byte slice
func DecodeImageData(data []byte) (*Image, error) {
	return DecodeImage(bytes.NewReader(data))
}

// Width returns the image width
func (i *Image) Width() int {
	return i.width
}

// Height returns the image height
func (i *Image) Height() int {
	return i.height
}

// Bounds returns the image bounds
func (i *Image) Bounds() image.Rectangle {
	return i.img.Bounds()
}

// Format returns the image format
func (i *Image) Format() string {
	return i.format
}

// Draw draws the image onto a target at the specified position
func (i *Image) Draw(target draw.Image, x, y int) {
	draw.Draw(target, image.Rect(x, y, x+i.width, y+i.height), i.img, image.Point{0, 0}, draw.Over)
}

// DrawScaled draws the image onto a target with scaling
func (i *Image) DrawScaled(target draw.Image, dst image.Rectangle) {
	// Simple nearest-neighbor scaling
	for y := dst.Min.Y; y < dst.Max.Y; y++ {
		for x := dst.Min.X; x < dst.Max.X; x++ {
			srcX := (x - dst.Min.X) * i.width / dst.Dx()
			srcY := (y - dst.Min.Y) * i.height / dst.Dy()
			c := i.img.At(srcX, srcY)
			target.Set(x, y, c)
		}
	}
}

// ToPNG encodes the image as PNG
func (i *Image) ToPNG() ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, i.img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ToJPEG encodes the image as JPEG
func (i *Image) ToJPEG(quality int) ([]byte, error) {
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(&buf, i.img, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GetImage returns the underlying image.Image
func (i *Image) GetImage() image.Image {
	return i.img
}

// Resize creates a resized version of the image
func (i *Image) Resize(width, height int) *Image {
	if width <= 0 || height <= 0 {
		return nil
	}

	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	i.DrawScaled(newImg, newImg.Bounds())
	return NewImage(newImg)
}

// Crop creates a cropped version of the image
func (i *Image) Crop(x, y, width, height int) *Image {
	if width <= 0 || height <= 0 {
		return nil
	}

	srcRect := image.Rect(x, y, x+width, y+height)
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(newImg, newImg.Bounds(), i.img, srcRect.Min, draw.Src)
	return NewImage(newImg)
}

// ConvertToRGBA converts the image to RGBA format
func (i *Image) ConvertToRGBA() *Image {
	if _, ok := i.img.(*image.RGBA); ok {
		return i
	}

	bounds := i.img.Bounds()
	newImg := image.NewRGBA(bounds)
	draw.Draw(newImg, bounds, i.img, bounds.Min, draw.Src)
	return NewImage(newImg)
}

// Fill creates a new image filled with a solid color
func Fill(width, height int, c color.Color) *Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{0, 0}, draw.Src)
	return NewImage(img)
}

// Overlay overlays one image onto another
func Overlay(base, overlay *Image, x, y int) {
	overlay.Draw(base.GetImage().(draw.Image), x, y)
}

// Blend blends two images together with the specified alpha
func Blend(base, overlay *Image, x, y int, alpha float64) {
	if alpha <= 0 {
		return
	}
	if alpha >= 1 {
		Overlay(base, overlay, x, y)
		return
	}

	bounds := base.Bounds()
	blended := image.NewRGBA(bounds)
	base.Draw(blended, 0, 0)

	for py := 0; py < overlay.height; py++ {
		for px := 0; px < overlay.width; px++ {
			dstX := x + px
			dstY := y + py
			if dstX < 0 || dstX >= bounds.Dx() || dstY < 0 || dstY >= bounds.Dy() {
				continue
			}

			overlayColor := overlay.img.At(px, py)
			dstColor := blended.At(dstX, dstY)

			r1, g1, b1, a1 := overlayColor.RGBA()
			r2, g2, b2, a2 := dstColor.RGBA()

			alphaNorm := float64(alpha) / 1.0
			r := uint8((float64(r1)*alphaNorm + float64(r2)*(1-alphaNorm)) / 255)
			g := uint8((float64(g1)*alphaNorm + float64(g2)*(1-alphaNorm)) / 255)
			b := uint8((float64(b1)*alphaNorm + float64(b2)*(1-alphaNorm)) / 255)
			a := uint8((float64(a1)*alphaNorm + float64(a2)*(1-alphaNorm)) / 255)

			blended.Set(dstX, dstY, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
}

// GetPixel returns the color of a pixel at the specified coordinates
func (i *Image) GetPixel(x, y int) color.Color {
	if x < 0 || x >= i.width || y < 0 || y >= i.height {
		return nil
	}
	return i.img.At(x, y)
}

// SetPixel sets the color of a pixel at the specified coordinates
func (i *Image) SetPixel(x, y int, c color.Color) {
	if rgba, ok := i.img.(*image.RGBA); ok {
		rgba.Set(x, y, c)
	}
}

// ImageData represents raw pixel data
type ImageData struct {
	Pixels []byte
	Width  int
	Height int
	Format string
}

// ToImageData converts an Image to raw pixel data
func (i *Image) ToImageData() *ImageData {
	bounds := i.img.Bounds()
	pixels := make([]byte, bounds.Dy()*bounds.Dx()*4)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := i.img.At(x, y)
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			idx := ((y-bounds.Min.Y)*bounds.Dx() + (x - bounds.Min.X)) * 4
			pixels[idx] = rgba.R
			pixels[idx+1] = rgba.G
			pixels[idx+2] = rgba.B
			pixels[idx+3] = rgba.A
		}
	}

	return &ImageData{
		Pixels: pixels,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Format: "RGBA",
	}
}

// ImageDataToImage creates an Image from raw pixel data
func ImageDataToImage(data *ImageData) *Image {
	rgba := image.NewRGBA(image.Rect(0, 0, data.Width, data.Height))
	for y := 0; y < data.Height; y++ {
		for x := 0; x < data.Width; x++ {
			idx := (y*data.Width + x) * 4
			rgba.Set(x, y, color.RGBA{
				R: data.Pixels[idx],
				G: data.Pixels[idx+1],
				B: data.Pixels[idx+2],
				A: data.Pixels[idx+3],
			})
		}
	}
	return NewImage(rgba)
}
