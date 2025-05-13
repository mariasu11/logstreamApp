package storage

import (
	"context"
	"time"

	"github.com/mariasu11/logstreamApp/pkg/models"
)

// Storage defines the interface for log storage backends
type Storage interface {
	// Store saves a log entry
	Store(ctx context.Context, entry *models.LogEntry) error
	
	// Query retrieves log entries based on a query
	Query(ctx context.Context, query models.Query) ([]*models.LogEntry, error)
	
	// GetSources returns a list of all log sources
	GetSources(ctx context.Context) ([]string, error)
	
	// GetStats returns statistics about the stored logs
	GetStats(ctx context.Context) (StorageStats, error)
	
	// Close closes the storage and performs cleanup
	Close() error
}

// StorageStats contains statistics about the stored logs
type StorageStats struct {
	TotalEntries     int64
	OldestEntry      time.Time
	NewestEntry      time.Time
	EntriesBySource  map[string]int64
	EntriesByLevel   map[string]int64
	StorageSize      int64  // Size in bytes (if applicable)
	CompressionRatio float64 // Compression ratio (if applicable)
}

// MetricsByTimeRange contains metrics over a time range
type MetricsByTimeRange struct {
	StartTime time.Time
	EndTime   time.Time
	Count     int64
	Sources   map[string]int64
	Levels    map[string]int64
}

// Builder is a fluent interface for building storage queries
type Builder interface {
	// WithTimeRange sets the time range for the query
	WithTimeRange(start, end time.Time) Builder
	
	// WithSources limits the query to specific sources
	WithSources(sources ...string) Builder
	
	// WithLevels limits the query to specific log levels
	WithLevels(levels ...string) Builder
	
	// WithFilter adds a filter expression to the query
	WithFilter(filter string) Builder
	
	// WithLimit sets the maximum number of results
	WithLimit(limit int) Builder
	
	// Build creates the final query
	Build() models.Query
}

// QueryBuilder implements the Builder interface
type QueryBuilder struct {
	query models.Query
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: models.Query{
			TimeRange: models.TimeRange{},
			Limit:     100, // Default limit
		},
	}
}

// WithTimeRange implements the Builder interface
func (b *QueryBuilder) WithTimeRange(start, end time.Time) Builder {
	b.query.TimeRange.From = start
	b.query.TimeRange.To = end
	return b
}

// WithSources implements the Builder interface
func (b *QueryBuilder) WithSources(sources ...string) Builder {
	b.query.Sources = sources
	return b
}

// WithLevels implements the Builder interface
func (b *QueryBuilder) WithLevels(levels ...string) Builder {
	b.query.Levels = levels
	return b
}

// WithFilter implements the Builder interface
func (b *QueryBuilder) WithFilter(filter string) Builder {
	b.query.Filter = filter
	return b
}

// WithLimit implements the Builder interface
func (b *QueryBuilder) WithLimit(limit int) Builder {
	b.query.Limit = limit
	return b
}

// Build implements the Builder interface
func (b *QueryBuilder) Build() models.Query {
	return b.query
}
