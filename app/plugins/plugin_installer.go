package plugins

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// PluginMetadata represents the metadata of a plugin
type PluginMetadata struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Version         string         `json:"version"`
	Author          string         `json:"author"`
	License         string         `json:"license"`
	Icon            string         `json:"icon"`
	Status          string         `json:"status"`
	UpdateAvailable bool           `json:"updateAvailable"`
	LatestVersion   string         `json:"latestVersion,omitempty"`
	Path            string         `json:"path,omitempty"`
	Dependencies    []Dependency   `json:"dependencies,omitempty"`
	GitInfo         GitVersionInfo `json:"gitInfo,omitempty"`
}

// GitVersionInfo represents Git version information for a plugin
type GitVersionInfo struct {
	CommitID       string `json:"commitID"`
	Branch         string `json:"branch"`
	LatestCommitID string `json:"latestCommitID,omitempty"`
	Repository     string `json:"repository,omitempty"`
	Organization   string `json:"organization,omitempty"`
}

// Dependency represents a plugin dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// PluginInstaller handles installation, updates and uninstallation of plugins
type PluginInstaller struct {
	pluginsDir string
	manager    *PluginManager
	config     *ConfigManager
	// List of GitHub organizations from which plugins can be installed
	pluginSources []PluginSource
}

// PluginSource represents a source for plugins
type PluginSource struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	IsDefault    bool   `json:"isDefault"`
	Pattern      string `json:"pattern"` // Naming pattern for plugins (e.g., "Plugin_*" or "plugin-*")
}

