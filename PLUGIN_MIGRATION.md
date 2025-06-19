# NetScout-Go Plugin Migration Guide

This document describes the process of migrating NetScout-Go plugins to individual GitHub repositories, installing them, and updating the application to use the new plugin system.

## Overview

NetScout-Go is transitioning from bundled plugins to external plugins hosted in individual GitHub repositories. This change allows for:

1. Independent development and versioning of plugins
2. Easier contribution from the community
3. More flexible plugin management
4. Simplified main codebase

## Plugin Repository Structure

Each plugin is now stored in its own GitHub repository with the following naming convention:

```text
Plugin_<plugin_name>
```

For example, the `ping` plugin is stored in the `Plugin_ping` repository.

## Migration Tools

Several scripts have been created to facilitate the migration process:

### 1. `list-plugins.sh`

Lists all local plugins and their migration status. It also shows remote plugins that are available in the GitHub organization.

```bash
./list-plugins.sh
```

### 2. `migrate-plugins.sh`

Migrates local plugins to GitHub repositories. This script:

- Creates a new GitHub repository for each plugin
- Pushes the plugin code to the repository
- Updates the local plugin to use the GitHub repository

```bash
./migrate-plugins.sh
```

### 3. `install-plugins.sh`

Installs plugins from GitHub repositories. This script:

- Lists all available plugins in the GitHub organization
- Allows you to install all plugins or select specific ones
- Clones the repositories to the appropriate location

```bash
./install-plugins.sh
```

### 4. `update-plugin-loader.sh`

Updates the plugin loader to use only external plugins. This script:

- Creates a backup of the original loader
- Replaces it with a new loader that supports external plugins

```bash
./update-plugin-loader.sh
```

## Migration Process

To migrate the plugin system, follow these steps:

1. List your current plugins to see what needs to be migrated:

   ```bash
   ./list-plugins.sh
   ```

2. Migrate local plugins to GitHub repositories:

   ```bash
   ./migrate-plugins.sh
   ```

3. Update the plugin loader to use only external plugins:

   ```bash
   ./update-plugin-loader.sh
   ```

4. Test the application to ensure everything works correctly

## Installing Plugins

To install plugins from GitHub repositories:

```bash
./install-plugins.sh
```

This script will show you a list of available plugins and allow you to install all or select specific ones.

## Creating New Plugins

To create a new plugin:

1. Create a new repository in the NetScout-Go organization with the name `Plugin_<plugin_name>`
2. Structure the repository with at least:
   - `plugin.go` - The main plugin code
   - `plugin.json` - Plugin metadata

3. Clone the repository to the appropriate location:

   ```bash
   git clone https://github.com/NetScout-Go/Plugin_<plugin_name>.git ~/NetTool/app/plugins/plugins/<plugin_name>
   ```

## Requirements

- GitHub CLI (`gh`) installed and authenticated
- Git installed
- Access to the NetScout-Go GitHub organization

## Troubleshooting

- If you encounter permission issues, make sure you're authenticated with GitHub CLI:

  ```bash
  gh auth login
  ```

- If the plugin loader doesn't recognize your plugins, make sure they're installed in the correct location and have the required `plugin.json` file.

- If a plugin doesn't work after migration, check the plugin code and ensure it follows the plugin interface requirements.
