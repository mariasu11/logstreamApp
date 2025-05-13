package storage

import (
        "context"
        "fmt"
        "sort"
        "strings"
        "sync"
        "time"

        "github.com/mariasu11/logstreamApp/pkg/models"
)

// MemoryStorage implements in-memory storage for log entries
type MemoryStorage struct {
        entries  []*models.LogEntry
        mutex    sync.RWMutex
        capacity int
}

// NewMemoryStorage creates a new memory storage instance
func NewMemoryStorage() *MemoryStorage {
        return &MemoryStorage{
                entries:  make([]*models.LogEntry, 0, 10000),
                capacity: 1000000, // Default capacity of 1 million entries
        }
}

// WithCapacity sets the maximum capacity of the memory storage
func (m *MemoryStorage) WithCapacity(capacity int) *MemoryStorage {
        m.capacity = capacity
        return m
}

// Store implements the Storage interface
func (m *MemoryStorage) Store(ctx context.Context, entry *models.LogEntry) error {
        m.mutex.Lock()
        defer m.mutex.Unlock()

        // Check if we're at capacity
        if len(m.entries) >= m.capacity {
                // Remove the oldest entry
                m.entries = m.entries[1:]
        }

        // Add the new entry
        m.entries = append(m.entries, entry)
        return nil
}

// Query implements the Storage interface
func (m *MemoryStorage) Query(ctx context.Context, query models.Query) ([]*models.LogEntry, error) {
        m.mutex.RLock()
        defer m.mutex.RUnlock()

        // Create a slice to hold the matching entries
        var result []*models.LogEntry

        // Process each entry
        for _, entry := range m.entries {
                // Check if the context is cancelled
                select {
                case <-ctx.Done():
                        return nil, ctx.Err()
                default:
                        // Continue processing
                }

                // Apply time range filter if set
                if !query.TimeRange.From.IsZero() && entry.Timestamp.Before(query.TimeRange.From) {
                        continue
                }
                if !query.TimeRange.To.IsZero() && entry.Timestamp.After(query.TimeRange.To) {
                        continue
                }

                // Apply source filter if set
                if len(query.Sources) > 0 {
                        sourceMatch := false
                        for _, source := range query.Sources {
                                if entry.Source == source {
                                        sourceMatch = true
                                        break
                                }
                        }
                        if !sourceMatch {
                                continue
                        }
                }

                // Apply level filter if set
                if len(query.Levels) > 0 {
                        levelMatch := false
                        for _, level := range query.Levels {
                                if strings.EqualFold(entry.Level, level) {
                                        levelMatch = true
                                        break
                                }
                        }
                        if !levelMatch {
                                continue
                        }
                }

                // Apply custom filter if set
                if query.Filter != "" {
                        filterLower := strings.ToLower(query.Filter)
                        messageLower := strings.ToLower(entry.Message)
                        
                        // Case-insensitive check if filter string is contained in message
                        if !strings.Contains(messageLower, filterLower) {
                                // Also check fields for filter match
                                fieldMatch := false
                                
                                // Search in source and level too
                                sourceLower := strings.ToLower(entry.Source)
                                levelLower := strings.ToLower(entry.Level)
                                
                                if strings.Contains(sourceLower, filterLower) || 
                                   strings.Contains(levelLower, filterLower) {
                                    fieldMatch = true
                                } else {
                                    // Check other fields
                                    for key, value := range entry.Fields {
                                        keyLower := strings.ToLower(key)
                                        if strings.Contains(keyLower, filterLower) {
                                            fieldMatch = true
                                            break
                                        }
                                        
                                        if strValue, ok := value.(string); ok {
                                            strValueLower := strings.ToLower(strValue)
                                            if strings.Contains(strValueLower, filterLower) {
                                                fieldMatch = true
                                                break
                                            }
                                        }
                                    }
                                }
                                
                                if !fieldMatch {
                                        continue
                                }
                        }
                }

                // If we get here, the entry matches all filters
                result = append(result, entry)

                // Check if we've reached the limit
                if query.Limit > 0 && len(result) >= query.Limit {
                        break
                }
        }

        // Sort results by timestamp (newest first)
        sort.Slice(result, func(i, j int) bool {
                return result[i].Timestamp.After(result[j].Timestamp)
        })

        return result, nil
}