// PluginListItem represents a plugin in the store listing
type PluginListItem struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Author       string                 `json:"author"`
	License      string                 `json:"license"`
	Category     string                 `json:"category"`
	Repository   string                 `json:"repository"`
	Icon         string                 `json:"icon"`
	Installed    bool                   `json:"installed"`
	DataJSON     map[string]interface{} `json:"dataJson,omitempty"`
	Screenshots  []string               `json:"screenshots,omitempty"`
	Requirements []string               `json:"requirements,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
}

// GitHubRepository represents a GitHub repository from the API
type GitHubRepository struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	UpdatedAt   string `json:"updated_at"`
}

// PluginDataJSON represents the data.json structure in plugin repositories
type PluginDataJSON struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Author       string   `json:"author"`
	License      string   `json:"license"`
	Category     string   `json:"category"`
	Icon         string   `json:"icon"`
	Screenshots  []string `json:"screenshots"`
	Requirements []string `json:"requirements"`
	Tags         []string `json:"tags"`
	MinVersion   string   `json:"minVersion"`
	Homepage     string   `json:"homepage"`
}

// PluginCatalog represents the catalog of available plugins
type PluginCatalog struct {
	Plugins []PluginListItem `json:"plugins"`
}

// NewPluginInstaller creates a new plugin installer
func NewPluginInstaller(pluginsDir string, manager *PluginManager) *PluginInstaller {
	// Initialize with NetScout-Go as the default plugin source
	defaultSources := []PluginSource{
		{
			Name:         "NetScout-Go",
			Organization: "NetScout-Go",
			IsDefault:    true,
			Pattern:      "Plugin_*",
		},
	}

	// Create config manager
	configManager := NewConfigManager("app/plugins/config.json")
	err := configManager.LoadConfiguration()
	if err != nil {
		fmt.Printf("Warning: Failed to load configuration: %v\n", err)
	}

	// Create plugin installer
	return &PluginInstaller{
		pluginsDir:    pluginsDir,
		manager:       manager,
		config:        configManager,
		pluginSources: defaultSources,
	}
}

// ListInstalledPlugins returns a list of installed plugins with metadata
func (pi *PluginInstaller) ListInstalledPlugins() ([]PluginMetadata, error) {
	// Get all plugin folders
	entries, err := os.ReadDir(pi.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %v", err)
	}

	var plugins []PluginMetadata

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginDir := filepath.Join(pi.pluginsDir, entry.Name())

		// Read plugin.json for metadata
		metadata, err := pi.readPluginMetadata(pluginDir)
		if err != nil {
			log.Printf("Warning: Failed to read metadata for plugin %s: %v", entry.Name(), err)
			continue
		}

		// Set plugin status
		metadata.Status = "active" // Default status

		// Check if plugin has an update available
		updateAvailable, latestVersion := pi.checkForUpdates(metadata.ID, metadata.Version)
		metadata.UpdateAvailable = updateAvailable
		metadata.LatestVersion = latestVersion

		// Set plugin path
		metadata.Path = pluginDir

		plugins = append(plugins, metadata)
	}

	return plugins, nil
}

// GetPluginDetails returns detailed information about a plugin
func (pi *PluginInstaller) GetPluginDetails(pluginID string) (PluginMetadata, error) {
	// Find the plugin directory
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin directory not found: %s", pluginID)
	}

	// Read plugin metadata
	metadata, err := pi.readPluginMetadata(pluginDir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Set plugin status
	metadata.Status = "active" // Default status

	// Check if plugin has an update available
	updateAvailable, latestVersion := pi.checkForUpdates(metadata.ID, metadata.Version)
	metadata.UpdateAvailable = updateAvailable
	metadata.LatestVersion = latestVersion

	// Set plugin path
	metadata.Path = pluginDir

	// Read dependencies
	dependencies, err := pi.readDependencies(pluginDir)
	if err == nil && dependencies != nil {
		metadata.Dependencies = dependencies
	}

	return metadata, nil
}

// InstallPlugin installs a plugin from a URL or Git repository
func (pi *PluginInstaller) InstallPlugin(url string) (PluginMetadata, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "nettool-plugin-")
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Check if URL is a Git repository
	if strings.HasSuffix(url, ".git") || strings.Contains(url, "github.com") || strings.Contains(url, "gitlab.com") {
		// Clone the Git repository
		cmd := exec.Command("git", "clone", "--depth", "1", url, tempDir)
		if err := cmd.Run(); err != nil {
			return PluginMetadata{}, fmt.Errorf("failed to clone repository: %v", err)
		}
	} else if strings.HasSuffix(url, ".zip") {
		// Download the ZIP file
		resp, err := http.Get(url)
		if err != nil {
			return PluginMetadata{}, fmt.Errorf("failed to download plugin: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return PluginMetadata{}, fmt.Errorf("failed to download plugin: HTTP %d", resp.StatusCode)
		}

		// Save the ZIP file
		zipPath := filepath.Join(tempDir, "plugin.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			return PluginMetadata{}, fmt.Errorf("failed to create temporary file: %v", err)
		}

		_, err = io.Copy(zipFile, resp.Body)
		zipFile.Close()
		if err != nil {
			return PluginMetadata{}, fmt.Errorf("failed to save plugin file: %v", err)
		}

		// Extract the ZIP file
		err = pi.extractZip(zipPath, tempDir)
		if err != nil {
			return PluginMetadata{}, fmt.Errorf("failed to extract plugin: %v", err)
		}
	} else {
		return PluginMetadata{}, fmt.Errorf("unsupported plugin source: %s", url)
	}

	// Validate the plugin
	metadata, err := pi.validatePlugin(tempDir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Create the plugin directory
	pluginDir := filepath.Join(pi.pluginsDir, metadata.ID)

	// Check if the plugin already exists
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin with ID %s already exists", metadata.ID)
	}

	// Copy the plugin files to the plugins directory
	err = pi.copyDir(tempDir, pluginDir)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to install plugin: %v", err)
	}

	// Build the plugin if needed
	if err := pi.buildPlugin(pluginDir); err != nil {
		// This is not a fatal error, just log it
		log.Printf("Warning: Failed to build plugin %s: %v", metadata.ID, err)
	}

	// Set plugin path
	metadata.Path = pluginDir

	// Reload plugins in the plugin manager
	pi.manager.RegisterPlugins()

	return metadata, nil
}

// UploadPlugin installs a plugin from an uploaded ZIP file
func (pi *PluginInstaller) UploadPlugin(file io.Reader) (PluginMetadata, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "nettool-plugin-upload-")
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save the uploaded file
	zipPath := filepath.Join(tempDir, "plugin.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to create temporary file: %v", err)
	}

	_, err = io.Copy(zipFile, file)
	zipFile.Close()
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to save uploaded file: %v", err)
	}

	// Extract the ZIP file
	extractDir := filepath.Join(tempDir, "extracted")
	err = os.MkdirAll(extractDir, 0755)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to create extraction directory: %v", err)
	}

	err = pi.extractZip(zipPath, extractDir)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to extract plugin: %v", err)
	}

	// Validate the plugin
	metadata, err := pi.validatePlugin(extractDir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Create the plugin directory
	pluginDir := filepath.Join(pi.pluginsDir, metadata.ID)

	// Check if the plugin already exists
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin with ID %s already exists", metadata.ID)
	}

	// Copy the plugin files to the plugins directory
	err = pi.copyDir(extractDir, pluginDir)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to install plugin: %v", err)
	}

	// Build the plugin if needed
	if err := pi.buildPlugin(pluginDir); err != nil {
		// This is not a fatal error, just log it
		log.Printf("Warning: Failed to build plugin %s: %v", metadata.ID, err)
	}

	// Set plugin path
	metadata.Path = pluginDir

	// Reload plugins in the plugin manager
	pi.manager.RegisterPlugins()

	return metadata, nil
}

// UpdatePlugin updates a plugin to the latest version
func (pi *PluginInstaller) UpdatePlugin(pluginID string) (PluginMetadata, error) {
	// Find the plugin directory
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin directory not found: %s", pluginID)
	}

	// Read plugin metadata
	metadata, err := pi.readPluginMetadata(pluginDir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Check if the plugin has a git repository
	gitDir := filepath.Join(pluginDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin was not installed from a Git repository, cannot update")
	}

	// Pull the latest changes
	cmd := exec.Command("git", "-C", pluginDir, "pull")
	if err := cmd.Run(); err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to update plugin: %v", err)
	}

	// Rebuild the plugin if needed
	if err := pi.buildPlugin(pluginDir); err != nil {
		// This is not a fatal error, just log it
		log.Printf("Warning: Failed to build plugin %s after update: %v", metadata.ID, err)
	}

	// Read updated metadata
	updatedMetadata, err := pi.readPluginMetadata(pluginDir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Set plugin path and status
	updatedMetadata.Path = pluginDir
	updatedMetadata.Status = "active"

	// Reload plugins in the plugin manager
	pi.manager.RegisterPlugins()

	return updatedMetadata, nil
}

// UninstallPlugin uninstalls a plugin
func (pi *PluginInstaller) UninstallPlugin(pluginID string) (PluginMetadata, error) {
	// Find the plugin directory
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin directory not found: %s", pluginID)
	}

	// Read plugin metadata before deletion
	metadata, err := pi.readPluginMetadata(pluginDir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Delete the plugin directory
	if err := os.RemoveAll(pluginDir); err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to uninstall plugin: %v", err)
	}

	// Set plugin status
	metadata.Status = "uninstalled"

	// Reload plugins in the plugin manager
	pi.manager.RegisterPlugins()

	return metadata, nil
}

// CheckForUpdates checks if a plugin has updates available (exported version of checkForUpdates)
func (pi *PluginInstaller) CheckForUpdates(pluginID string) (bool, string) {
	// Get plugin metadata
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)
	metadata, err := pi.readPluginMetadata(pluginDir)
	if err != nil {
		log.Printf("Warning: Failed to read metadata for plugin %s: %v", pluginID, err)
		return false, ""
	}

	return pi.checkForUpdates(pluginID, metadata.Version)
}

// UpdateVersionInfo updates the plugin.json file with the latest version information
func (pi *PluginInstaller) UpdateVersionInfo(pluginID string) error {
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)

	// Check if plugin.json exists
	jsonPath := filepath.Join(pluginDir, "plugin.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found for plugin %s", pluginID)
	}

	// Read existing plugin.json
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin.json: %v", err)
	}

	var pluginData map[string]interface{}
	if err := json.Unmarshal(data, &pluginData); err != nil {
		return fmt.Errorf("failed to parse plugin.json: %v", err)
	}

	// Check if the plugin has a Git repository
	gitDir := filepath.Join(pluginDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s is not a Git repository", pluginID)
	}

	// Get the latest tag
	cmd := exec.Command("git", "-C", pluginDir, "fetch", "--tags")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch tags for plugin %s: %v", pluginID, err)
	}

	cmd = exec.Command("git", "-C", pluginDir, "describe", "--tags", "--abbrev=0")
	tagOutput, err := cmd.Output()

	// If there are no tags, use the current commit hash
	if err != nil {
		cmd = exec.Command("git", "-C", pluginDir, "rev-parse", "--short", "HEAD")
		tagOutput, err = cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to get current commit hash for plugin %s: %v", pluginID, err)
		}
	}

	latestVersion := strings.TrimSpace(string(tagOutput))

	// Update the version in the plugin data
	pluginData["version"] = latestVersion

	// Get author information if not present
	if _, ok := pluginData["author"]; !ok {
		cmd = exec.Command("git", "-C", pluginDir, "config", "user.name")
		authorOutput, err := cmd.Output()
		if err == nil {
			author := strings.TrimSpace(string(authorOutput))
			if author != "" {
				pluginData["author"] = author
			} else {
				pluginData["author"] = "NetScout-Go"
			}
		} else {
			pluginData["author"] = "NetScout-Go"
		}
	}

	// Get license information if not present
	if _, ok := pluginData["license"]; !ok {
		// Check for LICENSE file
		licensePath := filepath.Join(pluginDir, "LICENSE")
		if _, err := os.Stat(licensePath); !os.IsNotExist(err) {
			pluginData["license"] = "See LICENSE file"
		} else {
			pluginData["license"] = "MIT"
		}
	}

	// Write updated plugin.json
	updatedData, err := json.MarshalIndent(pluginData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plugin data: %v", err)
	}

	if err := os.WriteFile(jsonPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated plugin.json: %v", err)
	}

	return nil
}

// updatePluginJsonWithGitInfo updates the plugin.json file with Git information
func (pi *PluginInstaller) updatePluginJsonWithGitInfo(pluginDir string, gitInfo GitVersionInfo) error {
	// Check if plugin.json exists
	jsonPath := filepath.Join(pluginDir, "plugin.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin.json not found")
	}

	// Read existing plugin.json
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin.json: %v", err)
	}

	var pluginData map[string]interface{}
	if err := json.Unmarshal(data, &pluginData); err != nil {
		return fmt.Errorf("failed to parse plugin.json: %v", err)
	}

	// Add Git information
	gitInfoMap := map[string]interface{}{
		"commitID":     gitInfo.CommitID,
		"branch":       gitInfo.Branch,
		"repository":   gitInfo.Repository,
		"organization": gitInfo.Organization,
	}

	pluginData["gitInfo"] = gitInfoMap

	// Write updated plugin.json
	updatedData, err := json.MarshalIndent(pluginData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plugin data: %v", err)
	}

	if err := os.WriteFile(jsonPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated plugin.json: %v", err)
	}

	return nil
}

// InstallFromGitHub installs a plugin from a GitHub repository in the specified organization
func (pi *PluginInstaller) InstallFromGitHub(org string, repo string, branch string) (PluginMetadata, error) {
	// If branch is empty, use main as default
	if branch == "" {
		branch = "main"
	}

	// Format the GitHub URL
	url := fmt.Sprintf("https://github.com/%s/%s.git", org, repo)

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "nettool-plugin-")
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone the Git repository with the specified branch
	cmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, url, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to clone repository: %v\nOutput: %s", err, string(output))
	}

	// Get the current commit ID
	cmd = exec.Command("git", "-C", tempDir, "rev-parse", "HEAD")
	commitOutput, err := cmd.Output()
	commitID := ""
	if err == nil {
		commitID = strings.TrimSpace(string(commitOutput))
	}

	// Validate the plugin
	metadata, err := pi.validatePlugin(tempDir)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("invalid plugin: %v", err)
	}

	// Create the plugin directory
	pluginDir := filepath.Join(pi.pluginsDir, metadata.ID)

	// Check if the plugin already exists
	if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin with ID %s already exists", metadata.ID)
	}

	// Copy the plugin files to the plugins directory
	err = pi.copyDir(tempDir, pluginDir)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to install plugin: %v", err)
	}

	// Add Git version information
	metadata.GitInfo = GitVersionInfo{
		CommitID:     commitID,
		Branch:       branch,
		Repository:   repo,
		Organization: org,
	}

	// Update the plugin.json file with Git information
	if err := pi.updatePluginJsonWithGitInfo(pluginDir, metadata.GitInfo); err != nil {
		log.Printf("Warning: Failed to update plugin.json with Git information: %v", err)
	}

	// Build the plugin if needed
	if err := pi.buildPlugin(pluginDir); err != nil {
		// This is not a fatal error, just log it
		log.Printf("Warning: Failed to build plugin %s: %v", metadata.ID, err)
	}

	// Set plugin path
	metadata.Path = pluginDir

	// Reload plugins in the plugin manager
	pi.manager.RegisterPlugins()

	return metadata, nil
}

// ListGitHubPlugins lists available plugins from a GitHub organization
func (pi *PluginInstaller) ListGitHubPlugins(org string) ([]map[string]interface{}, error) {
	// GitHub API URL for organization repositories
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", org)

	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication if available
	token, err := pi.config.GetTokenForOrganization(org)
	if err == nil && token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	// Add User-Agent header required by GitHub API
	req.Header.Add("User-Agent", "NetTool-Plugin-Installer")
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Check for rate limiting
		if resp.StatusCode == http.StatusForbidden {
			rateLimitRemaining := resp.Header.Get("X-RateLimit-Remaining")
			if rateLimitRemaining == "0" {
				resetTimeStr := resp.Header.Get("X-RateLimit-Reset")
				return nil, fmt.Errorf("GitHub API rate limit exceeded. Add a personal access token in config.json to increase the limit. Resets at %s", resetTimeStr)
			}
		}
		return nil, fmt.Errorf("GitHub API returned status code %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse the JSON response
	var repos []map[string]interface{}
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Find the source pattern for this organization
	pattern := "Plugin_*" // Default pattern for NetScout-Go
	for _, source := range pi.pluginSources {
		if source.Organization == org {
			pattern = source.Pattern
			break
		}
	}

	// Filter repositories to only include plugins based on naming pattern
	var plugins []map[string]interface{}
	for _, repo := range repos {
		name, ok := repo["name"].(string)
		if !ok {
			continue
		}

		// Check if the repository name matches the plugin pattern
		isPlugin := false

		// Match by specific pattern (e.g., "Plugin_*")
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			isPlugin = strings.HasSuffix(name, suffix)
		} else if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			isPlugin = strings.HasPrefix(name, prefix)
		} else if pattern == "*" {
			// Match any repo in this organization
			isPlugin = true
		} else {
			// Exact match
			isPlugin = (name == pattern)
		}

		// Fallback to generic plugin naming patterns
		if !isPlugin {
			isPlugin = strings.HasPrefix(name, "plugin-") ||
				strings.HasSuffix(name, "-plugin") ||
				strings.Contains(name, "plugin")
		}

		if isPlugin {
			// Add additional metadata to help the user
			repo["is_plugin"] = true
			repo["organization"] = org

			// Try to get commit information
			if url, ok := repo["html_url"].(string); ok {
				repo["repo_url"] = url
			}

			// Extract plugin ID from repository name
			pluginID := name
			if strings.HasPrefix(name, "Plugin_") {
				pluginID = strings.TrimPrefix(name, "Plugin_")
			}
			repo["plugin_id"] = pluginID

			plugins = append(plugins, repo)
		}
	}

	return plugins, nil
}

// ListAllGitHubPlugins lists available plugins from all registered organizations
func (pi *PluginInstaller) ListAllGitHubPlugins() ([]map[string]interface{}, error) {
	var allPlugins []map[string]interface{}

	for _, source := range pi.pluginSources {
		plugins, err := pi.ListGitHubPlugins(source.Organization)
		if err != nil {
			log.Printf("Warning: Failed to list plugins from %s: %v", source.Organization, err)
			continue
		}

		// Add source information to each plugin
		for _, plugin := range plugins {
			plugin["source_name"] = source.Name
			plugin["source_organization"] = source.Organization
			plugin["source_is_default"] = source.IsDefault
			allPlugins = append(allPlugins, plugin)
		}
	}

	return allPlugins, nil
}

// AddPluginSource adds a new plugin source organization
func (pi *PluginInstaller) AddPluginSource(name, organization, pattern string) error {
	// Check if the organization exists
	if !pi.isValidGitHubOrg(organization) {
		return fmt.Errorf("GitHub organization '%s' does not exist or is not accessible", organization)
	}

	// Check if source already exists
	for _, source := range pi.pluginSources {
		if source.Organization == organization {
			return fmt.Errorf("plugin source with organization '%s' already exists", organization)
		}
	}

	// Add the new source
	pi.pluginSources = append(pi.pluginSources, PluginSource{
		Name:         name,
		Organization: organization,
		IsDefault:    false,
		Pattern:      pattern,
	})

	return nil
}

// RemovePluginSource removes a plugin source organization
func (pi *PluginInstaller) RemovePluginSource(organization string) error {
	for i, source := range pi.pluginSources {
		if source.Organization == organization {
			// Cannot remove default source
			if source.IsDefault {
				return fmt.Errorf("cannot remove default plugin source")
			}

			// Remove the source
			pi.pluginSources = append(pi.pluginSources[:i], pi.pluginSources[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("plugin source with organization '%s' not found", organization)
}

// GetPluginSources returns the list of plugin sources
func (pi *PluginInstaller) GetPluginSources() []PluginSource {
	return pi.pluginSources
}

// isValidGitHubOrg checks if a GitHub organization exists
func (pi *PluginInstaller) isValidGitHubOrg(org string) bool {
	// GitHub API URL for organization
	url := fmt.Sprintf("https://api.github.com/orgs/%s", org)

	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	// Add authentication if available
	token, err := pi.config.GetTokenForOrganization(org)
	if err == nil && token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	// Add User-Agent header required by GitHub API
	req.Header.Add("User-Agent", "NetTool-Plugin-Installer")
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Check if the organization exists
	return resp.StatusCode == http.StatusOK
}

// ListAvailablePlugins returns a list of available plugins from the catalog and GitHub
func (pi *PluginInstaller) ListAvailablePlugins() ([]PluginListItem, error) {
	// Get installed plugins
	installedPlugins, err := pi.ListInstalledPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to list installed plugins: %v", err)
	}

	// Create map of installed plugins for quick lookup
	installedMap := make(map[string]bool)
	var pluginItems []PluginListItem

	// Add installed plugins to the list
	for _, plugin := range installedPlugins {
		installedMap[plugin.ID] = true

		// Convert each installed plugin to a PluginListItem
		item := PluginListItem{
			ID:          plugin.ID,
			Name:        plugin.Name,
			Description: plugin.Description,
			Version:     plugin.Version,
			Author:      plugin.Author,
			License:     plugin.License,
			Icon:        plugin.Icon,
			Installed:   true,
		}

		// Try to get repository information if available in the GitInfo
		if plugin.GitInfo.Repository != "" {
			if plugin.GitInfo.Organization != "" {
				item.Repository = fmt.Sprintf("https://github.com/%s/%s",
					plugin.GitInfo.Organization, plugin.GitInfo.Repository)
			}
		}

		pluginItems = append(pluginItems, item)
	}

	// Fetch plugins from GitHub sources
	for _, source := range pi.pluginSources {
		githubPlugins, err := pi.fetchPluginsFromGitHub(source)
		if err != nil {
			log.Printf("Error fetching plugins from %s: %v", source.Organization, err)
			continue
		}

		// Add GitHub plugins to the list
		for _, plugin := range githubPlugins {
			// Skip if already installed
			if _, exists := installedMap[plugin.ID]; exists {
				continue
			}

			pluginItems = append(pluginItems, plugin)
		}
	}

	return pluginItems, nil
}

// fetchPluginsFromGitHub fetches plugins from a GitHub organization that match the Plugin_ pattern
func (pi *PluginInstaller) fetchPluginsFromGitHub(source PluginSource) ([]PluginListItem, error) {
	// GitHub API URL for organization repositories
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100", source.Organization)

	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication if available
	token, err := pi.config.GetTokenForOrganization(source.Organization)
	if err == nil && token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	// Add User-Agent header required by GitHub API
	req.Header.Add("User-Agent", "NetTool-Plugin-Installer")
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			rateLimitRemaining := resp.Header.Get("X-RateLimit-Remaining")
			if rateLimitRemaining == "0" {
				resetTimeStr := resp.Header.Get("X-RateLimit-Reset")
				return nil, fmt.Errorf("GitHub API rate limit exceeded. Add a personal access token in config.json to increase the limit. Resets at %s", resetTimeStr)
			}
		}
		return nil, fmt.Errorf("GitHub API returned status code %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse the JSON response
	var repos []GitHubRepository
	if err := json.Unmarshal(body, &repos); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	var plugins []PluginListItem

	// Filter repositories to only include those with Plugin_ prefix
	for _, repo := range repos {
		if !strings.HasPrefix(repo.Name, "Plugin_") {
			continue
		}

		// Fetch data.json from the repository
		pluginData, err := pi.fetchPluginDataJSON(source.Organization, repo.Name)
		if err != nil {
			log.Printf("Error fetching data.json for %s/%s: %v", source.Organization, repo.Name, err)
			// Create a basic plugin item without data.json
			pluginID := strings.TrimPrefix(repo.Name, "Plugin_")
			plugins = append(plugins, PluginListItem{
				ID:          pluginID,
				Name:        repo.Name,
				Description: repo.Description,
				Version:     "unknown",
				Author:      source.Organization,
				License:     "unknown",
				Category:    "other",
				Repository:  repo.HTMLURL,
				Icon:        "plugin",
				Installed:   false,
			})
			continue
		}

		// Create plugin item from data.json
		pluginID := pluginData.ID
		if pluginID == "" {
			pluginID = strings.TrimPrefix(repo.Name, "Plugin_")
		}

		plugin := PluginListItem{
			ID:           pluginID,
			Name:         pluginData.Name,
			Description:  pluginData.Description,
			Version:      pluginData.Version,
			Author:       pluginData.Author,
			License:      pluginData.License,
			Category:     pluginData.Category,
			Repository:   repo.HTMLURL,
			Icon:         pluginData.Icon,
			Installed:    false,
			Screenshots:  pluginData.Screenshots,
			Requirements: pluginData.Requirements,
			Tags:         pluginData.Tags,
		}

		// Use repository description as fallback
		if plugin.Description == "" {
			plugin.Description = repo.Description
		}

		// Set defaults
		if plugin.Author == "" {
			plugin.Author = source.Organization
		}
		if plugin.License == "" {
			plugin.License = "unknown"
		}
		if plugin.Category == "" {
			plugin.Category = "other"
		}
		if plugin.Icon == "" {
			plugin.Icon = "plugin"
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// fetchPluginDataJSON fetches the data.json file from a GitHub repository
func (pi *PluginInstaller) fetchPluginDataJSON(org, repo string) (*PluginDataJSON, error) {
	// GitHub API URL for file contents
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/data.json", org, repo)

	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add authentication if available
	token, err := pi.config.GetTokenForOrganization(org)
	if err == nil && token != "" {
		req.Header.Add("Authorization", "token "+token)
	}

	// Add User-Agent header required by GitHub API
	req.Header.Add("User-Agent", "NetTool-Plugin-Installer")
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data.json: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("data.json not found in repository")
		}
		return nil, fmt.Errorf("GitHub API returned status code %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Parse the GitHub API response
	var fileResponse struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(body, &fileResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Decode the base64 content
	if fileResponse.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported encoding: %s", fileResponse.Encoding)
	}

	// GitHub API returns content with newlines, so we need to clean it
	cleanContent := strings.ReplaceAll(fileResponse.Content, "\n", "")

	// Decode base64
	decodedContent, err := base64.StdEncoding.DecodeString(cleanContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 content: %v", err)
	}

	// Parse the data.json content
	var pluginData PluginDataJSON
	if err := json.Unmarshal(decodedContent, &pluginData); err != nil {
		return nil, fmt.Errorf("failed to parse data.json: %v", err)
	}

	return &pluginData, nil
}

// RefreshPluginCatalog fetches the latest plugins from GitHub and updates individual plugin.json files
func (pi *PluginInstaller) RefreshPluginCatalog() error {
	// Fetch plugins from each configured source
	for _, source := range pi.pluginSources {
		// Construct GitHub API URL to list repositories
		apiURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos", source.Organization)

		// Make the request
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			log.Printf("Error creating request for %s: %v", source.Organization, err)
			continue
		}

		// Add authentication if available
		token, err := pi.config.GetTokenForOrganization(source.Organization)
		if err == nil && token != "" {
			req.Header.Add("Authorization", "token "+token)
		}

		// Add GitHub API version header
		req.Header.Add("Accept", "application/vnd.github.v3+json")
		req.Header.Add("User-Agent", "NetTool-Plugin-Installer")

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching repositories from %s: %v", source.Organization, err)
			continue
		}
		defer resp.Body.Close()

		// Check response
		if resp.StatusCode != http.StatusOK {
			// Check for rate limiting
			if resp.StatusCode == http.StatusForbidden {
				rateLimitRemaining := resp.Header.Get("X-RateLimit-Remaining")
				if rateLimitRemaining == "0" {
					resetTimeStr := resp.Header.Get("X-RateLimit-Reset")
					log.Printf("GitHub API rate limit exceeded for %s. Add a personal access token in config.json to increase the limit. Resets at %s", source.Organization, resetTimeStr)
					continue
				}
			}

			body, _ := io.ReadAll(resp.Body)
			log.Printf("GitHub API error (%d) for %s: %s", resp.StatusCode, source.Organization, string(body))
			continue
		}

		// Parse the response
		var repos []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			HTMLURL     string `json:"html_url"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			log.Printf("Error parsing GitHub response for %s: %v", source.Organization, err)
			continue
		}

		// Process each repository that matches the plugin pattern
		for _, repo := range repos {
			// Check if repo name matches plugin pattern (e.g., "Plugin_*")
			if !strings.HasPrefix(repo.Name, "Plugin_") {
				continue
			}

			// Extract plugin ID from repo name (e.g., "Plugin_ping" -> "ping")
			pluginID := strings.TrimPrefix(repo.Name, "Plugin_")

			// Check if plugin directory exists locally
			pluginDir := filepath.Join(pi.pluginsDir, pluginID)
			if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
				// Plugin not installed, skip updating its plugin.json
				continue
			}

			// Read existing plugin.json if it exists
			jsonPath := filepath.Join(pluginDir, "plugin.json")
			var pluginData map[string]interface{}

			existingData, err := os.ReadFile(jsonPath)
			if err == nil {
				// If plugin.json exists, unmarshal it
				if err := json.Unmarshal(existingData, &pluginData); err != nil {
					log.Printf("Error parsing existing plugin.json for %s: %v", pluginID, err)
					// Initialize empty map if parsing failed
					pluginData = make(map[string]interface{})
				}
			} else {
				// Initialize empty map if file doesn't exist
				pluginData = make(map[string]interface{})
			}

			// Fetch plugin.json from GitHub to get latest metadata
			pluginJSONURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/plugin.json",
				source.Organization, repo.Name)

			pluginResp, err := http.Get(pluginJSONURL)
			if err == nil && pluginResp.StatusCode == http.StatusOK {
				var remotePluginData map[string]interface{}

				if err := json.NewDecoder(pluginResp.Body).Decode(&remotePluginData); err == nil {
					// Update local plugin data with remote data
					for key, value := range remotePluginData {
						pluginData[key] = value
					}
				}
				pluginResp.Body.Close()
			}

			// Ensure basic fields are set
			if _, ok := pluginData["id"]; !ok {
				pluginData["id"] = pluginID
			}
			if _, ok := pluginData["name"]; !ok {
				pluginData["name"] = strings.ReplaceAll(pluginID, "_", " ")
			}
			if _, ok := pluginData["repository"]; !ok {
				pluginData["repository"] = repo.HTMLURL
			}
			if _, ok := pluginData["description"]; !ok && repo.Description != "" {
				pluginData["description"] = repo.Description
			}
			if _, ok := pluginData["author"]; !ok {
				pluginData["author"] = source.Organization
			}
			if _, ok := pluginData["license"]; !ok {
				pluginData["license"] = "GPL-3.0" // Default license
			}
			if _, ok := pluginData["version"]; !ok {
				pluginData["version"] = "1.0.0" // Default version
			}

			// Write updated plugin.json
			updatedData, err := json.MarshalIndent(pluginData, "", "  ")
			if err != nil {
				log.Printf("Error marshaling plugin.json for %s: %v", pluginID, err)
				continue
			}

			if err := os.WriteFile(jsonPath, updatedData, 0644); err != nil {
				log.Printf("Error writing plugin.json for %s: %v", pluginID, err)
				continue
			}
		}
	}

	return nil
}

