package pipeline

// Config represents the root structure of .rune.yaml
type Config struct {
	Version  string `yaml:"version"`
	Pipeline []Step `yaml:"pipeline"`
}

// Step represents a single task/job in the pipeline
type Step struct {
	Name string            `yaml:"name"`
	Run  string            `yaml:"run,omitempty"`
	Args []string          `yaml:"args,omitempty"`
	Uses string            `yaml:"uses,omitempty"`
	With map[string]string `yaml:"with,omitempty"`
}