// GetSources implements the Storage interface
func (m *MemoryStorage) GetSources(ctx context.Context) ([]string, error) {
        m.mutex.RLock()
        defer m.mutex.RUnlock()

        // Use a map to track unique sources
        sources := make(map[string]bool)

        for _, entry := range m.entries {
                // Check if the context is cancelled
                select {
                case <-ctx.Done():
                        return nil, ctx.Err()
                default:
                        // Continue processing
                }

                sources[entry.Source] = true
        }

        // Convert map keys to slice
        result := make([]string, 0, len(sources))
        for source := range sources {
                result = append(result, source)
        }

        return result, nil
}

// GetStats implements the Storage interface
func (m *MemoryStorage) GetStats(ctx context.Context) (StorageStats, error) {
        m.mutex.RLock()
        defer m.mutex.RUnlock()

        stats := StorageStats{
                TotalEntries:    int64(len(m.entries)),
                EntriesBySource: make(map[string]int64),
                EntriesByLevel:  make(map[string]int64),
        }

        // Initialize with zero time
        stats.OldestEntry = time.Time{}
        stats.NewestEntry = time.Time{}

        for _, entry := range m.entries {
                // Check if the context is cancelled
                select {
                case <-ctx.Done():
                        return StorageStats{}, ctx.Err()
                default:
                        // Continue processing
                }

                // Update oldest/newest timestamps
                if stats.OldestEntry.IsZero() || entry.Timestamp.Before(stats.OldestEntry) {
                        stats.OldestEntry = entry.Timestamp
                }
                if stats.NewestEntry.IsZero() || entry.Timestamp.After(stats.NewestEntry) {
                        stats.NewestEntry = entry.Timestamp
                }

                // Count by source
                stats.EntriesBySource[entry.Source]++

                // Count by level
                stats.EntriesByLevel[entry.Level]++
        }

        // Estimate storage size (very rough approximation)
        // Assuming average entry size of 500 bytes
        stats.StorageSize = int64(len(m.entries) * 500)
        stats.CompressionRatio = 1.0 // No compression in memory

        return stats, nil
}

// Close implements the Storage interface
func (m *MemoryStorage) Close() error {
        m.mutex.Lock()
        defer m.mutex.Unlock()

        // Just clear the entries
        m.entries = nil
        return nil
}

// GetMetricsByTimeRange returns metrics for entries in a given time range
func (m *MemoryStorage) GetMetricsByTimeRange(ctx context.Context, start, end time.Time, interval time.Duration) ([]MetricsByTimeRange, error) {
        m.mutex.RLock()
        defer m.mutex.RUnlock()

        if interval == 0 {
                return nil, fmt.Errorf("interval cannot be zero")
        }

        // If no end time specified, use current time
        if end.IsZero() {
                end = time.Now()
        }

        // If no start time specified or it's after end time, error
        if start.IsZero() || start.After(end) {
                return nil, fmt.Errorf("invalid time range: start must be before end")
        }

        // Calculate number of intervals
        numIntervals := int(end.Sub(start) / interval)
        if numIntervals == 0 {
                numIntervals = 1
        }

        // Initialize results
        results := make([]MetricsByTimeRange, numIntervals)
        for i := 0; i < numIntervals; i++ {
                intervalStart := start.Add(time.Duration(i) * interval)
                intervalEnd := start.Add(time.Duration(i+1) * interval)
                if i == numIntervals-1 {
                        intervalEnd = end // Ensure the last interval includes the end time
                }

                results[i] = MetricsByTimeRange{
                        StartTime: intervalStart,
                        EndTime:   intervalEnd,
                        Sources:   make(map[string]int64),
                        Levels:    make(map[string]int64),
                }
        }

        // Process each entry
        for _, entry := range m.entries {
                // Check if the context is cancelled
                select {
                case <-ctx.Done():
                        return nil, ctx.Err()
                default:
                        // Continue processing
                }

                // Skip entries outside the time range
                if entry.Timestamp.Before(start) || entry.Timestamp.After(end) {
                        continue
                }

                // Find which interval this entry belongs to
                interval := int(entry.Timestamp.Sub(start) / interval)
                if interval < 0 {
                        interval = 0
                }
                if interval >= numIntervals {
                        interval = numIntervals - 1
                }

                // Update metrics for this interval
                results[interval].Count++
                results[interval].Sources[entry.Source]++
                results[interval].Levels[entry.Level]++
        }

        return results, nil
}