// Helper functions

// readPluginMetadata reads and parses the plugin.json file
func (pi *PluginInstaller) readPluginMetadata(pluginDir string) (PluginMetadata, error) {
	// Read plugin.json
	jsonPath := filepath.Join(pluginDir, "plugin.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to read plugin.json: %v", err)
	}

	var metadata PluginMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return PluginMetadata{}, fmt.Errorf("failed to parse plugin.json: %v", err)
	}

	return metadata, nil
}

// readDependencies reads and parses the DEPENDENCIES.md file if it exists
func (pi *PluginInstaller) readDependencies(pluginDir string) ([]Dependency, error) {
	// Check if DEPENDENCIES.md exists
	depPath := filepath.Join(pluginDir, "DEPENDENCIES.md")
	if _, err := os.Stat(depPath); os.IsNotExist(err) {
		return nil, nil
	}

	// Read DEPENDENCIES.md
	data, err := os.ReadFile(depPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read DEPENDENCIES.md: %v", err)
	}

	// Parse dependencies
	var deps []Dependency
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 1 {
			deps = append(deps, Dependency{
				Name: parts[0],
			})
		} else {
			deps = append(deps, Dependency{
				Name:    parts[0],
				Version: strings.TrimSpace(parts[1]),
			})
		}
	}

	return deps, nil
}

