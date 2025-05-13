package collector

import (
        "bufio"
        "context"
        "fmt"
        "os"
        "path/filepath"
        "strings"
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
        // First try to use the path as provided
        cleanPath := path
        
        // Check if path exists, if not try various common path resolutions
        if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
                // Try without leading slash
                cleanPath = strings.TrimPrefix(cleanPath, "/")
                
                // If we're using fixtures directory, make sure the path is correct
                if strings.Contains(cleanPath, "fixtures/") {
                        // Try to use the path as specified
                        if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
                                // Log the path we're attempting to use
                                fmt.Printf("Trying to access: %s (does not exist)\n", cleanPath)
                        }
                }
        }

        return &FileCollector{
                BaseCollector: BaseCollector{
                        name:      filepath.Base(cleanPath),
                        source:    fmt.Sprintf("file://%s", cleanPath),
                        processor: processor,
                },
                filePath:     cleanPath,
                batchSize:    100,
                pollInterval: 1 * time.Second,
        }, nil
}

// Start implements the Collector interface
func (fc *FileCollector) Start(ctx context.Context) error {
        // Print debug info
        fmt.Printf("DEBUG: Attempting to access file at path: %s\n", fc.filePath)
        
        // Try with fixtures path if it's in the URL but not in the resolved path
        if strings.Contains(fc.source, "fixtures") && !strings.Contains(fc.filePath, "fixtures") {
            fc.filePath = "fixtures/logs/" + filepath.Base(fc.filePath)
            fmt.Printf("DEBUG: Adjusted path to: %s\n", fc.filePath)
        }
        
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

        // When we're starting collection, start from the beginning of the file
        // to load all existing logs. On a production system with large files,
        // we might want to seek to the end, but for demo purposes, let's read everything.
        // Seek to the beginning of the file
        if _, err := file.Seek(0, 0); err != nil {
                return fmt.Errorf("failed to seek to beginning of file %s: %w", fc.filePath, err)
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
                                
                                // Debug output for log parsing
                                fmt.Printf("DEBUG: Processing log line: %s\n", line)
                                
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
