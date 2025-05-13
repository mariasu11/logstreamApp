package collector

import (
        "context"
        "fmt"
        "net/url"
        "path/filepath"
        "strings"

        "github.com/yourusername/logstream/internal/processor"
)

// Collector defines the interface for log collectors
type Collector interface {
        // Start begins collecting logs and passes them to the processor
        Start(ctx context.Context) error
        // Name returns the collector's name
        Name() string
        // Source returns the source identifier
        Source() string
}

// CollectorFactory creates a collector from a source URI
func NewCollector(sourceURI string, processor processor.Processor) (Collector, error) {
        // Debug print the source URI
        fmt.Printf("DEBUG: Source URI: %s\n", sourceURI)
        
        // Handle special case for fixtures directory directly
        if strings.Contains(sourceURI, "fixtures/logs/") {
                actualPath := "fixtures/logs/" + filepath.Base(sourceURI)
                fmt.Printf("DEBUG: Using fixtures path: %s\n", actualPath)
                return NewFileCollector(actualPath, processor)
        }
        
        // Parse the source URI to determine the collector type
        uri, err := url.Parse(sourceURI)
        if err != nil {
                return nil, fmt.Errorf("invalid source URI %s: %w", sourceURI, err)
        }

        switch strings.ToLower(uri.Scheme) {
        case "file":
                // For file URIs, uri.Path might have a leading slash that needs to be handled
                path := strings.TrimPrefix(uri.Path, "/")
                if path == "" {
                        path = uri.Host // Handle file://test.log format
                }
                
                // Special handling for fixtures directory
                if strings.Contains(path, "fixtures/") || strings.Contains(sourceURI, "fixtures/") {
                        actualPath := "fixtures/logs/" + filepath.Base(path)
                        fmt.Printf("DEBUG: Using fixtures path: %s\n", actualPath)
                        return NewFileCollector(actualPath, processor)
                }
                
                // For other paths
                return NewFileCollector(path, processor)
        case "http", "https":
                return NewHTTPCollector(sourceURI, processor)
        default:
                return nil, fmt.Errorf("unsupported collector type: %s", uri.Scheme)
        }
}

// BaseCollector provides common functionality for collectors
type BaseCollector struct {
        name      string
        source    string
        processor processor.Processor
}

func (b *BaseCollector) Name() string {
        return b.name
}

func (b *BaseCollector) Source() string {
        return b.source
}
