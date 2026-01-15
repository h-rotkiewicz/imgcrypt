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

type EncryptionSession struct {
	EphemeralPriv *ecdh.PrivateKey
	SharedKey     []byte // The 16-byte AES key
}

func NewEncryptionSession(receiverPub *ecdh.PublicKey) (*EncryptionSession, error) {
	ephemeralPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	sharedSecret, err := ephemeralPriv.ECDH(receiverPub)
	if err != nil {
		return nil, err
	}

	fullHash := sha256.Sum256(sharedSecret)
	aesKey16 := fullHash[:16]

	return &EncryptionSession{
		EphemeralPriv: ephemeralPriv,
		SharedKey:     aesKey16,
	}, nil
}

func (s *EncryptionSession) BuildHeader(metadata []byte) ([]byte, error) {
	encryptedMetadata, err := encryptBits(metadata, s.SharedKey)
	if err != nil {
		return nil, err
	}

	return append(s.EphemeralPriv.PublicKey().Bytes(), encryptedMetadata...), nil
}

func ParseHeader(receiverPriv *ecdh.PrivateKey, headerBlob []byte) ([]byte, []byte, error) {
	const pubKeySize = 65
	if len(headerBlob) <= pubKeySize {
		return nil, nil, errors.New("header blob too short")
	}

	ephemPubBytes := headerBlob[:pubKeySize]
	encryptedMetadata := headerBlob[pubKeySize:]

	ephemPub, err := ecdh.P256().NewPublicKey(ephemPubBytes)
	if err != nil {
		return nil, nil, err
	}

	sharedSecret, err := receiverPriv.ECDH(ephemPub)
	if err != nil {
		return nil, nil, err
	}

	fullHash := sha256.Sum256(sharedSecret)
	aesKey16 := fullHash[:16]

	decryptedMetadata, err := decryptBits(encryptedMetadata, aesKey16)
	if err != nil {
		return nil, nil, err
	}

	return decryptedMetadata, aesKey16, nil
}
