package mysql

import (
	"context"
	"database/sql"
	"errors"
	"runtime"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/qor/validations"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/sirupsen/logrus"

	"soldr/pkg/secret"
)

type DB struct {
	session *sqlx.DB
	conf    Config
}

const (
	driverName             = "mysql"
	defaultMaxConnLifetime = time.Hour
	defaultMaxOpenConns    = 4
)

type Config struct {
	DSN             secret.String
	MaxConnLifetime time.Duration
	MaxOpenConns    int
}

func (c *Config) CheckAndSetDefaults() error {
	if c.DSN.Unmask() == "" {
		return errors.New("DSN is required")
	}
	if c.MaxConnLifetime == 0 {
		c.MaxConnLifetime = defaultMaxConnLifetime
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = defaultMaxOpenConns
	}
	if numCPU := runtime.NumCPU(); numCPU > c.MaxOpenConns {
		c.MaxOpenConns = numCPU
	}
	return nil
}

func New(cfg *Config) (*DB, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, cfg.DSN.Unmask())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetConnMaxLifetime(cfg.MaxConnLifetime)

	dbx := sqlx.NewDb(db, driverName)
	return &DB{session: dbx, conf: *cfg}, nil
}

// DSN returns DSN from the config.
func (d *DB) DSN() secret.String {
	return d.conf.DSN
}

func (d *DB) RetryConnect(ctx context.Context, maxAttempts int, backoff time.Duration) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err = d.Connect(ctx); err == nil {
				break
			}
			nextAttemptWait := time.Duration(attempt) * backoff
			logrus.Warnf(
				"attempt %v: could not establish a connection with the database, wait for %v.",
				attempt,
				nextAttemptWait,
			)
			time.Sleep(nextAttemptWait)
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (d *DB) Connect(ctx context.Context) error {
	return d.session.PingContext(ctx)
}

func (d *DB) Close() error {
	return d.session.Close()
}

func (d *DB) Migrate(path string) error {
	migrations := &migrate.FileMigrationSource{Dir: path}
	_, err := migrate.Exec(d.session.DB, driverName, migrations, migrate.Up)
	return err
}

func (d *DB) WithORM() (*gorm.DB, error) {
	conn, err := gorm.Open("mysql", d.DSN().Unmask())
	if err != nil {
		return nil, err
	}
	conn.SetLogger(&GormLogger{})
	validations.RegisterCallbacks(conn)
	ApplyGormMetrics(conn)
	return conn, nil
}
