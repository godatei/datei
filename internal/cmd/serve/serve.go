package serve

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/godatei/datei/internal/buildconfig"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/db/migrations"
	"github.com/godatei/datei/internal/server"
	middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/spf13/cobra"
)

type Options struct {
	Config string
}

func (opts *Options) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&opts.Config, "config", "", "config file")
}

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
	err := config.NewConfig(options.Config)
	if err != nil {
		slog.Warn("config error", "error", err)
	}

	swagger, err := server.GetSwagger()
	if err != nil {
		slog.Error("swagger error", "error", err)
		return err
	}

	r := chi.NewRouter()
	r.Use(middleware.OapiRequestValidator(swagger))

	db, err := db.NewPool(ctx, config.DatabaseURI())
	if err != nil {
		slog.Error("database init error", "error", err)
		return err
	}
	defer db.Close()

	if config.DatabaseMigrations() {
		if err := migrations.Up(db); err != nil {
			slog.Error("migrations failed", "error", err)
			return err
		}
	}

	strictHandler := server.NewStrictHandler(server.NewServer(db), nil)
	httpServer := &http.Server{Handler: server.HandlerFromMux(strictHandler, r), Addr: config.ServerAddr()}

	shutdownComplete := make(chan struct{})
	sigCtx, _ := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	context.AfterFunc(sigCtx, func() {
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		defer close(shutdownComplete)
		slog.Warn("shutting down")
		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	})

	go func() {
		slog.Info("server is listening", "addr", config.ServerAddr(), "version", buildconfig.Version())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen and serve failed", "error", err)
		}
	}()

	<-shutdownComplete
	slog.Info("shutdown complete")

	return nil
}