// checkForUpdates checks if a plugin has updates available
func (pi *PluginInstaller) checkForUpdates(pluginID, currentVersion string) (bool, string) {
	// Check if the plugin has a git repository
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)
	gitDir := filepath.Join(pluginDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false, ""
	}

	// Read plugin metadata to get Git info
	metadata, err := pi.readPluginMetadata(pluginDir)
	if err != nil {
		log.Printf("Warning: Failed to read metadata for plugin %s: %v", pluginID, err)
		return false, ""
	}

	// Get current branch (default to main)
	branch := "main"
	if metadata.GitInfo.Branch != "" {
		branch = metadata.GitInfo.Branch
	}

	// Fetch the latest changes without applying them
	cmd := exec.Command("git", "-C", pluginDir, "fetch", "origin", branch)
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: Failed to fetch updates for plugin %s: %v", pluginID, err)
		return false, ""
	}

	// Get the current commit ID
	currentCommitID := metadata.GitInfo.CommitID
	if currentCommitID == "" {
		// If not stored in metadata, get from git
		cmd = exec.Command("git", "-C", pluginDir, "rev-parse", "HEAD")
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Warning: Failed to get current commit ID for plugin %s: %v", pluginID, err)
			return false, ""
		}
		currentCommitID = strings.TrimSpace(string(output))
	}

	// Get the latest commit ID
	cmd = exec.Command("git", "-C", pluginDir, "rev-parse", "origin/"+branch)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to get latest commit ID for plugin %s: %v", pluginID, err)
		return false, ""
	}
	latestCommitID := strings.TrimSpace(string(output))

	// Update GitInfo with latest commit ID
	metadata.GitInfo.LatestCommitID = latestCommitID

	// Compare commit IDs
	if currentCommitID != latestCommitID {
		// Get number of commits behind
		cmd = exec.Command("git", "-C", pluginDir, "rev-list", "--count", currentCommitID+".."+latestCommitID)
		output, err := cmd.Output()
		commitsBehind := 0
		if err == nil {
			commitsBehind, _ = strconv.Atoi(strings.TrimSpace(string(output)))
		}

		// Get latest version string
		latestVersion := ""
		if commitsBehind > 0 {
			latestVersion = fmt.Sprintf("%s (%d commits newer)", currentVersion, commitsBehind)
		} else {
			latestVersion = fmt.Sprintf("%s (newer commit)", currentVersion)
		}

		// Try to get tag information if available
		cmd = exec.Command("git", "-C", pluginDir, "describe", "--tags", "origin/"+branch)
		tagOutput, err := cmd.Output()
		if err == nil {
			tag := strings.TrimSpace(string(tagOutput))
			latestVersion = tag
		}

		return true, latestVersion
	}

	return false, ""
}

