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
	"github.com/go-chi/httprate"
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

	// Create the unified server implementing StrictServerInterface
	srv := server.NewServer(db, store, dateiRepository, userRepository, m)
	strictHandler := server.NewStrictHandler(srv, nil)

	// Build a wrapper for per-route registration with different middleware groups
	wrapper := server.ServerInterfaceWrapper{
		Handler: strictHandler,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
	}

	commonMiddleware := chi.Chain(
		chimiddleware.RequestID,
		chimiddleware.RealIP,
		slogchi.New(slog.Default()),
	)

	rootMux := chi.NewRouter()
	rootMux.Use(chimiddleware.Recoverer)

	// Auth routes (public, rate-limited)
	rootMux.Group(func(r chi.Router) {
		r.Use(commonMiddleware...)
		r.Use(httprate.Limit(
			10, 1*time.Minute,
			httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
		))
		r.Post("/api/v1/auth/login", wrapper.Login)
		r.Get("/api/v1/auth/login/config", wrapper.GetLoginConfig)
		r.Post("/api/v1/auth/register", wrapper.Register)
		r.Post("/api/v1/auth/reset", wrapper.ResetPassword)
	})

	// Settings routes that accept action tokens (password reset, email verification)
	rootMux.Group(func(r chi.Router) {
		r.Use(commonMiddleware...)
		r.Use(authn.Middleware)
		r.Post("/api/v1/settings/user", wrapper.UpdateUser)
		r.Post("/api/v1/settings/verify/confirm", wrapper.ConfirmEmailVerification)
	})

	// Settings routes requiring session tokens only
	rootMux.Group(func(r chi.Router) {
		r.Use(commonMiddleware...)
		r.Use(authn.Middleware)
		r.Use(authn.RequireSessionTokenMiddleware)
		r.Patch("/api/v1/settings/user/email", wrapper.UpdateUserEmail)
		r.Post("/api/v1/settings/verify/request", wrapper.RequestEmailVerification)
		r.Post("/api/v1/settings/mfa/setup", wrapper.SetupMFA)
		r.Post("/api/v1/settings/mfa/enable", wrapper.EnableMFA)
		r.Post("/api/v1/settings/mfa/disable", wrapper.DisableMFA)
		r.Post("/api/v1/settings/mfa/recovery-codes/regenerate", wrapper.RegenerateMFARecoveryCodes)
		r.Get("/api/v1/settings/mfa/recovery-codes/status", wrapper.GetMFARecoveryCodesStatus)
		r.Get("/api/v1/settings/emails", wrapper.ListEmails)
		r.Post("/api/v1/settings/emails", wrapper.AddEmail)
		r.Delete("/api/v1/settings/emails/{emailId}", wrapper.RemoveEmail)
		r.Patch("/api/v1/settings/emails/{emailId}/primary", wrapper.SetPrimaryEmail)
	})

	// Datei routes (session token required + OpenAPI request validation)
	rootMux.Group(func(r chi.Router) {
		r.Use(commonMiddleware...)
		r.Use(authn.Middleware)
		r.Use(authn.RequireSessionTokenMiddleware)
		r.Use(oapimiddleware.OapiRequestValidator(swagger))
		r.Get("/api/v1/datei", wrapper.ListDatei)
		r.Post("/api/v1/datei", wrapper.CreateDatei)
		r.Delete("/api/v1/datei/{id}", wrapper.DeleteDatei)
		r.Patch("/api/v1/datei/{id}", wrapper.UpdateDatei)
		r.Get("/api/v1/datei/{id}/download", wrapper.DownloadDatei)
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
