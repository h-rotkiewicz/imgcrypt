package main

import (
	"image"
	"image/draw"
	"image/png"
	"os"
	"fmt"
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