// validatePlugin validates that a directory contains a valid plugin
func (pi *PluginInstaller) validatePlugin(dir string) (PluginMetadata, error) {
	// Check if plugin.json exists
	jsonPath := filepath.Join(dir, "plugin.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin.json not found, not a valid plugin")
	}

	// Check if plugin.go exists
	goPath := filepath.Join(dir, "plugin.go")
	if _, err := os.Stat(goPath); os.IsNotExist(err) {
		return PluginMetadata{}, fmt.Errorf("plugin.go not found, not a valid plugin")
	}

	// Read plugin.json
	metadata, err := pi.readPluginMetadata(dir)
	if err != nil {
		return PluginMetadata{}, err
	}

	// Validate required fields
	if metadata.ID == "" {
		return PluginMetadata{}, fmt.Errorf("plugin ID is missing in plugin.json")
	}

	if metadata.Name == "" {
		return PluginMetadata{}, fmt.Errorf("plugin name is missing in plugin.json")
	}

	if metadata.Description == "" {
		return PluginMetadata{}, fmt.Errorf("plugin description is missing in plugin.json")
	}

	// Set defaults for optional fields
	if metadata.Icon == "" {
		metadata.Icon = "plugin" // Default icon
	}

	if metadata.Version == "" {
		metadata.Version = "1.0.0" // Default version
	}

	return metadata, nil
}

