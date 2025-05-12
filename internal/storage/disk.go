package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/logstream/pkg/models"
)

// DiskStorage implements disk-based storage for log entries
type DiskStorage struct {
	basePath      string
	currentFile   *os.File
	currentDay    string
	mutex         sync.RWMutex
	inMemoryCache []*models.LogEntry // Small cache for fast queries
	maxCacheSize  int
}

// NewDiskStorage creates a new disk storage instance
func NewDiskStorage(basePath string) (*DiskStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &DiskStorage{
		basePath:     basePath,
		maxCacheSize: 10000, // Cache last 10,000 entries in memory for fast queries
	}, nil
}

// WithMaxCacheSize sets the maximum in-memory cache size
func (d *DiskStorage) WithMaxCacheSize(size int) *DiskStorage {
	d.maxCacheSize = size
	return d
}

// Store implements the Storage interface
func (d *DiskStorage) Store(ctx context.Context, entry *models.LogEntry) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Determine the current day for file naming
	day := entry.Timestamp.Format("2006-01-02")

	// If we're writing to a different day, close the current file
	if d.currentFile != nil && d.currentDay != day {
		d.currentFile.Close()
		d.currentFile = nil
	}

	// Open a file for the current day if needed
	if d.currentFile == nil {
		fileName := filepath.Join(d.basePath, fmt.Sprintf("logs-%s.json", day))
		fileExists := fileExists(fileName)

		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		// If it's a new file, start a JSON array
		if !fileExists {
			if _, err := file.WriteString("[\n"); err != nil {
				file.Close()
				return fmt.Errorf("failed to write file header: %w", err)
			}
		} else {
			// If the file exists, remove the closing bracket to continue the array
			info, err := file.Stat()
			if err != nil {
				file.Close()
				return fmt.Errorf("failed to get file info: %w", err)
			}

			if info.Size() > 2 { // Only try to truncate if file has content
				if err := file.Truncate(info.Size() - 2); err != nil {
					file.Close()
					return fmt.Errorf("failed to truncate file: %w", err)
				}
				if _, err := file.Seek(info.Size()-2, io.SeekStart); err != nil {
					file.Close()
					return fmt.Errorf("failed to seek in file: %w", err)
				}
				if _, err := file.WriteString(",\n"); err != nil {
					file.Close()
					return fmt.Errorf("failed to write separator: %w", err)
				}
			}
		}

		d.currentFile = file
		d.currentDay = day
	}

	// Serialize the entry to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize log entry: %w", err)
	}

	// Write the entry to the file
	if _, err := d.currentFile.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	// End the array
	if _, err := d.currentFile.WriteString("\n]"); err != nil {
		return fmt.Errorf("failed to write file footer: %w", err)
	}

	// Update in-memory cache
	d.updateCache(entry)

	return nil
}

// updateCache adds an entry to the in-memory cache, maintaining size limits
func (d *DiskStorage) updateCache(entry *models.LogEntry) {
	// Add to in-memory cache for quick queries
	d.inMemoryCache = append(d.inMemoryCache, entry)

	// Trim cache if it exceeds maximum size
	if len(d.inMemoryCache) > d.maxCacheSize {
		// Remove oldest entries
		excess := len(d.inMemoryCache) - d.maxCacheSize
		d.inMemoryCache = d.inMemoryCache[excess:]
	}
}

// Query implements the Storage interface
func (d *DiskStorage) Query(ctx context.Context, query models.Query) ([]*models.LogEntry, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// For simple queries that can be satisfied from cache, use the cache
	if d.canUseCache(query) {
		return d.queryCache(ctx, query)
	}

	// For more complex queries, we need to scan disk files
	return d.queryDisk(ctx, query)
}

