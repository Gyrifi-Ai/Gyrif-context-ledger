package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/config"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/engine"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/inference"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/cli"
	httpinterface "github.com/gyrifi/gyrif-context-ledger/runtime/internal/interfaces/http"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/repository"
	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/targets/qdrant"
)

const Version = "0.1.0"

func Run(ctx context.Context, args []string) error {
	settings, err := config.Load()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(settings.DataDirectory, 0o750); err != nil {
		return fmt.Errorf("initialize data directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(settings.SQLitePath), 0o750); err != nil {
		return err
	}
	level := slog.LevelInfo
	if settings.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	repo, err := repository.OpenSQLite(ctx, settings.SQLitePath, settings.ObjectsPath)
	if err != nil {
		return err
	}
	applicationClosed := false
	defer func() {
		if !applicationClosed {
			_ = repo.Close()
		}
	}()
	target, err := qdrant.New(settings.QdrantURL, settings.QdrantCollection, settings.QdrantAPIKey)
	if err != nil {
		return fmt.Errorf("configure Qdrant adapter: %w", err)
	}
	var provider inference.Provider
	var llama *inference.LlamaServer
	if settings.EvaluationProvider == "llamacpp" {
		if _, err := os.Stat(settings.ModelPath); err != nil {
			return fmt.Errorf("validate GGUF model: %w", err)
		}
		llama, err = inference.StartLlamaServer(ctx, settings.LlamaServerPath, settings.ModelPath, settings.LlamaServerPort)
		if err != nil {
			return err
		}
		provider = llama.Provider
		defer func() { _ = llama.Stop() }()
	}
	application := engine.New(repo, target, provider)
	defer func() {
		if !applicationClosed {
			_ = application.Close()
			applicationClosed = true
		}
	}()
	handled, err := cli.Run(ctx, args, application, os.Stdout)
	if handled {
		return err
	}
	if err := application.RecoverReleases(ctx); err != nil {
		logger.Error("release recovery needs attention", "error", err)
	}
	api := httpinterface.New(application, logger, Version)
	server := &http.Server{Addr: settings.HTTPAddress, Handler: api.Handler(), BaseContext: func(net.Listener) context.Context { return ctx }, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 2 * time.Minute, IdleTimeout: 2 * time.Minute}
	errChannel := make(chan error, 1)
	go func() {
		logger.Info("Gyrifi started", "address", settings.HTTPAddress, "data_directory", settings.DataDirectory, "inference", application.InferenceName())
		errChannel <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	case err := <-errChannel:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
