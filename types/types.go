package types

type Property struct {
	Name       string     `yaml:"name"`
	Type       string     `yaml:"type"`
	Required   bool       `yaml:"required,omitempty"`
	Enum 	   []string   `yaml:"enum,omitempty"`	
	Properties []Property `yaml:"properties,omitempty"`
}

type FlowStep struct {
	Validate string `yaml:"validate"`
	Next     string `yaml:"next"`
}

type Flow struct {
	Name         string     `yaml:"name"`
	Model        string     `yaml:"model"`
	Actions      []string   `yaml:"action"`
	Input        []Property `yaml:"input"`
	Output       []Property `yaml:"output"`
	SystemPrompt string     `yaml:"system-prompt"`
	Prompt       string     `yaml:"prompt"`
	FlowSteps    []FlowStep `yaml:"flow"`
}