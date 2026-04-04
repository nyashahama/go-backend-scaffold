package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	dbgen "github.com/nyashahama/go-backend-scaffold/db/gen"
)

// Pool wraps pgxpool.Pool with a ready-to-use *dbgen.Queries so
// callers only need to carry one value.
type Pool struct {
	*pgxpool.Pool
	Q *dbgen.Queries
}

// New creates a pgx connection pool with production-ready defaults.
func New(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Pool{
		Pool: pool,
		Q:    dbgen.New(pool),
	}, nil
}

// Ping implements health.Checker so *Pool can be passed directly to the
// health handler.
func (p *Pool) Ping(ctx context.Context) error {
	return p.Pool.Ping(ctx)
}
