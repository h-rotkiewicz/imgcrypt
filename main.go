package main

import (
  "fmt"
  "flag"
  "os"
	"math/rand"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'hide' or 'reveal' subcommand")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "hide":
		handleHide(os.Args[2:])
	case "reveal":
		handleReveal(os.Args[2:])
	default:
		fmt.Println("Expected 'hide' or 'reveal' subcommand")
		os.Exit(1)
	}
}

func handleHide(args []string) {
	cmd := flag.NewFlagSet("hide", flag.ExitOnError)
	text := cmd.String("t", "", "Text to hide")
	imgPath := cmd.String("i", "", "Path to input image")
	password := cmd.String("p", "", "Password")

	cmd.Parse(args)

	if *text == "" || *imgPath == "" {
		fmt.Println("Error: hide requires -t and -i")
		cmd.PrintDefaults()
		os.Exit(1)
	}

	image, err := load_png(*imgPath)
	if err != nil {
		fmt.Println("Error loading image:", err)
		return
	}

	textBits := TextToBits(*text)
	lengthHeader := IntTo32Bits(len(*text)) 
	allBits := append(lengthHeader, textBits...)

	fmt.Printf("Hiding %d bits...\n", len(allBits))
	
	err = write_bits_into_image(image, allBits, *password)
	if err != nil {
		fmt.Println("Error writing bits:", err)
		return
	}

	image.Save("output.png")
	fmt.Println("Done. Saved to output.png")
}

func handleReveal(args []string) {
	cmd := flag.NewFlagSet("reveal", flag.ExitOnError)
	imgPath := cmd.String("i", "", "Path to input image")
	password := cmd.String("p", "", "Password")
	cmd.Parse(args)

	if *imgPath == "" {
		fmt.Println("Error: reveal requires -i")
		cmd.PrintDefaults()
		os.Exit(1)
	}

	image, err := load_png(*imgPath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	extractedBits := read_bits_randomly(image, *password)

	lengthBits := extractedBits[:32]
	messageLength := BitsToInt(lengthBits)

	fmt.Printf("Debug: Header says message is %d bytes long.\n", messageLength)

	maxCapacity := (image.Width() * image.Height() * 3) / 8
	
	if messageLength < 0 || messageLength > maxCapacity {
		fmt.Println("ERROR: Extracted length is invalid.")
		fmt.Println("Possible causes:")
		fmt.Println(" - Wrong Password (Seed mismatch)")
		fmt.Println(" - Input image does not have a message")
		return
	}

	start := 32
	end := 32 + (messageLength * 8)
	
	if end > len(extractedBits) {
		fmt.Println("Error: Message length exceeds read bits.")
		return
	}

	messageBits := extractedBits[start:end]
	message := BitsToText(messageBits)
	
	fmt.Println("Hidden message:", message)
}


func write_bits_into_image(img *EditableImage, bits []int, password string) error {
	width := img.Width()
	height := img.Height()
	totalPixels := width * height

	pixelsNeeded := (len(bits) + 2) / 3 
	if pixelsNeeded > totalPixels {
		return fmt.Errorf("image too small: need %d pixels, have %d", pixelsNeeded, totalPixels)
	}

	seed := passwordToSeed(password)
	r := rand.New(rand.NewSource(seed))
	
	shuffledIndices := r.Perm(totalPixels)

	pixelCounter := 0 // Tracks which random pixel we are using
	
	for i := 0; i < len(bits); i += 3 {
		randomIndex := shuffledIndices[pixelCounter]
		pixelCounter++

		x := randomIndex % width
		y := randomIndex / width

		pixel := img.GetPixel(x, y)

		var chunk [3]int
		if i < len(bits)   { chunk[0] = bits[i] }
		if i+1 < len(bits) { chunk[1] = bits[i+1] }
		if i+2 < len(bits) { chunk[2] = bits[i+2] }

		err := pixel.set_LSB(chunk)
		if err != nil {
			return err
		}
		img.SetPixel(x, y, pixel)
	}
	return nil
}

func read_bits_randomly(img *EditableImage, password string) []int {
	width := img.Width()
	height := img.Height()
	totalPixels := width * height

	seed := passwordToSeed(password)
	r := rand.New(rand.NewSource(seed))
	shuffledIndices := r.Perm(totalPixels)

	var extractedBits []int

	for _, randomIndex := range shuffledIndices {
		x := randomIndex % width
		y := randomIndex / width

		pixel := img.GetPixel(x, y)

		extractedBits = append(extractedBits, int(pixel.R&1))
		extractedBits = append(extractedBits, int(pixel.G&1))
		extractedBits = append(extractedBits, int(pixel.B&1))
	}

	return extractedBits
}
