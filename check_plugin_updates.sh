#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}========================================================${NC}"
echo -e "${BLUE}         NetScout-Go Plugin Update Checker              ${NC}"
echo -e "${BLUE}========================================================${NC}"

# Configuration
PLUGINS_DIR="app/plugins/plugins"
GITHUB_ORG="NetScout-Go"
PLUGIN_PREFIX="Plugin_"

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
    echo -e "${RED}Error: Plugins directory not found: $PLUGINS_DIR${NC}"
    exit 1
fi

# Get list of plugin directories
plugin_dirs=$(find "$PLUGINS_DIR" -maxdepth 1 -type d | grep -v "^$PLUGINS_DIR$")

# Function to convert plugin directory name to repository name
# Example: ping -> Plugin_ping
function get_repo_name() {
    local plugin_name="$1"
    echo "${PLUGIN_PREFIX}${plugin_name}"
}

# Function to check if a repository has updates
function check_for_updates() {
    local plugin_dir="$1"
    local plugin_name=$(basename "$plugin_dir")
    local plugin_json="$plugin_dir/plugin.json"
    local repo_name=$(get_repo_name "$plugin_name")
    
    echo -e "${BLUE}Checking updates for ${plugin_name}...${NC}"
    
    # Check if plugin.json exists
    if [ ! -f "$plugin_json" ]; then
        echo -e "${YELLOW}Warning: plugin.json not found in $plugin_dir${NC}"
        return
    fi
    
    # Get current version from plugin.json
    local current_version=$(grep -o '"version": *"[^"]*"' "$plugin_json" | cut -d'"' -f4)
    if [ -z "$current_version" ]; then
        echo -e "${YELLOW}Warning: No version found in $plugin_json${NC}"
        return
    fi
    
    echo -e "Current version: ${GREEN}$current_version${NC}"
    
    # Check if the repository exists
    if ! gh repo view "$GITHUB_ORG/$repo_name" &> /dev/null; then
        echo -e "${YELLOW}Repository $GITHUB_ORG/$repo_name not found${NC}"
        return
    }
    
    # Check for tags in the repository
    local latest_tag=$(gh api "repos/$GITHUB_ORG/$repo_name/tags" --jq '.[0].name' 2>/dev/null)
    
    if [ -z "$latest_tag" ]; then
        echo -e "${YELLOW}No tags found in repository $GITHUB_ORG/$repo_name${NC}"
        return
    }
    
    echo -e "Latest version: ${GREEN}$latest_tag${NC}"
    
    # Compare versions
    if [ "$current_version" != "$latest_tag" ]; then
        echo -e "${GREEN}Update available: $current_version -> $latest_tag${NC}"
    else
        echo -e "${GREEN}Plugin is up to date${NC}"
    fi
}

# Process each plugin directory
for plugin_dir in $plugin_dirs; do
    check_for_updates "$plugin_dir"
    echo -e "${BLUE}--------------------------------------------------------${NC}"
done

echo -e "${BLUE}========================================================${NC}"
echo -e "${GREEN}Plugin update check completed${NC}"
echo -e "${BLUE}========================================================${NC}"
