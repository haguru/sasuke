package userservice

import (
	"context"
	"fmt"

	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models"

	"golang.org/x/crypto/bcrypt"
)


type UserService struct {
	UserRepo interfaces.UserRepository
}

// NewUserService creates a new UserService instance.
func NewUserService(repo interfaces.UserRepository) *UserService {
	return &UserService{UserRepo: repo}
}

// RegisterUser hashes the password and adds the user via the repository.
func (s *UserService) RegisterUser(ctx context.Context, username, password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	user := models.User{
		Username:       username,
		HashedPassword: string(hashedPassword), // Pass hashed password to repository
	}

	userID, err := s.UserRepo.AddUser(ctx, user)
	if err != nil {
		return "", fmt.Errorf("failed to register user: %w", err)
	}
	return userID, nil
}

// AuthenticateUser verifies a user's credentials and returns their ID or an error.
func (s *UserService) AuthenticateUser(ctx context.Context, username, password string) (bool, error) {
	user, err := s.UserRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("error retrieving user: %w", err)
	}
	if user == nil {
		return false, fmt.Errorf("user not found")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	if err != nil {
		return false, fmt.Errorf("invalid password")
	}

	return true, nil // Authentication successful, return true
}
