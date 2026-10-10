// Command api is the workshop portal HTTP API: it loads the configuration,
// opens the PostgreSQL pool, applies the schema, ensures the first employee and
// serves the HTTP routes.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop-api/internal/config"
	"workshop-api/internal/httpapi"
	"workshop-api/internal/store"
)

const migrationsPath = "migrations/0001_init.sql"

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)

	cfg, err := config.Load()
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := openPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("database error: %v", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := applyMigrations(ctx, pool, migrationsPath); err != nil {
		log.Printf("migration error: %v", err)
		os.Exit(1)
	}

	st := store.New(pool)
	if err := st.EnsureFirstEmployee(ctx, cfg.WorkshopEmployeeEmail, cfg.WorkshopEmployeePassword); err != nil {
		log.Printf("could not ensure the first employee: %v", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(cfg, st),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("workshop api listening on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

// openPool opens and verifies the PostgreSQL connection pool.
func openPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// applyMigrations executes the idempotent DDL file statement by statement.
func applyMigrations(ctx context.Context, pool *pgxpool.Pool, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, stmt := range strings.Split(string(raw), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
