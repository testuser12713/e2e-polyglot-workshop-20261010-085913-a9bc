package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"workshop-api/internal/model"
)

// ErrNotImplemented is returned by the seams the feature tickets fill in.
var ErrNotImplemented = errors.New("not_implemented")

// IssueToken signs a new HS256 bearer token for the employee and returns the
// token together with its expiry time (AC-14). The claims carry the employee id
// (sub), the employee e-mail and the expiry (exp), signed with secret.
func IssueToken(emp model.Employee, secret string, ttl time.Duration) (string, time.Time, error) {
	if secret == "" {
		return "", time.Time{}, errors.New("auth token secret is empty")
	}
	if ttl <= 0 {
		return "", time.Time{}, errors.New("auth token ttl must be positive")
	}

	expiresAt := time.Now().UTC().Add(ttl)
	claims := &Claims{
		Sub:   strconv.FormatInt(emp.ID, 10),
		Email: emp.Email,
		Exp:   expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}
