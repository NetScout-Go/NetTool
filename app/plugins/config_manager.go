package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ConfigManager handles loading and saving plugin system configuration
type ConfigManager struct {
	configPath     string
	configuration  *Configuration
	mu             sync.RWMutex
	loadedCallback func()
}

// Configuration represents the main configuration structure
type Configuration struct {
	GitHub  GitHubConfig   `json:"github"`
	Sources []PluginSource `json:"sources"`
}

// GitHubConfig represents GitHub-specific configuration
type GitHubConfig struct {
	Tokens []GitHubToken `json:"tokens"`
}

// GitHubToken represents a GitHub personal access token
type GitHubToken struct {
	Name         string `json:"name"`
	Token        string `json:"token"`
	Organization string `json:"organization"`
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(configPath string) *ConfigManager {
	if configPath == "" {
		configPath = "app/plugins/config.json"
	}

	return &ConfigManager{
		configPath: configPath,
		configuration: &Configuration{
			GitHub: GitHubConfig{
				Tokens: []GitHubToken{},
			},
			Sources: []PluginSource{},
		},
	}
}

// LoadConfiguration loads the configuration from the config file
func (cm *ConfigManager) LoadConfiguration() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if the config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Create default configuration
		defaultConfig := &Configuration{
			GitHub: GitHubConfig{
				Tokens: []GitHubToken{
					{
						Name:         "default",
						Token:        "",
						Organization: "NetScout-Go",
					},
				},
			},
			Sources: []PluginSource{
				{
					Name:         "NetScout-Go",
					Organization: "NetScout-Go",
					IsDefault:    true,
					Pattern:      "Plugin_*",
				},
			},
		}

		// Ensure the directory exists
		dir := filepath.Dir(cm.configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}

		// Save the default configuration
		data, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %v", err)
		}

		if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write default config: %v", err)
		}

		cm.configuration = defaultConfig
		return nil
	}

	// Read the config file
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	// Parse the config file
	var config Configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	cm.configuration = &config

	// Call the loaded callback if set
	if cm.loadedCallback != nil {
		cm.loadedCallback()
	}

	return nil
}

// SaveConfiguration saves the configuration to the config file
func (cm *ConfigManager) SaveConfiguration() error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Marshal the configuration
	data, err := json.MarshalIndent(cm.configuration, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Ensure the directory exists
	dir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	// Write the config file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

// AddGitHubToken adds a GitHub token to the configuration
func (cm *ConfigManager) AddGitHubToken(name, token, organization string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if a token with this name already exists
	for i, t := range cm.configuration.GitHub.Tokens {
		if t.Name == name {
			// Update the existing token
			cm.configuration.GitHub.Tokens[i].Token = token
			cm.configuration.GitHub.Tokens[i].Organization = organization
			return cm.SaveConfiguration()
		}
	}

	// Add a new token
	cm.configuration.GitHub.Tokens = append(cm.configuration.GitHub.Tokens, GitHubToken{
		Name:         name,
		Token:        token,
		Organization: organization,
	})

	return cm.SaveConfiguration()
}

// RemoveGitHubToken removes a GitHub token from the configuration
func (cm *ConfigManager) RemoveGitHubToken(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find the token index
	index := -1
	for i, t := range cm.configuration.GitHub.Tokens {
		if t.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("token '%s' not found", name)
	}

	// Remove the token
	cm.configuration.GitHub.Tokens = append(
		cm.configuration.GitHub.Tokens[:index],
		cm.configuration.GitHub.Tokens[index+1:]...,
	)

	return cm.SaveConfiguration()
}

// GetGitHubToken returns the GitHub token for the specified name
func (cm *ConfigManager) GetGitHubToken(name string) (GitHubToken, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, t := range cm.configuration.GitHub.Tokens {
		if t.Name == name {
			return t, nil
		}
	}

	return GitHubToken{}, fmt.Errorf("token '%s' not found", name)
}

// GetGitHubTokens returns all GitHub tokens
func (cm *ConfigManager) GetGitHubTokens() []GitHubToken {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	tokens := make([]GitHubToken, len(cm.configuration.GitHub.Tokens))
	copy(tokens, cm.configuration.GitHub.Tokens)

	return tokens
}

// GetTokenForOrganization returns a GitHub token for the specified organization
func (cm *ConfigManager) GetTokenForOrganization(org string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// First try to find a token specifically for this organization
	for _, t := range cm.configuration.GitHub.Tokens {
		if t.Organization == org && t.Token != "" {
			return t.Token, nil
		}
	}

	// Then try to find any token
	for _, t := range cm.configuration.GitHub.Tokens {
		if t.Token != "" {
			return t.Token, nil
		}
	}

	return "", fmt.Errorf("no token found for organization '%s'", org)
}

// GetSources returns all plugin sources
func (cm *ConfigManager) GetSources() []PluginSource {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	sources := make([]PluginSource, len(cm.configuration.Sources))
	copy(sources, cm.configuration.Sources)

	return sources
}

// AddSource adds a plugin source to the configuration
func (cm *ConfigManager) AddSource(name, organization, pattern string, isDefault bool) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if a source with this name already exists
	for i, s := range cm.configuration.Sources {
		if s.Name == name {
			// Update the existing source
			cm.configuration.Sources[i].Organization = organization
			cm.configuration.Sources[i].Pattern = pattern
			cm.configuration.Sources[i].IsDefault = isDefault
			return cm.SaveConfiguration()
		}
	}

	// Add a new source
	cm.configuration.Sources = append(cm.configuration.Sources, PluginSource{
		Name:         name,
		Organization: organization,
		Pattern:      pattern,
		IsDefault:    isDefault,
	})

	return cm.SaveConfiguration()
}

// RemoveSource removes a plugin source from the configuration
func (cm *ConfigManager) RemoveSource(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Find the source index
	index := -1
	for i, s := range cm.configuration.Sources {
		if s.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("source '%s' not found", name)
	}

	// Remove the source
	cm.configuration.Sources = append(
		cm.configuration.Sources[:index],
		cm.configuration.Sources[index+1:]...,
	)

	return cm.SaveConfiguration()
}

// SetLoadedCallback sets a callback function to be called when the configuration is loaded
func (cm *ConfigManager) SetLoadedCallback(callback func()) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.loadedCallback = callback
}

// GetConfiguration returns the current configuration
func (cm *ConfigManager) GetConfiguration() *Configuration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.configuration
}
