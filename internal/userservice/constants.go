package userservice

const (
	// Error messages for user service operations
	ErrFailedToHashPassword = "failed to hash password" // #nosec G101
	ErrFailedToRegisterUser = "failed to register user"
	ErrRetrievingUser       = "error retrieving user"
	ErrUserNotFound         = "user not found"
	ErrInvalidPassword      = "invalid password"
)
