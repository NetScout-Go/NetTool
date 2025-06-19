#!/bin/bash

# Script to list and clone all plugins from the NetScout-Go GitHub organization
# and install them in the appropriate location

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}========================================================${NC}"
echo -e "${BLUE}         NetScout-Go Plugin Installation Tool           ${NC}"
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

# Create plugins directory if it doesn't exist
mkdir -p "$PLUGINS_DIR"

# Create a temp directory
mkdir -p "$TEMP_DIR"

# Function to convert plugin repo name to plugin directory name
# Example: Plugin_ping -> ping
function get_plugin_name() {
    local repo_name="$1"
    echo "$repo_name" | sed "s/^${PLUGIN_PREFIX//\//\\/}//i"
}

# Get list of repos from the organization
echo -e "${BLUE}Fetching plugin repositories from ${GITHUB_ORG}...${NC}"
repos=$(gh repo list "$GITHUB_ORG" --limit 100 --json name --jq '.[].name' | grep "^$PLUGIN_PREFIX" || true)

if [ -z "$repos" ]; then
    echo -e "${RED}No plugin repositories found in ${GITHUB_ORG} organization${NC}"
    echo -e "Make sure you have access to the organization and the repositories exist"
    exit 1
fi

# Count repositories
repo_count=$(echo "$repos" | wc -l)
echo -e "${GREEN}Found ${repo_count} plugin repositories${NC}"

# Print the list of plugins
echo -e "${BLUE}Available plugins:${NC}"
counter=1
for repo in $repos; do
    plugin_name=$(get_plugin_name "$repo")
    echo -e "${counter}. ${GREEN}${plugin_name}${NC} (${repo})"
    ((counter++))
done

# Ask if user wants to install all plugins
echo
echo -e "${YELLOW}What would you like to do?${NC}"
echo "1. Install all plugins"
echo "2. Select specific plugins to install"
echo "3. Exit"
read -p "Enter your choice (1-3): " choice

case $choice in
    1)
        # Install all plugins
        echo -e "${BLUE}Installing all plugins...${NC}"
        for repo in $repos; do
            plugin_name=$(get_plugin_name "$repo")
            plugin_dir="$PLUGINS_DIR/$plugin_name"
            
            echo -e "${YELLOW}Installing ${plugin_name}...${NC}"
            
            # Check if plugin directory already exists
            if [ -d "$plugin_dir" ]; then
                echo -e "${YELLOW}Plugin directory already exists. Updating...${NC}"
                # Pull latest changes if it's a git repository
                if [ -d "$plugin_dir/.git" ]; then
                    (cd "$plugin_dir" && git pull)
                else
                    # If not a git repo, remove and clone
                    rm -rf "$plugin_dir"
                    gh repo clone "$GITHUB_ORG/$repo" "$plugin_dir"
                fi
            else
                # Clone the repository
                gh repo clone "$GITHUB_ORG/$repo" "$plugin_dir"
            fi
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✓ ${plugin_name} installed successfully${NC}"
            else
                echo -e "${RED}✗ Failed to install ${plugin_name}${NC}"
            fi
            echo
        done
        ;;
    2)
        # Select specific plugins
        echo -e "${BLUE}Enter the numbers of the plugins you want to install (space-separated):${NC}"
        read -p "Plugin numbers: " plugin_numbers
        
        for num in $plugin_numbers; do
            if ! [[ "$num" =~ ^[0-9]+$ ]] || [ "$num" -lt 1 ] || [ "$num" -gt "$repo_count" ]; then
                echo -e "${RED}Invalid selection: ${num}${NC}"
                continue
            fi
            
            # Get repo name by index
            repo=$(echo "$repos" | sed -n "${num}p")
            plugin_name=$(get_plugin_name "$repo")
            plugin_dir="$PLUGINS_DIR/$plugin_name"
            
            echo -e "${YELLOW}Installing ${plugin_name}...${NC}"
            
            # Check if plugin directory already exists
            if [ -d "$plugin_dir" ]; then
                echo -e "${YELLOW}Plugin directory already exists. Updating...${NC}"
                # Pull latest changes if it's a git repository
                if [ -d "$plugin_dir/.git" ]; then
                    (cd "$plugin_dir" && git pull)
                else
                    # If not a git repo, remove and clone
                    rm -rf "$plugin_dir"
                    gh repo clone "$GITHUB_ORG/$repo" "$plugin_dir"
                fi
            else
                # Clone the repository
                gh repo clone "$GITHUB_ORG/$repo" "$plugin_dir"
            fi
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}✓ ${plugin_name} installed successfully${NC}"
            else
                echo -e "${RED}✗ Failed to install ${plugin_name}${NC}"
            fi
            echo
        done
        ;;
    3)
        echo -e "${BLUE}Exiting without installing any plugins${NC}"
        exit 0
        ;;
    *)
        echo -e "${RED}Invalid choice${NC}"
        exit 1
        ;;
esac

# Clean up
rm -rf "$TEMP_DIR"

echo -e "${GREEN}Plugin installation complete!${NC}"
echo -e "${YELLOW}Don't forget to update the plugin loader if needed.${NC}"
