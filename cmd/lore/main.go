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
	"github.com/mtlprog/lore/internal/handler"
	"github.com/mtlprog/lore/internal/logger"
	"github.com/mtlprog/lore/internal/service"
	"github.com/mtlprog/lore/internal/template"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "lore",
		Usage: "Stellar Token Explorer for MTLAP & MTLAC",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   config.DefaultPort,
				Usage:   "HTTP server port",
				EnvVars: []string{"PORT"},
			},
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
		},
		Before: func(c *cli.Context) error {
			logger.Setup(logger.ParseLevel(c.String("log-level")))
			return nil
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	port := c.String("port")
	horizonURL := c.String("horizon-url")

	tmpl, err := template.New()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	stellar := service.NewStellarService(horizonURL)
	h := handler.New(stellar, tmpl)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

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
