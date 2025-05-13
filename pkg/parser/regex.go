package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mariasu11/logstream/pkg/models"
)

// RegexParser is a parser that uses regular expressions to parse log entries
type RegexParser struct {
	patterns []*regexPattern
}

// regexPattern defines a regex pattern with named capture groups
type regexPattern struct {
	name        string
	regex       *regexp.Regexp
	timeFormats []string
	timeField   string
	msgField    string
	levelField  string
	sourceField string
}

// NewRegexParser creates a new regex parser with default patterns
func NewRegexParser() *RegexParser {
	parser := &RegexParser{
		patterns: make([]*regexPattern, 0),
	}
	
	// Add common log formats
	
	// Apache/Nginx access log format
	parser.AddPattern(
		"apache",
		`^(?P<ip>\S+) \S+ \S+ \[(?P<timestamp>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+) (?P<protocol>\S+)" (?P<status>\d+) (?P<bytes>\d+)`,
		[]string{"02/Jan/2006:15:04:05 -0700"},
		"timestamp",
		"",
		"",
		"",
	)
	
	// Common log format with timestamp, level, source, message
	parser.AddPattern(
		"common",
		`^(?P<timestamp>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)?) (?P<level>[A-Z]+) (?P<source>[^:]+): (?P<message>.+)$`,
		[]string{"2006-01-02 15:04:05", "2006-01-02 15:04:05.000"},
		"timestamp",
		"message",
		"level",
		"source",
	)
	
	// Kubernetes log format
	parser.AddPattern(
		"kubernetes",
		`^(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z) (?P<level>[A-Z]+) +(?P<source>\S+) +(?P<message>.+)$`,
		[]string{time.RFC3339, "2006-01-02T15:04:05Z"},
		"timestamp",
		"message",
		"level",
		"source",
	)
	
	// Simple format with bracket-enclosed timestamp and level
	parser.AddPattern(
		"brackets",
		`^\[(?P<timestamp>[^\]]+)\] \[(?P<level>[^\]]+)\] (?P<message>.+)$`,
		[]string{time.RFC3339, "2006-01-02 15:04:05", "Jan 02 15:04:05", "02 Jan 06 15:04 MST"},
		"timestamp",
		"message",
		"level",
		"",
	)
	
	return parser
}

// Name returns the parser name
func (p *RegexParser) Name() string {
	return "regex"
}

// AddPattern adds a new regex pattern to the parser
func (p *RegexParser) AddPattern(name, pattern string, timeFormats []string, timeField, msgField, levelField, sourceField string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}
	
	p.patterns = append(p.patterns, &regexPattern{
		name:        name,
		regex:       regex,
		timeFormats: timeFormats,
		timeField:   timeField,
		msgField:    msgField,
		levelField:  levelField,
		sourceField: sourceField,
	})
	
	return nil
}

// CanParse checks if any pattern can parse the given log line
func (p *RegexParser) CanParse(raw string) bool {
	for _, pattern := range p.patterns {
		if pattern.regex.MatchString(raw) {
			return true
		}
	}
	return false
}

// Parse parses a log entry using the first matching regex pattern
func (p *RegexParser) Parse(entry *models.LogEntry) error {
	for _, pattern := range p.patterns {
		matches := pattern.regex.FindStringSubmatch(entry.RawData)
		if matches == nil {
			continue
		}
		
		// Get capture group names
		names := pattern.regex.SubexpNames()
		
		// Extract all captured fields
		fields := make(map[string]string)
		for i, name := range names {
			if i > 0 && name != "" {
				fields[name] = matches[i]
			}
		}
		
		// Process timestamp
		if pattern.timeField != "" {
			if timeStr, ok := fields[pattern.timeField]; ok {
				for _, format := range pattern.timeFormats {
					if t, err := time.Parse(format, timeStr); err == nil {
						entry.Timestamp = t
						break
					}
				}
			}
		}
		
		// Process message
		if pattern.msgField != "" {
			if msg, ok := fields[pattern.msgField]; ok {
				entry.Message = msg
			}
		} else if entry.Message == "" {
			// If no message field is defined, use the entire log line
			entry.Message = entry.RawData
		}
		
		// Process level
		if pattern.levelField != "" {
			if level, ok := fields[pattern.levelField]; ok {
				entry.Level = strings.ToLower(level)
			}
		}
		
		// Process source
		if pattern.sourceField != "" {
			if source, ok := fields[pattern.sourceField]; ok {
				entry.Source = source
			}
		}
		
		// Initialize fields map if needed
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{})
		}
		
		// Add all captured fields to the Fields map
		for name, value := range fields {
			// Skip fields already handled
			if name == pattern.timeField || name == pattern.msgField || 
			   name == pattern.levelField || name == pattern.sourceField {
				continue
			}
			entry.Fields[name] = value
		}
		
		// Add the pattern name for reference
		entry.Fields["pattern"] = pattern.name
		
		return nil
	}
	
	// If no pattern matched, just use the raw line as the message
	if entry.Message == "" {
		entry.Message = entry.RawData
	}
	
	return nil
}

// PatternNames returns the names of all registered patterns
func (p *RegexParser) PatternNames() []string {
	names := make([]string, len(p.patterns))
	for i, pattern := range p.patterns {
		names[i] = pattern.name
	}
	return names
}
