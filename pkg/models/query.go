package models

import (
        "encoding/json"
        "time"
)

// TimeRange represents a time range for filtering log entries
type TimeRange struct {
        From time.Time `json:"from,omitempty"`
        To   time.Time `json:"to,omitempty"`
}

// Query represents a query for filtering and retrieving log entries
type Query struct {
        // TimeRange limits logs to a specific time range
        TimeRange TimeRange `json:"time_range"`
        
        // Sources limits logs to specific sources
        Sources []string `json:"sources,omitempty"`
        
        // Levels limits logs to specific severity levels
        Levels []string `json:"levels,omitempty"`
        
        // Filter is a free-text filter expression
        Filter string `json:"filter,omitempty"`
        
        // FilterFields contains field-specific filters
        FilterFields map[string]string `json:"filter_fields,omitempty"`
        
        // Limit sets the maximum number of results
        Limit int `json:"limit,omitempty"`
        
        // SortBy specifies the field to sort by
        SortBy string `json:"sort_by,omitempty"`
        
        // SortOrder specifies the sort order (asc or desc)
        SortOrder string `json:"sort_order,omitempty"`
}

// NewQuery creates a new query with default values
func NewQuery() Query {
        return Query{
                Limit:        100,
                SortBy:       "timestamp",
                SortOrder:    "desc",
                FilterFields: make(map[string]string),
        }
}

// WithTimeRange sets the time range for the query
func (q Query) WithTimeRange(from, to time.Time) Query {
        q.TimeRange.From = from
        q.TimeRange.To = to
        return q
}

// WithSources sets the sources for the query
func (q Query) WithSources(sources ...string) Query {
        q.Sources = sources
        return q
}

// WithLevels sets the levels for the query
func (q Query) WithLevels(levels ...string) Query {
        q.Levels = levels
        return q
}

// WithFilter sets the filter expression for the query
func (q Query) WithFilter(filter string) Query {
        q.Filter = filter
        return q
}

// WithFilterField adds a field filter to the query
func (q Query) WithFilterField(field, value string) Query {
        if q.FilterFields == nil {
                q.FilterFields = make(map[string]string)
        }
        q.FilterFields[field] = value
        return q
}

// WithLimit sets the maximum number of results
func (q Query) WithLimit(limit int) Query {
        q.Limit = limit
        return q
}

// WithSort sets the sort field and order
func (q Query) WithSort(field, order string) Query {
        q.SortBy = field
        q.SortOrder = order
        return q
}

// Analysis represents parameters for log analysis
type Analysis struct {
        // Type specifies the analysis type
        Type string `json:"type"`
        
        // TimeRange limits logs to a specific time range
        TimeRange TimeRange `json:"time_range"`
        
        // Sources limits logs to specific sources
        Sources []string `json:"sources,omitempty"`
        
        // Levels limits logs to specific severity levels
        Levels []string `json:"levels,omitempty"`
        
        // Filter is a free-text filter expression
        Filter string `json:"filter,omitempty"`
        
        // GroupBy specifies the field to group by for frequency analysis
        GroupBy string `json:"group_by,omitempty"`
        
        // Interval specifies the time interval for time series analysis
        Interval string `json:"interval,omitempty"`
        
        // PatternConfig contains configuration for pattern analysis
        PatternConfig PatternConfig `json:"pattern_config,omitempty"`
        
        // CorrelationFields specifies fields to correlate
        CorrelationFields []string `json:"correlation_fields,omitempty"`
}

// PatternConfig contains configuration for pattern analysis
type PatternConfig struct {
        ReplaceNumbers bool `json:"replace_numbers"`
        ReplaceIPs     bool `json:"replace_ips"`
        ReplaceUUIDs   bool `json:"replace_uuids"`
        MinCount       int  `json:"min_count"`
}

// AnalysisResult contains the results of a log analysis
type AnalysisResult struct {
        // Type specifies the analysis type
        Type string `json:"type"`
        
        // TimeRange shows the time range that was analyzed
        TimeRange TimeRange `json:"time_range"`
        
        // Count is the total number of matching entries
        Count int64 `json:"count,omitempty"`
        
        // Frequency contains frequency distribution data
        Frequency map[string]int64 `json:"frequency,omitempty"`
        
        // TimeSeries contains time series data
        TimeSeries map[string]int64 `json:"time_series,omitempty"`
        
        // Patterns contains pattern analysis results
        Patterns []Pattern `json:"patterns,omitempty"`
        
        // Correlation contains correlation analysis results
        Correlation map[string]map[string]int64 `json:"correlation,omitempty"`
}

// Pattern represents a message pattern found in logs
type Pattern struct {
        // Pattern is the pattern template
        Pattern string `json:"pattern"`
        
        // Count is the number of messages matching this pattern
        Count int `json:"count"`
        
        // Examples contains example messages that match the pattern
        Examples []string `json:"examples,omitempty"`
}

// Analysis types
const (
        AnalysisTypeCount       = "count"
        AnalysisTypeFrequency   = "frequency"
        AnalysisTypeTimeSeries  = "time_series"
        AnalysisTypePatterns    = "patterns"
        AnalysisTypeCorrelation = "correlation"
)

// MarshalJSON implements custom JSON marshaling for TimeRange
func (tr TimeRange) MarshalJSON() ([]byte, error) {
        // We don't need the actual Alias type but just the struct
        return json.Marshal(&struct {
                From string `json:"from,omitempty"`
                To   string `json:"to,omitempty"`
        }{
                From: func() string {
                        if tr.From.IsZero() {
                                return ""
                        }
                        return tr.From.Format(time.RFC3339)
                }(),
                To: func() string {
                        if tr.To.IsZero() {
                                return ""
                        }
                        return tr.To.Format(time.RFC3339)
                }(),
        })
}

// UnmarshalJSON implements custom JSON unmarshaling for TimeRange
func (tr *TimeRange) UnmarshalJSON(data []byte) error {
        // Just use a simple struct without the Alias type
        aux := &struct {
                From string `json:"from"`
                To   string `json:"to"`
        }{}
        
        if err := json.Unmarshal(data, &aux); err != nil {
                return err
        }
        
        if aux.From != "" {
                parsedTime, err := time.Parse(time.RFC3339, aux.From)
                if err != nil {
                        return err
                }
                tr.From = parsedTime
        }
        
        if aux.To != "" {
                parsedTime, err := time.Parse(time.RFC3339, aux.To)
                if err != nil {
                        return err
                }
                tr.To = parsedTime
        }
        
        return nil
}
