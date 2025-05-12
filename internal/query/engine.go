package query

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/logstream/pkg/models"
)

// LogQueryLanguage (LQL) parser for advanced query syntax
type LQLParser struct {
	tokenizer *regexp.Regexp
}

// NewLQLParser creates a new LQL parser
func NewLQLParser() *LQLParser {
	// Regex to tokenize LQL query string
	tokenizer := regexp.MustCompile(`("[^"]*"|\S+)`)
	
	return &LQLParser{
		tokenizer: tokenizer,
	}
}

// Parse parses an LQL query string into a structured query
func (p *LQLParser) Parse(queryStr string) (models.Query, error) {
	query := models.Query{
		Limit:        100, // Default limit
		FilterFields: make(map[string]string),
	}

	// Extract tokens from the query string
	tokens := p.tokenizer.FindAllString(queryStr, -1)
	if len(tokens) == 0 {
		return query, nil
	}

	// Process tokens
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]
		
		// Remove quotes if present
		if strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"") {
			token = token[1 : len(token)-1]
		}

		// Check for operators
		if i+1 < len(tokens) {
			nextToken := tokens[i+1]
			if strings.HasPrefix(nextToken, "\"") && strings.HasSuffix(nextToken, "\"") {
				nextToken = nextToken[1 : len(nextToken)-1]
			}

			switch strings.ToLower(token) {
			case "from":
				t, err := p.parseTime(nextToken)
				if err == nil {
					query.TimeRange.From = t
					i++
					continue
				}

			case "to":
				t, err := p.parseTime(nextToken)
				if err == nil {
					query.TimeRange.To = t
					i++
					continue
				}

			case "source:":
			case "source":
				query.Sources = append(query.Sources, strings.Split(nextToken, ",")...)
				i++
				continue

			case "level:":
			case "level":
				query.Levels = append(query.Levels, strings.Split(nextToken, ",")...)
				i++
				continue

			case "limit:":
			case "limit":
				if limit, err := parseInt(nextToken); err == nil {
					query.Limit = limit
					i++
					continue
				}
			}
		}

		// Check for field expressions (field=value)
		if strings.Contains(token, "=") {
			parts := strings.SplitN(token, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				query.FilterFields[key] = value
				continue
			}
		}

		// If we get here, treat as part of the free-text filter
		if query.Filter != "" {
			query.Filter += " "
		}
		query.Filter += token
	}

	return query, nil
}

// parseTime parses a time string into a timestamp
func (p *LQLParser) parseTime(timeStr string) (time.Time, error) {
	// Check for relative time expressions
	if strings.HasPrefix(timeStr, "-") {
		return p.parseRelativeTime(timeStr)
	}

	// Try common formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
		"15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse time: %s", timeStr)
}

// parseRelativeTime parses relative time expressions like "-1h", "-30m", etc.
func (p *LQLParser) parseRelativeTime(relTime string) (time.Time, error) {
	// Remove the leading minus sign
	durationStr := strings.TrimPrefix(relTime, "-")

	// Parse the duration
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid relative time: %s", relTime)
	}

	// Calculate the time
	return time.Now().Add(-duration), nil
}

// parseInt parses an integer with error handling
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// BuildPlanExecutor implements query plan optimization and execution
type BuildPlanExecutor struct {
	originalQuery models.Query
	optimizedPlan QueryPlan
}

// QueryPlan represents an optimized execution plan for a query
type QueryPlan struct {
	TimeFilter  bool
	SourceFilter bool
	LevelFilter  bool
	FieldFilters []FieldFilter
	TextFilter   bool
	EstimatedCost int // Lower is better
}

// FieldFilter represents a filter on a specific field
type FieldFilter struct {
	Field string
	Value string
	Op    string // "eq", "contains", "regex"
}

// NewBuildPlanExecutor creates a new query plan executor
func NewBuildPlanExecutor(query models.Query) *BuildPlanExecutor {
	executor := &BuildPlanExecutor{
		originalQuery: query,
	}
	
	executor.optimizedPlan = executor.buildQueryPlan()
	return executor
}

// buildQueryPlan creates an optimized query execution plan
func (e *BuildPlanExecutor) buildQueryPlan() QueryPlan {
	plan := QueryPlan{
		TimeFilter:   !e.originalQuery.TimeRange.From.IsZero() || !e.originalQuery.TimeRange.To.IsZero(),
		SourceFilter: len(e.originalQuery.Sources) > 0,
		LevelFilter:  len(e.originalQuery.Levels) > 0,
		TextFilter:   e.originalQuery.Filter != "",
	}
	
	// Add field filters
	for field, value := range e.originalQuery.FilterFields {
		op := "eq"
		if strings.Contains(value, "*") {
			op = "regex"
		} else if strings.HasPrefix(value, "~") {
			op = "contains"
			value = strings.TrimPrefix(value, "~")
		}
		
		plan.FieldFilters = append(plan.FieldFilters, FieldFilter{
			Field: field,
			Value: value,
			Op:    op,
		})
	}
	
	// Estimate cost based on selectivity
	plan.EstimatedCost = e.estimateQueryCost(plan)
	
	return plan
}

// estimateQueryCost assigns a cost to the query plan for optimization
func (e *BuildPlanExecutor) estimateQueryCost(plan QueryPlan) int {
	cost := 100 // Base cost
	
	// Time filters are usually very selective
	if plan.TimeFilter {
		cost -= 40
	}
	
	// Source filters can be selective
	if plan.SourceFilter {
		cost -= 20
	}
	
	// Level filters are less selective
	if plan.LevelFilter {
		cost -= 10
	}
	
	// Field filters vary in selectivity
	for _, filter := range plan.FieldFilters {
		switch filter.Op {
		case "eq":
			cost -= 15
		case "regex":
			cost -= 5 // Regex is expensive
		case "contains":
			cost -= 8
		}
	}
	
	// Text filters are usually expensive and less selective
	if plan.TextFilter {
		cost -= 5
	}
	
	return cost
}

// GetOptimizedPlan returns the optimized query plan
func (e *BuildPlanExecutor) GetOptimizedPlan() QueryPlan {
	return e.optimizedPlan
}

// EstimateResultSize estimates the number of results the query will return
func (e *BuildPlanExecutor) EstimateResultSize() int {
	// This would use statistics from the storage layer in a real implementation
	baseSize := 10000 // Assume 10,000 total logs
	
	estimatedSize := baseSize
	
	// Apply filter estimates
	if e.optimizedPlan.TimeFilter {
		estimatedSize = estimatedSize / 4
	}
	
	if e.optimizedPlan.SourceFilter {
		estimatedSize = estimatedSize / 2
	}
	
	if e.optimizedPlan.LevelFilter {
		estimatedSize = estimatedSize / 2
	}
	
	for range e.optimizedPlan.FieldFilters {
		estimatedSize = estimatedSize / 3
	}
	
	if e.optimizedPlan.TextFilter {
		estimatedSize = estimatedSize / 2
	}
	
	// Respect the query limit
	if e.originalQuery.Limit > 0 && estimatedSize > e.originalQuery.Limit {
		estimatedSize = e.originalQuery.Limit
	}
	
	return estimatedSize
}
