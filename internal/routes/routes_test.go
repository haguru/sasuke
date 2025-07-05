package routes

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces/mocks"
	"github.com/haguru/sasuke/internal/metrics"
	"github.com/haguru/sasuke/internal/models"
	"github.com/haguru/sasuke/internal/userservice"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	// Generate a new ECDSA private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic("failed to generate ECDSA key: " + err.Error())
	}

	// Marshal the private key to DER format
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		panic("failed to marshal ECDSA key: " + err.Error())
	}

	// Create the PEM block
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}

	// Ensure the directory exists
	_ = os.MkdirAll("../../res", 0o755)

	// Write the PEM file
	pemPath := "validKey.pem"
	f, err := os.Create(pemPath)
	if err != nil {
		panic("failed to create PEM file: " + err.Error())
	}
	if err := pem.Encode(f, block); err != nil {
		f.Close()
		_ = os.Remove(pemPath)
		panic("failed to encode PEM: " + err.Error())
	}
	f.Close()

	// Run the tests
	code := m.Run()

	// Clean up the PEM file after tests
	_ = os.Remove(pemPath)

	os.Exit(code)
}

func TestRoute_Login(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		wantStatusCode int
	}{
		{
			name:           "Valid login request",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"username":"%s","password":"%s"}`, "testuser", "testpass"),
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			contentType:    "application/json",
			body:           "",
			wantStatusCode: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Content-Type",
			method:         http.MethodPost,
			contentType:    "",
			body:           fmt.Sprintf(`{"username":"%s","password":"%s"}`, "testuser", "testpass"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON body",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"username":"%s""password":"%s"}`, "testuser", "testpass"),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/login", nil)
		if tt.body != "" {
			req = httptest.NewRequest(tt.method, "/login",
				bytes.NewBufferString(tt.body))
		}
		if tt.contentType != "" {
			req.Header.Set("Content-Type", tt.contentType)
		}
		rr := httptest.NewRecorder()

		// create a mock userrepository or use a real one if available
		userRepo := mocks.NewMockUserRepository(t)

		// Hash the password for the mock user
		// This is necessary because the user service expects a hashed password
		// and the GetUserByUsername method will return a user with a hashed password.
		hashedPassword, err := HashString("testpass")
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		// Mock the GetUserByUsername method to return a user with a hashed password
		userRepo.On("GetUserByUsername", mock.Anything, "testuser").Return(&models.User{
			Username: "testuser",
			Password: hashedPassword,
		}, nil).Maybe()

		userService := &userservice.UserService{
			UserRepo: userRepo, // Use a mock or a real implementation
		}

		// Set up expectations for the mock if needed

		privateKey, err := auth.LoadECDSAPrivateKey("validKey.pem") // Mock or set up your private key as needed
		if err != nil {
			t.Fatalf("Failed to load private key: %v", err)
		}
		r := &Route{
			Metrics:     &metrics.Metrics{},
			UserService: userService,
			PrivateKey:  privateKey,
		}
		r.Login(rr, req)
		if rr.Code != tt.wantStatusCode {
			t.Errorf("got status %d, want %d", rr.Code, tt.wantStatusCode)
		}
	}
}

// HashString creates a bcrypt hash of the input string
func HashString(input string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash string: %w", err)
	}
	return string(hashedBytes), nil
}
