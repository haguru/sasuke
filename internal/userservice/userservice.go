// userservice.go
package userservice

import (
	"context"
	"fmt"

	"github.com/haguru/sasuke/pkg/helper"
	"github.com/haguru/sasuke/internal/interfaces"
	"github.com/haguru/sasuke/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	UserRepo interfaces.UserRepository
	Logger   interfaces.Logger
}

// NewUserService creates a new UserService instance.
func NewUserService(repo interfaces.UserRepository, logger interfaces.Logger) *UserService {
	return &UserService{
		UserRepo: repo,
		Logger:   logger,
	}
}

// RegisterUser hashes the password and adds the user via the repository.
func (s *UserService) RegisterUser(ctx context.Context, username, password string) (string, error) {
	funcName := helper.GetFuncName()
	s.Logger.Debug("Entering function", "func", funcName, "user", username)
	s.Logger.Info("Registering user", "func", funcName, "user", username)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.Logger.Error(ErrFailedToHashPassword, "func", funcName, "user", username, "error", err)
		return "", fmt.Errorf("%s: %w", ErrFailedToHashPassword, err)
	}

	user := models.User{
		Username:       username,
		HashedPassword: string(hashedPassword),
	}

	userID, err := s.UserRepo.AddUser(ctx, user)
	if err != nil {
		s.Logger.Error(ErrFailedToRegisterUser, "func", funcName, "user", username, "error", err)
		return "", fmt.Errorf("%s: %w", ErrFailedToRegisterUser, err)
	}
	s.Logger.Info("User registered successfully", "func", funcName, "user", username, "ID", userID)
	s.Logger.Debug("Exiting function", "func", funcName, "user", username)
	return userID, nil
}

// AuthenticateUser verifies a user's credentials and returns their ID or an error.
func (s *UserService) AuthenticateUser(ctx context.Context, username, password string) (bool, error) {
	funcName := helper.GetFuncName()
	s.Logger.Debug("Entering function", "func", funcName, "user", username)
	user, err := s.UserRepo.GetUserByUsername(ctx, username)
	if err != nil {
		s.Logger.Error(ErrRetrievingUser, "func", funcName, "user", username, "error", err)
		s.Logger.Debug("Exiting function", "func", funcName, "user", username)
		return false, fmt.Errorf("%s: %w", ErrRetrievingUser, err)
	}
	if user == nil {
		s.Logger.Error(ErrUserNotFound, "func", funcName, "user", username)
		s.Logger.Debug("Exiting function", "func", funcName, "user", username)
		return false, fmt.Errorf("%s: %w", ErrUserNotFound, err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	if err != nil {
		s.Logger.Error(ErrInvalidPassword, "func", funcName, "user", username)
		s.Logger.Debug("Exiting function", "func", funcName, "user", username)
		return false, fmt.Errorf("%s: %w", ErrInvalidPassword, err)
	}

	s.Logger.Info("User authenticated successfully", "func", funcName, "user", username)
	s.Logger.Debug("Exiting function", "func", funcName, "user", username)
	return true, nil
}
