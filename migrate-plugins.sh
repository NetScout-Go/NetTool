#!/bin/bash

# Script to migrate plugins to GitHub repositories under the NetScout-Go organization

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}========================================================${NC}"
echo -e "${BLUE}          NetScout-Go Plugin Migration Tool            ${NC}"
echo -e "${BLUE}========================================================${NC}"

# Configuration
GITHUB_ORG="NetScout-Go"
PLUGIN_PREFIX="Plugin_"
PLUGINS_DIR="$HOME/NetTool/app/plugins/plugins"
TEMP_DIR="/tmp/netscout_plugins"

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI (gh) is not installed${NC}"
    echo -e "Please install it with: ${YELLOW}sudo apt install gh${NC} or see https://cli.github.com/"
    echo -e "After installation, authenticate with: ${YELLOW}gh auth login${NC}"
    exit 1
fi

# Check if gh is authenticated
if ! gh auth status &> /dev/null; then
    echo -e "${RED}Error: GitHub CLI is not authenticated${NC}"
    echo -e "Please run: ${YELLOW}gh auth login${NC}"
    exit 1
fi

# Check if plugins directory exists
if [ ! -d "$PLUGINS_DIR" ]; then
    echo -e "${RED}Error: Plugins directory not found: ${PLUGINS_DIR}${NC}"
    exit 1
fi

# Create temp directory
mkdir -p "$TEMP_DIR"

