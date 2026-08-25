package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlesixDev/MCAIA/backend/internal/ai/ollama"
	"github.com/AlesixDev/MCAIA/backend/internal/auth"
	"github.com/AlesixDev/MCAIA/backend/internal/config"
	"github.com/AlesixDev/MCAIA/backend/internal/database"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter/bedrock"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter/blockbench"
	"github.com/AlesixDev/MCAIA/backend/internal/exporter/geckolib"
	"github.com/AlesixDev/MCAIA/backend/internal/httpapi"
	"github.com/AlesixDev/MCAIA/backend/internal/importer"
	"github.com/AlesixDev/MCAIA/backend/internal/pipeline"
	"github.com/AlesixDev/MCAIA/backend/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	exporter.Register(blockbench.New())
	exporter.Register(bedrock.New())
	exporter.Register(geckolib.New())

	importer.Register(importer.NewBlockbench())
	importer.Register(importer.NewGLTF())
	importer.Register(importer.NewOBJ())

	engine := ollama.New(ollama.Options{
		BaseURL:     cfg.OllamaBaseURL,
		Model:       cfg.OllamaModel,
		Temperature: cfg.Temperature,
		NumCtx:      cfg.NumCtx,
		Think:       cfg.Think,
		Timeout:     cfg.RequestTimeout,
	})

	db, err := database.Open(cfg.DataDir)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}

	defer db.Close()

	accounts, err := auth.NewService(db, cfg.DataDir)
	if err != nil {
		slog.Error("accounts", "error", err)
		os.Exit(1)
	}

	projects := store.New(db)
	flow := pipeline.New(projects, engine)

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           httpapi.NewServer(cfg, projects, flow, engine, accounts).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      cfg.RequestTimeout + 30*time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("listening", "addr", cfg.Address, "model", cfg.OllamaModel)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server", "error", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
	}

	slog.Info("stopped")
}
