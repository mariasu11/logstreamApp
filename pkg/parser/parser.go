package parser

import (
	"github.com/mariasu11/logstream/pkg/models"
)

// Parser defines the interface for log parsers
type Parser interface {
	// Parse parses a log entry from the raw data
	Parse(entry *models.LogEntry) error
	
	// CanParse checks if this parser can handle the given log format
	CanParse(raw string) bool
	
	// Name returns the parser name
	Name() string
}

// ParserRegistry manages available parsers
type ParserRegistry struct {
	parsers []Parser
}

// NewParserRegistry creates a new parser registry with default parsers
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: []Parser{
			NewJSONParser(),
			NewRegexParser(),
			// Add other default parsers here
		},
	}
}

// AddParser adds a parser to the registry
func (r *ParserRegistry) AddParser(parser Parser) {
	r.parsers = append(r.parsers, parser)
}

// ParseLogEntry attempts to parse a log entry using all registered parsers
func (r *ParserRegistry) ParseLogEntry(entry *models.LogEntry) error {
	if entry.RawData == "" {
		return nil // Nothing to parse
	}
	
	// Try each parser in order
	for _, parser := range r.parsers {
		if parser.CanParse(entry.RawData) {
			return parser.Parse(entry)
		}
	}
	
	// If no parser can handle it, just use the raw data as the message
	if entry.Message == "" {
		entry.Message = entry.RawData
	}
	
	return nil
}

// GetParsers returns all registered parsers
func (r *ParserRegistry) GetParsers() []Parser {
	return r.parsers
}

// GetParserByName returns a parser by name
func (r *ParserRegistry) GetParserByName(name string) Parser {
	for _, parser := range r.parsers {
		if parser.Name() == name {
			return parser
		}
	}
	return nil
}
