package migrations

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed sql/*
var fs embed.FS

type migrateSlogAdapter struct{}

// Printf implements migrate.Logger.
func (l *migrateSlogAdapter) Printf(format string, v ...interface{}) {
	if strings.HasPrefix(format, "error") {
		slog.Error(fmt.Sprintf(strings.TrimSpace(format), v...))
	} else {
		slog.Debug(fmt.Sprintf(strings.TrimSpace(format), v...))
	}
}

// Verbose implements migrate.Logger.
func (l *migrateSlogAdapter) Verbose() bool {
	return slog.Default().Enabled(context.TODO(), slog.LevelDebug)
}

var _ migrate.Logger = &migrateSlogAdapter{}

func Up(db *pgxpool.Pool) (err error) {
	instance, err := getInstance(db)
	if err != nil {
		return err
	}

	defer func() {
		a, b := instance.Close()
		err = errors.Join(err, a, b)
	}()

	if err := instance.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		slog.Info("migrations completed", "error", err)
	}
	return nil
}

func Down(db *pgxpool.Pool) (err error) {
	instance, err := getInstance(db)
	if err != nil {
		return err
	}

	defer func() {
		a, b := instance.Close()
		err = errors.Join(err, a, b)
	}()

	if err := instance.Down(); err != nil {
		return err
	}
	return nil
}

func Migrate(db *pgxpool.Pool, to uint) (err error) {
	instance, err := getInstance(db)
	if err != nil {
		return err
	}

	defer func() {
		a, b := instance.Close()
		err = errors.Join(err, a, b)
	}()

	if err := instance.Migrate(to); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		slog.Info("migrations completed", "error", err)
	}
	return nil
}

func getInstance(db *pgxpool.Pool) (*migrate.Migrate, error) {
	if driver, err := postgres.WithInstance(stdlib.OpenDBFromPool(db), &postgres.Config{}); err != nil {
		return nil, err
	} else if sourceInstance, err := iofs.New(fs, "sql"); err != nil {
		return nil, err
	} else if instance, err := migrate.NewWithInstance("", sourceInstance, "datei", driver); err != nil {
		return nil, err
	} else {
		instance.Log = &migrateSlogAdapter{}
		return instance, nil
	}
}
