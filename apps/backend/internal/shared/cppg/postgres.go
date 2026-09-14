// Package cppg is the Postgres client, ported from on-core-platform's onpg and
// cut down to what this process uses: no gorm, no tunnel, no tracing, no lazy
// config. Queries are plain database/sql over lib/pq.
package cppg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // registers the postgres driver for migrate
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq" // registers the postgres driver for database/sql
)

type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Beginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// QuerierBeginner is what a store that also needs a transaction depends on — narrower than *Postgres.
type QuerierBeginner interface {
	Querier
	Beginner
}

var _ QuerierBeginner = (*Postgres)(nil)

// PoolConfig leaves nil at the database/sql default rather than setting it to zero.
type PoolConfig struct {
	MaxOpenConns    *int
	MaxIdleConns    *int
	ConnMaxLifetime *time.Duration
	ConnMaxIdleTime *time.Duration
}

// Config is the `database:` block. Every module that stores something reads the same one and opens its own pool.
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	Pool     PoolConfig
}

// String keeps the password out of the boot's config log line.
func (c Config) String() string {
	return fmt.Sprintf("{Host:%s Port:%s User:%s DBName:%s SSLMode:%s}", c.Host, c.Port, c.User, c.DBName, c.SSLMode)
}

func (c Config) Validate() error {
	var missing []string
	for _, field := range []struct{ key, value string }{
		{"host", c.Host},
		{"port", c.Port},
		{"user", c.User},
		{"dbName", c.DBName},
		{"sslMode", c.SSLMode},
	} {
		if field.value == "" {
			missing = append(missing, "database."+field.key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%v is empty: the process keeps its state in postgres and cannot start without it", missing)
	}

	return nil
}

func New(config Config) *Postgres {
	return &Postgres{config: config}
}

type Postgres struct {
	config Config

	sqlClient *sql.DB
}

func (p *Postgres) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	tx, err := p.sqlClient.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

func (p *Postgres) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := p.sqlClient.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to exec query: %w", err)
	}
	return result, nil
}

func (p *Postgres) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := p.sqlClient.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	return rows, nil
}

func (p *Postgres) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return p.sqlClient.QueryRowContext(ctx, query, args...)
}

func (p *Postgres) dsn() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.config.Host,
		p.config.Port,
		p.config.User,
		p.config.Password,
		p.config.DBName,
		p.config.SSLMode,
	)
}

func (p *Postgres) url() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(p.config.User, p.config.Password),
		Host:   fmt.Sprintf("%s:%s", p.config.Host, p.config.Port),
		Path:   "/" + p.config.DBName,
	}

	q := u.Query()
	q.Set("sslmode", p.config.SSLMode)
	u.RawQuery = q.Encode()

	return u.String()
}

func applyPoolConfig(db *sql.DB, cfg PoolConfig) {
	if cfg.MaxOpenConns != nil {
		db.SetMaxOpenConns(*cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != nil {
		db.SetMaxIdleConns(*cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != nil {
		db.SetConnMaxLifetime(*cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != nil {
		db.SetConnMaxIdleTime(*cfg.ConnMaxIdleTime)
	}
}

// ConnectCtx pings before returning: sql.Open is lazy, and a database that is not there should refuse the boot rather than the first query.
func (p *Postgres) ConnectCtx(ctx context.Context) error {
	sqlDB, err := sql.Open("postgres", p.dsn())
	if err != nil {
		return fmt.Errorf("open sql: %w", err)
	}

	applyPoolConfig(sqlDB, p.config.Pool)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return fmt.Errorf("ping postgres at %s:%s: %w", p.config.Host, p.config.Port, err)
	}

	p.sqlClient = sqlDB

	return nil
}

func (p *Postgres) Close() error {
	if p.sqlClient == nil {
		return nil
	}

	return p.sqlClient.Close()
}

// Migrate applies every migration in migrations not applied yet. It opens a connection of its own, so it needs no ConnectCtx first.
func (p *Postgres) Migrate(migrations fs.FS) error {
	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("failed to create iofs source: %w", err)
	}
	defer func() { _ = source.Close() }()

	m, err := migrate.NewWithSourceInstance("iofs", source, p.url())
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
