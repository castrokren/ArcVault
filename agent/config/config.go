package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AgentID        string   `yaml:"agent_id"`
	Hostname       string   `yaml:"hostname"`
	OS             string   `yaml:"os"`
	CoordinatorURL string   `yaml:"coordinator_url"`
	Coordinators   []string `yaml:"coordinators,omitempty"`
	AuthToken      string   `yaml:"auth_token"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("could not parse config file: %w", err)
	}

	if cfg.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required in config")
	}
	// Either coordinator_url or coordinators list must be provided (backward compat).
	if cfg.CoordinatorURL == "" && len(cfg.Coordinators) == 0 {
		return nil, fmt.Errorf("coordinator_url or coordinators list is required in config")
	}
	// If coordinators list is provided, use it; otherwise fall back to single coordinator_url.
	if len(cfg.Coordinators) > 0 && cfg.CoordinatorURL == "" {
		cfg.CoordinatorURL = cfg.Coordinators[0]
	}
	if cfg.AuthToken == "" {
		return nil, fmt.Errorf("auth_token is required in config")
	}

	return &cfg, nil
}
