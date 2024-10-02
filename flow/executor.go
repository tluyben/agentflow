// flow/executor.go

package flow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tluyben/agentflow/plugin"
	"github.com/tluyben/agentflow/types"
)

type FlowExecutor struct {
	PluginRegistry *plugin.PluginRegistry
	LLMCaller      func(model, systemPrompt, userPrompt string) (string, error)
}

func NewFlowExecutor(pluginRegistry *plugin.PluginRegistry, llmCaller func(model, systemPrompt, userPrompt string) (string, error)) *FlowExecutor {
	return &FlowExecutor{
		PluginRegistry: pluginRegistry,
		LLMCaller:      llmCaller,
	}
}

func (fe *FlowExecutor) ExecuteFlow(flow types.Flow, input string) error {
	context := make(map[string]interface{})
	// var err error

	for {
		// Prepare input for LLM
		inputData := map[string]interface{}{
			"user_prompt": input,
			"context":     context,
		}
		inputJSON, err := json.Marshal(inputData)
		if err != nil {
			return fmt.Errorf("error marshaling input data: %w", err)
		}

		// Call LLM
		llmOutput, err := fe.LLMCaller(flow.Model, flow.SystemPrompt, strings.ReplaceAll(flow.Prompt, "{USER}", string(inputJSON)))
		if err != nil {
			return fmt.Errorf("error calling LLM: %w", err)
		}

		// Parse LLM output
		var output map[string]interface{}
		if err := json.Unmarshal([]byte(llmOutput), &output); err != nil {
			return fmt.Errorf("error parsing LLM output: %w", err)
		}

		// Check for plugin executions
		for pluginName, pluginInput := range output {
			if strings.HasPrefix(pluginName, "cmd:") {
				cmdName := strings.TrimPrefix(pluginName, "cmd:")
				pluginOutput, err := fe.PluginRegistry.ExecutePlugin(cmdName, pluginInput.(map[string]interface{}))
				if err != nil {
					return fmt.Errorf("error executing plugin %s: %w", cmdName, err)
				}
				context[cmdName+"_output"] = pluginOutput
				delete(output, pluginName)
			}
		}

		// Check for response
		if response, ok := output["response"].(string); ok && response != "" {
			fmt.Println(response)
			return nil
		}

		// Update context with any remaining output
		for k, v := range output {
			context[k] = v
		}

		// If we reach here, continue the loop with updated context
	}
}