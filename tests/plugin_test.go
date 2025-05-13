package tests

import (
        "fmt"
        "io/ioutil"
        "testing"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
        "github.com/hashicorp/go-hclog"

        "github.com/mariasu11/logstream/pkg/models"
        "github.com/mariasu11/logstream/pkg/plugin"
)

// MockPlugin is a test implementation of the Plugin interface
type MockPlugin struct {
        plugin.BasePlugin
        processCount int
        lastEntry    *models.LogEntry
        shouldError  bool
}

// NewMockPlugin creates a new mock plugin
func NewMockPlugin() *MockPlugin {
        return &MockPlugin{
                BasePlugin: plugin.NewBasePlugin(plugin.PluginInfo{
                        Name:        "mock",
                        Description: "Mock plugin for testing",
                        Version:     "1.0.0",
                        Author:      "LogStream Team",
                }),
                processCount: 0,
        }
}

// ProcessLogEntry implements the Plugin interface
func (p *MockPlugin) ProcessLogEntry(entry *models.LogEntry) error {
        p.processCount++
        p.lastEntry = entry.Clone()
        
        // Add a field to indicate the plugin processed this entry
        if entry.Fields == nil {
                entry.Fields = make(map[string]interface{})
        }
        entry.Fields["processed_by"] = p.Name()
        
        if p.shouldError {
                return fmt.Errorf("mock plugin error")
        }
        return nil
}

func TestPluginRegistry(t *testing.T) {
        // Create a logger that discards output
        logger := hclog.New(&hclog.LoggerOptions{
                Output: ioutil.Discard,
                Level:  hclog.Debug,
        })

        // Create registry
        registry := plugin.NewRegistry(logger)

        // Create and register a mock plugin
        mockPlugin := NewMockPlugin()
        err := registry.RegisterPlugin(mockPlugin)
        require.NoError(t, err)

        // Test plugin retrieval
        retrieved, err := registry.GetPlugin("mock")
        require.NoError(t, err)
        assert.Equal(t, "mock", retrieved.Name())
        assert.Equal(t, "1.0.0", retrieved.Version())

        // Test plugin configuration
        config := map[string]string{
                "setting1": "value1",
                "setting2": "value2",
        }
        err = registry.ConfigurePlugin("mock", config)
        require.NoError(t, err)
        
        // Verify config was stored
        assert.Equal(t, "value1", mockPlugin.GetConfig("setting1"))
        assert.Equal(t, "value2", mockPlugin.GetConfig("setting2"))
        assert.Equal(t, "default", mockPlugin.GetConfigWithDefault("nonexistent", "default"))

        // Test plugin list
        plugins := registry.ListPlugins()
        assert.Equal(t, 1, len(plugins))
        assert.Equal(t, "mock", plugins[0].Name)
}

func TestPluginProcessing(t *testing.T) {
        // Create a mock plugin
        mockPlugin := NewMockPlugin()

        // Create a test log entry
        entry := &models.LogEntry{
                Source:  "test",
                Level:   "info",
                Message: "Test message",
                Fields: map[string]interface{}{
                        "field1": "value1",
                },
        }

        // Process the entry
        err := mockPlugin.ProcessLogEntry(entry)
        require.NoError(t, err)

        // Verify the plugin processed the entry
        assert.Equal(t, 1, mockPlugin.processCount)
        assert.Equal(t, "Test message", mockPlugin.lastEntry.Message)
        assert.Equal(t, "mock", entry.Fields["processed_by"])
}

func TestPluginChain(t *testing.T) {
        // Create multiple plugins
        plugin1 := NewMockPlugin()
        plugin2 := NewMockPlugin()
        plugin3 := NewMockPlugin()

        // Create a test log entry
        entry := &models.LogEntry{
                Source:  "test",
                Level:   "info",
                Message: "Test message",
        }

        // Process the entry through the plugin chain
        err := plugin1.ProcessLogEntry(entry)
        require.NoError(t, err)
        
        err = plugin2.ProcessLogEntry(entry)
        require.NoError(t, err)
        
        err = plugin3.ProcessLogEntry(entry)
        require.NoError(t, err)

        // Verify all plugins processed the entry
        assert.Equal(t, 1, plugin1.processCount)
        assert.Equal(t, 1, plugin2.processCount)
        assert.Equal(t, 1, plugin3.processCount)
        
        // Verify the plugins added their markers
        assert.Equal(t, "mock", entry.Fields["processed_by"])
}

func TestPluginError(t *testing.T) {
        // Create a plugin that will return an error
        mockPlugin := NewMockPlugin()
        mockPlugin.shouldError = true

        // Create a test log entry
        entry := &models.LogEntry{
                Source:  "test",
                Level:   "info",
                Message: "Test message",
        }

        // Process the entry
        err := mockPlugin.ProcessLogEntry(entry)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "mock plugin error")

        // Verify the plugin attempted to process the entry
        assert.Equal(t, 1, mockPlugin.processCount)
}

func TestBasePlugin(t *testing.T) {
        // Create a base plugin
        info := plugin.PluginInfo{
                Name:        "base",
                Description: "Base plugin for testing",
                Version:     "1.0.0",
                Author:      "LogStream Team",
        }
        basePlugin := plugin.NewBasePlugin(info)

        // Test basic interface methods
        assert.Equal(t, "base", basePlugin.Name())
        assert.Equal(t, "Base plugin for testing", basePlugin.Description())
        assert.Equal(t, "1.0.0", basePlugin.Version())

        // Test configuration
        config := map[string]string{
                "key1": "value1",
                "key2": "value2",
        }
        err := basePlugin.Init(config)
        require.NoError(t, err)
        assert.Equal(t, "value1", basePlugin.GetConfig("key1"))
        assert.Equal(t, "value2", basePlugin.GetConfig("key2"))
        assert.Equal(t, "", basePlugin.GetConfig("nonexistent"))
        assert.Equal(t, "default", basePlugin.GetConfigWithDefault("nonexistent", "default"))

        // Test close
        err = basePlugin.Close()
        require.NoError(t, err)
}
