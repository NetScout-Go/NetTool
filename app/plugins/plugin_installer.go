package plugins

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Description     string       `json:"description"`
	Version         string       `json:"version"`
	Author          string       `json:"author"`
	License         string       `json:"license"`
	Icon            string       `json:"icon"`
	Status          string       `json:"status"`
	UpdateAvailable bool         `json:"updateAvailable"`
	LatestVersion   string       `json:"latestVersion,omitempty"`
	Path            string       `json:"path,omitempty"`
	Dependencies    []Dependency `json:"dependencies,omitempty"`
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
}

// NewPluginInstaller creates a new plugin installer
func NewPluginInstaller(pluginsDir string, manager *PluginManager) *PluginInstaller {
	return &PluginInstaller{
		pluginsDir: pluginsDir,
		manager:    manager,
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
	tempDir, err := ioutil.TempDir("", "nettool-plugin-")
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
	tempDir, err := ioutil.TempDir("", "nettool-plugin-upload-")
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
	data, err := ioutil.ReadFile(jsonPath)
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

	if err := ioutil.WriteFile(jsonPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated plugin.json: %v", err)
	}

	return nil
}

// Helper functions

// readPluginMetadata reads and parses the plugin.json file
func (pi *PluginInstaller) readPluginMetadata(pluginDir string) (PluginMetadata, error) {
	// Read plugin.json
	jsonPath := filepath.Join(pluginDir, "plugin.json")
	data, err := ioutil.ReadFile(jsonPath)
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
	data, err := ioutil.ReadFile(depPath)
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

	// Fetch the latest changes without applying them
	cmd := exec.Command("git", "-C", pluginDir, "fetch")
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: Failed to fetch updates for plugin %s: %v", pluginID, err)
		return false, ""
	}

	// Check if there are changes between local and remote
	cmd = exec.Command("git", "-C", pluginDir, "rev-list", "HEAD..origin/main", "--count")
	output, err := cmd.Output()
	if err != nil {
		// Try with master branch instead
		cmd = exec.Command("git", "-C", pluginDir, "rev-list", "HEAD..origin/master", "--count")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Warning: Failed to check updates for plugin %s: %v", pluginID, err)
			return false, ""
		}
	}

	// Parse the number of commits behind
	commitsBehind, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		log.Printf("Warning: Failed to parse update count for plugin %s: %v", pluginID, err)
		return false, ""
	}

	if commitsBehind > 0 {
		// Get the latest tag if available
		cmd = exec.Command("git", "-C", pluginDir, "describe", "--tags", "origin/main")
		tagOutput, err := cmd.Output()
		if err != nil {
			// Try with master branch
			cmd = exec.Command("git", "-C", pluginDir, "describe", "--tags", "origin/master")
			tagOutput, err = cmd.Output()
		}

		latestVersion := ""
		if err == nil {
			latestVersion = strings.TrimSpace(string(tagOutput))
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
	entries, err := ioutil.ReadDir(src)
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
			if err := os.Chmod(dstPath, entry.Mode()); err != nil {
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
