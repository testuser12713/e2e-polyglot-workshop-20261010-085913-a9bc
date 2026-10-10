package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned for a token that is malformed, carries an
// unexpected signing algorithm, was signed with the wrong secret or has expired.
var ErrInvalidToken = errors.New("invalid token")

// secretFromEnv reads the HS256 signing secret used to verify tokens. It is
// read lazily on every verification so the process boots without a hard import
// failure and a missing value is reported by name.
func secretFromEnv() (string, error) {
	secret := strings.TrimSpace(os.Getenv("AUTH_TOKEN_SECRET"))
	if secret == "" {
		return "", errors.New("AUTH_TOKEN_SECRET is not set")
	}
	return secret, nil
}

// VerifyToken parses and validates an HS256 bearer token signed with
// AUTH_TOKEN_SECRET. It rejects a wrong signature, a malformed token and an
// expired exp, and returns the contract claims (sub, email, exp) on success.
func VerifyToken(token string) (*Claims, error) {
	secret, err := secretFromEnv()
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
