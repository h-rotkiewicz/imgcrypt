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

	sessionPass := GenerateRandomPassword()
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

	const SplitPoint = 5000 // Reserve first 5000 pixels for header

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
	fmt.Println("Done.")
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

	keyObj, kType, err := LoadECCKey(keyPath)
	if err != nil {
		fmt.Println("Key Error:", err)
		return
	}
	if kType != KeyTypePrivate {
		fmt.Println("Error: To reveal, you need YOUR PRIVATE KEY.")
		return
	}
	privKey := keyObj.(*ecdh.PrivateKey)

	const SplitPoint = 5000

	// EphemeralKey(65) + AES_GCM_Nonce(12) + AES_GCM_Tag(16) + Payload(36) = 129 bytes
	const EncryptedHeaderSize = 129 
	headerBitsToRead := EncryptedHeaderSize * 8
	headerPixelsNeeded := (headerBitsToRead + 2) / 3

	headerPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), MasterSeed, headerPixelsNeeded, 0, SplitPoint)
	headerRawBits := ReadBitsAtPoints(img, headerPoints)
	
	encryptedHeaderBytes := BitsToBytes(headerRawBits[:headerBitsToRead])

	fmt.Println("Decrypting header...")
	decryptedPayload, err := DecryptHeader(privKey, encryptedHeaderBytes)
	if err != nil {
		fmt.Println("Decryption Failed (Wrong Key?):", err)
		return
	}

	buf := bytes.NewReader(decryptedPayload)
	var bodyLength int32
	binary.Read(buf, binary.LittleEndian, &bodyLength)
	
	passBytes := make([]byte, 32)
	buf.Read(passBytes)
	sessionPass := string(passBytes)

	fmt.Printf("Header Decrypted! Body Length: %d, Pass: %s\n", bodyLength, sessionPass)

	sessionSeed := passwordToSeed(sessionPass)
	totalBodyBits := int(bodyLength) * 8
	bodyPixelsNeeded := (totalBodyBits + 2) / 3
	
	bodyPoints, _ := GeneratePointsInRange(img.Width(), img.Height(), sessionSeed, bodyPixelsNeeded, SplitPoint, img.Width()*img.Height())
	bodyRawBits := ReadBitsAtPoints(img, bodyPoints)

	if len(bodyRawBits) < totalBodyBits {
		fmt.Println("Error: Image corrupt or not enough pixels read.")
		return
	}

	encryptedBodyBytes := BitsToBytes(bodyRawBits[:totalBodyBits])

	fmt.Println("Decrypting Body...")
	decryptedMessage, err := DecryptAES(encryptedBodyBytes, sessionPass)
	if err != nil {
		fmt.Println("Body Decryption Failed:", err)
		return
	}

	fmt.Println("HIDDEN MESSAGE:", decryptedMessage)
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
