// Package auth issues and verifies the HS256 bearer tokens of the workshop area.
package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the payload of a workshop bearer token: the employee id (sub), the
// employee email and the expiry (exp).
type Claims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

// GetExpirationTime implements jwt.Claims.
func (c *Claims) GetExpirationTime() (*jwt.NumericDate, error) {
	if c.Exp == 0 {
		return nil, nil
	}
	return jwt.NewNumericDate(time.Unix(c.Exp, 0)), nil
}

// GetIssuedAt implements jwt.Claims.
func (c *Claims) GetIssuedAt() (*jwt.NumericDate, error) { return nil, nil }

// GetNotBefore implements jwt.Claims.
func (c *Claims) GetNotBefore() (*jwt.NumericDate, error) { return nil, nil }

// GetIssuer implements jwt.Claims.
func (c *Claims) GetIssuer() (string, error) { return "", nil }

// GetSubject implements jwt.Claims.
func (c *Claims) GetSubject() (string, error) { return c.Sub, nil }

// GetAudience implements jwt.Claims.
func (c *Claims) GetAudience() (jwt.ClaimStrings, error) { return nil, nil }
