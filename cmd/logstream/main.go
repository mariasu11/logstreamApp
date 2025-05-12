package main

import (
        "fmt"
        "os"

        "github.com/hashicorp/go-hclog"
)

// Global logger instance
var logger hclog.Logger

func main() {
        // Initialize the logger
        logger = hclog.New(&hclog.LoggerOptions{
                Name:       "logstream",
                Level:      hclog.LevelFromString(os.Getenv("LOG_LEVEL")),
                JSONFormat: true,
        })

        // Execute the root command
        if err := rootCmd.Execute(); err != nil {
                logger.Error("command execution failed", "error", err)
                fmt.Fprintln(os.Stderr, err)
                os.Exit(1)
        }
}
