package config

import (
"fmt"
"os"

"gopkg.in/yaml.v3"
)

type Config struct {
AgentID        string `yaml:"agent_id"`
Hostname       string `yaml:"hostname"`
OS             string `yaml:"os"`
CoordinatorURL string `yaml:"coordinator_url"`
AuthToken      string `yaml:"auth_token"`
Version        string `yaml:"version"`
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
if cfg.CoordinatorURL == "" {
return nil, fmt.Errorf("coordinator_url is required in config")
}
if cfg.AuthToken == "" {
return nil, fmt.Errorf("auth_token is required in config")
}
if cfg.Version == "" {
cfg.Version = "0.1.0"
}

return &cfg, nil
}