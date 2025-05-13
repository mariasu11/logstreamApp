package tests

import (
        "context"
        "testing"
        "time"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/mock"
        "github.com/stretchr/testify/require"

        "github.com/mariasu11/logstream/internal/query"
        "github.com/mariasu11/logstream/internal/storage"
        "github.com/mariasu11/logstream/pkg/models"
)

// MockStorage implements a mock storage.Storage interface for testing
type MockStorage struct {
        mock.Mock
}

func (m *MockStorage) Store(ctx context.Context, entry *models.LogEntry) error {
        args := m.Called(ctx, entry)
        return args.Error(0)
}

func (m *MockStorage) Query(ctx context.Context, query models.Query) ([]*models.LogEntry, error) {
        args := m.Called(ctx, query)
        return args.Get(0).([]*models.LogEntry), args.Error(1)
}

func (m *MockStorage) GetStats(ctx context.Context) (storage.StorageStats, error) {
        args := m.Called(ctx)
        return args.Get(0).(storage.StorageStats), args.Error(1)
}

func (m *MockStorage) GetSources(ctx context.Context) ([]string, error) {
        args := m.Called(ctx)
        return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) GetLevels(ctx context.Context) ([]string, error) {
        args := m.Called(ctx)
        return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) Close() error {
        args := m.Called()
        return args.Error(0)
}

func TestQueryEngineExecute(t *testing.T) {
        // Create mock storage
        mockStorage := new(MockStorage)
        
        // Create test log entries
        now := time.Now()
        testEntries := []*models.LogEntry{
                {
                        Timestamp: now.Add(-5 * time.Minute),
                        Source:    "app1",
                        Level:     "info",
                        Message:   "Test message 1",
                },
                {
                        Timestamp: now.Add(-3 * time.Minute),
                        Source:    "app1",
                        Level:     "error",
                        Message:   "Test error message",
                },
                {
                        Timestamp: now.Add(-1 * time.Minute),
                        Source:    "app2",
                        Level:     "info",
                        Message:   "Test message 2",
                },
        }
        
        // Create test query
        testQuery := models.Query{
                Sources: []string{"app1"},
                Limit:   10,
        }
        
        // Set up mock expectations
        mockStorage.On("Query", mock.Anything, testQuery).Return(testEntries[:2], nil)
        
        // Create query engine
        engine := query.NewEngine(mockStorage)
        
        // Execute query
        result, err := engine.Execute(testQuery)
        
        // Verify results
        require.NoError(t, err)
        assert.Equal(t, 2, len(result))
        assert.Equal(t, "app1", result[0].Source)
        assert.Equal(t, "app1", result[1].Source)
        
        // Verify expectations
        mockStorage.AssertExpectations(t)
}

