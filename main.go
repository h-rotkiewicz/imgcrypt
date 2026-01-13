package main

import (
	"flag"
	"fmt"
	"os"
	"bytes"
	"encoding/binary"
	"crypto/ecdh"
)
const MasterSeed int64 = 1234567890
const SplitPoint = 5000

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
	key := cmd.String("k", "", "Path to Receiver's Public Key")
	text := cmd.String("t", "", "Text to hide")
	imgPath := cmd.String("i", "", "Path to input image")
	cmd.Parse(args)
	keyPath := *key

	if *text == "" || *imgPath == "" || keyPath == "" {
		fmt.Println("Error: -t, -i, and -k are all required.")
		cmd.PrintDefaults()
		return
	}

	img, err := load_png(*imgPath)
	if err != nil {
		fmt.Println("Image Load Error:", err)
		return
	}

	keyObj, kType, err := LoadECCKey(keyPath)
	if err != nil {
		fmt.Println("Key Error:", err)
		return
	}
	if kType != KeyTypePublic {
		fmt.Println("Error: To hide, you need the RECEIVER'S PUBLIC KEY.")
		return
	}
	pubKey := keyObj.(*ecdh.PublicKey)

	sessionPass := GenerateRandomPassword() // 32B random password
	fmt.Println("Generated Session Password:", sessionPass)

	fmt.Println("Encrypting Body with AES...")
	encryptedBodyBytes, err := EncryptAES(*text, sessionPass)
	if err != nil {
		fmt.Println("Body Encryption Failed:", err)
		return
	}

	payloadBuf := new(bytes.Buffer)
	
	binary.Write(payloadBuf, binary.LittleEndian, int32(len(encryptedBodyBytes)))
	
	payloadBuf.Write([]byte(sessionPass))

	fmt.Println("Encrypting Header with ECC...")
	encryptedHeaderBytes, err := EncryptHeader(pubKey, payloadBuf.Bytes())
	if err != nil {
		fmt.Println("Header Encryption Failed:", err)
		return
	}

	encryptedHeaderBits := BytesToBits(encryptedHeaderBytes) // Helper function
	headerPixelsNeeded := (len(encryptedHeaderBits) + 2) / 3
	
	headerPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), MasterSeed, headerPixelsNeeded, 0, SplitPoint)
	
	fmt.Printf("Writing %d encrypted header bits...\n", len(encryptedHeaderBits))
	WriteBitsAtPoints(img, encryptedHeaderBits, headerPoints)

	bodyBits := BytesToBits(encryptedBodyBytes) 
	sessionSeed := passwordToSeed(sessionPass)
	
	bodyPixelsNeeded := (len(bodyBits) + 2) / 3
	bodyPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), sessionSeed, bodyPixelsNeeded, SplitPoint, img.Width()*img.Height())

	fmt.Printf("Writing %d encrypted body bits...\n", len(bodyBits))
	WriteBitsAtPoints(img, bodyBits, bodyPoints)

	img.Save("output.png")
// Paint Header Points PINK
	for _, p := range headerPoints {
    	px := img.GetPixel(p.X, p.Y) // Assuming you have a getter
    	px.R = 255; px.G = 0; px.B = 255
    	img.SetPixel(p.X, p.Y, px)   // Assuming you have a setter
	}

	for _, p := range bodyPoints {
    	px := img.GetPixel(p.X, p.Y)
    	px.R = 0; px.G = 0; px.B = 255
    	img.SetPixel(p.X, p.Y, px)
	}

	img.Save("output_debug.png")
}

func handleReveal(args []string) {
	cmd := flag.NewFlagSet("reveal", flag.ExitOnError)
	key := cmd.String("k", "", "Path to Your Private Key")
	imgPath := cmd.String("i", "", "Path to input image")
	cmd.Parse(args)
	keyPath := *key

	if *imgPath == "" || keyPath == "" {
		fmt.Println("Error: -i and -k are required.")
		return
	}
	img, err := load_png(*imgPath)
	if err != nil {
		fmt.Println("Image Load Error:", err)
		return
	}


	headerBytesLen := 97
	headerPixels := ((headerBytesLen * 8) + 2) / 3 

	headerPoints, err := GeneratePointsInRange(img.Width(), img.Height(), MasterSeed, headerPixels, 0, SplitPoint)
	headerBits := ReadBitsAtPoints(img, headerPoints)

	exactHeaderBits := headerBytesLen * 8
	if len(headerBits) > exactHeaderBits {
    	headerBits = headerBits[:exactHeaderBits]
	}

	encryptedHeaderBytes := BitsToBytes(headerBits)

	keyObj, kType, _ := LoadECCKey(keyPath)
	if kType != KeyTypePrivate {
		fmt.Println("Error: To reveal, you need private key")
		return
	}

	privKey := keyObj.(*ecdh.PrivateKey)
	decryptedHeaderBytes, err:= DecryptHeader(privKey, encryptedHeaderBytes)
	if err != nil {
		fmt.Println("Header Decryption Failed:", err)
		return
	}

	headerBuf := bytes.NewReader(decryptedHeaderBytes)
	var bodySize int32
	binary.Read(headerBuf, binary.LittleEndian, &bodySize)

	sessionPassBytes := make([]byte, 16)
	headerBuf.Read(sessionPassBytes)
	sessionPass := string(sessionPassBytes)
	fmt.Println("Recovered Session Password:", sessionPass)

	sessionSeed := passwordToSeed(sessionPass)

	bodyPixels := ((int(bodySize)*8)+2)/3
	bodyPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), sessionSeed, bodyPixels, SplitPoint, img.Width()*img.Height())
	bodyBits := ReadBitsAtPoints(img, bodyPoints)
	bodyBits = bodyBits[:bodySize*8] // Trim to actual size

	encryptedBodyBytes := BitsToBytes(bodyBits)
	decryptedBody, err := DecryptAES(encryptedBodyBytes, sessionPass)
	if err != nil {
		fmt.Println("Body Decryption Failed:", err)
		return
	}
	fmt.Println("Hidden Text:")
	fmt.Println(decryptedBody)

}


// Converts a byte slice (e.g., encrypted data) into a slice of bits (0s and 1s)
func BytesToBits(data []byte) []int {
	var bits []int
	for _, b := range data {
		for i := 7; i >= 0; i-- {
			bits = append(bits, int((b>>i)&1))
		}
	}
	return bits
}

// Converts a slice of bits back into a byte slice
func BitsToBytes(bits []int) []byte {
	var bytes []byte
	for i := 0; i < len(bits); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if i+j < len(bits) && bits[i+j] == 1 {
				b |= 1 << (7 - j)
			}
		}
		bytes = append(bytes, b)
	}
	return bytes
}
