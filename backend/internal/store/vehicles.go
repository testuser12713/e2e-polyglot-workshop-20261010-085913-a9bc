package store

import (
	"context"
	"strings"

	"workshop-api/internal/model"
)

// NormalizePlate canonicalizes a plate for duplicate detection: upper case,
// with all whitespace removed (AC-02). It is a pure helper shared by the
// vehicle and order-lookup queries.
func NormalizePlate(plate string) string {
	return strings.ToUpper(strings.Join(strings.Fields(plate), ""))
}

// UpsertVehicle returns the vehicle for the given normalized plate, creating it
// when it does not exist yet. The public appointment ticket implements the body.
func (s *Store) UpsertVehicle(ctx context.Context, plate, brand, modelName string, mileage int) (model.Vehicle, error) {
	return model.Vehicle{}, ErrNotImplemented
}

// VehicleByPlate returns the vehicle with the given normalized plate.
func (s *Store) VehicleByPlate(ctx context.Context, plateNormalized string) (model.Vehicle, error) {
	return model.Vehicle{}, ErrNotImplemented
}
