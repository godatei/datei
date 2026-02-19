package serve

import (
	"context"
	"os"

	"github.com/spf13/cobra"
)

type Options struct{}

func (opts *Options) Bind(cmd *cobra.Command) {}

func NewCommand() *cobra.Command {
	opts := Options{}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "run the Datei server",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if err := run(cmd.Context(), opts); err != nil {
				os.Exit(1)
			}
		},
	}
	opts.Bind(cmd)
	return cmd
}

func run(ctx context.Context, options Options) error {
	// TODO: Code for serve command goes here
	return nil
}
