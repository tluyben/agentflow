// plugin/plugin.go

package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"reflect"

	"github.com/tluyben/agentflow/types"
	"gopkg.in/yaml.v2"
)

type PluginInfo struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Input       []types.Property `yaml:"input"`
	Output      []types.Property `yaml:"output"`
}

type PluginRegistry struct {
	Plugins map[string]*LoadedPlugin
}

type LoadedPlugin struct {
	Info   PluginInfo
	Execute reflect.Value
}

type PluginFunc func(input map[string]interface{}) (map[string]interface{}, error)

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		Plugins: make(map[string]*LoadedPlugin),
	}
}

func (r *PluginRegistry) LoadPlugins(pluginsDir string) error {
	configPath := filepath.Join(pluginsDir, "plugins.yml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading plugins.yml: %w", err)
	}

	var pluginsConfig struct {
		Plugins []PluginInfo `yaml:"plugins"`
	}
	err = yaml.Unmarshal(configData, &pluginsConfig)
	if err != nil {
		return fmt.Errorf("error parsing plugins.yml: %w", err)
	}

	for _, pluginInfo := range pluginsConfig.Plugins {
		pluginPath := filepath.Join(pluginsDir, pluginInfo.Name+".so")
		p, err := plugin.Open(pluginPath)
		if err != nil {
			return fmt.Errorf("error loading plugin %s: %w", pluginInfo.Name, err)
		}

		executeSymbol, err := p.Lookup("Execute")
		if err != nil {
			return fmt.Errorf("plugin %s does not export Execute function: %w", pluginInfo.Name, err)
		}

		execute := reflect.ValueOf(executeSymbol)
		if execute.Type() != reflect.TypeOf((*PluginFunc)(nil)).Elem() {
			return fmt.Errorf("plugin %s: Execute function has wrong signature", pluginInfo.Name)
		}

		r.Plugins[pluginInfo.Name] = &LoadedPlugin{
			Info:    pluginInfo,
			Execute: execute,
		}
	}

	return nil
}

func (r *PluginRegistry) ExecutePlugin(name string, input map[string]interface{}) (map[string]interface{}, error) {
	plugin, ok := r.Plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	results := plugin.Execute.Call([]reflect.Value{reflect.ValueOf(input)})
	if len(results) != 2 {
		return nil, fmt.Errorf("plugin %s: Execute function returned unexpected number of results", name)
	}

	outputValue := results[0].Interface()
	errValue := results[1].Interface()

	output, ok := outputValue.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("plugin %s: Execute function returned invalid output type", name)
	}

	if errValue != nil {
		return nil, errValue.(error)
	}

	return output, nil
}

func (r *PluginRegistry) GetPluginSchema(name string) ([]types.Property, []types.Property, error) {
	plugin, ok := r.Plugins[name]
	if !ok {
		return nil, nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin.Info.Input, plugin.Info.Output, nil
}