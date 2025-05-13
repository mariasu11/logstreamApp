package processor

import (
        "fmt"
        "regexp"
        "strings"
        "time"

        "github.com/mariasu11/logstreamApp/pkg/models"
)

// Transformer defines the interface for transforming log entries
type Transformer interface {
        // Transform modifies the log entry in place
        Transform(entry *models.LogEntry)
}

// AddFieldTransformer adds a field to log entries
type AddFieldTransformer struct {
        FieldName  string
        FieldValue interface{}
}

// NewAddFieldTransformer creates a new transformer to add a field
func NewAddFieldTransformer(name string, value interface{}) *AddFieldTransformer {
        return &AddFieldTransformer{
                FieldName:  name,
                FieldValue: value,
        }
}

// Transform implements the Transformer interface
func (t *AddFieldTransformer) Transform(entry *models.LogEntry) {
        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }
        entry.Fields[t.FieldName] = t.FieldValue
}

// RemoveFieldTransformer removes a field from log entries
type RemoveFieldTransformer struct {
        FieldName string
}

// NewRemoveFieldTransformer creates a new transformer to remove a field
func NewRemoveFieldTransformer(name string) *RemoveFieldTransformer {
        return &RemoveFieldTransformer{
                FieldName: name,
        }
}

// Transform implements the Transformer interface
func (t *RemoveFieldTransformer) Transform(entry *models.LogEntry) {
        if entry.Fields != nil {
                delete(entry.Fields, t.FieldName)
        }
}

// RenameFieldTransformer renames a field in log entries
type RenameFieldTransformer struct {
        OldName string
        NewName string
}

// NewRenameFieldTransformer creates a new transformer to rename a field
func NewRenameFieldTransformer(oldName, newName string) *RenameFieldTransformer {
        return &RenameFieldTransformer{
                OldName: oldName,
                NewName: newName,
        }
}

// Transform implements the Transformer interface
func (t *RenameFieldTransformer) Transform(entry *models.LogEntry) {
        if entry.Fields == nil {
                return
        }
        
        if value, exists := entry.Fields[t.OldName]; exists {
                entry.Fields[t.NewName] = value
                delete(entry.Fields, t.OldName)
        }
}

// RegexExtractTransformer extracts field values using regex
type RegexExtractTransformer struct {
        Pattern *regexp.Regexp
        Fields  []string
}

// NewRegexExtractTransformer creates a new transformer that extracts fields using a regex
func NewRegexExtractTransformer(pattern string, fields []string) (*RegexExtractTransformer, error) {
        re, err := regexp.Compile(pattern)
        if err != nil {
                return nil, err
        }

        return &RegexExtractTransformer{
                Pattern: re,
                Fields:  fields,
        }, nil
}

// Transform implements the Transformer interface
func (t *RegexExtractTransformer) Transform(entry *models.LogEntry) {
        matches := t.Pattern.FindStringSubmatch(entry.Message)
        if matches == nil || len(matches) <= 1 {
                return
        }

        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }

        // Start at index 1 to skip the full match
        for i, name := range t.Fields {
                if i+1 < len(matches) {
                        entry.Fields[name] = matches[i+1]
                }
        }
}

// TimestampFormatTransformer formats the timestamp of log entries
type TimestampFormatTransformer struct {
        Format string
}

// NewTimestampFormatTransformer creates a new transformer that formats timestamps
func NewTimestampFormatTransformer(format string) *TimestampFormatTransformer {
        return &TimestampFormatTransformer{
                Format: format,
        }
}

// Transform implements the Transformer interface
func (t *TimestampFormatTransformer) Transform(entry *models.LogEntry) {
        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }
        
        entry.Fields["formatted_timestamp"] = entry.Timestamp.Format(t.Format)
}

// EnrichIPTransformer enriches IP addresses with location data
type EnrichIPTransformer struct {
        IPFieldName string
}

// NewEnrichIPTransformer creates a new transformer that enriches IP addresses
func NewEnrichIPTransformer(fieldName string) *EnrichIPTransformer {
        return &EnrichIPTransformer{
                IPFieldName: fieldName,
        }
}

// Transform implements the Transformer interface
func (t *EnrichIPTransformer) Transform(entry *models.LogEntry) {
        if entry.Fields == nil {
                return
        }

        ipVal, exists := entry.Fields[t.IPFieldName]
        if !exists {
                return
        }

        // Check if the IP value is a string and return if not
        if _, ok := ipVal.(string); !ok {
                return
        }

        // In a real implementation, this would call a geolocation service with the IP string
        // Here we just add placeholder data
        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }
        
        // Add geo fields with placeholder data (in a real implementation, this would be real data)
        entry.Fields["ip_geo_country"] = "Unknown"
        entry.Fields["ip_geo_city"] = "Unknown"
        entry.Fields["ip_geo_coordinates"] = "0,0"
}

// MessageFormatTransformer formats the log message
type MessageFormatTransformer struct {
        Template string
}

// NewMessageFormatTransformer creates a new transformer that formats the message
func NewMessageFormatTransformer(template string) *MessageFormatTransformer {
        return &MessageFormatTransformer{
                Template: template,
        }
}

// Transform implements the Transformer interface
func (t *MessageFormatTransformer) Transform(entry *models.LogEntry) {
        if entry.Fields == nil {
                return
        }

        // Replace placeholders in template with field values
        message := t.Template
        for key, value := range entry.Fields {
                placeholder := fmt.Sprintf("{%s}", key)
                if strings.Contains(message, placeholder) {
                        message = strings.ReplaceAll(message, placeholder, fmt.Sprintf("%v", value))
                }
        }

        // Replace timestamp placeholder
        if strings.Contains(message, "{timestamp}") {
                message = strings.ReplaceAll(message, "{timestamp}", entry.Timestamp.Format(time.RFC3339))
        }

        // Replace source placeholder
        if strings.Contains(message, "{source}") {
                message = strings.ReplaceAll(message, "{source}", entry.Source)
        }

        // Replace level placeholder
        if strings.Contains(message, "{level}") {
                message = strings.ReplaceAll(message, "{level}", entry.Level)
        }

        entry.Message = message
}

// CompositeTransformer applies multiple transformers in sequence
type CompositeTransformer struct {
        Transformers []Transformer
}

// NewCompositeTransformer creates a new composite transformer
func NewCompositeTransformer(transformers ...Transformer) *CompositeTransformer {
        return &CompositeTransformer{
                Transformers: transformers,
        }
}

// Transform implements the Transformer interface
func (t *CompositeTransformer) Transform(entry *models.LogEntry) {
        for _, transformer := range t.Transformers {
                transformer.Transform(entry)
        }
}
