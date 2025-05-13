package api

import (
        "context"
        "fmt"
        "net/http"
        "time"

        "github.com/go-chi/chi/v5"
        "github.com/go-chi/chi/v5/middleware"
        "github.com/hashicorp/go-hclog"
        "github.com/prometheus/client_golang/prometheus/promhttp"

        "github.com/mariasu11/logstreamApp/internal/storage"
)

// Server represents the LogStream API server
type Server struct {
        host       string
        port       int
        Router     *chi.Mux  // Exported for testing
        logger     hclog.Logger
        storage    storage.Storage
        httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(host string, port int, storage storage.Storage, logger hclog.Logger) *Server {
        r := chi.NewRouter()

        // Create the server
        server := &Server{
                host:    host,
                port:    port,
                Router:  r,
                logger:  logger,
                storage: storage,
        }

        // Set up middleware
        server.setupMiddleware()
        
        // Set up routes
        server.setupRoutes()

        return server
}

// setupMiddleware configures the middleware stack
func (s *Server) setupMiddleware() {
        // Standard middleware
        s.Router.Use(middleware.RequestID)
        s.Router.Use(middleware.RealIP)
        s.Router.Use(LoggerMiddleware(s.logger))
        s.Router.Use(middleware.Recoverer)
        s.Router.Use(middleware.Timeout(60 * time.Second))
        
        // CORS middleware
        s.Router.Use(middleware.AllowContentType("application/json"))
        s.Router.Use(middleware.SetHeader("Content-Type", "application/json"))
        
        // Custom middleware
        s.Router.Use(MetricsMiddleware)
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
        // Create API handlers
        handlers := NewHandlers(s.storage, s.logger)
        
        // Create Web UI handler
        webHandler := NewWebHandler(s.logger)
        
        // API v1 routes
        s.Router.Route("/api/v1", func(r chi.Router) {
                // Log routes
                r.Route("/logs", func(r chi.Router) {
                        r.Get("/", handlers.GetLogs)
                        r.Post("/", handlers.StoreLog)
                        r.Post("/batch", handlers.StoreLogs)
                        r.Get("/sources", handlers.GetSources)
                        r.Get("/stats", handlers.GetStats)
                        r.Get("/export", handlers.GetLogs) // Alias for logs export
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
        s.Router.Handle("/metrics", promhttp.Handler())
        
        // API documentation
        s.Router.Get("/api", handlers.GetDocs)
        
        // Web UI routes
        webHandler.RegisterRoutes(s.Router)
}

// Start begins the HTTP server
func (s *Server) Start() error {
        addr := fmt.Sprintf("%s:%d", s.host, s.port)
        s.logger.Info("Starting LogStream API server", "address", addr)
        
        s.httpServer = &http.Server{
                Addr:    addr,
                Handler: s.Router,
        }
        
        return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the server
func (s *Server) Stop(ctx context.Context) error {
        if s.httpServer != nil {
                s.logger.Info("Shutting down API server")
                return s.httpServer.Shutdown(ctx)
        }
        return nil
}
