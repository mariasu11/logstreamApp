package main

import (
        "context"
        "os"
        "os/signal"
        "syscall"
        "time"

        "github.com/spf13/cobra"
        "github.com/spf13/viper"
        "golang.org/x/sync/errgroup"

        "github.com/mariasu11/logstream/internal/collector"
        "github.com/mariasu11/logstream/internal/config"
        "github.com/mariasu11/logstream/internal/processor"
        "github.com/mariasu11/logstream/internal/storage"
        "github.com/mariasu11/logstream/pkg/worker"
)

var (
        collectCmd = &cobra.Command{
                Use:   "collect",
                Short: "Start log collection from configured sources",
                Long:  `Start collecting logs from all configured sources, process them according to rules, and store them for analysis.`,
                Run:   runCollect,
        }
)

func init() {
        rootCmd.AddCommand(collectCmd)

        // Collect command flags
        collectCmd.Flags().StringP("sources", "s", "", "Comma-separated list of log sources to collect from")
        collectCmd.Flags().IntP("workers", "w", 4, "Number of worker goroutines for processing")
        collectCmd.Flags().StringP("storage", "d", "memory", "Storage backend (memory, disk)")
        collectCmd.Flags().StringP("storage-path", "p", "./logs", "Path for disk storage")
        
        // Bind flags to viper
        viper.BindPFlag("collect.sources", collectCmd.Flags().Lookup("sources"))
        viper.BindPFlag("collect.workers", collectCmd.Flags().Lookup("workers"))
        viper.BindPFlag("collect.storage", collectCmd.Flags().Lookup("storage"))
        viper.BindPFlag("collect.storage-path", collectCmd.Flags().Lookup("storage-path"))
}

func runCollect(cmd *cobra.Command, args []string) {
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
        switch cfg.Collect.Storage {
        case "memory":
                store = storage.NewMemoryStorage()
        case "disk":
                store, err = storage.NewDiskStorage(cfg.Collect.StoragePath)
                if err != nil {
                        logger.Error("Failed to initialize disk storage", "error", err)
                        os.Exit(1)
                }
        default:
                logger.Error("Unknown storage type", "type", cfg.Collect.Storage)
                os.Exit(1)
        }

        // Create worker pool
        wp := worker.NewPool(cfg.Collect.Workers)

        // Create processor
        proc := processor.NewProcessor(store, wp)

        // Set up collectors based on configuration
        collectors := []collector.Collector{}
        for _, src := range cfg.Collect.Sources {
                coll, err := collector.NewCollector(src, proc)
                if err != nil {
                        logger.Error("Failed to initialize collector", "source", src, "error", err)
                        continue
                }
                collectors = append(collectors, coll)
        }

        if len(collectors) == 0 {
                logger.Error("No valid collectors configured")
                os.Exit(1)
        }

        // Start the collectors
        g, ctx := errgroup.WithContext(ctx)
        for _, c := range collectors {
                c := c // capture variable for goroutine
                g.Go(func() error {
                        return c.Start(ctx)
                })
        }

        // Start the worker pool
        wp.Start(ctx)

        // Wait for either error from collectors or context cancellation
        if err := g.Wait(); err != nil && err != context.Canceled {
                logger.Error("Collector failed", "error", err)
        }

        // Graceful shutdown
        logger.Info("Shutting down...")
        
        // Allow up to 5 seconds for graceful shutdown
        shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer shutdownCancel()
        
        // Stop the worker pool
        wp.Stop(shutdownCtx)
        
        // Flush storage
        if err := store.Close(); err != nil {
                logger.Error("Error closing storage", "error", err)
        }

        logger.Info("LogStream collector shutdown complete")
}
