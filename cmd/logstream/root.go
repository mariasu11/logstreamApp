package main

import (
        "fmt"
        "os"

        "github.com/spf13/cobra"
        "github.com/spf13/viper"
)

var (
        cfgFile string
        rootCmd = &cobra.Command{
                Use:   "logstream",
                Short: "LogStream - A high-performance log aggregation and analysis tool",
                Long: `LogStream is a distributed log aggregation and analysis system built in Go.
It collects logs from multiple sources, processes them concurrently, 
and provides powerful query capabilities through a REST API and CLI interface.

Complete documentation is available at https://github.com/mariasu11/logstream`,
                Run: func(cmd *cobra.Command, args []string) {
                        // If no subcommand is provided, print help
                        cmd.Help()
                },
        }
)

// Execute adds all child commands to the root command and sets flags appropriately.
func init() {
        cobra.OnInitialize(initConfig)

        // Global flags
        rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.logstream.yaml)")
        rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
        
        // Bind flags to viper
        viper.BindPFlag("log.level", rootCmd.PersistentFlags().Lookup("log-level"))
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
        if cfgFile != "" {
                // Use config file from the flag
                viper.SetConfigFile(cfgFile)
        } else {
                // Find home directory
                home, err := os.UserHomeDir()
                if err != nil {
                        fmt.Println(err)
                        os.Exit(1)
                }

                // Search config in home directory with name ".logstream" (without extension)
                viper.AddConfigPath(home)
                viper.AddConfigPath(".")
                viper.SetConfigName(".logstream")
        }

        // Read environment variables prefixed with LOGSTREAM_
        viper.SetEnvPrefix("LOGSTREAM")
        viper.AutomaticEnv() 

        // If a config file is found, read it in
        if err := viper.ReadInConfig(); err == nil {
                fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
        }
}
