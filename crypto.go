package main

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

type KeyType int

const (
	KeyTypeUnknown KeyType = iota
	KeyTypePrivate
	KeyTypePublic
)

func LoadECCKey(path string) (any, KeyType, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, KeyTypeUnknown, fmt.Errorf("could not read key file: %v", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, KeyTypeUnknown, errors.New("failed to parse PEM block")
	}

	switch block.Type {
	case "EC PRIVATE KEY":
		privECDSA, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, KeyTypeUnknown, fmt.Errorf("failed to parse EC Private Key: %v", err)
		}
		privECDH, err := privECDSA.ECDH()
		if err != nil {
			return nil, KeyTypeUnknown, fmt.Errorf("failed to convert to ECDH: %v", err)
		}
		return privECDH, KeyTypePrivate, nil
	case "PUBLIC KEY":
		pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, KeyTypeUnknown, fmt.Errorf("failed to parse Public Key: %v", err)
		}
		pubECDSA, ok := pubInterface.(*ecdsa.PublicKey)
		if !ok {
			return nil, KeyTypeUnknown, errors.New("key is not an ECC Public Key")
		}
		pubECDH, err := pubECDSA.ECDH()
		if err != nil {
			return nil, KeyTypeUnknown, fmt.Errorf("failed to convert to ECDH: %v", err)
		}
		return pubECDH, KeyTypePublic, nil
	default:
		return nil, KeyTypeUnknown, fmt.Errorf("unknown key type: %s", block.Type)
	}
}

func EncryptHeader(receiverPub *ecdh.PublicKey, data []byte) ([]byte, error) {
	ephemeralPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := ephemeralPriv.ECDH(receiverPub)
	if err != nil {
		return nil, err
	}


	ciphertext, err := EncryptAES(string(data), string(sharedSecret))
	if err != nil {
		return nil, err
	}

	return append(ephemeralPriv.PublicKey().Bytes(), ciphertext...), nil
}

func DecryptHeader(privKey *ecdh.PrivateKey, blob []byte) ([]byte, error) {
	const pubKeySize = 65
	if len(blob) <= pubKeySize {
		return nil, errors.New("header blob too short")
	}

	ephemPubBytes := blob[:pubKeySize]
	ciphertext := blob[pubKeySize:]


	ephemPub, err := ecdh.P256().NewPublicKey(ephemPubBytes)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := privKey.ECDH(ephemPub)
	if err != nil {
		return nil, err
	}

	fullHash := sha256.Sum256(sharedSecret)
	symmetricKey := fullHash[:16]

	return decryptBits(ciphertext, symmetricKey)
}

func EncryptAES(plaintext string, password string) ([]byte, error) {
	keyHash := sha256.Sum256([]byte(password))
	key := keyHash[:16]

	return encryptBits([]byte(plaintext), key)
}

func DecryptAES(encryptedData []byte, password string) (string, error) {
	keyHash := sha256.Sum256([]byte(password))
	key := keyHash[:16]

	plaintextBytes, err := decryptBits(encryptedData, key)
	if err != nil {
		return "", fmt.Errorf("AES decryption failed: %v", err)
	}
	return string(plaintextBytes), nil
}
