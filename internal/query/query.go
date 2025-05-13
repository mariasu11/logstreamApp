package query

import (
        "context"
        "fmt"
        "regexp"
        "strconv"
        "strings"

        "github.com/mariasu11/logstreamApp/internal/storage"
        "github.com/mariasu11/logstreamApp/pkg/models"
)

// QueryEngine performs log queries and analysis
type QueryEngine interface {
        // Execute runs a query and returns matching log entries
        Execute(query models.Query) ([]*models.LogEntry, error)
        
        // Analyze performs analysis on log data
        Analyze(analysis models.Analysis) (*models.AnalysisResult, error)
        
        // ParseQuery parses a query string into a structured query
        ParseQuery(queryString string) (models.Query, error)
}

// Engine implements the QueryEngine interface
type Engine struct {
        storage storage.Storage
}

// NewEngine creates a new query engine
func NewEngine(storage storage.Storage) *Engine {
        return &Engine{
                storage: storage,
        }
}

// Execute implements the QueryEngine interface
func (e *Engine) Execute(query models.Query) ([]*models.LogEntry, error) {
        ctx := context.Background()
        return e.storage.Query(ctx, query)
}

// Analyze implements the QueryEngine interface
func (e *Engine) Analyze(analysis models.Analysis) (*models.AnalysisResult, error) {
        ctx := context.Background()
        
        // Create a query from the analysis parameters
        query := models.Query{
                TimeRange: analysis.TimeRange,
                Sources:   analysis.Sources,
                Levels:    analysis.Levels,
                Filter:    analysis.Filter,
                Limit:     0, // No limit for analysis
        }
        
        // Execute the query
        entries, err := e.storage.Query(ctx, query)
        if err != nil {
                return nil, fmt.Errorf("query execution failed: %w", err)
        }
        
        // Process the analysis based on type
        result := &models.AnalysisResult{
                Type:      analysis.Type,
                TimeRange: analysis.TimeRange,
        }
        
        switch analysis.Type {
        case models.AnalysisTypeCount:
                result.Count = int64(len(entries))
                
        case models.AnalysisTypeFrequency:
                if analysis.GroupBy == "" {
                        return nil, fmt.Errorf("frequency analysis requires a GroupBy field")
                }
                result.Frequency = e.calculateFrequency(entries, analysis.GroupBy)
                
        case models.AnalysisTypeTimeSeries:
                result.TimeSeries = e.calculateTimeSeries(entries, analysis.Interval)
                
        case models.AnalysisTypePatterns:
                result.Patterns = e.findPatterns(entries, analysis.PatternConfig)
                
        case models.AnalysisTypeCorrelation:
                if len(analysis.CorrelationFields) < 2 {
                        return nil, fmt.Errorf("correlation analysis requires at least two fields")
                }
                result.Correlation = e.calculateCorrelation(entries, analysis.CorrelationFields)
                
        default:
                return nil, fmt.Errorf("unsupported analysis type: %s", analysis.Type)
        }
        
        return result, nil
}

// ParseQuery implements the QueryEngine interface
func (e *Engine) ParseQuery(queryString string) (models.Query, error) {
        query := models.Query{
                Limit: 100, // Default limit
        }
        
        // Split the query string into tokens
        tokens := strings.Fields(queryString)
        
        for i := 0; i < len(tokens); i++ {
                token := tokens[i]
                
                // Check for special keywords
                switch strings.ToLower(token) {
                case "from":
                        if i+1 < len(tokens) {
                                i++
                                // Skip time parsing for now
                        }
                        
                case "to":
                        if i+1 < len(tokens) {
                                i++
                                // Skip time parsing for now
                        }
                        
                case "source:":
                case "source":
                        if i+1 < len(tokens) {
                                i++
                                sources := strings.Split(tokens[i], ",")
                                query.Sources = append(query.Sources, sources...)
                        }
                        
                case "level:":
                case "level":
                        if i+1 < len(tokens) {
                                i++
                                levels := strings.Split(tokens[i], ",")
                                query.Levels = append(query.Levels, levels...)
                        }
                        
                case "limit:":
                case "limit":
                        if i+1 < len(tokens) {
                                i++
                                limit, err := strconv.Atoi(tokens[i])
                                if err == nil && limit > 0 {
                                        query.Limit = limit
                                }
                        }
                        
                default:
                        // Check if it's a field:value pair
                        if strings.Contains(token, ":") {
                                parts := strings.SplitN(token, ":", 2)
                                if len(parts) == 2 {
                                        if query.FilterFields == nil {
                                                query.FilterFields = make(map[string]string)
                                        }
                                        query.FilterFields[parts[0]] = parts[1]
                                }
                        } else {
                                // Treat as part of general filter
                                if query.Filter != "" {
                                        query.Filter += " "
                                }
                                query.Filter += token
                        }
                }
        }
        
        return query, nil
}

