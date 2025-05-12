package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yourusername/logstream/internal/api"
	"github.com/yourusername/logstream/internal/config"
	"github.com/yourusername/logstream/internal/storage"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the LogStream API server",
		Long:  `Start the LogStream API server to provide query and analysis capabilities through HTTP endpoints.`,
		Run:   runServe,
	}
)

func init() {
	rootCmd.AddCommand(serveCmd)

	// Serve command flags
	serveCmd.Flags().StringP("host", "H", "0.0.0.0", "Host to bind the server to")
	serveCmd.Flags().IntP("port", "P", 8000, "Port to listen on")
	serveCmd.Flags().StringP("storage", "s", "memory", "Storage backend (memory, disk)")
	serveCmd.Flags().StringP("storage-path", "p", "./logs", "Path for disk storage")
	
	// Bind flags to viper
	viper.BindPFlag("api.host", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("api.port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("api.storage", serveCmd.Flags().Lookup("storage"))
	viper.BindPFlag("api.storage-path", serveCmd.Flags().Lookup("storage-path"))
}

func runServe(cmd *cobra.Command, args []string) {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize storage
	var store storage.Storage
	switch cfg.API.Storage {
	case "memory":
		store = storage.NewMemoryStorage()
	case "disk":
		store, err = storage.NewDiskStorage(cfg.API.StoragePath)
		if err != nil {
			logger.Error("Failed to initialize disk storage", "error", err)
			os.Exit(1)
		}
	default:
		logger.Error("Unknown storage type", "type", cfg.API.Storage)
		os.Exit(1)
	}

	// Create and configure the API server
	server := api.NewServer(cfg.API.Host, cfg.API.Port, store, logger)
	
	// Start the server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("Server failed", "error", err)
			cancel()
		}
	}()

	logger.Info("API server started", "host", cfg.API.Host, "port", cfg.API.Port)

	// Wait for context cancellation (from signal handler)
	<-ctx.Done()

	// Graceful shutdown
	logger.Info("Shutting down API server...")
	
	// Allow up to 10 seconds for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	if err := server.Stop(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
	}

	// Close storage
	if err := store.Close(); err != nil {
		logger.Error("Error closing storage", "error", err)
	}

	logger.Info("LogStream API server shutdown complete")
}
