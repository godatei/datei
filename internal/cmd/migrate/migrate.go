package migrate

import (
	"context"
	"os"

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
	// TODO: Code for migrate command goes here
	return nil
}
