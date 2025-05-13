package api

import (
        "encoding/json"
        "net/http"
        "strconv"
        "strings"
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

// StoreLog stores a log entry
func (h *Handlers) StoreLog(w http.ResponseWriter, r *http.Request) {
        // Parse log entry from request body
        var entry models.LogEntry
        if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
                h.respondWithError(w, http.StatusBadRequest, "Invalid log entry: "+err.Error())
                return
        }

        // Set timestamp if not provided
        if entry.Timestamp.IsZero() {
                entry.Timestamp = time.Now()
        }

        // Store the log entry
        if err := h.storage.Store(r.Context(), &entry); err != nil {
                h.respondWithError(w, http.StatusInternalServerError, "Failed to store log entry: "+err.Error())
                return
        }

        h.respondWithJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// StoreLogs stores multiple log entries
func (h *Handlers) StoreLogs(w http.ResponseWriter, r *http.Request) {
        // Parse log entries from request body
        var entries []*models.LogEntry
        if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
                h.respondWithError(w, http.StatusBadRequest, "Invalid log entries: "+err.Error())
                return
        }

        // Set timestamp for any entries without one
        for _, entry := range entries {
                if entry.Timestamp.IsZero() {
                        entry.Timestamp = time.Now()
                }
        }

        // Store all log entries
        for _, entry := range entries {
                if err := h.storage.Store(r.Context(), entry); err != nil {
                        h.respondWithError(w, http.StatusInternalServerError, "Failed to store log entries: "+err.Error())
                        return
                }
        }

        h.respondWithJSON(w, http.StatusCreated, map[string]string{"status": "ok", "count": strconv.Itoa(len(entries))})
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

        // Handle sources as comma-separated list
        sources := []string{}
        if sourcesList := r.URL.Query().Get("sources"); sourcesList != "" {
                sources = strings.Split(sourcesList, ",")
        } else if source := r.URL.Query().Get("source"); source != "" {
                sources = []string{source}
        }

        // Handle levels as comma-separated list
        levels := []string{}
        if levelsList := r.URL.Query().Get("levels"); levelsList != "" {
                levels = strings.Split(levelsList, ",")
        } else if level := r.URL.Query().Get("level"); level != "" {
                levels = []string{level}
        }

        filter := r.URL.Query().Get("filter")

        // Debug log request parameters
        h.logger.Debug("GetLogs request parameters", 
                "limit", limit, 
                "from", from, 
                "to", to, 
                "sources", sources, 
                "levels", levels, 
                "filter", filter)

        // Build query
        qb := storage.NewQueryBuilder().
                WithLimit(limit).
                WithTimeRange(from, to)

        if len(sources) > 0 {
                qb.WithSources(sources...)
        }

        if len(levels) > 0 {
                qb.WithLevels(levels...)
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

        // Debug response
        h.logger.Debug("GetLogs response", "count", len(logs))

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
                Filter     string    `json:"filter"`
                Limit      int       `json:"limit"`
                From       string    `json:"from,omitempty"`
                To         string    `json:"to,omitempty"`
                Sources    []string  `json:"sources,omitempty"`
                Levels     []string  `json:"levels,omitempty"`
                SortBy     string    `json:"sort_by,omitempty"`
                SortOrder  string    `json:"sort_order,omitempty"`
        }

        if err := json.NewDecoder(r.Body).Decode(&queryRequest); err != nil {
                h.respondWithError(w, http.StatusBadRequest, "Invalid request: "+err.Error())
                return
        }

        // Build query
        query := models.NewQuery()
        
        // Parse source:value filter format
        if queryRequest.Filter != "" {
                if strings.HasPrefix(queryRequest.Filter, "source:") {
                        source := strings.TrimPrefix(queryRequest.Filter, "source:")
                        query = query.WithSources(source)
                } else if strings.HasPrefix(queryRequest.Filter, "level:") {
                        level := strings.TrimPrefix(queryRequest.Filter, "level:")
                        query = query.WithLevels(level)
                } else {
                        // Use as general text filter
                        query = query.WithFilter(queryRequest.Filter)
                }
        }
        
        // Apply sources filter if provided
        if len(queryRequest.Sources) > 0 {
                query = query.WithSources(queryRequest.Sources...)
        }

        // Apply levels filter if provided
        if len(queryRequest.Levels) > 0 {
                query = query.WithLevels(queryRequest.Levels...)
        }

        // Parse time range if provided
        var from, to time.Time
        var err error
        
        if queryRequest.From != "" {
                from, err = time.Parse(time.RFC3339, queryRequest.From)
                if err != nil {
                        h.respondWithError(w, http.StatusBadRequest, "Invalid 'from' timestamp: "+err.Error())
                        return
                }
        }
        
        if queryRequest.To != "" {
                to, err = time.Parse(time.RFC3339, queryRequest.To)
                if err != nil {
                        h.respondWithError(w, http.StatusBadRequest, "Invalid 'to' timestamp: "+err.Error())
                        return
                }
        }
        
        if !from.IsZero() || !to.IsZero() {
                query = query.WithTimeRange(from, to)
        }
        
        // Apply sorting if provided
        if queryRequest.SortBy != "" && queryRequest.SortOrder != "" {
                query = query.WithSort(queryRequest.SortBy, queryRequest.SortOrder)
        }
        
        // Apply limit if provided
        if queryRequest.Limit > 0 {
                query = query.WithLimit(queryRequest.Limit)
        }

        // Execute query
        results, err := h.storage.Query(r.Context(), query)
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
                        {"path": "/api/v1/logs", "method": "POST", "description": "Store a single log entry"},
                        {"path": "/api/v1/logs/batch", "method": "POST", "description": "Store multiple log entries"},
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
