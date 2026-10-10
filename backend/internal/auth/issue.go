package auth

import (
	"errors"
	"time"

	"workshop-api/internal/model"
)

// ErrNotImplemented is returned by the seams the feature tickets fill in.
var ErrNotImplemented = errors.New("not_implemented")

// IssueToken signs a new HS256 bearer token for the employee and returns the
// token together with its expiry time. The login ticket implements the body.
func IssueToken(emp model.Employee, secret string, ttl time.Duration) (string, time.Time, error) {
	return "", time.Time{}, ErrNotImplemented
}
