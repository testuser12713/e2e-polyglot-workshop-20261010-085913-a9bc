package auth

// VerifyToken parses and validates an HS256 bearer token and returns its
// claims. The auth ticket implements the body.
func VerifyToken(token string) (*Claims, error) {
	return nil, ErrNotImplemented
}
