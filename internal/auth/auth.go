package auth

import (
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	ISSUER  = "github.com/haguru/sasuke.com"
	SUBJECT = "AUTHENTICATION"
	// this should not be hard coded. create a new ecdsa private key and store it to be reused.
	// this is just for practice for now
	SECRETKEY = "secret-key-this_should_be_32_bytes_long"
)

// var jwtSecret = []byte(SECRETKEY)

type CustomClaims struct {
	UserID string `json:"userid"`
	jwt.RegisteredClaims
}

func CreateToken(userName string, privateKey *ecdsa.PrivateKey) (string, error) {
	claims := CustomClaims{
		UserID: userName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    ISSUER,
			Subject:   SUBJECT,
			Audience:  []string{"api" + ISSUER},
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	signToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}

	return signToken, nil
}

func VerifyToken(tokenString string, publicKey *ecdsa.PublicKey) (*CustomClaims, error) {
	// check key type for the correct signing method
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token parsing error: %v", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token or claims")
}
