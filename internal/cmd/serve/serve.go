package serve

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

// authRoutes are public endpoints that do not require authentication.
var authRoutes = map[string]bool{
	"/api/v1/auth/login":        true,
	"/api/v1/auth/login/config": true,
	"/api/v1/auth/register":     true,
	"/api/v1/auth/reset":        true,
}

// actionTokenRoutes accept both session tokens and action tokens (password reset, email verification).
// All other authenticated routes require session tokens only.
var actionTokenRoutes = map[string]bool{
	"/api/v1/settings/user":           true,
	"/api/v1/settings/verify/confirm": true,
}

// routeAwareAuthMiddleware applies JWT auth + session-token checks
// based on route classification. Public auth routes are passed through,
// action-token routes get JWT auth only, all others also require session tokens.
func routeAwareAuthMiddleware(next http.Handler) http.Handler {
	jwtMiddleware := authn.Middleware
	sessionMiddleware := authn.RequireSessionTokenMiddleware

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")

		// Public auth routes — no authentication required
		if authRoutes[path] {
			next.ServeHTTP(w, r)
			return
		}

		// All other API routes require JWT authentication
		authenticated := jwtMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Action-token routes accept any valid JWT
			if actionTokenRoutes[path] {
				next.ServeHTTP(w, r)
				return
			}

			// Everything else requires a session token (not action/reset tokens)
			sessionMiddleware(next).ServeHTTP(w, r)
		}))

		authenticated.ServeHTTP(w, r)
	})
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

	rootMux := chi.NewRouter()
	rootMux.Use(chimiddleware.Recoverer)
	rootMux.Use(chimiddleware.RequestID)
	rootMux.Use(chimiddleware.RealIP)
	rootMux.Use(slogchi.New(slog.Default()))

	// API sub-router: OpenAPI validation + auth middleware for all API routes
	apiRouter := chi.NewRouter()
	apiRouter.Use(oapimiddleware.OapiRequestValidator(swagger))
	apiRouter.Use(routeAwareAuthMiddleware)
	apiRouter.Use(httprate.Limit(
		10, 1*time.Minute,
		httprate.WithKeyFuncs(httprate.KeyByRealIP, httprate.KeyByEndpoint),
	))

	// Let the generated code register all routes
	server.HandlerWithOptions(strictHandler, server.ChiServerOptions{
		BaseRouter: apiRouter,
	})

	rootMux.Mount("/", apiRouter)
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
