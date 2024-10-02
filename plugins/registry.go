// plugins/registry.go

package plugins

import (
	"fmt"

	"github.com/tluyben/agentflow/types"
)

type Plugin func(map[string]interface{}) (map[string]interface{}, error)

type PluginInfo struct {
	Execute      Plugin
	InputSchema  []types.Property
	OutputSchema []types.Property
}

var PluginRegistry = make(map[string]PluginInfo)

func RegisterPlugin(name string, plugin Plugin, inputSchema, outputSchema []types.Property) {
	PluginRegistry[name] = PluginInfo{
		Execute:      plugin,
		InputSchema:  inputSchema,
		OutputSchema: outputSchema,
	}
}

func ExecutePlugin(name string, input map[string]interface{}) (map[string]interface{}, error) {
	pluginInfo, ok := PluginRegistry[name]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return pluginInfo.Execute(input)
}

func GetPluginSchema(name string) ([]types.Property, []types.Property, error) {
	pluginInfo, ok := PluginRegistry[name]
	if !ok {
		return nil, nil, fmt.Errorf("plugin %s not found", name)
	}
	return pluginInfo.InputSchema, pluginInfo.OutputSchema, nil
}