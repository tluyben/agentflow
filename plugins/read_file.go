// plugins/read_file.go

package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tluyben/agentflow/types"
)

func init() {
	inputSchema := []types.Property{
        {Name: "filepath", Type: "string"},
    }
    outputSchema := []types.Property{
        {Name: "filepath", Type: "string"},
        {Name: "content", Type: "string"},
        {Name: "file-type", Type: "string"},
    }
    RegisterPlugin("read-file", ReadFile, inputSchema, outputSchema)
}

func ReadFile(input map[string]interface{}) (map[string]interface{}, error) {
	filePath, ok := input["filepath"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid input: filepath must be a string")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return map[string]interface{}{
		"filepath":  filePath,
		"content":   string(content),
		"file-type": filepath.Ext(filePath),
	}, nil
}