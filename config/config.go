package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the .ci.yaml structure
type Config struct {
	Name string         `yaml:"name"`
	On   []string       `yaml:"on"` //  e.g., push, pull_request
	Jobs map[string]Job `yaml:"jobs"`
}

type Job struct {
	Needs []string `yaml:"needs"`
	Steps []Step   `yaml:"steps"`
	Name  string   `yaml:"name"`
}

type Step struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// LoadConfig reads and parses the .ci.yaml file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
