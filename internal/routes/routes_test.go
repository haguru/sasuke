package routes

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	structValidator "github.com/go-playground/validator/v10"
	"github.com/haguru/sasuke/internal/auth"
	"github.com/haguru/sasuke/internal/interfaces/mocks"
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
		userrepoError  error
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
		{
			name:           "short username",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"username":"%s","password":"%s"}`, "short", "validpass123!"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "long password 65 characters",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"username":"%s","password":"%s"}`, "validuser", "longpassword12345678901234567890123456789012345678901234567890345"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing username",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"password":"%s"}`, "validpass123!"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "missing password",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"username":"%s"}`, "validuser"),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "User not found",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"username":"%s","password":"%s"}`, "nonexistentuser", "testpass"),
			userrepoError:  fmt.Errorf("user not found"),
			wantStatusCode: http.StatusUnauthorized,
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

		// get username and password from the request body
		username, password, _ := extractCredentials(tt.body)

		// Hash the password for the mock user
		// This is necessary because the user service expects a hashed password
		// and the GetUserByUsername method will return a user with a hashed password.
		hashedPassword, err := HashString(password)
		if err != nil {
			t.Fatalf("Failed to hash password: %v", err)
		}

		// Create a user object to return from the mock repository
		// If the test expects an unauthorized status, we return nil for the user.
		// Otherwise, we return a user with the hashed password.
		// This simulates the behavior of the user repository when a user is found.
		var returnedUser *models.User
		if tt.wantStatusCode != http.StatusUnauthorized {
			returnedUser = &models.User{
				Username: username,
				Password: hashedPassword,
			}
		}

		// Mock the GetUserByUsername method to return a user with a hashed password
		userRepo.On("GetUserByUsername", mock.Anything, username).Return(returnedUser, tt.userrepoError).Maybe()

		userService := &userservice.UserService{
			UserRepo: userRepo, // Use a mock or a real implementation
		}

		// Load the ECDSA private key for signing JWTs
		privateKey, err := auth.LoadECDSAPrivateKey("validKey.pem") // Mock or set up your private key as needed
		if err != nil {
			t.Fatalf("Failed to load private key: %v", err)
		}
		mockedMetrics := mocks.NewMockMetrics(t)
		mockedMetrics.On("IncCounter", mock.AnythingOfType("string")).Return().Maybe()
		mockedMetrics.On("ObserveHistogram", mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return().Maybe()
		// Create a new Route instance with the mock user service and private key
		r := &Route{
			Metrics:     mockedMetrics,
			UserService: userService,
			PrivateKey:  privateKey,
			validator:   structValidator.New(),
		}
		// Call the Login method with the recorder and request
		r.Login(rr, req)
		if rr.Code != tt.wantStatusCode {
			t.Errorf("%s: got status %d, want %d", tt.name, rr.Code, tt.wantStatusCode)
		}
	}
}

func TestRoute_Signup(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		userrepoError  error
		wantStatusCode int
	}{
		{
			name:           "Valid signup request",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"username":"validuser1","password":"validPass123!"}`,
			wantStatusCode: http.StatusCreated,
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
			body:           `{"username":"validuser2","password":"validPass123!"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON body",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"username":"validuser3""password":"validPass123!"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Short username",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"username":"short","password":"validPass123!"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Long password",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"username":"validuser4","password":"` + string(make([]byte, 65)) + `"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Missing username",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"password":"validPass123!"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Missing password",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"username":"validuser5"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "User already exists",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"username":"existinguser","password":"validPass123!"}`,
			userrepoError:  fmt.Errorf("user already exists"),
			wantStatusCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/signup", nil)
		if tt.body != "" {
			req = httptest.NewRequest(tt.method, "/signup", bytes.NewBufferString(tt.body))
		}
		if tt.contentType != "" {
			req.Header.Set("Content-Type", tt.contentType)
		}
		rr := httptest.NewRecorder()

		// create a mock userrepository or use a real one if available
		userRepo := mocks.NewMockUserRepository(t)
		userRepo.On("AddUser", mock.Anything, mock.AnythingOfType("models.User")).
			Return("", tt.userrepoError).Maybe()

		userService := &userservice.UserService{
			UserRepo: userRepo,
		}

		// Load the ECDSA private key for signing JWTs
		privateKey, err := auth.LoadECDSAPrivateKey("validKey.pem")
		if err != nil {
			t.Fatalf("Failed to load private key: %v", err)
		}

		// Create a mock metrics instance
		// This is necessary to avoid nil pointer dereference in the Route methods
		mockedMetrics := mocks.NewMockMetrics(t)
		mockedMetrics.On("IncCounter", mock.AnythingOfType("string")).Return().Maybe()
		mockedMetrics.On("ObserveHistogram", mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return().Maybe()

		// Create a new Route instance with the mock user service and private key
		r := &Route{
			Metrics:     mockedMetrics,
			UserService: userService,
			PrivateKey:  privateKey,
			validator:   structValidator.New(),
		}
		r.Signup(rr, req)
		if rr.Code != tt.wantStatusCode {
			t.Errorf("%s: got status %d, want %d", tt.name, rr.Code, tt.wantStatusCode)
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

// Extract username and password from request body
// if
func extractCredentials(body string) (string, string, error) {
	var creds map[string]string
	if err := json.Unmarshal([]byte(body), &creds); err != nil {
		return "", "", fmt.Errorf("failed to parse request body: %w", err)
	}

	username := creds["username"]
	password := creds["password"]

	return username, password, nil
}
