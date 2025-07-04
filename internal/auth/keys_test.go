package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
)

func TestLoadECDSAPrivateKey(t *testing.T) {
	tests := []struct {
		name    string
		keyPath string
		wantErr bool
	}{
		{
			name:    "load valid key",
			keyPath: "test_valid_private.pem",
			wantErr: false,
		},
		{
			name:    "load invalid key",
			keyPath: "test_invalid_private.pem",
			wantErr: true,
		},
		{
			name:    "file does not exist",
			keyPath: "non_existent_key.pem",
			wantErr: true,
		},
		{
			name:    "empty key path",
			keyPath: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadECDSAPrivateKey(tt.keyPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadECDSAPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != nil {
				checkECDSAPrivateKey(t, got)
			}
		})
	}
}

func checkECDSAPrivateKey(t *testing.T, key *ecdsa.PrivateKey) {
	if key == nil {
		t.Error("Expected non-nil key")
		return
	}
	if key.Curve != elliptic.P256() {
		t.Errorf("Expected P256 curve")
	}
	// Verify we can sign and verify with the key
	hash := []byte("test message")
	r, s, err := ecdsa.Sign(rand.Reader, key, hash)
	if err != nil {
		t.Errorf("Failed to sign with loaded key: %v", err)
	}
	if !ecdsa.Verify(&key.PublicKey, hash, r, s) {
		t.Errorf("Failed to verify signature with loaded key: %v", err)
	}
}