// calculateFrequency calculates frequency distribution
func (e *Engine) calculateFrequency(entries []*models.LogEntry, groupBy string) map[string]int64 {
        frequency := make(map[string]int64)
        
        for _, entry := range entries {
                var value string
                
                switch strings.ToLower(groupBy) {
                case "source":
                        value = entry.Source
                case "level":
                        value = entry.Level
                default:
                        // Try to find in fields
                        if fieldValue, ok := entry.Fields[groupBy]; ok {
                                value = fmt.Sprintf("%v", fieldValue)
                        } else {
                                value = "unknown"
                        }
                }
                
                frequency[value]++
        }
        
        return frequency
}

// calculateTimeSeries creates a time series analysis
func (e *Engine) calculateTimeSeries(entries []*models.LogEntry, interval string) map[string]int64 {
        timeSeries := make(map[string]int64)
        
        // Determine format based on interval
        var format string
        switch strings.ToLower(interval) {
        case "minute":
                format = "2006-01-02 15:04"
        case "hour":
                format = "2006-01-02 15"
        case "day":
                format = "2006-01-02"
        case "month":
                format = "2006-01"
        default:
                format = "2006-01-02" // Default to day
        }
        
        for _, entry := range entries {
                timeKey := entry.Timestamp.Format(format)
                timeSeries[timeKey]++
        }
        
        return timeSeries
}

// findPatterns identifies common patterns in log messages
func (e *Engine) findPatterns(entries []*models.LogEntry, config models.PatternConfig) []models.Pattern {
        // Map to track pattern frequencies
        patternFrequency := make(map[string]int)
        
        // Regex to replace numbers with placeholders
        numberRegex := regexp.MustCompile(`\b\d+\b`)
        
        // Track which original messages are associated with which pattern
        patternExamples := make(map[string][]string)
        
        for _, entry := range entries {
                // Skip empty messages
                if entry.Message == "" {
                        continue
                }
                
                // Create a pattern by replacing variable parts
                pattern := entry.Message
                
                // Replace numbers with a placeholder
                if config.ReplaceNumbers {
                        pattern = numberRegex.ReplaceAllString(pattern, "{number}")
                }
                
                // Replace IDs, IPs, etc. with more sophisticated regex if needed
                // ...
                
                // Update frequency
                patternFrequency[pattern]++
                
                // Store original message as an example (up to 3 examples per pattern)
                if len(patternExamples[pattern]) < 3 {
                        patternExamples[pattern] = append(patternExamples[pattern], entry.Message)
                }
        }
        
        // Convert to result format
        patterns := make([]models.Pattern, 0, len(patternFrequency))
        for pattern, count := range patternFrequency {
                patterns = append(patterns, models.Pattern{
                        Pattern:  pattern,
                        Count:    count,
                        Examples: patternExamples[pattern],
                })
        }
        
        // Sort by frequency (in a real implementation)
        // sort.Slice(patterns, func(i, j int) bool {
        //     return patterns[i].Count > patterns[j].Count
        // })
        
        // Limit to top N patterns
        maxPatterns := 10
        if len(patterns) > maxPatterns {
                patterns = patterns[:maxPatterns]
        }
        
        return patterns
}

// calculateCorrelation finds relationships between fields
func (e *Engine) calculateCorrelation(entries []*models.LogEntry, fields []string) map[string]map[string]int64 {
        correlation := make(map[string]map[string]int64)
        
        // Initialize correlation map
        for _, field1 := range fields {
                correlation[field1] = make(map[string]int64)
        }
        
        // Count occurrences of each value combination
        for _, entry := range entries {
                for _, field1 := range fields {
                        for _, field2 := range fields {
                                if field1 == field2 {
                                        continue
                                }
                                
                                var value2 string
                                
                                // Get value for field2
                                switch strings.ToLower(field2) {
                                case "source":
                                        value2 = entry.Source
                                case "level":
                                        value2 = entry.Level
                                default:
                                        if fieldValue, ok := entry.Fields[field2]; ok {
                                                value2 = fmt.Sprintf("%v", fieldValue)
                                        } else {
                                                value2 = "unknown"
                                        }
                                }
                                
                                key := fmt.Sprintf("%s=%s", field2, value2)
                                correlation[field1][key]++
                        }
                }
        }
        
        return correlation
}

// Note: Using lint directives to mark intentionally unused code for future implementation
// Helper function to parse time
// nolint:unused,deadcode
func parseTime(timeStr string) models.Timestamp {
        // A real implementation would include more sophisticated time parsing
        return models.Timestamp{}
}
