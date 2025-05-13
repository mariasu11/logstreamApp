package main

import (
        "encoding/json"
        "fmt"
        "os"
        "time"

        "github.com/spf13/cobra"
        "github.com/spf13/viper"

        "github.com/mariasu11/logstreamApp/internal/config"
        "github.com/mariasu11/logstreamApp/internal/query"
        "github.com/mariasu11/logstreamApp/internal/storage"
        "github.com/mariasu11/logstreamApp/pkg/models"
)

var (
        queryCmd = &cobra.Command{
                Use:   "query",
                Short: "Query collected logs",
                Long:  `Execute queries against collected logs using various filters and parameters.`,
                Run:   runQuery,
        }
)

func init() {
        rootCmd.AddCommand(queryCmd)

        // Query command flags
        queryCmd.Flags().StringP("filter", "f", "", "Filter expression (e.g., 'level=error')")
        queryCmd.Flags().StringP("time-from", "F", "", "Start time (RFC3339 format)")
        queryCmd.Flags().StringP("time-to", "T", "", "End time (RFC3339 format)")
        queryCmd.Flags().StringP("sources", "s", "", "Comma-separated list of sources to query")
        queryCmd.Flags().IntP("limit", "l", 100, "Maximum number of results")
        queryCmd.Flags().StringP("storage", "d", "memory", "Storage backend (memory, disk)")
        queryCmd.Flags().StringP("storage-path", "p", "./logs", "Path for disk storage")
        queryCmd.Flags().StringP("output", "o", "json", "Output format (json, text)")
        
        // Bind flags to viper
        viper.BindPFlag("query.filter", queryCmd.Flags().Lookup("filter"))
        viper.BindPFlag("query.time-from", queryCmd.Flags().Lookup("time-from"))
        viper.BindPFlag("query.time-to", queryCmd.Flags().Lookup("time-to"))
        viper.BindPFlag("query.sources", queryCmd.Flags().Lookup("sources"))
        viper.BindPFlag("query.limit", queryCmd.Flags().Lookup("limit"))
        viper.BindPFlag("query.storage", queryCmd.Flags().Lookup("storage"))
        viper.BindPFlag("query.storage-path", queryCmd.Flags().Lookup("storage-path"))
        viper.BindPFlag("query.output", queryCmd.Flags().Lookup("output"))
}

func runQuery(cmd *cobra.Command, args []string) {
        // Load configuration
        cfg, err := config.Load()
        if err != nil {
                logger.Error("Failed to load configuration", "error", err)
                os.Exit(1)
        }

        // Initialize storage
        var store storage.Storage
        switch cfg.Query.Storage {
        case "memory":
                store = storage.NewMemoryStorage()
        case "disk":
                store, err = storage.NewDiskStorage(cfg.Query.StoragePath)
                if err != nil {
                        logger.Error("Failed to initialize disk storage", "error", err)
                        os.Exit(1)
                }
        default:
                logger.Error("Unknown storage type", "type", cfg.Query.Storage)
                os.Exit(1)
        }
        defer store.Close()

        // Parse time range
        var fromTime, toTime time.Time
        if cfg.Query.TimeFrom != "" {
                fromTime, err = time.Parse(time.RFC3339, cfg.Query.TimeFrom)
                if err != nil {
                        logger.Error("Invalid from time format", "error", err)
                        os.Exit(1)
                }
        }

        if cfg.Query.TimeTo != "" {
                toTime, err = time.Parse(time.RFC3339, cfg.Query.TimeTo)
                if err != nil {
                        logger.Error("Invalid to time format", "error", err)
                        os.Exit(1)
                }
        } else {
                toTime = time.Now()
        }

        // Build query
        qry := models.Query{
                Filter:    cfg.Query.Filter,
                TimeRange: models.TimeRange{From: fromTime, To: toTime},
                Sources:   cfg.Query.Sources,
                Limit:     cfg.Query.Limit,
        }

        // Execute query
        engine := query.NewEngine(store)
        results, err := engine.Execute(qry)
        if err != nil {
                logger.Error("Query execution failed", "error", err)
                os.Exit(1)
        }

        // Output results
        switch cfg.Query.Output {
        case "json":
                outputJSON(results)
        case "text":
                outputText(results)
        default:
                logger.Error("Unknown output format", "format", cfg.Query.Output)
                os.Exit(1)
        }
}

func outputJSON(results []*models.LogEntry) {
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if err := encoder.Encode(results); err != nil {
                logger.Error("Failed to encode results as JSON", "error", err)
                os.Exit(1)
        }
}

func outputText(results []*models.LogEntry) {
        if len(results) == 0 {
                fmt.Println("No results found")
                return
        }

        fmt.Printf("Found %d results:\n\n", len(results))
        for i, entry := range results {
                fmt.Printf("[%d] Timestamp: %s\n", i+1, entry.Timestamp.Format(time.RFC3339))
                fmt.Printf("    Source: %s\n", entry.Source)
                fmt.Printf("    Level: %s\n", entry.Level)
                fmt.Printf("    Message: %s\n", entry.Message)
                if len(entry.Fields) > 0 {
                        fmt.Println("    Fields:")
                        for k, v := range entry.Fields {
                                fmt.Printf("      %s: %v\n", k, v)
                        }
                }
                fmt.Println()
        }
}
