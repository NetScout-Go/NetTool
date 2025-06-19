#!/bin/bash

# Script to list all current plugins and their status

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}========================================================${NC}"
echo -e "${BLUE}          NetScout-Go Plugin Status Tool               ${NC}"
echo -e "${BLUE}========================================================${NC}"

# Configuration
GITHUB_ORG="NetScout-Go"
PLUGIN_PREFIX="Plugin_"
PLUGINS_DIR="$HOME/NetTool/app/plugins/plugins"

# Check if plugins directory exists
if [ ! -d "$PLUGINS_DIR" ]; then
    echo -e "${RED}Error: Plugins directory not found: ${PLUGINS_DIR}${NC}"
    exit 1
fi

# Check if gh CLI is installed (for remote plugin status)
has_gh=true
if ! command -v gh &> /dev/null; then
    echo -e "${YELLOW}Warning: GitHub CLI (gh) is not installed${NC}"
    echo -e "Install it with: ${YELLOW}sudo apt install gh${NC} for complete functionality"
    has_gh=false
fi

# Check if gh is authenticated
gh_auth=true
if $has_gh && ! gh auth status &> /dev/null; then
    echo -e "${YELLOW}Warning: GitHub CLI is not authenticated${NC}"
    echo -e "Run: ${YELLOW}gh auth login${NC} for complete functionality"
    gh_auth=false
fi

# List local plugins
echo -e "${BLUE}Local plugins:${NC}"
local_plugins=$(find "$PLUGINS_DIR" -mindepth 1 -maxdepth 1 -type d -printf "%f\n" | sort)

# Display local plugins
for plugin in $local_plugins; do
    # Skip if directory name starts with a dot
    [[ "$plugin" == .* ]] && continue
    
    # Check if plugin.json exists
    if [ -f "$PLUGINS_DIR/$plugin/plugin.json" ]; then
        if [ -d "$PLUGINS_DIR/$plugin/.git" ]; then
            echo -e "${GREEN}✓ ${plugin}${NC} (Git repository)"
        else
            echo -e "${YELLOW}! ${plugin}${NC} (Local only)"
        fi
    else
        echo -e "${RED}✗ ${plugin}${NC} (Invalid plugin - missing plugin.json)"
    fi
done

# List remote plugins if gh is available
if $has_gh && $gh_auth; then
    echo -e "\n${BLUE}Remote plugins in ${GITHUB_ORG} organization:${NC}"
    remote_repos=$(gh repo list "$GITHUB_ORG" --limit 100 --json name --jq '.[].name' | grep "^$PLUGIN_PREFIX" || true)
    
    if [ -z "$remote_repos" ]; then
        echo -e "${YELLOW}No plugin repositories found in ${GITHUB_ORG} organization${NC}"
    else
        for repo in $remote_repos; do
            plugin_name=$(echo "$repo" | sed "s/^${PLUGIN_PREFIX//\//\\/}//i")
            if [ -d "$PLUGINS_DIR/$plugin_name" ]; then
                echo -e "${GREEN}✓ ${plugin_name}${NC} (Installed)"
            else
                echo -e "${RED}✗ ${plugin_name}${NC} (Not installed)"
            fi
        done
    fi
    
    # Show plugins that need to be migrated
    echo -e "\n${BLUE}Plugins that need to be migrated to GitHub:${NC}"
    for plugin in $local_plugins; do
        # Skip if directory name starts with a dot
        [[ "$plugin" == .* ]] && continue
        
        # Skip if not a valid plugin
        [ ! -f "$PLUGINS_DIR/$plugin/plugin.json" ] && continue
        
        # Check if it's already a git repo
        if [ ! -d "$PLUGINS_DIR/$plugin/.git" ]; then
            repo_name="${PLUGIN_PREFIX}${plugin}"
            remote_exists=$(echo "$remote_repos" | grep -x "$repo_name" || true)
            
            if [ -z "$remote_exists" ]; then
                echo -e "${YELLOW}! ${plugin}${NC} (Needs migration)"
            fi
        fi
    done
fi

echo -e "\n${GREEN}Total local plugins: $(echo "$local_plugins" | grep -v '^\.' | wc -l)${NC}"
if $has_gh && $gh_auth; then
    echo -e "${GREEN}Total remote plugins: $(echo "$remote_repos" | wc -l || echo 0)${NC}"
fi
