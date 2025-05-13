package api

import (
        "github.com/go-chi/chi/v5"
        "github.com/hashicorp/go-hclog"
        "github.com/prometheus/client_golang/prometheus/promhttp"
        
        "github.com/mariasu11/logstream/internal/storage"
)

// SetupRoutes configures the routes for the chi router
func SetupRoutes(router *chi.Mux, storage storage.Storage, logger hclog.Logger) {
        // Create handlers
        handlers := NewHandlers(storage, logger)

        // API v1 routes
        router.Route("/api/v1", func(r chi.Router) {
                // Log routes
                r.Route("/logs", func(r chi.Router) {
                        r.Get("/", handlers.GetLogs)
                        r.Post("/", handlers.StoreLog)
                        r.Post("/batch", handlers.StoreLogs)
                        r.Get("/sources", handlers.GetSources)
                        r.Get("/stats", handlers.GetStats)
                })

                // Query routes
                r.Route("/query", func(r chi.Router) {
                        r.Post("/", handlers.ExecuteQuery)
                        r.Post("/analyze", handlers.AnalyzeLogs)
                })

                // Health routes
                r.Get("/health", handlers.HealthCheck)
        })

        // Prometheus metrics endpoint
        router.Handle("/metrics", promhttp.Handler())

        // Documentation
        router.Get("/", handlers.GetDocs)
}
