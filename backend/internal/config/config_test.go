package config

import (
	"strings"
	"testing"
)

func TestLoadReadsAllValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:pw@localhost:5432/db")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("HOURLY_RATE_CENTS", "9500")
	t.Setenv("WORKSHOP_EMPLOYEE_EMAIL", "chef@werkstatt.test")
	t.Setenv("WORKSHOP_EMPLOYEE_PASSWORD", "s3cret")
	t.Setenv("AUTH_TOKEN_SECRET", "topsecret")
	t.Setenv("AUTH_TOKEN_TTL_MINUTES", "120")
	t.Setenv("FRONTEND_ORIGIN", "http://localhost:5173/")
	t.Setenv("PORT", "9000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.DatabaseURL != "postgresql://user:pw@localhost:5432/db" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.HourlyRateCents != 9500 {
		t.Errorf("HourlyRateCents = %d", cfg.HourlyRateCents)
	}
	if cfg.AuthTokenTTLMinutes != 120 {
		t.Errorf("AuthTokenTTLMinutes = %d", cfg.AuthTokenTTLMinutes)
	}
	if cfg.FrontendOrigin != "http://localhost:5173" {
		t.Errorf("FrontendOrigin = %q, want trailing slash trimmed", cfg.FrontendOrigin)
	}
	if cfg.Port != "9000" {
		t.Errorf("Port = %q", cfg.Port)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:pw@localhost:5432/db")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("WORKSHOP_EMPLOYEE_EMAIL", "chef@werkstatt.test")
	t.Setenv("WORKSHOP_EMPLOYEE_PASSWORD", "s3cret")
	t.Setenv("AUTH_TOKEN_SECRET", "topsecret")
	t.Setenv("AUTH_TOKEN_TTL_MINUTES", "")
	t.Setenv("HOURLY_RATE_CENTS", "")
	t.Setenv("FRONTEND_ORIGIN", "")
	t.Setenv("PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.AuthTokenTTLMinutes != 480 {
		t.Errorf("AuthTokenTTLMinutes default = %d, want 480", cfg.AuthTokenTTLMinutes)
	}
	if cfg.FrontendOrigin != DefaultFrontendOrigin {
		t.Errorf("FrontendOrigin default = %q, want %q", cfg.FrontendOrigin, DefaultFrontendOrigin)
	}
	if cfg.Port != "8000" {
		t.Errorf("Port default = %q, want 8000", cfg.Port)
	}
}

// TestLoadMissingRequiredNamesVariable proves a missing required value names the
// variable in the error so the operator knows what to set (AC-31).
func TestLoadMissingRequiredNamesVariable(t *testing.T) {
	for _, name := range required {
		t.Setenv(name, "")
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded with all required variables missing")
	}
	for _, name := range required {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err.Error(), name)
		}
	}
}

// TestFrontendOriginPlaceholderFallsBack proves an unresolved service
// placeholder (the web service is another ticket) falls back to the dev default
// instead of becoming a bogus allowed origin.
func TestFrontendOriginPlaceholderFallsBack(t *testing.T) {
	t.Setenv("FRONTEND_ORIGIN", "${service:web.origin}")
	if got := frontendOrigin(); got != DefaultFrontendOrigin {
		t.Errorf("frontendOrigin() = %q, want %q", got, DefaultFrontendOrigin)
	}
}
