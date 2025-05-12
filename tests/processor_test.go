package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/logstream/internal/processor"
	"github.com/yourusername/logstream/internal/storage"
	"github.com/yourusername/logstream/pkg/models"
	"github.com/yourusername/logstream/pkg/worker"
)

func TestProcessor(t *testing.T) {
	// Create storage and worker pool
	memStorage := storage.NewMemoryStorage()
	workerPool := worker.NewPool(2)

	// Start the worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerPool.Start(ctx)

	// Create processor
	proc := processor.NewProcessor(memStorage, workerPool)

	// Create test log entries
	entries := []*models.LogEntry{
		{
			Timestamp: time.Now(),
			Source:    "test",
			Level:     "info",
			Message:   "Test message 1",
			Fields: map[string]interface{}{
				"key1": "value1",
			},
		},
		{
			Timestamp: time.Now().Add(-1 * time.Hour),
			Source:    "test",
			Level:     "error",
			Message:   "Test message 2",
			Fields: map[string]interface{}{
				"key2": "value2",
			},
		},
	}

	// Process the entries
	err := proc.Process(ctx, entries)
	require.NoError(t, err)

	// Allow some time for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Query the storage to verify entries were processed and stored
	query := models.Query{
		Limit: 10,
	}
	results, err := memStorage.Query(ctx, query)
	require.NoError(t, err)

	// Should have both log entries
	assert.Equal(t, 2, len(results))
	assert.Equal(t, "Test message 1", results[0].Message)
	assert.Equal(t, "Test message 2", results[1].Message)
}

func TestProcessorWithFilters(t *testing.T) {
	// Create storage and worker pool
	memStorage := storage.NewMemoryStorage()
	workerPool := worker.NewPool(2)

	// Start the worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerPool.Start(ctx)

	// Create processor with a level filter
	proc := processor.NewProcessor(memStorage, workerPool)
	levelFilter := processor.NewLevelFilter([]string{"error"}, true)
	proc.AddFilter(levelFilter)

	// Create test log entries with different levels
	entries := []*models.LogEntry{
		{
			Timestamp: time.Now(),
			Source:    "test",
			Level:     "info",
			Message:   "Info message",
		},
		{
			Timestamp: time.Now(),
			Source:    "test",
			Level:     "error",
			Message:   "Error message",
		},
		{
			Timestamp: time.Now(),
			Source:    "test",
			Level:     "warn",
			Message:   "Warning message",
		},
	}

	// Process the entries
	err := proc.Process(ctx, entries)
	require.NoError(t, err)

	// Allow some time for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Query the storage to verify only error entries were stored
	query := models.Query{
		Limit: 10,
	}
	results, err := memStorage.Query(ctx, query)
	require.NoError(t, err)

	// Should have only the error message
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "Error message", results[0].Message)
	assert.Equal(t, "error", results[0].Level)
}

func TestProcessorWithTransformers(t *testing.T) {
	// Create storage and worker pool
	memStorage := storage.NewMemoryStorage()
	workerPool := worker.NewPool(2)

	// Start the worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerPool.Start(ctx)

	// Create processor with a field transformer
	proc := processor.NewProcessor(memStorage, workerPool)
	transformer := processor.NewAddFieldTransformer("environment", "test")
	proc.AddTransformer(transformer)

	// Create test log entry
	entries := []*models.LogEntry{
		{
			Timestamp: time.Now(),
			Source:    "test",
			Level:     "info",
			Message:   "Test message",
		},
	}

	// Process the entry
	err := proc.Process(ctx, entries)
	require.NoError(t, err)

	// Allow some time for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Query the storage to verify the transformer was applied
	query := models.Query{
		Limit: 10,
	}
	results, err := memStorage.Query(ctx, query)
	require.NoError(t, err)

	// Should have the environment field added
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "Test message", results[0].Message)
	assert.Equal(t, "test", results[0].Fields["environment"])
}

func TestRegexExtractTransformer(t *testing.T) {
	// Create storage and worker pool
	memStorage := storage.NewMemoryStorage()
	workerPool := worker.NewPool(2)

	// Start the worker pool
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	workerPool.Start(ctx)

	// Create processor with a regex extract transformer
	proc := processor.NewProcessor(memStorage, workerPool)
	
	// Extract user ID and action from a message like "User 12345 performed action: login"
	regexTransformer, err := processor.NewRegexExtractTransformer(
		`User (\d+) performed action: (\w+)`,
		[]string{"user_id", "action"},
	)
	require.NoError(t, err)
	proc.AddTransformer(regexTransformer)

	// Create test log entry
	entries := []*models.LogEntry{
		{
			Timestamp: time.Now(),
			Source:    "test",
			Level:     "info",
			Message:   "User 12345 performed action: login",
		},
	}

	// Process the entry
	err = proc.Process(ctx, entries)
	require.NoError(t, err)

	// Allow some time for processing to complete
	time.Sleep(100 * time.Millisecond)

	// Query the storage to verify the transformer was applied
	query := models.Query{
		Limit: 10,
	}
	results, err := memStorage.Query(ctx, query)
	require.NoError(t, err)

	// Should have the extracted fields
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "12345", results[0].Fields["user_id"])
	assert.Equal(t, "login", results[0].Fields["action"])
}

func TestCompositeFilter(t *testing.T) {
	// Create composite filter with level and regex filters
	levelFilter := processor.NewLevelFilter([]string{"error", "warn"}, true)
	regexFilter, err := processor.NewRegexFilter("database|connection", true)
	require.NoError(t, err)
	
	compositeFilter := processor.NewCompositeFilter(levelFilter, regexFilter)
	
	// Test with various log entries
	entries := []*models.LogEntry{
		{
			Level:   "error",
			Message: "Database connection failed",
		},
		{
			Level:   "error",
			Message: "Application crashed",
		},
		{
			Level:   "info",
			Message: "Database connection established",
		},
	}
	
	// Only the first entry should pass both filters
	assert.True(t, compositeFilter.Apply(entries[0]))
	assert.False(t, compositeFilter.Apply(entries[1])) // Passes level but not regex
	assert.False(t, compositeFilter.Apply(entries[2])) // Passes regex but not level
}
