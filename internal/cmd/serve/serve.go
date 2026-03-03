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
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/godatei/datei/internal/aggregate"
	"github.com/godatei/datei/internal/authn"
	"github.com/godatei/datei/internal/buildconfig"
	"github.com/godatei/datei/internal/config"
	"github.com/godatei/datei/internal/db"
	"github.com/godatei/datei/internal/db/migrations"
	"github.com/godatei/datei/internal/events"
	"github.com/godatei/datei/internal/frontend"
	"github.com/godatei/datei/internal/mailer"
	"github.com/godatei/datei/internal/server"
	"github.com/godatei/datei/internal/storage"
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

	dateiEventStore := events.NewPostgresEventStore(db)
	dateiRepository := aggregate.NewPostgresDateiRepository(db, dateiEventStore)

	userEventStore := events.NewUserAccountEventStore(db)
	userRepository := aggregate.NewPostgresUserRepository(db, userEventStore)

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

	// Datei API routes (oapi-codegen generated, auth required)
	apiMux := chi.NewRouter()
	apiMux.Use(
		chimiddleware.RequestID,
		chimiddleware.RealIP,
		slogchi.New(slog.Default()),
		authn.Middleware,
		oapimiddleware.OapiRequestValidator(swagger),
	)
	strictHandler := server.NewStrictHandler(server.NewServer(db, store, dateiRepository), nil)
	server.HandlerFromMux(strictHandler, apiMux)

	// Auth routes (public, rate-limited)
	authMux := chi.NewRouter()
	authMux.Use(
		chimiddleware.RequestID,
		chimiddleware.RealIP,
		slogchi.New(slog.Default()),
	)
	authMux.Mount("/", server.AuthRoutes(db, userRepository, m))

	// Settings routes (auth required)
	settingsMux := chi.NewRouter()
	settingsMux.Use(
		chimiddleware.RequestID,
		chimiddleware.RealIP,
		slogchi.New(slog.Default()),
	)
	settingsMux.Mount("/", server.SettingsRoutes(db, userRepository, m))

	rootMux := chi.NewRouter()
	rootMux.Use(chimiddleware.Recoverer)
	rootMux.Mount("/api/v1/auth", authMux)
	rootMux.Mount("/api/v1/settings", settingsMux)
	rootMux.Handle("/api/*", apiMux)
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
