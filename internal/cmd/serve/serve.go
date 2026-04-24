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

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/buildconfig"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/datei"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/db/migrations"
	"github.com/godatei/datei/internal/frontend"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/server"
	"github.com/godatei/datei/internal/storage"
	"github.com/godatei/datei/internal/users"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
	slogchi "github.com/samber/slog-chi"
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

	if config.LoggingLevel() == "debug" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	db, err := db.NewPool(ctx, config.DatabaseURI())
	if err != nil {
		slog.Error("database init error", "error", err)
		return err
	}
	defer db.Close()

	if config.DatabaseMigrations() {
		slog.Info("running migrations")
		if err := migrations.Up(db); err != nil {
			slog.Error("migrations failed", "error", err)
			return err
		}
	}

	sc, err := config.StorageS3()
	if err != nil {
		slog.Error("invalid storage config", "error", err)
		return err
	}
	store := storage.NewS3Store(ctx, sc)

	if err := store.Initialize(ctx); err != nil {
		slog.Error("failed to initialize storage", "error", err)
		return err
	}

	dateiEventStore := datei.NewEventStore(db)
	dateiRepository := datei.NewRepository(db, dateiEventStore)

	userEventStore := users.NewEventStore(db)
	userRepository := users.NewRepository(db, userEventStore)

	// Create mailer
	var m mailer.Mailer
	mc := config.Mailer()
	if mc.Enabled {
		m = mailer.NewSMTPMailer(mc.SMTP.Host, mc.SMTP.Port, mc.SMTP.Username, mc.SMTP.Password, mc.SMTP.From)
	} else {
		m = mailer.NewNoopMailer()
	}

	swagger, err := server.GetSwagger()
	if err != nil {
		slog.Error("swagger error", "error", err)
		return err
	}

	// Create the unified server implementing StrictServerInterface
	srv := server.NewServer(db, store, dateiRepository, userRepository, m)
	strictHandler := server.NewStrictHandler(srv, nil)

	rootMux := chi.NewRouter()
	rootMux.Use(chimiddleware.Recoverer)
	rootMux.Use(chimiddleware.RequestID)
	rootMux.Use(chimiddleware.RealIP)
	rootMux.Use(slogchi.New(slog.Default()))

	// API routes: OpenAPI validator handles auth via security schemes in the spec
	rootMux.Group(func(r chi.Router) {
		r.Use(oapimiddleware.OapiRequestValidatorWithOptions(swagger, &oapimiddleware.Options{
			SilenceServersWarning: true,
			Options: openapi3filter.Options{
				AuthenticationFunc: authn.OpenAPIAuthFunc(),
			},
		}))
		r.Use(httprate.Limit(
			100, 1*time.Minute,
			httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
		))
		server.HandlerWithOptions(strictHandler, server.ChiServerOptions{
			BaseRouter: r,
		})
	})

	rootMux.Handle("/*", frontend.NewHandler())

	httpServer := &http.Server{Handler: rootMux, Addr: config.ServerAddr()}

	shutdownComplete := make(chan struct{})
	sigCtx, sigCancel := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer sigCancel()
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
