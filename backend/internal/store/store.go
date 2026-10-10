// Package store is the data access layer of the workshop API. It wraps a pgx
// connection pool; one file per feature holds that feature's queries. The
// skeleton ships the method signatures the feature tickets fill in.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotImplemented is returned by the stub methods the feature tickets fill in.
var ErrNotImplemented = errors.New("not_implemented")

// Store provides access to the workshop database.
type Store struct {
	pool *pgxpool.Pool
}

// New builds a Store around an open pgx pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Pool exposes the underlying connection pool.
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// Ping checks that the database is reachable. It backs the health endpoint so
// health exercises the same configuration the real routes use.
func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}
