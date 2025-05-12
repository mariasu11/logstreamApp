package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yourusername/logstream/internal/processor"
	"github.com/yourusername/logstream/pkg/models"
)

// FileCollector collects logs from files
type FileCollector struct {
	BaseCollector
	filePath    string
	batchSize   int
	pollInterval time.Duration
}

// NewFileCollector creates a new file collector
func NewFileCollector(path string, processor processor.Processor) (*FileCollector, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid file path %s: %w", path, err)
	}

	return &FileCollector{
		BaseCollector: BaseCollector{
			name:      filepath.Base(path),
			source:    fmt.Sprintf("file://%s", absPath),
			processor: processor,
		},
		filePath:     absPath,
		batchSize:    100,
		pollInterval: 1 * time.Second,
	}, nil
}

// Start implements the Collector interface
func (fc *FileCollector) Start(ctx context.Context) error {
	// Check if file exists
	info, err := os.Stat(fc.filePath)
	if err != nil {
		return fmt.Errorf("cannot access file %s: %w", fc.filePath, err)
	}

	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", fc.filePath)
	}

	// Start monitoring the file
	file, err := os.Open(fc.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fc.filePath, err)
	}
	defer file.Close()

	// Seek to the end for new files
	if _, err := file.Seek(0, 2); err != nil {
		return fmt.Errorf("failed to seek to end of file %s: %w", fc.filePath, err)
	}

	// Watch for new content
	scanner := bufio.NewScanner(file)
	ticker := time.NewTicker(fc.pollInterval)
	defer ticker.Stop()

	batch := make([]*models.LogEntry, 0, fc.batchSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Read new lines
			hasNewContent := false
			for scanner.Scan() {
				hasNewContent = true
				line := scanner.Text()
				
				// Create a log entry
				entry := &models.LogEntry{
					Timestamp: time.Now(),
					Source:    fc.Source(),
					RawData:   line,
					Message:   line, // Use raw line as message until processed
				}
				
				batch = append(batch, entry)
				
				// Process batch if it's full
				if len(batch) >= fc.batchSize {
					if err := fc.processor.Process(ctx, batch); err != nil {
						return fmt.Errorf("failed to process batch: %w", err)
					}
					batch = batch[:0] // Clear batch but keep capacity
				}
			}
			
			// Check for scanner errors
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading file %s: %w", fc.filePath, err)
			}
			
			// Process any remaining entries in the batch
			if hasNewContent && len(batch) > 0 {
				if err := fc.processor.Process(ctx, batch); err != nil {
					return fmt.Errorf("failed to process batch: %w", err)
				}
				batch = batch[:0] // Clear batch but keep capacity
			}
		}
	}
}
