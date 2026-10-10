package store

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"workshop-api/internal/model"
)

// ErrDuplicatePlate signals that a vehicle with the same normalized plate
// already exists (AC-02). The handler maps it to HTTP 409.
var ErrDuplicatePlate = errors.New("duplicate_plate")

// NormalizePlate canonicalizes a plate for duplicate detection: upper case,
// with all whitespace and hyphens removed (AC-02). It is a pure helper shared
// by the vehicle and order-lookup queries.
func NormalizePlate(plate string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r == '-' || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, plate)
	return strings.ToUpper(cleaned)
}

// CreateVehicle inserts a vehicle for the given plate. The plate is normalized
// into plate_normalized, which the unique index enforces: a second vehicle with
// the same plate in any casing or spacing returns ErrDuplicatePlate (AC-02).
// The query is parametrized (AC-28); no plate is ever logged (AC-35).
func (s *Store) CreateVehicle(ctx context.Context, plate, brand, modelName string, mileage int) (model.Vehicle, error) {
	const query = `
		INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, plate, plate_normalized, brand, model, mileage, created_at`

	var v model.Vehicle
	err := s.pool.QueryRow(ctx, query, strings.TrimSpace(plate), NormalizePlate(plate), brand, modelName, mileage).
		Scan(&v.ID, &v.Plate, &v.PlateNormalized, &v.Brand, &v.Model, &v.Mileage, &v.CreatedAt)
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return model.Vehicle{}, ErrDuplicatePlate
		}
		return model.Vehicle{}, err
	}
	return v, nil
}

// VehicleByPlate returns the vehicle with the given normalized plate.
func (s *Store) VehicleByPlate(ctx context.Context, plateNormalized string) (model.Vehicle, error) {
	const query = `
		SELECT id, plate, plate_normalized, brand, model, mileage, created_at
		FROM vehicles
		WHERE plate_normalized = $1`

	var v model.Vehicle
	err := s.pool.QueryRow(ctx, query, plateNormalized).
		Scan(&v.ID, &v.Plate, &v.PlateNormalized, &v.Brand, &v.Model, &v.Mileage, &v.CreatedAt)
	if err != nil {
		return model.Vehicle{}, err
	}
	return v, nil
}
