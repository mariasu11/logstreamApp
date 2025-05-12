package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/yourusername/logstream/internal/metrics"
)

// LoggerMiddleware creates a custom middleware that logs requests using go-hclog
func LoggerMiddleware(logger hclog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Create a wrapped response writer to capture status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			
			// Process the request
			next.ServeHTTP(ww, r)
			
			// Log after request is processed
			logger.Info("HTTP Request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", ww.Status(),
				"size", ww.BytesWritten(),
				"duration", time.Since(start).String(),
				"remote_addr", r.RemoteAddr,
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}

// MetricsMiddleware records Prometheus metrics for API requests
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a wrapped response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		
		// Process the request
		next.ServeHTTP(ww, r)
		
		// Record metrics
		duration := time.Since(start).Seconds()
		
		// Get metric collectors
		m := metrics.GetMetrics()
		
		// Increment request counter
		m.APIRequestsTotal.With(prometheus.Labels{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": http.StatusText(ww.Status()),
		}).Inc()
		
		// Record request duration
		m.APIRequestDuration.With(prometheus.Labels{
			"method": r.Method,
			"path":   r.URL.Path,
		}).Observe(duration)
	})
}

// RateLimitMiddleware implements basic rate limiting
func RateLimitMiddleware(rps int) func(next http.Handler) http.Handler {
	// Token bucket implementation would go here
	// For simplicity, using a very basic implementation
	var tokens = rps
	ticker := time.NewTicker(time.Second)
	
	go func() {
		for range ticker.C {
			tokens = rps
		}
	}()
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we have tokens
			if tokens <= 0 {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			// Consume a token
			tokens--
			
			// Process the request
			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware implements basic authentication
// In a real application, you'd use a more sophisticated auth system
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get API key from header
		apiKey := r.Header.Get("X-API-Key")
		
		// Check if API key is valid
		// This is a placeholder - you would validate against a real auth system
		if apiKey == "" {
			http.Error(w, "Unauthorized - API key required", http.StatusUnauthorized)
			return
		}
		
		// Process the request
		next.ServeHTTP(w, r)
	})
}

// TraceMiddleware adds distributed tracing capabilities
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract trace ID from headers or generate a new one
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = middleware.GetReqID(r.Context())
		}
		
		// Add trace ID to response headers
		w.Header().Set("X-Trace-ID", traceID)
		
		// Process the request
		next.ServeHTTP(w, r)
	})
}
