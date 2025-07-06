package auth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadECDSAPrivateKey loads ECDSA private key from file or environment
func LoadECDSAPrivateKey(keyPath string) (*ecdsa.PrivateKey, error) {
	// check if keyPath exists
	if _, err := os.Stat(keyPath); err != nil {
		return nil, fmt.Errorf("private key path does not exist: %v", err)
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ECDSA private key: %w", err)
	}

	return privateKey, nil
}