// extractZip extracts a ZIP file to a directory
func (pi *PluginInstaller) extractZip(zipPath, destDir string) error {
	// Open the ZIP file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %v", err)
	}
	defer reader.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	// Extract each file
	for _, file := range reader.File {
		// Validate file path to prevent zip slip vulnerability
		filePath := filepath.Join(destDir, file.Name)
		if !strings.HasPrefix(filePath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path in zip: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// Create directory
			if err := os.MkdirAll(filePath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
			continue
		}

		// Create parent directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %v", err)
		}

		// Extract file
		fileReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %v", err)
		}

		targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fileReader.Close()
			return fmt.Errorf("failed to create file: %v", err)
		}

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			fileReader.Close()
			targetFile.Close()
			return fmt.Errorf("failed to extract file: %v", err)
		}

		fileReader.Close()
		targetFile.Close()
	}

	return nil
}

// copyDir copies a directory recursively
func (pi *PluginInstaller) copyDir(src, dst string) error {
	// Get file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Skip .git directory if it exists
			if entry.Name() == ".git" {
				continue
			}

			// Recursively copy subdirectory
			if err := pi.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			srcFile, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			dstFile, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer dstFile.Close()

			if _, err := io.Copy(dstFile, srcFile); err != nil {
				return err
			}

			// Set file mode
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if err := os.Chmod(dstPath, info.Mode()); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildPlugin builds a plugin if it has a go.mod file
func (pi *PluginInstaller) buildPlugin(pluginDir string) error {
	// Check if go.mod exists
	goModPath := filepath.Join(pluginDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return nil // No go.mod, no need to build
	}

	// Run go mod tidy
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = pluginDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run go mod tidy: %v", err)
	}

	// Check if there's a Makefile
	makefilePath := filepath.Join(pluginDir, "Makefile")
	if _, err := os.Stat(makefilePath); !os.IsNotExist(err) {
		// Run make
		cmd = exec.Command("make")
		cmd.Dir = pluginDir
		return cmd.Run()
	}

	// Otherwise try to build with go build
	cmd = exec.Command("go", "build", "-o", "plugin.so", "-buildmode=plugin", ".")
	cmd.Dir = pluginDir
	return cmd.Run()
}

