package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Global variable for the JWT private key for testing purposes
// This will be initialized in TestMain
var testJwtPrivateKey *ecdsa.PrivateKey

// TestMain runs before any tests in the package.
// It's used for setup and teardown.
func TestMain(m *testing.M) {
	fmt.Println("Running TestMain: Setting up test environment...")

	// Generate a valid ECDSA private key for ES256 signing
	validKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate ECDSA private key for tests: %v", err)
	}
	testJwtPrivateKey = validKey

	// Save the valid private key to a PEM file
	validKeyFile := "test_valid_private.pem"
	validKeyOut, err := os.Create(validKeyFile)
	if err != nil {
		log.Fatalf("Failed to create valid private key file: %v", err)
	}
	defer func() {
		if err := validKeyOut.Close(); err != nil {
			log.Fatalf("Failed to close valid private key file: %v", err)
		}
	}()
	if err := encodeECDSAPrivateKeyToPEM(validKeyOut, validKey); err != nil {
		log.Fatalf("Failed to write valid private key to PEM: %v", err)
	}
	fmt.Printf("Valid private key PEM written to: %s\n", validKeyFile)

	// Create an invalid PEM file (not a private key)
	invalidKeyFile := "test_invalid_private.pem"
	invalidKeyOut, err := os.Create(invalidKeyFile)
	if err != nil {
		log.Fatalf("Failed to create invalid private key file: %v", err)
	}
	defer func() {
		if err := invalidKeyOut.Close(); err != nil {
			log.Fatalf("Failed to close invalid private key file: %v", err)
		}
	}()
	if _, err := invalidKeyOut.WriteString("-----BEGIN INVALID KEY-----\nnot-a-real-key\n-----END INVALID KEY-----\n"); err != nil {
		log.Fatalf("Failed to write invalid key to PEM: %v", err)
	}
	fmt.Printf("Invalid private key PEM written to: %s\n", invalidKeyFile)

	// Run all tests in the package
	code := m.Run()

	// Teardown: remove generated files
	if err := os.Remove(validKeyFile); err != nil {
		log.Printf("Warning: failed to remove %s: %v", validKeyFile, err)
	}
	if err := os.Remove(invalidKeyFile); err != nil {
		log.Printf("Warning: failed to remove %s: %v", invalidKeyFile, err)
	}

	os.Exit(code)
}

// encodeECDSAPrivateKeyToPEM writes an ECDSA private key to the given writer in PEM format.
func encodeECDSAPrivateKeyToPEM(out *os.File, key *ecdsa.PrivateKey) error {
	// Import encoding/pem and crypto/x509
	// Write the PEM-encoded key to the file
	// (This function is only used in tests, so error handling is simple)
	// Place imports at the top if not already present:
	// "encoding/pem"
	// "crypto/x509"
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal ECDSA private key: %w", err)
	}
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}
	if err := pem.Encode(out, block); err != nil {
		return fmt.Errorf("failed to encode PEM: %w", err)
	}
	return nil
}

