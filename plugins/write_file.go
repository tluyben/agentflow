// plugins/write_file.go

package plugins

import (
	"fmt"
	"os"

	"github.com/tluyben/agentflow/types"
)

func init() {
	inputSchema := []types.Property{
        {Name: "filepath", Type: "string"},
        {Name: "content", Type: "string"},
    }
    outputSchema := []types.Property{
        {Name: "success", Type: "boolean"},
        {Name: "message", Type: "string"},
    }
    RegisterPlugin("write-file", WriteFile, inputSchema, outputSchema)
}
func WriteFile(input map[string]interface{}) (map[string]interface{}, error) {
	filepath, ok := input["filepath"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid input: filepath must be a string")
	}

	content, ok := input["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid input: content must be a string")
	}

	err := os.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		return nil, fmt.Errorf("error writing file: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"message": "File written successfully",
	}, nil
}