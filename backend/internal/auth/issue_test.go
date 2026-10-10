package auth

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"workshop-api/internal/model"
)

const issueTestSecret = "issue-test-secret-please-change-me"

func TestIssueTokenSignsHS256WithExpectedClaims(t *testing.T) {
	emp := model.Employee{ID: 42, Email: "chef@werkstatt.example"}
	ttl := 90 * time.Minute

	before := time.Now()
	token, expiresAt, err := IssueToken(emp, issueTestSecret, ttl)
	after := time.Now()
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if token == "" {
		t.Fatal("IssueToken returned an empty token")
	}

	lower := before.Add(ttl).Add(-time.Minute)
	upper := after.Add(ttl).Add(time.Minute)
	if expiresAt.Before(lower) || expiresAt.After(upper) {
		t.Fatalf("expiresAt = %v, want a value near %v", expiresAt, before.Add(ttl))
	}

	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", tok.Header["alg"])
		}
		return []byte(issueTestSecret), nil
	})
	if err != nil {
		t.Fatalf("parse issued token: %v", err)
	}
	if !parsed.Valid {
		t.Fatal("issued token is not valid")
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		t.Fatalf("claims have type %T, want *Claims", parsed.Claims)
	}
	if claims.Email != emp.Email {
		t.Fatalf("email claim = %q, want %q", claims.Email, emp.Email)
	}
	if claims.Sub != strconv.FormatInt(emp.ID, 10) {
		t.Fatalf("sub claim = %q, want %q", claims.Sub, strconv.FormatInt(emp.ID, 10))
	}
	if claims.Exp != expiresAt.Unix() {
		t.Fatalf("exp claim = %d, want %d", claims.Exp, expiresAt.Unix())
	}

	if _, err := jwt.ParseWithClaims(token, &Claims{}, func(tok *jwt.Token) (any, error) {
		return []byte("a-different-secret"), nil
	}); err == nil {
		t.Fatal("token verified under a foreign secret; want verification failure")
	}
}

func TestIssueTokenRejectsEmptySecret(t *testing.T) {
	if _, _, err := IssueToken(model.Employee{ID: 1}, "", time.Minute); err == nil {
		t.Fatal("IssueToken with an empty secret returned no error")
	}
}
