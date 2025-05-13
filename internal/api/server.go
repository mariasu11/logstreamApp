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

        "github.com/yourusername/logstream/internal/storage"
)

// Server represents the LogStream API server
type Server struct {
        host       string
        port       int
        router     *chi.Mux
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
                router:  r,
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
        s.router.Use(middleware.RequestID)
        s.router.Use(middleware.RealIP)
        s.router.Use(LoggerMiddleware(s.logger))
        s.router.Use(middleware.Recoverer)
        s.router.Use(middleware.Timeout(60 * time.Second))
        
        // CORS middleware
        s.router.Use(middleware.AllowContentType("application/json"))
        s.router.Use(middleware.SetHeader("Content-Type", "application/json"))
        
        // Custom middleware
        s.router.Use(MetricsMiddleware)
}

// setupRoutes configures the API routes
func (s *Server) setupRoutes() {
        // Create handlers
        handlers := NewHandlers(s.storage, s.logger)
        
        // API v1 routes
        s.router.Route("/api/v1", func(r chi.Router) {
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
        s.router.Handle("/metrics", promhttp.Handler())
        
        // Documentation
        s.router.Get("/", handlers.GetDocs)
}

// Start begins the HTTP server
func (s *Server) Start() error {
        addr := fmt.Sprintf("%s:%d", s.host, s.port)
        s.logger.Info("Starting LogStream API server", "address", addr)
        
        s.httpServer = &http.Server{
                Addr:    addr,
                Handler: s.router,
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
