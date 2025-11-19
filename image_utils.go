package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
)

// Pixel represents the RGBA values of a single pixel.
// We use uint8 (byte) so you can perform bit-wise operations.
type Pixel struct {
	R, G, B, A uint8
}

// EditableImage wraps the standard Go image to provide easy pixel manipulation.
type EditableImage struct {
	Img *image.RGBA
}

// ImageEditor defines the interface for modifying pixels.
type ImageEditor interface {
	GetPixel(x, y int) Pixel
	SetPixel(x, y int, p Pixel)
	Width() int
	Height() int
	Save(filename string) error
}

func load_png(filename string) (*EditableImage, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	src, err := png.Decode(file)
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)

	draw.Draw(dst, bounds, src, bounds.Min, draw.Src)

	return &EditableImage{Img: dst}, nil
}

func (e *EditableImage) GetPixel(x, y int) Pixel {
	// Pix is an array of []uint8. The stride is the width * 4 bytes.
	idx := e.Img.PixOffset(x, y)
	
	return Pixel{
		R: e.Img.Pix[idx+0],
		G: e.Img.Pix[idx+1],
		B: e.Img.Pix[idx+2],
		A: e.Img.Pix[idx+3],
	}
}

func (e *EditableImage) SetPixel(x, y int, p Pixel) {
	idx := e.Img.PixOffset(x, y)
	
	e.Img.Pix[idx+0] = p.R
	e.Img.Pix[idx+1] = p.G
	e.Img.Pix[idx+2] = p.B
	e.Img.Pix[idx+3] = p.A
}

func (e *EditableImage) Width() int {
	return e.Img.Bounds().Dx()
}

func (e *EditableImage) Height() int {
	return e.Img.Bounds().Dy()
}

func (e *EditableImage) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, e.Img)
}

func (e *EditableImage) ApplyFilter(modifier func(x, y int, p Pixel) Pixel) {
	width := e.Width()
	height := e.Height()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			currentPixel := e.GetPixel(x, y)
			newPixel := modifier(x, y, currentPixel)
			e.SetPixel(x, y, newPixel)
		}
	}
}
