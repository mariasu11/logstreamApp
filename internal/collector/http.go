package collector

import (
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net/http"
        "strings"
        "time"

        "github.com/mariasu11/logstreamApp/internal/processor"
        "github.com/mariasu11/logstreamApp/pkg/models"
)

// HTTPCollector collects logs from HTTP endpoints
type HTTPCollector struct {
        BaseCollector
        url          string
        method       string
        headers      map[string]string
        pollInterval time.Duration
        client       *http.Client
}

// NewHTTPCollector creates a new HTTP collector
func NewHTTPCollector(url string, processor processor.Processor) (*HTTPCollector, error) {
        // Basic validation
        if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
                return nil, fmt.Errorf("invalid HTTP URL: %s", url)
        }

        return &HTTPCollector{
                BaseCollector: BaseCollector{
                        name:      fmt.Sprintf("http-%s", url),
                        source:    url,
                        processor: processor,
                },
                url:          url,
                method:       "GET",
                headers:      make(map[string]string),
                pollInterval: 10 * time.Second,
                client: &http.Client{
                        Timeout: 30 * time.Second,
                },
        }, nil
}

// WithMethod sets the HTTP method
func (hc *HTTPCollector) WithMethod(method string) *HTTPCollector {
        hc.method = method
        return hc
}

// WithHeader adds an HTTP header
func (hc *HTTPCollector) WithHeader(key, value string) *HTTPCollector {
        hc.headers[key] = value
        return hc
}

// WithPollInterval sets the polling interval
func (hc *HTTPCollector) WithPollInterval(interval time.Duration) *HTTPCollector {
        hc.pollInterval = interval
        return hc
}

// Start implements the Collector interface
func (hc *HTTPCollector) Start(ctx context.Context) error {
        ticker := time.NewTicker(hc.pollInterval)
        defer ticker.Stop()

        for {
                select {
                case <-ctx.Done():
                        return ctx.Err()
                case <-ticker.C:
                        // Fetch logs from the HTTP endpoint
                        if err := hc.fetch(ctx); err != nil {
                                // Log error but continue - don't fail the collector on transient errors
                                fmt.Printf("Error fetching logs from %s: %v\n", hc.url, err)
                        }
                }
        }
}

// fetch retrieves logs from the HTTP endpoint
func (hc *HTTPCollector) fetch(ctx context.Context) error {
        // Create a new request
        req, err := http.NewRequestWithContext(ctx, hc.method, hc.url, nil)
        if err != nil {
                return fmt.Errorf("failed to create request: %w", err)
        }

        // Add headers
        for key, value := range hc.headers {
                req.Header.Add(key, value)
        }

        // Execute the request
        resp, err := hc.client.Do(req)
        if err != nil {
                return fmt.Errorf("HTTP request failed: %w", err)
        }
        defer resp.Body.Close()

        // Check status code
        if resp.StatusCode < 200 || resp.StatusCode >= 300 {
                return fmt.Errorf("HTTP request returned non-success status: %d", resp.StatusCode)
        }

        // Read the response body
        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return fmt.Errorf("failed to read response body: %w", err)
        }

        // Parse response based on content type
        contentType := resp.Header.Get("Content-Type")
        if strings.Contains(contentType, "application/json") {
                return hc.processJSONResponse(ctx, body)
        } else {
                return hc.processTextResponse(ctx, body)
        }
}

// processJSONResponse handles JSON-formatted log data
func (hc *HTTPCollector) processJSONResponse(ctx context.Context, data []byte) error {
        // Try to parse as array of log entries
        var entries []*models.LogEntry
        err := json.Unmarshal(data, &entries)
        if err == nil && len(entries) > 0 {
                // Set source for each entry
                for _, entry := range entries {
                        if entry.Source == "" {
                                entry.Source = hc.Source()
                        }
                        if entry.Timestamp.IsZero() {
                                entry.Timestamp = time.Now()
                        }
                }
                
                // Process the entries
                return hc.processor.Process(ctx, entries)
        }

        // Try to parse as single log entry
        var entry models.LogEntry
        if err := json.Unmarshal(data, &entry); err == nil {
                if entry.Source == "" {
                        entry.Source = hc.Source()
                }
                if entry.Timestamp.IsZero() {
                        entry.Timestamp = time.Now()
                }
                
                return hc.processor.Process(ctx, []*models.LogEntry{&entry})
        }

        // If we can't parse as structured log entries, create a raw entry
        rawEntry := &models.LogEntry{
                Timestamp: time.Now(),
                Source:    hc.Source(),
                RawData:   string(data),
                Message:   string(data),
        }
        
        return hc.processor.Process(ctx, []*models.LogEntry{rawEntry})
}

// processTextResponse handles plain text log data
func (hc *HTTPCollector) processTextResponse(ctx context.Context, data []byte) error {
        // Split text into lines
        lines := strings.Split(string(data), "\n")
        entries := make([]*models.LogEntry, 0, len(lines))
        
        // Create a log entry for each non-empty line
        for _, line := range lines {
                if len(strings.TrimSpace(line)) == 0 {
                        continue
                }
                
                entry := &models.LogEntry{
                        Timestamp: time.Now(),
                        Source:    hc.Source(),
                        RawData:   line,
                        Message:   line,
                }
                
                entries = append(entries, entry)
        }
        
        if len(entries) > 0 {
                return hc.processor.Process(ctx, entries)
        }
        
        return nil
}
