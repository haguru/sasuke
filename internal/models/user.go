package models

type User struct {
	Username       string `bson:"username" mapstructure:"username" db:"username"`
	HashedPassword string `bson:"hashed_password" mapstructure:"hashed_password" db:"hashed_password"`
}


func NewUser(username string, hashedPassword string) *User {
	return &User{
		Username:       username,
		HashedPassword: hashedPassword,
	}
}
