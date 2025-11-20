package main

import (
	"flag"
	"fmt"
	"os"
)
const MasterSeed int64 = 1234567890

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
	cmd.Parse(args)

	if *text == "" || *imgPath == "" {
		fmt.Println("Error: -t and -i are required")
		return
	}

	img, _ := load_png(*imgPath)
	
	sessionPass := GenerateRandomPassword()
	fmt.Println("Generated Session Password:", sessionPass)
	
	passBits := TextToBits(sessionPass) 
	textBits := TextToBits(*text)      
	
	lengthBits := IntTo32Bits(len(*text)) 
	headerBits := append(lengthBits, passBits...)

	const SplitPoint = 5000
	totalPixels := img.Width() * img.Height()

	// A. Header Points (Using MasterSeed)
	// enough pixels for 288 bits (approx 96 pixels)
	headerPixelsNeeded := (len(headerBits) + 2) / 3
	headerPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), MasterSeed, headerPixelsNeeded, 0, SplitPoint)

	// B. Body Points (Using Session Password as Seed)
	// enough pixels for the text
	bodyPixelsNeeded := (len(textBits) + 2) / 3
	sessionSeed := passwordToSeed(sessionPass)
	
	bodyPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), sessionSeed, bodyPixelsNeeded, SplitPoint, totalPixels)


	fmt.Println("Writing Header...")
	WriteBitsAtPoints(img, headerBits, headerPoints)

	fmt.Println("Writing Body...")
	WriteBitsAtPoints(img, textBits, bodyPoints)

	img.Save("output.png")
	fmt.Println("Done.")
}

func handleReveal(args []string) {
	cmd := flag.NewFlagSet("reveal", flag.ExitOnError)
	imgPath := cmd.String("i", "", "Path to input image")
	cmd.Parse(args)

	if *imgPath == "" {
		fmt.Println("Error: -i is required")
		return
	}

	img, _ := load_png(*imgPath)
	const SplitPoint = 5000

	
	// Header is ALWAYS 36 bytes (4 bytes len + 32 bytes pass) = 288 bits.
	headerPixelsToRead := 96 
	
	headerPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), MasterSeed, headerPixelsToRead, 0, SplitPoint)
	headerRawBits := ReadBitsAtPoints(img, headerPoints)

	msgLenBits := headerRawBits[:32]
	msgLen := BitsToInt(msgLenBits) // Length in BYTES
	fmt.Printf("Header says message is %d bytes long.\n", msgLen)

	// B. Extract Session Password (Next 256 bits)
	// 32 bytes * 8 bits = 256 bits
	passStart := 32
	passEnd   := 32 + 256
	passBits  := headerRawBits[passStart:passEnd]
	sessionPass := BitsToText(passBits)
	fmt.Println("Recovered Session Password:", sessionPass)

	// --- 3. Read Body ---

	// Now we use the recovered password to generate the body points
	sessionSeed := passwordToSeed(sessionPass)
	
	// Calculate how many pixels we need to read for the message
	totalMsgBits := msgLen * 8
	bodyPixelsNeeded := (totalMsgBits + 2) / 3 // integer math ceiling
	
	// Generate points in the Body Zone
	bodyPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), sessionSeed, bodyPixelsNeeded, SplitPoint, img.Width()*img.Height())
	
	// Read bits
	bodyRawBits := ReadBitsAtPoints(img, bodyPoints)
	
	// Trim to exact size
	if len(bodyRawBits) < totalMsgBits {
		fmt.Println("Error: Image corrupt, not enough bits read")
		return
	}
	finalMsgBits := bodyRawBits[:totalMsgBits]
	
	message := BitsToText(finalMsgBits)
	fmt.Println("Hidden Message:", message)
}