# Function to generate README file for a plugin
generate_readme() {
    local plugin_name="$1"
    local plugin_dir="$2"
    local description=""
    
    # Extract description from plugin.json if it exists
    if [ -f "$plugin_dir/plugin.json" ]; then
        description=$(grep -o '"description"[[:space:]]*:[[:space:]]*"[^"]*"' "$plugin_dir/plugin.json" | cut -d '"' -f 4 || echo "")
    fi
    
    # Use description or default text
    if [ -z "$description" ]; then
        description="A plugin for NetScout-Go network toolkit."
    fi
    
    # Generate README.md
    cat > "$plugin_dir/README.md" << EOF
# ${PLUGIN_PREFIX}${plugin_name}

${description}

## Description

This plugin is part of the NetScout-Go network toolkit. It provides network diagnostics and utilities for the main NetScout-Go application.

## Installation

To install this plugin, use the plugin installer from the main NetScout-Go repository:

\`\`\`bash
cd ~/NetTool
./install-plugins.sh
\`\`\`

Or clone it directly:

\`\`\`bash
git clone https://github.com/${GITHUB_ORG}/${PLUGIN_PREFIX}${plugin_name}.git ~/NetTool/app/plugins/plugins/${plugin_name}
\`\`\`

## Usage

This plugin is loaded automatically by the NetScout-Go application when installed in the correct directory.
EOF
}

# Function to create and push a plugin repository
migrate_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    local repo_name="${PLUGIN_PREFIX}${plugin_name}"
    local repo_temp_dir="$TEMP_DIR/$repo_name"
    
    echo -e "${BLUE}Migrating plugin: ${plugin_name}${NC}"
    
    # Check if plugin directory exists
    if [ ! -d "$plugin_dir" ]; then
        echo -e "${RED}Error: Plugin directory not found: ${plugin_dir}${NC}"
        return 1
    fi
    
    # Check if plugin.json exists
    if [ ! -f "$plugin_dir/plugin.json" ]; then
        echo -e "${RED}Error: Missing plugin.json in ${plugin_dir}${NC}"
        return 1
    fi
    
    # Check if repository already exists on GitHub
    if gh repo view "$GITHUB_ORG/$repo_name" &> /dev/null; then
        echo -e "${YELLOW}Repository already exists: ${GITHUB_ORG}/${repo_name}${NC}"
        
        # Ask if user wants to update existing repository
        read -p "Update existing repository? (y/n): " update_repo
        if [ "$update_repo" != "y" ]; then
            echo -e "${YELLOW}Skipping ${plugin_name}${NC}"
            return 0
        fi
        
        # Check if local plugin is already a git repository
        if [ -d "$plugin_dir/.git" ]; then
            # Pull latest changes from remote
            echo -e "${BLUE}Updating local repository...${NC}"
            (cd "$plugin_dir" && git pull)
            return $?
        else
            # Remove local plugin and clone from GitHub
            echo -e "${BLUE}Replacing local files with GitHub repository...${NC}"
            rm -rf "$plugin_dir"
            gh repo clone "$GITHUB_ORG/$repo_name" "$plugin_dir"
            return $?
        fi
    fi
    
    # Create temporary directory for the plugin
    mkdir -p "$repo_temp_dir"
    
    # Copy plugin files to temporary directory
    cp -r "$plugin_dir"/* "$repo_temp_dir/"
    
    # Generate README.md if it doesn't exist
    if [ ! -f "$repo_temp_dir/README.md" ]; then
        generate_readme "$plugin_name" "$repo_temp_dir"
    fi
    
    # Create git repository
    (
        cd "$repo_temp_dir" && 
        git init && 
        git add . && 
        git commit -m "Initial commit: Migrated from NetTool core repository"
    )
    
    # Create GitHub repository
    echo -e "${BLUE}Creating GitHub repository: ${GITHUB_ORG}/${repo_name}${NC}"
    gh repo create "$GITHUB_ORG/$repo_name" --public --description "NetScout-Go plugin: $plugin_name" --source "$repo_temp_dir" --push
    
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to create GitHub repository${NC}"
        return 1
    fi
    
    # Replace local plugin with the GitHub repository
    echo -e "${BLUE}Replacing local plugin with GitHub repository...${NC}"
    rm -rf "$plugin_dir"
    gh repo clone "$GITHUB_ORG/$repo_name" "$plugin_dir"
    
    echo -e "${GREEN}Plugin ${plugin_name} successfully migrated to GitHub${NC}"
    return 0
}

# List available plugins
echo -e "${BLUE}Available plugins for migration:${NC}"
plugins=$(find "$PLUGINS_DIR" -mindepth 1 -maxdepth 1 -type d -not -path "*/\.*" -printf "%f\n" | sort)

# Check if any plugins were found
if [ -z "$plugins" ]; then
    echo -e "${RED}No plugins found in ${PLUGINS_DIR}${NC}"
    exit 1
fi

# Display plugins with numbers
counter=1
for plugin in $plugins; do
    # Check if it's already a git repository
    if [ -d "$PLUGINS_DIR/$plugin/.git" ]; then
        echo -e "${counter}. ${GREEN}${plugin}${NC} (Already a Git repository)"
    else
        echo -e "${counter}. ${YELLOW}${plugin}${NC} (Needs migration)"
    fi
    ((counter++))
done

# Ask what user wants to do
echo
echo -e "${YELLOW}What would you like to do?${NC}"
echo "1. Migrate all plugins"
echo "2. Migrate specific plugins"
echo "3. Exit"
read -p "Enter your choice (1-3): " choice

case $choice in
    1)
        # Migrate all plugins
        echo -e "${BLUE}Migrating all plugins...${NC}"
        for plugin in $plugins; do
            # Skip if it's already a git repository
            if [ -d "$PLUGINS_DIR/$plugin/.git" ]; then
                echo -e "${GREEN}Skipping ${plugin} (Already a Git repository)${NC}"
                continue
            fi
            
            migrate_plugin "$plugin"
            echo
        done
        ;;
    2)
        # Migrate specific plugins
        echo -e "${BLUE}Enter the numbers of the plugins you want to migrate (space-separated):${NC}"
        read -p "Plugin numbers: " plugin_numbers
        
        for num in $plugin_numbers; do
            if ! [[ "$num" =~ ^[0-9]+$ ]] || [ "$num" -lt 1 ] || [ "$num" -gt "$(echo "$plugins" | wc -l)" ]; then
                echo -e "${RED}Invalid selection: ${num}${NC}"
                continue
            fi
            
            # Get plugin name by index
            plugin=$(echo "$plugins" | sed -n "${num}p")
            
            # Skip if it's already a git repository
            if [ -d "$PLUGINS_DIR/$plugin/.git" ]; then
                echo -e "${GREEN}Skipping ${plugin} (Already a Git repository)${NC}"
                continue
            fi
            
            migrate_plugin "$plugin"
            echo
        done
        ;;
    3)
        echo -e "${BLUE}Exiting without migrating any plugins${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

# Clean up
rm -rf "$TEMP_DIR"

echo -e "${GREEN}Plugin migration complete!${NC}"
echo -e "${YELLOW}Don't forget to update the plugin loader to use external plugins.${NC}"