// InstallPluginFromRepository installs a plugin from a GitHub repository
func (pi *PluginInstaller) InstallPluginFromRepository(repository string) error { // Extract organization and repo name from repository URL
	// Example: https://github.com/NetScout-Go/Plugin_ping
	parts := strings.Split(repository, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid repository URL format")
	}

	// Get the repo name (last part)
	repoName := parts[len(parts)-1]

	// Check if the repository is a valid plugin repository
	if !strings.HasPrefix(repoName, "Plugin_") {
		return fmt.Errorf("repository name must start with 'Plugin_'")
	}

	// Extract plugin ID
	pluginID := strings.TrimPrefix(repoName, "Plugin_")

	// Create plugin directory if it doesn't exist
	pluginDir := filepath.Join(pi.pluginsDir, pluginID)
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			return fmt.Errorf("failed to create plugin directory: %v", err)
		}
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", repository, pluginDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %v", err)
	}

	// Refresh the plugins
	if err := pi.manager.RefreshPlugins(); err != nil {
		return fmt.Errorf("failed to refresh plugins: %v", err)
	}

	return nil
}

// BulkInstallResult represents the result of a bulk installation
type BulkInstallResult struct {
	PluginID string         `json:"pluginId"`
	Plugin   PluginMetadata `json:"plugin,omitempty"`
	Success  bool           `json:"success"`
	Error    string         `json:"error,omitempty"`
}

