package config

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

// NewDatabase membuka koneksi PostgreSQL via bun dan memastikan DB bisa di-ping.
func NewDatabase(cfg DBConfig) (*bun.DB, error) {
	opts := []pgdriver.Option{
		pgdriver.WithAddr(net.JoinHostPort(cfg.Host, cfg.Port)),
		pgdriver.WithUser(cfg.User),
		pgdriver.WithPassword(cfg.Pass),
		pgdriver.WithDatabase(cfg.Name),
		pgdriver.WithApplicationName("autojurnal"),
		pgdriver.WithTimeout(5 * time.Second),
	}
	if cfg.SSLMode == "disable" {
		opts = append(opts, pgdriver.WithInsecure(true))
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(opts...))
	sqldb.SetMaxOpenConns(cfg.MaxConns)
	sqldb.SetMaxIdleConns(cfg.MaxConns)
	sqldb.SetConnMaxIdleTime(5 * time.Minute)
	sqldb.SetConnMaxLifetime(time.Hour)

	db := bun.NewDB(sqldb, pgdialect.New())
	if cfg.Debug {
		db.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
