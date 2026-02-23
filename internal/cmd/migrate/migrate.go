package migrate

import (
	"context"
	"log/slog"
	"os"

	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/db/migrations"
	"github.com/spf13/cobra"
)

type Options struct {
	Down bool
	To   uint
}

func (opts *Options) Bind(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&opts.Down, "down", opts.Down,
		"run all down migrations. DANGER: This will purge the database!")
	cmd.Flags().UintVar(&opts.To, "to", opts.To,
		"run all up/down migrations to reach specified schema revision")
	cmd.MarkFlagsMutuallyExclusive("down", "to")
}

func NewCommand() *cobra.Command {
	opts := Options{}
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "execute database migrations",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMigrate(cmd.Context(), opts); err != nil {
				os.Exit(1)
			}
		},
	}
	opts.Bind(cmd)
	return cmd
}

func runMigrate(ctx context.Context, options Options) error {
	if err := config.NewConfig(""); err != nil {
		slog.Warn("config error", "error", err)
	}

	db, err := db.NewPool(ctx, config.DatabaseURI())
	if err != nil {
		slog.Error("database init error", "error", err)
		return err
	}
	defer db.Close()

	if options.To > 0 {
		slog.Info("run migrations", "to", options.To)
		err = migrations.Migrate(db, options.To)
	} else if options.Down {
		slog.Info("run down migrations")
		err = migrations.Down(db)
	} else {
		slog.Info("run up migrations")
		err = migrations.Up(db)
	}

	if err != nil {
		slog.Error("an error occurred during migrations", "error", err)
	}

	return err
}
