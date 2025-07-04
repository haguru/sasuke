package models

// User represents a user in the application.
// It contains fields for the user's ID, username, and password.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// NewUser creates a new User instance with the specified username and password.
// It initializes the ID to an empty string, which can be set later by the database or
// application logic.
// This function is typically used when creating a new user in the application.
// It returns a pointer to the newly created User instance.
// Example usage:
// user := NewUser("john_doe", "securepassword123")
// This function does not perform any validation on the username or password.
// It is the responsibility of the caller to ensure that the provided values meet any
// necessary criteria (e.g., length, format, etc.).
// The returned User instance can be used to interact with the application or database,
// such as saving the user to a database or performing authentication checks.
// Note: The ID field is left empty for the caller to set or for the database to
// populate it upon insertion.
func NewUser(username string, password string) *User {
	return &User{
		Username: username,
		Password: password,
	}
}