// canUseCache determines if a query can be satisfied from the in-memory cache
func (d *DiskStorage) canUseCache(query models.Query) bool {
	// If there's no time range, or the range is very recent, use cache
	if query.TimeRange.From.IsZero() {
		// Calculate the earliest timestamp in cache
		if len(d.inMemoryCache) == 0 {
			return false
		}
		
		earliest := d.inMemoryCache[0].Timestamp
		timeLimit := time.Now().Add(-24 * time.Hour) // Arbitrary limit

		// If cache doesn't go back far enough, can't use it
		if earliest.After(timeLimit) {
			return true
		}
	}
	
	// If the query specifically requests older data, don't use cache
	// This is a simplification - a more sophisticated implementation would check
	// if the cache contains the entire requested time range
	return false
}

// queryCache performs a query using the in-memory cache
func (d *DiskStorage) queryCache(ctx context.Context, query models.Query) ([]*models.LogEntry, error) {
	// Apply filters to in-memory cache
	var result []*models.LogEntry

	for _, entry := range d.inMemoryCache {
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
			// Simple implementation - check if filter string is contained in message
			if !strings.Contains(entry.Message, query.Filter) {
				// Also check fields for filter match
				fieldMatch := false
				for _, value := range entry.Fields {
					if strValue, ok := value.(string); ok {
						if strings.Contains(strValue, query.Filter) {
							fieldMatch = true
							break
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

// queryDisk performs a query by scanning log files on disk
func (d *DiskStorage) queryDisk(ctx context.Context, query models.Query) ([]*models.LogEntry, error) {
	// Create a slice to hold the matching entries
	var result []*models.LogEntry

	// Determine which files need to be scanned based on the time range
	fileNames, err := d.getRelevantFiles(query.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get relevant files: %w", err)
	}

	// Process each file
	for _, fileName := range fileNames {
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue processing
		}

		// Load and process the file
		entries, err := d.loadEntriesFromFile(ctx, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to load entries from file %s: %w", fileName, err)
		}

		// Apply filters
		for _, entry := range entries {
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
				// Simple implementation - check if filter string is contained in message
				if !strings.Contains(entry.Message, query.Filter) {
					// Also check fields for filter match
					fieldMatch := false
					for _, value := range entry.Fields {
						if strValue, ok := value.(string); ok {
							if strings.Contains(strValue, query.Filter) {
								fieldMatch = true
								break
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

		// If we've reached the limit, stop processing files
		if query.Limit > 0 && len(result) >= query.Limit {
			break
		}
	}

	// Sort results by timestamp (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.After(result[j].Timestamp)
	})

	// Limit the results if needed
	if query.Limit > 0 && len(result) > query.Limit {
		result = result[:query.Limit]
	}

	return result, nil
}

// getRelevantFiles returns a list of log files that might contain entries in the given time range
func (d *DiskStorage) getRelevantFiles(timeRange models.TimeRange) ([]string, error) {
	// List all log files
	files, err := filepath.Glob(filepath.Join(d.basePath, "logs-*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	// If no time range is specified, return all files
	if timeRange.From.IsZero() && timeRange.To.IsZero() {
		return files, nil
	}

	// Filter files by date in filename
	var relevantFiles []string
	for _, file := range files {
		// Extract date from filename (format: logs-YYYY-MM-DD.json)
		baseName := filepath.Base(file)
		if !strings.HasPrefix(baseName, "logs-") || !strings.HasSuffix(baseName, ".json") {
			continue
		}

		dateStr := strings.TrimPrefix(baseName, "logs-")
		dateStr = strings.TrimSuffix(dateStr, ".json")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			// If date can't be parsed, include the file to be safe
			relevantFiles = append(relevantFiles, file)
			continue
		}

		// Check if file's date is in the time range
		if (!timeRange.From.IsZero() && fileDate.Before(timeRange.From.AddDate(0, 0, -1))) ||
			(!timeRange.To.IsZero() && fileDate.After(timeRange.To.AddDate(0, 0, 1))) {
			// File is outside the time range, skip it
			continue
		}

		relevantFiles = append(relevantFiles, file)
	}

	return relevantFiles, nil
}

// loadEntriesFromFile loads log entries from a file
func (d *DiskStorage) loadEntriesFromFile(ctx context.Context, fileName string) ([]*models.LogEntry, error) {
	// Open the file
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the entire file
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var entries []*models.LogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return entries, nil
}

// GetSources implements the Storage interface
func (d *DiskStorage) GetSources(ctx context.Context) ([]string, error) {
	// Use in-memory cache for quick source listing
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// Use a map to track unique sources
	sources := make(map[string]bool)

	// First check the cache
	for _, entry := range d.inMemoryCache {
		sources[entry.Source] = true
	}

	// If cache is empty, scan disk
	if len(sources) == 0 {
		// List all log files
		files, err := filepath.Glob(filepath.Join(d.basePath, "logs-*.json"))
		if err != nil {
			return nil, fmt.Errorf("failed to list log files: %w", err)
		}

		// Scan a sample of files to find sources
		maxFilesToScan := 5
		filesScanned := 0
		
		for _, file := range files {
			if filesScanned >= maxFilesToScan {
				break
			}
			
			entries, err := d.loadEntriesFromFile(ctx, file)
			if err != nil {
				// Skip problematic files
				continue
			}
			
			for _, entry := range entries {
				sources[entry.Source] = true
			}
			
			filesScanned++
		}
	}

	// Convert map keys to slice
	result := make([]string, 0, len(sources))
	for source := range sources {
		result = append(result, source)
	}

	return result, nil
}

// GetStats implements the Storage interface
func (d *DiskStorage) GetStats(ctx context.Context) (StorageStats, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	stats := StorageStats{
		EntriesBySource: make(map[string]int64),
		EntriesByLevel:  make(map[string]int64),
	}

	// List all log files
	files, err := filepath.Glob(filepath.Join(d.basePath, "logs-*.json"))
	if err != nil {
		return stats, fmt.Errorf("failed to list log files: %w", err)
	}

	// Calculate stats
	for _, file := range files {
		// Check if the context is cancelled
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
			// Continue processing
		}

		// Get file stats
		fileInfo, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Update storage size
		stats.StorageSize += fileInfo.Size()

		// Extract date from filename to determine oldest/newest
		baseName := filepath.Base(file)
		if !strings.HasPrefix(baseName, "logs-") || !strings.HasSuffix(baseName, ".json") {
			continue
		}

		dateStr := strings.TrimPrefix(baseName, "logs-")
		dateStr = strings.TrimSuffix(dateStr, ".json")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Update oldest/newest timestamps
		if stats.OldestEntry.IsZero() || fileDate.Before(stats.OldestEntry) {
			stats.OldestEntry = fileDate
		}
		if stats.NewestEntry.IsZero() || fileDate.After(stats.NewestEntry) {
			stats.NewestEntry = fileDate
		}

		// Sample a few entries from each file to estimate counts
		// For a more accurate count, we'd need to read all files completely
		entries, err := d.loadEntriesFromFile(ctx, file)
		if err != nil {
			continue
		}

		// Update counts
		entriesInFile := int64(len(entries))
		stats.TotalEntries += entriesInFile

		// Sample first 100 entries to estimate distribution
		sampleSize := min(100, len(entries))
		for i := 0; i < sampleSize; i++ {
			entry := entries[i]
			sourceRatio := float64(entriesInFile) / float64(sampleSize)
			stats.EntriesBySource[entry.Source] += int64(sourceRatio)
			stats.EntriesByLevel[entry.Level] += int64(sourceRatio)
		}
	}

	// Estimate compression ratio (very rough)
	if stats.StorageSize > 0 {
		// Assume each entry is about 500 bytes uncompressed
		uncompressedSize := stats.TotalEntries * 500
		stats.CompressionRatio = float64(uncompressedSize) / float64(stats.StorageSize)
	} else {
		stats.CompressionRatio = 1.0
	}

	return stats, nil
}

// Close implements the Storage interface
func (d *DiskStorage) Close() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Close current file if open
	if d.currentFile != nil {
		err := d.currentFile.Close()
		d.currentFile = nil
		return err
	}

	return nil
}

// Helper functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