func TestCreateToken(t *testing.T) {
	type args struct {
		userName string
		// We'll pass the private key to the function, so it's part of the args for the test
		privateKey *ecdsa.PrivateKey
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Successful token creation for valid user",
			args: args{
				userName:   "testuser123",
				privateKey: testJwtPrivateKey, // Use the generated test private key
			},
			wantErr: false,
		},
		{
			name: "Token creation with empty username",
			args: args{
				userName:   "", // An empty username is technically valid for JWT claims, but you might want to add validation in CreateToken
				privateKey: testJwtPrivateKey,
			},
			wantErr: false,
		},
		// Note: Testing `wantErr` for ES256 with a bad private key is tricky.
		// `jwt.Token.SignedString` with ES256 will panic if the key is not `*ecdsa.PrivateKey`.
		// If you want to handle this gracefully in `CreateToken`, add checks there.
		// For example, if CreateToken were modified to return an error for a nil key:
		// {
		// 	name:    "Error with nil private key",
		// 	args: args{
		// 		userName:   "someuser",
		// 		privateKey: nil, // This would cause a panic in the current `CreateToken` without internal checks
		// 	},
		// 	wantErr: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTokenString, err := CreateToken(tt.args.userName, tt.args.privateKey)

			// Check if the error expectation matches
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If no error is expected, proceed to validate the token content
			if !tt.wantErr {
				if gotTokenString == "" {
					t.Error("CreateToken() returned an empty token string for a successful case")
					return
				}

				// Get the public key for verification
				publicKey := &tt.args.privateKey.PublicKey

				// Parse and validate the token
				parsedToken, parseErr := jwt.ParseWithClaims(gotTokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
					// Ensure the signing method is what we expect
					if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					}
					return publicKey, nil // Return the public key for validation
				}, jwt.WithValidMethods([]string{"ES256"})) // Explicitly state valid method

				if parseErr != nil {
					t.Fatalf("Failed to parse or validate token: %v", parseErr)
				}

				if !parsedToken.Valid {
					t.Error("Parsed token is not valid")
				}

				// Verify claims
				claims, ok := parsedToken.Claims.(*CustomClaims)
				if !ok {
					t.Fatal("Failed to cast claims to *CustomClaims")
				}

				// Check custom claim (UserID)
				if claims.UserID != tt.args.userName {
					t.Errorf("Expected UserID to be %s, got %s", tt.args.userName, claims.UserID)
				}

				// Check standard registered claims (with time tolerance)
				now := time.Now()
				// ExpiresAt should be roughly 15 minutes from now
				if claims.ExpiresAt == nil || claims.ExpiresAt.Before(now.Add(14*time.Minute)) || claims.ExpiresAt.After(now.Add(16*time.Minute)) {
					t.Errorf("ExpiresAt claim is not within expected range. Expected around 15min from now, got %v", claims.ExpiresAt)
				}
				// IssuedAt and NotBefore should be very close to now
				if claims.IssuedAt == nil || claims.IssuedAt.After(now.Add(5*time.Second)) || claims.IssuedAt.Before(now.Add(-5*time.Second)) {
					t.Errorf("IssuedAt claim is not recent enough. Expected around now, got %v", claims.IssuedAt)
				}
				if claims.NotBefore == nil || claims.NotBefore.After(now.Add(5*time.Second)) || claims.NotBefore.Before(now.Add(-5*time.Second)) {
					t.Errorf("NotBefore claim is not recent enough. Expected around now, got %v", claims.NotBefore)
				}
				if claims.Issuer != ISSUER {
					t.Errorf("Expected Issuer to be %s, got %s", ISSUER, claims.Issuer)
				}
				if claims.Subject != SUBJECT {
					t.Errorf("Expected Subject to be %s, got %s", SUBJECT, claims.Subject)
				}
				expectedAudience := []string{"api" + ISSUER}
				if len(claims.Audience) != len(expectedAudience) || claims.Audience[0] != expectedAudience[0] {
					t.Errorf("Expected Audience to be %v, got %v", expectedAudience, claims.Audience)
				}
				if claims.ID == "" {
					t.Error("ID (JTI) claim is empty")
				}
				if _, err := uuid.Parse(claims.ID); err != nil {
					t.Errorf("ID (JTI) claim is not a valid UUID: %v", err)
				}
			}
		})
	}
}

func TestVerifyToken(t *testing.T) {
	type args struct {
		tokenString string
		privateKey  *ecdsa.PrivateKey
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Successful token verification with valid token",
			args: args{
				tokenString: "valid-token-string", // Placeholder, will be replaced in the test
				privateKey:  testJwtPrivateKey,
			},
			wantErr: false,
		},
		{
			name: "Error with invalid token format",
			args: args{
				tokenString: "invalid-token-format",
				privateKey:  testJwtPrivateKey,
			},
			wantErr: true,
		},
		{
			name: "Error with tampered token",
			args: args{
				tokenString: "tampered-token-string", // Placeholder, will be replaced in the test
				privateKey:  testJwtPrivateKey,
			},
			wantErr: true,
		},
		{
			name: "Error with expired token",
			args: args{
				tokenString: "expired-token-string", // Placeholder, will be replaced in the test
				privateKey:  testJwtPrivateKey,
			},
			wantErr: true,
		},
		{
			name: "Error with token signed by different key",
			args: args{
				tokenString: "different-key-token-string", // Placeholder, will be replaced in the test
				privateKey:  testJwtPrivateKey,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Successful token verification with valid token" {
				var err error
				// Create a valid token for this test case
				tt.args.tokenString, err = CreateToken("testuser123", tt.args.privateKey)
				if err != nil {
					t.Fatalf("Failed to create token for test: %v", err)
				}
			}

			gotClaims, err := VerifyToken(tt.args.tokenString, &tt.args.privateKey.PublicKey)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && gotClaims == nil {
				t.Error("VerifyToken() returned nil claims for a successful case")
				return
			}

			if !tt.wantErr {
				if gotClaims.UserID != "testuser123" {
					t.Errorf("Expected UserID to be 'testuser123', got %s", gotClaims.UserID)
				}
				if gotClaims.Issuer != ISSUER {
					t.Errorf("Expected Issuer to be %s, got %s", ISSUER, gotClaims.Issuer)
				}
				if gotClaims.Subject != SUBJECT {
					t.Errorf("Expected Subject to be %s, got %s", SUBJECT, gotClaims.Subject)
				}
			}
		})
	}
}
