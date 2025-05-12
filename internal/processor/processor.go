package processor

import (
	"context"
	"sync"

	"github.com/yourusername/logstream/internal/metrics"
	"github.com/yourusername/logstream/internal/storage"
	"github.com/yourusername/logstream/pkg/models"
	"github.com/yourusername/logstream/pkg/parser"
	"github.com/yourusername/logstream/pkg/plugin"
	"github.com/yourusername/logstream/pkg/worker"
)

// Processor interface defines the methods for processing log entries
type Processor interface {
	// Process handles a batch of log entries
	Process(ctx context.Context, entries []*models.LogEntry) error
	// AddFilter adds a filter to the processing pipeline
	AddFilter(filter Filter) Processor
	// AddTransformer adds a transformer to the processing pipeline
	AddTransformer(transformer Transformer) Processor
	// AddPlugin adds a plugin to the processing pipeline
	AddPlugin(p plugin.Plugin) Processor
}

// LogProcessor implements the Processor interface
type LogProcessor struct {
	storage     storage.Storage
	workerPool  *worker.Pool
	filters     []Filter
	transformers []Transformer
	plugins     []plugin.Plugin
	parsers     []parser.Parser
	mu          sync.RWMutex
	metrics     *metrics.Metrics
}

// NewProcessor creates a new LogProcessor
func NewProcessor(storage storage.Storage, workerPool *worker.Pool) Processor {
	// Initialize with default parsers
	parsers := []parser.Parser{
		parser.NewJSONParser(),
		parser.NewRegexParser(),
	}

	return &LogProcessor{
		storage:     storage,
		workerPool:  workerPool,
		filters:     make([]Filter, 0),
		transformers: make([]Transformer, 0),
		plugins:     make([]plugin.Plugin, 0),
		parsers:     parsers,
		metrics:     metrics.GetMetrics(),
	}
}

// Process implements the Processor interface
func (p *LogProcessor) Process(ctx context.Context, entries []*models.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	p.metrics.LogBatchesReceived.Inc()
	p.metrics.LogEntriesReceived.Add(float64(len(entries)))

	// Submit each entry to the worker pool for processing
	for _, entry := range entries {
		entry := entry // capture for goroutine
		
		// Submit processing job to worker pool
		p.workerPool.Submit(func() {
			p.processEntry(ctx, entry)
		})
	}

	return nil
}

// processEntry handles processing of an individual log entry
func (p *LogProcessor) processEntry(ctx context.Context, entry *models.LogEntry) {
	// Parse the raw log data if needed
	if entry.RawData != "" && (entry.Message == "" || len(entry.Fields) == 0) {
		for _, parser := range p.parsers {
			if parser.CanParse(entry.RawData) {
				if err := parser.Parse(entry); err == nil {
					break // Successfully parsed
				}
			}
		}
	}

	// Apply filters
	p.mu.RLock()
	filters := p.filters
	p.mu.RUnlock()

	for _, filter := range filters {
		if !filter.Apply(entry) {
			p.metrics.LogEntriesFiltered.Inc()
			return // Entry filtered out
		}
	}

	// Apply transformers
	p.mu.RLock()
	transformers := p.transformers
	p.mu.RUnlock()

	for _, transformer := range transformers {
		transformer.Transform(entry)
	}

	// Apply plugins
	p.mu.RLock()
	plugins := p.plugins
	p.mu.RUnlock()

	for _, plugin := range plugins {
		plugin.ProcessLogEntry(entry)
	}

	// Store the processed entry
	if err := p.storage.Store(ctx, entry); err != nil {
		p.metrics.LogEntriesErrored.Inc()
		return
	}

	p.metrics.LogEntriesProcessed.Inc()
}

// AddFilter adds a filter to the processing pipeline
func (p *LogProcessor) AddFilter(filter Filter) Processor {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.filters = append(p.filters, filter)
	return p
}

// AddTransformer adds a transformer to the processing pipeline
func (p *LogProcessor) AddTransformer(transformer Transformer) Processor {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transformers = append(p.transformers, transformer)
	return p
}

// AddPlugin adds a plugin to the processing pipeline
func (p *LogProcessor) AddPlugin(plugin plugin.Plugin) Processor {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plugins = append(p.plugins, plugin)
	return p
}
