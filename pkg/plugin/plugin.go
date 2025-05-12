package plugin

import (
	"github.com/yourusername/logstream/pkg/models"
)

// Plugin defines the interface for LogStream plugins
type Plugin interface {
	// Init initializes the plugin with configuration
	Init(config map[string]string) error
	
	// Name returns the plugin name
	Name() string
	
	// Description returns the plugin description
	Description() string
	
	// Version returns the plugin version
	Version() string
	
	// ProcessLogEntry processes a log entry
	ProcessLogEntry(entry *models.LogEntry) error
	
	// Close performs cleanup when plugin is unloaded
	Close() error
}

// PluginInfo provides metadata about a plugin
type PluginInfo struct {
	Name        string
	Description string
	Version     string
	Author      string
}

// BasePlugin provides a common base for plugin implementations
type BasePlugin struct {
	info   PluginInfo
	config map[string]string
}

// NewBasePlugin creates a new base plugin with the given info
func NewBasePlugin(info PluginInfo) BasePlugin {
	return BasePlugin{
		info:   info,
		config: make(map[string]string),
	}
}

// Init initializes the base plugin
func (p *BasePlugin) Init(config map[string]string) error {
	p.config = config
	return nil
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.info.Name
}

// Description returns the plugin description
func (p *BasePlugin) Description() string {
	return p.info.Description
}

// Version returns the plugin version
func (p *BasePlugin) Version() string {
	return p.info.Version
}

// GetConfig returns a configuration value
func (p *BasePlugin) GetConfig(key string) string {
	return p.config[key]
}

// GetConfigWithDefault returns a configuration value with a default
func (p *BasePlugin) GetConfigWithDefault(key, defaultValue string) string {
	if value, exists := p.config[key]; exists {
		return value
	}
	return defaultValue
}

// Close performs cleanup
func (p *BasePlugin) Close() error {
	return nil
}

// PluginFactory is a function that creates a plugin instance
type PluginFactory func() Plugin

// ProcessingPlugin is a plugin that processes log entries
type ProcessingPlugin interface {
	Plugin
	// Process applies processing to a log entry
	Process(entry *models.LogEntry) error
}

// OutputPlugin is a plugin that outputs log entries to external systems
type OutputPlugin interface {
	Plugin
	// Output sends log entries to an external system
	Output(entries []*models.LogEntry) error
}

// InputPlugin is a plugin that provides log entries from external sources
type InputPlugin interface {
	Plugin
	// Start begins collecting logs
	Start() error
	// Stop stops collecting logs
	Stop() error
}
