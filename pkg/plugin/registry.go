package plugin

import (
	"fmt"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/hashicorp/go-hclog"
)

// Registry manages the available plugins
type Registry struct {
	plugins map[string]Plugin
	logger  hclog.Logger
	mutex   sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry(logger hclog.Logger) *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		logger:  logger,
	}
}

// RegisterPlugin registers a plugin with the registry
func (r *Registry) RegisterPlugin(plugin Plugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := plugin.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s is already registered", name)
	}

	r.plugins[name] = plugin
	r.logger.Info("Registered plugin", "name", name, "version", plugin.Version())
	return nil
}

// GetPlugin returns a plugin by name
func (r *Registry) GetPlugin(name string) (Plugin, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// LoadPlugins loads plugins from a directory
func (r *Registry) LoadPlugins(directory string, enabled []string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Create a map of enabled plugins for fast lookup
	enabledMap := make(map[string]bool)
	for _, name := range enabled {
		enabledMap[name] = true
	}

	// Find all .so files in the directory
	plugins, err := filepath.Glob(filepath.Join(directory, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugins: %w", err)
	}

	for _, pluginPath := range plugins {
		// Get plugin name from filename
		name := filepath.Base(pluginPath)
		name = name[:len(name)-3] // Remove .so extension

		// Skip disabled plugins
		if len(enabled) > 0 && !enabledMap[name] {
			r.logger.Debug("Skipping disabled plugin", "name", name)
			continue
		}

		// Attempt to load the plugin
		r.logger.Debug("Loading plugin", "path", pluginPath)
		p, err := plugin.Open(pluginPath)
		if err != nil {
			r.logger.Error("Failed to load plugin", "path", pluginPath, "error", err)
			continue
		}

		// Look up the plugin factory symbol
		symbol, err := p.Lookup("New")
		if err != nil {
			r.logger.Error("Plugin does not export 'New' symbol", "path", pluginPath, "error", err)
			continue
		}

		// Assert that the symbol is a plugin factory
		factory, ok := symbol.(func() Plugin)
		if !ok {
			r.logger.Error("Plugin 'New' symbol is not a factory function", "path", pluginPath)
			continue
		}

		// Create a plugin instance
		instance := factory()
		r.plugins[name] = instance
		r.logger.Info("Loaded plugin", "name", name, "version", instance.Version())
	}

	return nil
}

// ConfigurePlugin configures a plugin with the given options
func (r *Registry) ConfigurePlugin(name string, config map[string]string) error {
	plugin, err := r.GetPlugin(name)
	if err != nil {
		return err
	}

	r.logger.Debug("Configuring plugin", "name", name)
	return plugin.Init(config)
}

// ListPlugins returns a list of all registered plugins
func (r *Registry) ListPlugins() []PluginInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugins := make([]PluginInfo, 0, len(r.plugins))
	for _, p := range r.plugins {
		plugins = append(plugins, PluginInfo{
			Name:        p.Name(),
			Description: p.Description(),
			Version:     p.Version(),
		})
	}

	return plugins
}

// ClosePlugins closes all plugins
func (r *Registry) ClosePlugins() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for name, p := range r.plugins {
		r.logger.Debug("Closing plugin", "name", name)
		if err := p.Close(); err != nil {
			r.logger.Error("Error closing plugin", "name", name, "error", err)
		}
	}

	r.plugins = make(map[string]Plugin)
}
