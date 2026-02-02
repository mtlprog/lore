package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/database"
	"github.com/mtlprog/lore/internal/handler"
	"github.com/mtlprog/lore/internal/logger"
	"github.com/mtlprog/lore/internal/repository"
	"github.com/mtlprog/lore/internal/reputation"
	"github.com/mtlprog/lore/internal/service"
	"github.com/mtlprog/lore/internal/static"
	"github.com/mtlprog/lore/internal/sync"
	"github.com/mtlprog/lore/internal/template"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "lore",
		Usage: "Stellar Token Explorer for MTLAP & MTLAC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "horizon-url",
				Aliases: []string{"H"},
				Value:   config.DefaultHorizonURL,
				Usage:   "Stellar Horizon API URL",
				EnvVars: []string{"HORIZON_URL"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Aliases: []string{"l"},
				Value:   "info",
				Usage:   "Log level (debug, info, warn, error)",
				EnvVars: []string{"LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:     "database-url",
				Aliases:  []string{"d"},
				Value:    config.DefaultDatabaseURL,
				Usage:    "PostgreSQL database URL",
				EnvVars:  []string{"DATABASE_URL"},
				Required: true,
			},
		},
		Before: func(c *cli.Context) error {
			logger.Setup(logger.ParseLevel(c.String("log-level")))
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the web server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "port",
						Aliases: []string{"p"},
						Value:   config.DefaultPort,
						Usage:   "HTTP server port",
						EnvVars: []string{"PORT"},
					},
				},
				Action: runServe,
			},
			{
				Name:  "sync",
				Usage: "Sync data from Stellar Horizon to PostgreSQL",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "full",
						Usage: "Full resync (truncate tables before sync)",
					},
				},
				Action: runSync,
			},
		},
		Action: runServe,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func runServe(c *cli.Context) error {
	ctx := c.Context

	port := c.String("port")
	if port == "" {
		port = config.DefaultPort
	}
	horizonURL := c.String("horizon-url")
	databaseURL := c.String("database-url")

	db, err := database.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db.Pool()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	tmpl, err := template.New()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	stellar := service.NewStellarService(horizonURL)
	accounts, err := repository.NewAccountRepository(db.Pool())
	if err != nil {
		return fmt.Errorf("failed to create account repository: %w", err)
	}

	// Create reputation service (optional, non-critical feature)
	repService, err := reputation.NewService(db.Pool())
	if err != nil {
		slog.Warn("failed to create reputation service, feature will be disabled", "error", err)
	}

	h, err := handler.New(stellar, accounts, repService, tmpl)
	if err != nil {
		return fmt.Errorf("failed to create handler: %w", err)
	}

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	handler.RegisterStaticRoutes(mux, static.Handler())

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("starting server", "server_addr", "http://localhost:"+port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-done:
		slog.Info("shutting down server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	slog.Info("server stopped")
	return nil
}

func runSync(c *cli.Context) error {
	ctx := c.Context

	horizonURL := c.String("horizon-url")
	databaseURL := c.String("database-url")
	full := c.Bool("full")

	db, err := database.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db.Pool()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	syncer, err := sync.New(db.Pool(), horizonURL)
	if err != nil {
		return fmt.Errorf("failed to create syncer: %w", err)
	}
	result, err := syncer.Run(ctx, full)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	if result != nil && (len(result.FailedAccounts) > 0 || len(result.FailedPrices) > 0) {
		slog.Warn("sync completed with failures",
			"failed_accounts", len(result.FailedAccounts),
			"failed_prices", len(result.FailedPrices),
		)
	}

	return nil
}
