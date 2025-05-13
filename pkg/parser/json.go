package parser

import (
        "encoding/json"
        "strings"
        "time"

        "github.com/mariasu11/logstreamApp/pkg/models"
)

// JSONParser is a parser for JSON-formatted logs
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
        return &JSONParser{}
}

// Name returns the parser name
func (p *JSONParser) Name() string {
        return "json"
}

// CanParse checks if the given log line is in JSON format
func (p *JSONParser) CanParse(raw string) bool {
        trimmed := strings.TrimSpace(raw)
        if len(trimmed) == 0 {
            return false
        }
        return trimmed[0] == '{'
}

// Parse parses a JSON log entry
func (p *JSONParser) Parse(entry *models.LogEntry) error {
        // Parse the JSON
        var jsonData map[string]interface{}
        if err := json.Unmarshal([]byte(entry.RawData), &jsonData); err != nil {
                return err
        }
        
        // Extract common fields
        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }
        
        // Process each field
        for key, value := range jsonData {
                switch strings.ToLower(key) {
                case "timestamp", "time", "@timestamp", "date":
                        p.parseTimestamp(entry, value)
                case "message", "msg":
                        if strVal, ok := value.(string); ok {
                                entry.Message = strVal
                        }
                case "level", "severity", "loglevel":
                        if strVal, ok := value.(string); ok {
                                entry.Level = strVal
                        }
                case "source", "logger", "origin":
                        if strVal, ok := value.(string); ok {
                                entry.Source = strVal
                        }
                default:
                        // Add all other fields to the Fields map
                        entry.Fields[key] = value
                }
        }
        
        // If no message was found, create one from the JSON
        if entry.Message == "" {
                bytes, _ := json.Marshal(jsonData)
                entry.Message = string(bytes)
        }
        
        return nil
}

// parseTimestamp attempts to parse a timestamp value
func (p *JSONParser) parseTimestamp(entry *models.LogEntry, value interface{}) {
        switch v := value.(type) {
        case string:
                // Try common time formats
                formats := []string{
                        time.RFC3339,
                        time.RFC3339Nano,
                        "2006-01-02T15:04:05",
                        "2006-01-02 15:04:05",
                        "2006/01/02 15:04:05",
                        time.UnixDate,
                        time.RubyDate,
                }
                
                for _, format := range formats {
                        if t, err := time.Parse(format, v); err == nil {
                                entry.Timestamp = t
                                return
                        }
                }
        case float64:
                // Assume Unix timestamp in seconds or milliseconds
                if v > 1e12 {
                        // Milliseconds
                        entry.Timestamp = time.Unix(0, int64(v)*int64(time.Millisecond))
                } else {
                        // Seconds
                        entry.Timestamp = time.Unix(int64(v), 0)
                }
        case int64:
                // Assume Unix timestamp in seconds or milliseconds
                if v > 1e12 {
                        // Milliseconds
                        entry.Timestamp = time.Unix(0, v*int64(time.Millisecond))
                } else {
                        // Seconds
                        entry.Timestamp = time.Unix(v, 0)
                }
        }
}

// JSONStructuredParser parses specific JSON log formats like logrus, zap, etc.
type JSONStructuredParser struct {
        format string // Format identifier: "logrus", "zap", etc.
}

// NewJSONStructuredParser creates a new structured JSON parser
func NewJSONStructuredParser(format string) *JSONStructuredParser {
        return &JSONStructuredParser{
                format: format,
        }
}

// Name returns the parser name
func (p *JSONStructuredParser) Name() string {
        return "json_" + p.format
}

// CanParse checks if the given log line is in the expected JSON format
func (p *JSONStructuredParser) CanParse(raw string) bool {
        trimmed := strings.TrimSpace(raw)
        if len(trimmed) == 0 || trimmed[0] != '{' {
                return false
        }
        
        // For specific format detection, look for format-specific fields
        switch p.format {
        case "logrus":
                return strings.Contains(raw, "\"level\"") && strings.Contains(raw, "\"msg\"")
        case "zap":
                return strings.Contains(raw, "\"level\"") && strings.Contains(raw, "\"msg\"")
        case "hclog":
                return strings.Contains(raw, "\"@level\"") && strings.Contains(raw, "\"@message\"")
        default:
                return false
        }
}

// Parse parses a structured JSON log entry
func (p *JSONStructuredParser) Parse(entry *models.LogEntry) error {
        // Parse the basic JSON first
        var jsonData map[string]interface{}
        if err := json.Unmarshal([]byte(entry.RawData), &jsonData); err != nil {
                return err
        }
        
        // Initialize fields if necessary
        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }
        
        // Process based on the format
        switch p.format {
        case "logrus":
                return p.parseLogrus(entry, jsonData)
        case "zap":
                return p.parseZap(entry, jsonData)
        case "hclog":
                return p.parseHCLog(entry, jsonData)
        default:
                // Fall back to generic JSON parsing
                return NewJSONParser().Parse(entry)
        }
}

// parseLogrus parses a logrus-formatted JSON log
func (p *JSONStructuredParser) parseLogrus(entry *models.LogEntry, data map[string]interface{}) error {
        // Extract standard logrus fields
        if msg, ok := data["msg"].(string); ok {
                entry.Message = msg
                delete(data, "msg")
        }
        
        if level, ok := data["level"].(string); ok {
                entry.Level = level
                delete(data, "level")
        }
        
        if timeStr, ok := data["time"].(string); ok {
                if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
                        entry.Timestamp = t
                }
                delete(data, "time")
        }
        
        // Add remaining fields
        for k, v := range data {
                entry.Fields[k] = v
        }
        
        return nil
}

// parseZap parses a zap-formatted JSON log
func (p *JSONStructuredParser) parseZap(entry *models.LogEntry, data map[string]interface{}) error {
        // Extract standard zap fields
        if msg, ok := data["msg"].(string); ok {
                entry.Message = msg
                delete(data, "msg")
        }
        
        if level, ok := data["level"].(string); ok {
                entry.Level = level
                delete(data, "level")
        }
        
        if ts, ok := data["ts"].(float64); ok {
                // Zap timestamps are often Unix seconds with fractional part
                secs := int64(ts)
                nsecs := int64((ts - float64(secs)) * 1e9)
                entry.Timestamp = time.Unix(secs, nsecs)
                delete(data, "ts")
        }
        
        // Add remaining fields
        for k, v := range data {
                entry.Fields[k] = v
        }
        
        return nil
}

// parseHCLog parses a HashiCorp HCLog-formatted JSON log
func (p *JSONStructuredParser) parseHCLog(entry *models.LogEntry, data map[string]interface{}) error {
        // Extract standard hclog fields
        if msg, ok := data["@message"].(string); ok {
                entry.Message = msg
                delete(data, "@message")
        }
        
        if level, ok := data["@level"].(string); ok {
                entry.Level = level
                delete(data, "@level")
        }
        
        if timestamp, ok := data["@timestamp"].(string); ok {
                if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
                        entry.Timestamp = t
                }
                delete(data, "@timestamp")
        }
        
        if module, ok := data["@module"].(string); ok {
                entry.Source = module
                delete(data, "@module")
        }
        
        // Add remaining fields
        for k, v := range data {
                entry.Fields[k] = v
        }
        
        return nil
}
