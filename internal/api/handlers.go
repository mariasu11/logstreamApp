package api

import (
        "encoding/json"
        "net/http"
        "strconv"
        "time"

        "github.com/hashicorp/go-hclog"

        "github.com/yourusername/logstream/internal/query"
        "github.com/yourusername/logstream/internal/storage"
        "github.com/yourusername/logstream/pkg/models"
)

// Handlers contains the HTTP handlers for the API
type Handlers struct {
        storage  storage.Storage
        logger   hclog.Logger
        queryEngine *query.Engine
}

// NewHandlers creates a new set of API handlers
func NewHandlers(storage storage.Storage, logger hclog.Logger) *Handlers {
        return &Handlers{
                storage:     storage,
                logger:      logger,
                queryEngine: query.NewEngine(storage),
        }
}

// GetLogs returns log entries
func (h *Handlers) GetLogs(w http.ResponseWriter, r *http.Request) {
        // Parse query parameters
        limit := 100
        if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
                if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
                        limit = parsedLimit
                }
        }

        from := time.Time{}
        if fromStr := r.URL.Query().Get("from"); fromStr != "" {
                if parsedTime, err := time.Parse(time.RFC3339, fromStr); err == nil {
                        from = parsedTime
                }
        }

        to := time.Time{}
        if toStr := r.URL.Query().Get("to"); toStr != "" {
                if parsedTime, err := time.Parse(time.RFC3339, toStr); err == nil {
                        to = parsedTime
                }
        }

        source := r.URL.Query().Get("source")
        level := r.URL.Query().Get("level")
        filter := r.URL.Query().Get("filter")

        // Build query
        qb := storage.NewQueryBuilder().
                WithLimit(limit).
                WithTimeRange(from, to)

        if source != "" {
                qb.WithSources(source)
        }

        if level != "" {
                qb.WithLevels(level)
        }

        if filter != "" {
                qb.WithFilter(filter)
        }

        query := qb.Build()

        // Execute query
        logs, err := h.storage.Query(r.Context(), query)
        if err != nil {
                h.respondWithError(w, http.StatusInternalServerError, "Query failed: "+err.Error())
                return
        }

        // Respond with results
        h.respondWithJSON(w, http.StatusOK, logs)
}

// GetSources returns a list of all log sources
func (h *Handlers) GetSources(w http.ResponseWriter, r *http.Request) {
        sources, err := h.storage.GetSources(r.Context())
        if err != nil {
                h.respondWithError(w, http.StatusInternalServerError, "Failed to get sources: "+err.Error())
                return
        }

        h.respondWithJSON(w, http.StatusOK, sources)
}

// GetStats returns storage statistics
func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
        stats, err := h.storage.GetStats(r.Context())
        if err != nil {
                h.respondWithError(w, http.StatusInternalServerError, "Failed to get stats: "+err.Error())
                return
        }

        h.respondWithJSON(w, http.StatusOK, stats)
}

// ExecuteQuery executes a custom query
func (h *Handlers) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
        // Parse request body
        var queryRequest struct {
                Query models.Query `json:"query"`
        }

        if err := json.NewDecoder(r.Body).Decode(&queryRequest); err != nil {
                h.respondWithError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
                return
        }

        // Execute query
        results, err := h.queryEngine.Execute(queryRequest.Query)
        if err != nil {
                h.respondWithError(w, http.StatusInternalServerError, "Query failed: "+err.Error())
                return
        }

        // Respond with results
        h.respondWithJSON(w, http.StatusOK, results)
}

// AnalyzeLogs performs analysis on log data
func (h *Handlers) AnalyzeLogs(w http.ResponseWriter, r *http.Request) {
        // Parse request body
        var analysisRequest struct {
                Analysis models.Analysis `json:"analysis"`
        }

        if err := json.NewDecoder(r.Body).Decode(&analysisRequest); err != nil {
                h.respondWithError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
                return
        }

        // Perform analysis
        result, err := h.queryEngine.Analyze(analysisRequest.Analysis)
        if err != nil {
                h.respondWithError(w, http.StatusInternalServerError, "Analysis failed: "+err.Error())
                return
        }

        // Respond with analysis results
        h.respondWithJSON(w, http.StatusOK, result)
}

// HealthCheck returns the health status of the API
func (h *Handlers) HealthCheck(w http.ResponseWriter, r *http.Request) {
        // Check storage connectivity
        _, err := h.storage.GetSources(r.Context())
        if err != nil {
                h.respondWithJSON(w, http.StatusServiceUnavailable, map[string]string{
                        "status":  "unavailable",
                        "message": "Storage unavailable: " + err.Error(),
                })
                return
        }

        h.respondWithJSON(w, http.StatusOK, map[string]string{
                "status":  "ok",
                "version": "1.0.0",
        })
}

// GetDocs returns API documentation
func (h *Handlers) GetDocs(w http.ResponseWriter, r *http.Request) {
        docs := map[string]interface{}{
                "name":        "LogStream API",
                "version":     "1.0.0",
                "description": "A high-performance log aggregation and analysis API",
                "endpoints": []map[string]string{
                        {"path": "/api/v1/logs", "method": "GET", "description": "Get log entries"},
                        {"path": "/api/v1/logs/sources", "method": "GET", "description": "Get log sources"},
                        {"path": "/api/v1/logs/stats", "method": "GET", "description": "Get storage statistics"},
                        {"path": "/api/v1/query", "method": "POST", "description": "Execute a custom query"},
                        {"path": "/api/v1/query/analyze", "method": "POST", "description": "Perform log analysis"},
                        {"path": "/api/v1/health", "method": "GET", "description": "Check API health"},
                        {"path": "/metrics", "method": "GET", "description": "Prometheus metrics"},
                },
        }

        h.respondWithJSON(w, http.StatusOK, docs)
}

// respondWithError sends an error response
func (h *Handlers) respondWithError(w http.ResponseWriter, code int, message string) {
        h.respondWithJSON(w, code, map[string]string{"error": message})
}

// respondWithJSON sends a JSON response
func (h *Handlers) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
        // Set headers
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(code)

        // Encode and send response
        if err := json.NewEncoder(w).Encode(payload); err != nil {
                h.logger.Error("Failed to encode response", "error", err)
        }
}
