package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mtlprog/lore/internal/config"
	"github.com/mtlprog/lore/internal/handler"
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
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
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

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
