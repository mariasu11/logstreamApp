package collector

import (
	"context"
	"fmt"
	"net/url"
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
	// Parse the source URI to determine the collector type
	uri, err := url.Parse(sourceURI)
	if err != nil {
		return nil, fmt.Errorf("invalid source URI %s: %w", sourceURI, err)
	}

	switch strings.ToLower(uri.Scheme) {
	case "file":
		return NewFileCollector(uri.Path, processor)
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
