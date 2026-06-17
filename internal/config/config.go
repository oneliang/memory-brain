package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Graph  GraphConfig  `yaml:"graph"`
	Server ServerConfig `yaml:"server"`
}

// GraphConfig represents graph module configuration
type GraphConfig struct {
	Enabled         bool         `yaml:"enabled"`
	InjectToProfile bool         `yaml:"inject_to_profile"` // Inject graph context to profile
	Ollama          OllamaConfig `yaml:"ollama"`
	Query           QueryConfig  `yaml:"query"`
}

// OllamaConfig represents Ollama configuration for NER+RE
type OllamaConfig struct {
	Endpoint string `yaml:"endpoint"`
	Model    string `yaml:"model"`
	Timeout  int    `yaml:"timeout"` // seconds
}

// QueryConfig represents query configuration
type QueryConfig struct {
	MaxHops      int `yaml:"max_hops"`
	DefaultLimit int `yaml:"default_limit"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int `yaml:"port"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Graph: GraphConfig{
			Enabled:         true,
			InjectToProfile: true, // Default: inject graph context to profile
			Ollama: OllamaConfig{
				Endpoint: "http://localhost:11434",
				Model:    "qwen3.5:2b",
				Timeout:  30,
			},
			Query: QueryConfig{
				MaxHops:      3,
				DefaultLimit: 10,
			},
		},
		Server: ServerConfig{
			Port: 12321,
		},
	}
}

// Load loads configuration from file or returns default
func Load() (*Config, error) {
	configPath := getConfigPath()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, create default
		cfg := DefaultConfig()
		if err := createDefaultConfig(configPath); err != nil {
			// Failed to create config, but we can still use default
			fmt.Printf("Warning: failed to create default config: %v\n", err)
		} else {
			fmt.Printf("Created default config at: %s\n", configPath)
		}
		return cfg, nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	cfg := DefaultConfig() // Start with defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".memory-brain", "config.yaml")
}

// createDefaultConfig creates a default config file
func createDefaultConfig(configPath string) error {
	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate default config YAML
	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %w", err)
	}

	// Add header comment
	header := "# Memory Brain Configuration\n# Generated automatically. Edit as needed.\n\n"
	content := header + string(data)

	// Write to file
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	configPath := getConfigPath()

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
