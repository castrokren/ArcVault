package config

import (
"encoding/json"
"fmt"
"os"
"path/filepath"
)

type Config struct {
Port         int    `json:"port"`
DatabasePath string `json:"database_path"`
AdminToken   string `json:"admin_token"`
Environment  string `json:"environment"`
}

func GetConfigPath() (string, error) {
home, err := os.UserHomeDir()
if err != nil {
return "", err
}
return filepath.Join(home, ".arcvault", "config.json"), nil
}

func Save(cfg *Config) error {
path, err := GetConfigPath()
if err != nil {
return fmt.Errorf("could not determine config path: %w", err)
}
if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
return fmt.Errorf("could not create config directory: %w", err)
}
data, err := json.MarshalIndent(cfg, "", "  ")
if err != nil {
return fmt.Errorf("could not marshal config: %w", err)
}
return os.WriteFile(path, data, 0600)
}

func Load() (*Config, error) {
path, err := GetConfigPath()
if err != nil {
return nil, err
}
data, err := os.ReadFile(path)
if err != nil {
return nil, fmt.Errorf("could not read config (run 'coordinator init' first): %w", err)
}
var cfg Config
if err := json.Unmarshal(data, &cfg); err != nil {
return nil, fmt.Errorf("could not parse config: %w", err)
}
return &cfg, nil
}