package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// LogEntry represents a single log message with associated metadata
type LogEntry struct {
	// Timestamp is the time when the log entry was created
	Timestamp time.Time `json:"timestamp"`
	
	// Source identifies where the log came from
	Source string `json:"source"`
	
	// Level is the severity level (info, warn, error, etc.)
	Level string `json:"level,omitempty"`
	
	// Message is the main log message
	Message string `json:"message"`
	
	// Fields contains additional structured data fields
	Fields map[string]interface{} `json:"fields,omitempty"`
	
	// RawData contains the original unparsed log entry
	RawData string `json:"-"`
}

// NewLogEntry creates a new log entry with the current timestamp
func NewLogEntry(source, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now(),
		Source:    source,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}
}

// Clone creates a deep copy of the log entry
func (e *LogEntry) Clone() *LogEntry {
	clone := &LogEntry{
		Timestamp: e.Timestamp,
		Source:    e.Source,
		Level:     e.Level,
		Message:   e.Message,
		RawData:   e.RawData,
	}
	
	// Copy fields map
	if e.Fields != nil {
		clone.Fields = make(map[string]interface{}, len(e.Fields))
		for k, v := range e.Fields {
			clone.Fields[k] = v
		}
	}
	
	return clone
}

// AddField adds a field to the log entry
func (e *LogEntry) AddField(key string, value interface{}) *LogEntry {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// GetField retrieves a field from the log entry, returning the value and whether it exists
func (e *LogEntry) GetField(key string) (interface{}, bool) {
	if e.Fields == nil {
		return nil, false
	}
	value, exists := e.Fields[key]
	return value, exists
}

// GetStringField gets a field as a string
func (e *LogEntry) GetStringField(key string) (string, bool) {
	if value, exists := e.GetField(key); exists {
		if strValue, ok := value.(string); ok {
			return strValue, true
		}
		// Try to convert to string
		return fmt.Sprintf("%v", value), true
	}
	return "", false
}

// SetLevel sets the log level
func (e *LogEntry) SetLevel(level string) *LogEntry {
	e.Level = level
	return e
}

// String returns a string representation of the log entry
func (e *LogEntry) String() string {
	return fmt.Sprintf("[%s] %s: %s", e.Timestamp.Format(time.RFC3339), e.Source, e.Message)
}

// ToJSON converts the log entry to a JSON string
func (e *LogEntry) ToJSON() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// MustToJSON converts the log entry to a JSON string, panicking on error
func (e *LogEntry) MustToJSON() string {
	bytes, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// Timestamp is a type alias for time.Time with custom JSON marshaling
type Timestamp time.Time

// MarshalJSON implements json.Marshaler
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

// UnmarshalJSON implements json.Unmarshaler
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var timeStr string
	if err := json.Unmarshal(data, &timeStr); err != nil {
		return err
	}
	
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return err
	}
	
	*t = Timestamp(parsedTime)
	return nil
}
