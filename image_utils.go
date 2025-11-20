package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
)

type Pixel struct {
	R, G, B, A uint8
}

func (p *Pixel) set_LSB(bits [3]int) error {
	channels := []*uint8{&p.R, &p.G, &p.B}
	for i,bit := range bits {
		if bit != 0 && bit != 1 {
			return fmt.Errorf("Bit value must be 0 or 1")
		}
		if bit == 1 {
			*channels[i] |= 1
		} else {
			*channels[i] &^= 1
		}
	}
	return nil
}

type EditableImage struct {
	Img *image.RGBA
}

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

	for y := range height {
		for x := range width {
			currentPixel := e.GetPixel(x, y)
			newPixel := modifier(x, y, currentPixel)
			e.SetPixel(x, y, newPixel)
		}
	}
}

func WriteBitsAtPoints(img *EditableImage, bits []int, points []image.Point) error {
	if len(points) * 3 < len(bits) {
		return fmt.Errorf("not enough points to hold all bits")
	}

	pixelCounter := 0

	for i := 0; i < len(bits); i += 3 {
		pt := points[pixelCounter]
		pixelCounter++

		pixel := img.GetPixel(pt.X, pt.Y)

		var chunk [3]int
		if i < len(bits)   { chunk[0] = bits[i] }
		if i+1 < len(bits) { chunk[1] = bits[i+1] }
		if i+2 < len(bits) { chunk[2] = bits[i+2] }

		pixel.set_LSB(chunk) 
		img.SetPixel(pt.X, pt.Y, pixel)
	}
	return nil
}

func ReadBitsAtPoints(img *EditableImage, points []image.Point) []int {
	var bits []int
	
	for _, pt := range points {
		pixel := img.GetPixel(pt.X, pt.Y)
		
		// Extract 3 bits per pixel
		bits = append(bits, int(pixel.R&1))
		bits = append(bits, int(pixel.G&1))
		bits = append(bits, int(pixel.B&1))
	}
	return bits
}
