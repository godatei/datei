package cmd

import (
	"github.com/godatei/datei/internal/buildconfig"
	"github.com/godatei/datei/internal/cmd/migrate"
	"github.com/godatei/datei/internal/cmd/serve"
	"github.com/spf13/cobra"
)

func NewCLI() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "datei",
		Version: buildconfig.Version(),
	}

	cmd.AddCommand(
		serve.NewCommand(),
		migrate.NewCommand(),
	)

	return cmd
}
