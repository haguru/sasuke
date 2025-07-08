package models

// User represents an internal user model for the application/database.
type User struct {
	Username       string `bson:"username" mapstructure:"username" db:"username"`
	HashedPassword string `bson:"hashed_password" mapstructure:"hashed_password" db:"hashed_password"`
}

// NewUser creates a new User instance with the given username and password.
// Note: No validation is performed here.
func NewUser(username string, hashedPassword string) *User {
	return &User{
		Username:       username,
		HashedPassword: hashedPassword,
	}
}
