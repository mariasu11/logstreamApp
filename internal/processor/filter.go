package processor

import (
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/logstream/pkg/models"
)

// Filter defines the interface for filtering log entries
type Filter interface {
	// Apply returns true if the entry passes the filter, false if it should be filtered out
	Apply(entry *models.LogEntry) bool
}

// LevelFilter filters log entries based on log level
type LevelFilter struct {
	Levels     map[string]bool
	IncludeLevels bool
}

// NewLevelFilter creates a new level filter
func NewLevelFilter(levels []string, include bool) *LevelFilter {
	levelSet := make(map[string]bool)
	for _, level := range levels {
		levelSet[strings.ToLower(level)] = true
	}

	return &LevelFilter{
		Levels:     levelSet,
		IncludeLevels: include,
	}
}

// Apply implements the Filter interface
func (f *LevelFilter) Apply(entry *models.LogEntry) bool {
	if entry.Level == "" {
		// If no level is set and we're filtering for specific levels, filter it out
		return !f.IncludeLevels
	}

	levelExists := f.Levels[strings.ToLower(entry.Level)]
	
	// If include levels is true, we want to include entries with matching levels
	// If include levels is false, we want to exclude entries with matching levels
	return levelExists == f.IncludeLevels
}

// RegexFilter filters log entries based on a regular expression applied to the message
type RegexFilter struct {
	Pattern *regexp.Regexp
	IncludeMatches bool
}

// NewRegexFilter creates a new regex filter
func NewRegexFilter(pattern string, include bool) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &RegexFilter{
		Pattern:        re,
		IncludeMatches: include,
	}, nil
}

// Apply implements the Filter interface
func (f *RegexFilter) Apply(entry *models.LogEntry) bool {
	matches := f.Pattern.MatchString(entry.Message)
	return matches == f.IncludeMatches
}

// TimeRangeFilter filters log entries based on a time range
type TimeRangeFilter struct {
	Start time.Time
	End   time.Time
}

// NewTimeRangeFilter creates a new time range filter
func NewTimeRangeFilter(start, end time.Time) *TimeRangeFilter {
	return &TimeRangeFilter{
		Start: start,
		End:   end,
	}
}

// Apply implements the Filter interface
func (f *TimeRangeFilter) Apply(entry *models.LogEntry) bool {
	// If start time is not set, don't filter by start time
	startCheck := f.Start.IsZero() || !entry.Timestamp.Before(f.Start)
	
	// If end time is not set, don't filter by end time
	endCheck := f.End.IsZero() || !entry.Timestamp.After(f.End)
	
	return startCheck && endCheck
}

// SourceFilter filters log entries based on their source
type SourceFilter struct {
	Sources    map[string]bool
	IncludeSources bool
}

// NewSourceFilter creates a new source filter
func NewSourceFilter(sources []string, include bool) *SourceFilter {
	sourceSet := make(map[string]bool)
	for _, source := range sources {
		sourceSet[source] = true
	}

	return &SourceFilter{
		Sources:       sourceSet,
		IncludeSources: include,
	}
}

// Apply implements the Filter interface
func (f *SourceFilter) Apply(entry *models.LogEntry) bool {
	sourceExists := f.Sources[entry.Source]
	return sourceExists == f.IncludeSources
}

// FieldFilter filters log entries based on a field value
type FieldFilter struct {
	Field     string
	Value     string
	Exact     bool
}

// NewFieldFilter creates a new field filter
func NewFieldFilter(field, value string, exact bool) *FieldFilter {
	return &FieldFilter{
		Field: field,
		Value: value,
		Exact: exact,
	}
}

// Apply implements the Filter interface
func (f *FieldFilter) Apply(entry *models.LogEntry) bool {
	fieldVal, exists := entry.Fields[f.Field]
	if !exists {
		return false
	}

	strVal, ok := fieldVal.(string)
	if !ok {
		// Try to compare directly if it's not a string
		return fieldVal == f.Value
	}

	if f.Exact {
		return strVal == f.Value
	}
	
	return strings.Contains(strVal, f.Value)
}

// CompositeFilter combines multiple filters with AND logic
type CompositeFilter struct {
	Filters []Filter
}

// NewCompositeFilter creates a new composite filter
func NewCompositeFilter(filters ...Filter) *CompositeFilter {
	return &CompositeFilter{
		Filters: filters,
	}
}

// Apply implements the Filter interface
func (f *CompositeFilter) Apply(entry *models.LogEntry) bool {
	for _, filter := range f.Filters {
		if !filter.Apply(entry) {
			return false
		}
	}
	return true
}
