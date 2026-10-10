// Package config loads the API configuration from the environment.
//
// Every value is read lazily by Load, which is called once from main during
// startup. A missing required value is reported by NAME so the operator knows
// exactly what to set; there is never a literal fallback for a secret.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the fully resolved configuration of the API service.
type Config struct {
	DatabaseURL              string
	ValkeyURL                string
	HourlyRateCents          int
	WorkshopEmployeeEmail    string
	WorkshopEmployeePassword string
	AuthTokenSecret          string
	AuthTokenTTLMinutes      int
	FrontendOrigin           string
	Port                     string
}

// DefaultFrontendOrigin is used when FRONTEND_ORIGIN is not configured yet
// (for example before the web service is declared by its own ticket).
const DefaultFrontendOrigin = "http://localhost:5173"

// required lists the variables the API cannot boot without. The workshop
// employee credentials are required because the API creates the first employee
// from them at startup (AC-15).
var required = []string{
	"DATABASE_URL",
	"VALKEY_URL",
	"AUTH_TOKEN_SECRET",
	"WORKSHOP_EMPLOYEE_EMAIL",
	"WORKSHOP_EMPLOYEE_PASSWORD",
}

// Load reads and validates the configuration. It returns an error naming the
// first missing required variable; callers (main) log it and exit non-zero.
func Load() (Config, error) {
	missing := make([]string, 0, len(required))
	for _, name := range required {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	hourlyRate := 0
	if raw := strings.TrimSpace(os.Getenv("HOURLY_RATE_CENTS")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("HOURLY_RATE_CENTS must be an integer number of cents: %q", raw)
		}
		hourlyRate = v
	}

	ttlMinutes := 480
	if raw := strings.TrimSpace(os.Getenv("AUTH_TOKEN_TTL_MINUTES")); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			return Config{}, fmt.Errorf("AUTH_TOKEN_TTL_MINUTES must be an integer number of minutes: %q", raw)
		}
		ttlMinutes = v
	}

	return Config{
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		ValkeyURL:                os.Getenv("VALKEY_URL"),
		HourlyRateCents:          hourlyRate,
		WorkshopEmployeeEmail:    os.Getenv("WORKSHOP_EMPLOYEE_EMAIL"),
		WorkshopEmployeePassword: os.Getenv("WORKSHOP_EMPLOYEE_PASSWORD"),
		AuthTokenSecret:          os.Getenv("AUTH_TOKEN_SECRET"),
		AuthTokenTTLMinutes:      ttlMinutes,
		FrontendOrigin:           frontendOrigin(),
		Port:                     port(),
	}, nil
}

// frontendOrigin returns the configured frontend origin, falling back to the
// dev default when it is unset or still an unresolved service placeholder (the
// web service is declared by a different ticket and may not exist yet).
func frontendOrigin() string {
	raw := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN"))
	if raw == "" || strings.Contains(raw, "${") {
		return DefaultFrontendOrigin
	}
	return strings.TrimRight(raw, "/")
}

func port() string {
	if raw := strings.TrimSpace(os.Getenv("PORT")); raw != "" {
		return raw
	}
	return "8000"
}