func TestQueryEngineAnalyze(t *testing.T) {
        // Create mock storage
        mockStorage := new(MockStorage)
        
        // Create test log entries
        now := time.Now()
        testEntries := []*models.LogEntry{
                {
                        Timestamp: now.Add(-5 * time.Minute),
                        Source:    "app1",
                        Level:     "info",
                        Message:   "Test message 1",
                },
                {
                        Timestamp: now.Add(-3 * time.Minute),
                        Source:    "app1",
                        Level:     "error",
                        Message:   "Test error message",
                },
                {
                        Timestamp: now.Add(-1 * time.Minute),
                        Source:    "app2",
                        Level:     "info",
                        Message:   "Test message 2",
                },
        }
        
        // Test frequency analysis
        t.Run("FrequencyAnalysis", func(t *testing.T) {
                // Create analysis request
                analysis := models.Analysis{
                        Type:    models.AnalysisTypeFrequency,
                        GroupBy: "level",
                }
                
                // Set up mock expectations - match any query derived from the analysis
                mockStorage.On("Query", mock.Anything, mock.AnythingOfType("models.Query")).Return(testEntries, nil).Once()
                
                // Create query engine
                engine := query.NewEngine(mockStorage)
                
                // Execute analysis
                result, err := engine.Analyze(analysis)
                
                // Verify results
                require.NoError(t, err)
                assert.Equal(t, models.AnalysisTypeFrequency, result.Type)
                assert.Equal(t, int64(2), result.Frequency["info"])
                assert.Equal(t, int64(1), result.Frequency["error"])
                
                // Verify expectations
                mockStorage.AssertExpectations(t)
        })
        
        // Test count analysis
        t.Run("CountAnalysis", func(t *testing.T) {
                // Create analysis request
                analysis := models.Analysis{
                        Type: models.AnalysisTypeCount,
                }
                
                // Reset mock
                mockStorage = new(MockStorage)
                
                // Set up mock expectations
                mockStorage.On("Query", mock.Anything, mock.AnythingOfType("models.Query")).Return(testEntries, nil).Once()
                
                // Create query engine
                engine := query.NewEngine(mockStorage)
                
                // Execute analysis
                result, err := engine.Analyze(analysis)
                
                // Verify results
                require.NoError(t, err)
                assert.Equal(t, models.AnalysisTypeCount, result.Type)
                assert.Equal(t, int64(3), result.Count)
                
                // Verify expectations
                mockStorage.AssertExpectations(t)
        })
        
        // Test time series analysis
        t.Run("TimeSeriesAnalysis", func(t *testing.T) {
                // Create analysis request
                analysis := models.Analysis{
                        Type:     models.AnalysisTypeTimeSeries,
                        Interval: "hour",
                }
                
                // Reset mock
                mockStorage = new(MockStorage)
                
                // Set up mock expectations
                mockStorage.On("Query", mock.Anything, mock.AnythingOfType("models.Query")).Return(testEntries, nil).Once()
                
                // Create query engine
                engine := query.NewEngine(mockStorage)
                
                // Execute analysis
                result, err := engine.Analyze(analysis)
                
                // Verify results
                require.NoError(t, err)
                assert.Equal(t, models.AnalysisTypeTimeSeries, result.Type)
                
                // Check that the time series contains data
                assert.NotEmpty(t, result.TimeSeries)
                
                // The time series should have entries bucketed by hour
                // But we don't want to hard-code the exact format since the test log entries
                // all have the same timestamp and will end up in the same bucket
                var total int64
                for _, count := range result.TimeSeries {
                        total += count
                }
                assert.Equal(t, int64(3), total)
                
                // Verify expectations
                mockStorage.AssertExpectations(t)
        })
        
        // Test pattern analysis
        t.Run("PatternAnalysis", func(t *testing.T) {
                // Create analysis request
                analysis := models.Analysis{
                        Type: models.AnalysisTypePatterns,
                        PatternConfig: models.PatternConfig{
                                ReplaceNumbers: true,
                        },
                }
                
                // Create test entries with patterns
                patternEntries := []*models.LogEntry{
                        {
                                Timestamp: now,
                                Source:    "app1",
                                Level:     "info",
                                Message:   "User 123 logged in",
                        },
                        {
                                Timestamp: now,
                                Source:    "app1",
                                Level:     "info",
                                Message:   "User 456 logged in",
                        },
                        {
                                Timestamp: now,
                                Source:    "app1",
                                Level:     "error",
                                Message:   "Failed to process request 789",
                        },
                }
                
                // Reset mock
                mockStorage = new(MockStorage)
                
                // Set up mock expectations
                mockStorage.On("Query", mock.Anything, mock.AnythingOfType("models.Query")).Return(patternEntries, nil).Once()
                
                // Create query engine
                engine := query.NewEngine(mockStorage)
                
                // Execute analysis
                result, err := engine.Analyze(analysis)
                
                // Verify results
                require.NoError(t, err)
                assert.Equal(t, models.AnalysisTypePatterns, result.Type)
                
                // Check that we have the patterns with numbers replaced
                foundUserPattern := false
                foundErrorPattern := false
                
                for _, pattern := range result.Patterns {
                        if pattern.Pattern == "User {number} logged in" {
                                foundUserPattern = true
                                assert.Equal(t, 2, pattern.Count)
                                assert.Len(t, pattern.Examples, 2)
                        }
                        if pattern.Pattern == "Failed to process request {number}" {
                                foundErrorPattern = true
                                assert.Equal(t, 1, pattern.Count)
                                assert.Len(t, pattern.Examples, 1)
                        }
                }
                
                assert.True(t, foundUserPattern, "User pattern not found")
                assert.True(t, foundErrorPattern, "Error pattern not found")
                
                // Verify expectations
                mockStorage.AssertExpectations(t)
        })
}

func TestParseQuery(t *testing.T) {
        // Create mock storage
        mockStorage := new(MockStorage)
        
        // Create query engine
        engine := query.NewEngine(mockStorage)
        
        // Test cases
        testCases := []struct {
                name        string
                queryString string
                expected    models.Query
        }{
                {
                        name:        "Empty Query",
                        queryString: "",
                        expected: models.Query{
                                Limit: 100, // Default limit
                        },
                },
                {
                        name:        "Source Filter",
                        queryString: "source app1",
                        expected: models.Query{
                                Sources: []string{"app1"},
                                Limit:   100,
                        },
                },
                {
                        name:        "Level Filter",
                        queryString: "level error",
                        expected: models.Query{
                                Levels: []string{"error"},
                                Limit:  100,
                        },
                },
                {
                        name:        "Multiple Sources",
                        queryString: "source app1,app2",
                        expected: models.Query{
                                Sources: []string{"app1", "app2"},
                                Limit:   100,
                        },
                },
                {
                        name:        "Text Filter",
                        queryString: "error database",
                        expected: models.Query{
                                Filter: "error database",
                                Limit:  100,
                        },
                },
                {
                        name:        "Field Filter",
                        queryString: "status:500",
                        expected: models.Query{
                                FilterFields: map[string]string{"status": "500"},
                                Limit:        100,
                        },
                },
                {
                        name:        "Combined Filters",
                        queryString: "source app1 level error connection",
                        expected: models.Query{
                                Sources: []string{"app1"},
                                Levels:  []string{"error"},
                                Filter:  "connection",
                                Limit:   100,
                        },
                },
                {
                        name:        "With Limit",
                        queryString: "source app1 limit 50",
                        expected: models.Query{
                                Sources: []string{"app1"},
                                Limit:   50,
                        },
                },
        }
        
        for _, tc := range testCases {
                t.Run(tc.name, func(t *testing.T) {
                        result, err := engine.ParseQuery(tc.queryString)
                        
                        require.NoError(t, err)
                        
                        // Check specific fields that we expect
                        if len(tc.expected.Sources) > 0 {
                                assert.Equal(t, tc.expected.Sources, result.Sources)
                        }
                        if len(tc.expected.Levels) > 0 {
                                assert.Equal(t, tc.expected.Levels, result.Levels)
                        }
                        if tc.expected.Filter != "" {
                                assert.Equal(t, tc.expected.Filter, result.Filter)
                        }
                        if tc.expected.FilterFields != nil {
                                for k, v := range tc.expected.FilterFields {
                                        assert.Equal(t, v, result.FilterFields[k])
                                }
                        }
                        assert.Equal(t, tc.expected.Limit, result.Limit)
                })
        }
}