// BulkInstallResponse represents the response from bulk installation
type BulkInstallResponse struct {
	Results        []BulkInstallResult `json:"results"`
	TotalPlugins   int                 `json:"totalPlugins"`
	SuccessCount   int                 `json:"successCount"`
	FailureCount   int                 `json:"failureCount"`
	OverallSuccess bool                `json:"overallSuccess"`
}

// BulkInstallPlugins installs multiple plugins from a list of repositories
func (pi *PluginInstaller) BulkInstallPlugins(repositories []string) BulkInstallResponse {
	results := make([]BulkInstallResult, 0, len(repositories))
	successCount := 0
	failureCount := 0

	for _, repo := range repositories {
		result := BulkInstallResult{
			PluginID: extractPluginIDFromRepo(repo),
		}

		plugin, err := pi.InstallPlugin(repo)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
			failureCount++
		} else {
			result.Success = true
			result.Plugin = plugin
			successCount++
		}

		results = append(results, result)
	}

	return BulkInstallResponse{
		Results:        results,
		TotalPlugins:   len(repositories),
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		OverallSuccess: failureCount == 0,
	}
}

// extractPluginIDFromRepo extracts plugin ID from repository URL
func extractPluginIDFromRepo(repo string) string {
	// Extract plugin name from repository URL
	parts := strings.Split(repo, "/")
	if len(parts) > 0 {
		pluginName := parts[len(parts)-1]
		// Remove .git suffix if present
		if strings.HasSuffix(pluginName, ".git") {
			pluginName = pluginName[:len(pluginName)-4]
		}
		// Remove Plugin_ prefix if present to get clean ID
		if strings.HasPrefix(pluginName, "Plugin_") {
			pluginName = pluginName[7:]
		}
		return strings.ToLower(pluginName)
	}
	return repo
}
