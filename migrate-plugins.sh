#!/bin/bash

# Script to migrate plugins to GitHub repositories under the NetScout-Go organization
# Also supports pulling and developing plugins

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}========================================================${NC}"
echo -e "${BLUE}      NetScout-Go Plugin Migration & Development Tool   ${NC}"
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

# Function to pull latest changes for a plugin
pull_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    local repo_name="${PLUGIN_PREFIX}${plugin_name}"
    
    echo -e "${BLUE}Pulling latest changes for plugin: ${plugin_name}${NC}"
    
    # Check if plugin directory exists
    if [ ! -d "$plugin_dir" ]; then
        echo -e "${RED}Error: Plugin directory not found: ${plugin_dir}${NC}"
        return 1
    fi
    
    # Check if it's a git repository
    if [ ! -d "$plugin_dir/.git" ]; then
        echo -e "${YELLOW}Plugin ${plugin_name} is not a Git repository.${NC}"
        
        # Check if repository exists on GitHub
        if gh repo view "$GITHUB_ORG/$repo_name" &> /dev/null; then
            echo -e "${BLUE}Repository exists on GitHub. Cloning it now...${NC}"
            # Remove local files and clone from GitHub
            rm -rf "$plugin_dir"
            gh repo clone "$GITHUB_ORG/$repo_name" "$plugin_dir"
            
            if [ $? -eq 0 ]; then
                echo -e "${GREEN}Successfully cloned repository for ${plugin_name}${NC}"
                return 0
            else
                echo -e "${RED}Failed to clone repository for ${plugin_name}${NC}"
                return 1
            fi
        else
            echo -e "${RED}No repository found for ${plugin_name} on GitHub.${NC}"
            echo -e "${YELLOW}You may want to migrate it first using option 2 in the main menu.${NC}"
            return 1
        fi
    fi
    
    # Pull latest changes
    echo -e "${BLUE}Pulling latest changes from remote repository...${NC}"
    (cd "$plugin_dir" && git pull)
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Successfully pulled latest changes for ${plugin_name}${NC}"
        return 0
    else
        echo -e "${RED}Failed to pull latest changes for ${plugin_name}${NC}"
        echo -e "${YELLOW}You may have local changes that conflict with remote changes.${NC}"
        return 1
    fi
}

# Function to handle development workflow for a plugin
develop_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    local repo_name="${PLUGIN_PREFIX}${plugin_name}"
    
    echo -e "${BLUE}Development workflow for plugin: ${plugin_name}${NC}"
    
    # Check if plugin directory exists
    if [ ! -d "$plugin_dir" ]; then
        echo -e "${RED}Error: Plugin directory not found: ${plugin_dir}${NC}"
        return 1
    fi
    
    # Check if it's a git repository
    if [ ! -d "$plugin_dir/.git" ]; then
        echo -e "${YELLOW}Plugin ${plugin_name} is not a Git repository.${NC}"
        
        # Check if repository exists on GitHub
        if gh repo view "$GITHUB_ORG/$repo_name" &> /dev/null; then
            echo -e "${BLUE}Repository exists on GitHub. Cloning it now...${NC}"
            # Remove local files and clone from GitHub
            rm -rf "$plugin_dir"
            gh repo clone "$GITHUB_ORG/$repo_name" "$plugin_dir"
            
            if [ $? -ne 0 ]; then
                echo -e "${RED}Failed to clone repository for ${plugin_name}${NC}"
                return 1
            fi
        else
            echo -e "${RED}No repository found for ${plugin_name} on GitHub.${NC}"
            echo -e "${YELLOW}You may want to migrate it first using option 2 in the main menu.${NC}"
            return 1
        fi
    fi
    
    # Development menu
    while true; do
        echo
        echo -e "${YELLOW}Development options for ${plugin_name}:${NC}"
        echo "1. Show status (git status)"
        echo "2. Pull latest changes (git pull)"
        echo "3. Stage all changes (git add .)"
        echo "4. Commit changes (git commit)"
        echo "5. Push changes (git push)"
        echo "6. View commit history (git log)"
        echo "7. Create a new branch"
        echo "8. Switch branch"
        echo "9. Return to main menu"
        read -p "Enter your choice (1-9): " dev_choice
        
        case $dev_choice in
            1)
                # Show status
                echo -e "${BLUE}Git status for ${plugin_name}:${NC}"
                (cd "$plugin_dir" && git status)
                ;;
            2)
                # Pull latest changes
                echo -e "${BLUE}Pulling latest changes for ${plugin_name}...${NC}"
                (cd "$plugin_dir" && git pull)
                ;;
            3)
                # Stage all changes
                echo -e "${BLUE}Staging all changes for ${plugin_name}...${NC}"
                (cd "$plugin_dir" && git add .)
                echo -e "${GREEN}Changes staged.${NC}"
                ;;
            4)
                # Commit changes
                echo -e "${BLUE}Committing changes for ${plugin_name}...${NC}"
                read -p "Enter commit message: " commit_msg
                (cd "$plugin_dir" && git commit -m "$commit_msg")
                ;;
            5)
                # Push changes
                echo -e "${BLUE}Pushing changes for ${plugin_name}...${NC}"
                (cd "$plugin_dir" && git push)
                ;;
            6)
                # View commit history
                echo -e "${BLUE}Commit history for ${plugin_name}:${NC}"
                (cd "$plugin_dir" && git log --oneline -n 10)
                read -p "Press Enter to continue..." dummy
                ;;
            7)
                # Create a new branch
                read -p "Enter new branch name: " branch_name
                echo -e "${BLUE}Creating branch ${branch_name} for ${plugin_name}...${NC}"
                (cd "$plugin_dir" && git checkout -b "$branch_name")
                ;;
            8)
                # Switch branch
                echo -e "${BLUE}Available branches for ${plugin_name}:${NC}"
                (cd "$plugin_dir" && git branch)
                read -p "Enter branch name to switch to: " branch_name
                echo -e "${BLUE}Switching to branch ${branch_name}...${NC}"
                (cd "$plugin_dir" && git checkout "$branch_name")
                ;;
            9)
                # Return to main menu
                return 0
                ;;
            *)
                echo -e "${RED}Invalid choice${NC}"
                ;;
        esac
    done
}

# Function to pull all plugins
pull_all_plugins() {
    echo -e "${BLUE}Pulling all plugins from GitHub...${NC}"
    local plugin_count=0
    local success_count=0
    
    for plugin in $plugins; do
        ((plugin_count++))
        
        echo -e "${BLUE}[${plugin_count}/${total_plugins}] Pulling ${plugin}...${NC}"
        
        pull_plugin "$plugin"
        
        if [ $? -eq 0 ]; then
            ((success_count++))
        fi
        
        echo
    done
    
    echo -e "${GREEN}Pull complete! ${success_count}/${plugin_count} plugins updated successfully.${NC}"
}

# List available plugins
echo -e "${BLUE}Available plugins:${NC}"
plugins=$(find "$PLUGINS_DIR" -mindepth 1 -maxdepth 1 -type d -not -path "*/\.*" -printf "%f\n" | sort)
total_plugins=$(echo "$plugins" | wc -l)

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
        echo -e "${counter}. ${GREEN}${plugin}${NC} (Git repository)"
    else
        echo -e "${counter}. ${YELLOW}${plugin}${NC} (Local files only)"
    fi
    ((counter++))
done

# Main menu
while true; do
    echo
    echo -e "${YELLOW}What would you like to do?${NC}"
    echo "1. Pull all plugins from GitHub"
    echo "2. Migrate plugins to GitHub"
    echo "3. Develop a plugin"
    echo "4. Exit"
    read -p "Enter your choice (1-4): " choice

    case $choice in
        1)
            # Pull all plugins
            pull_all_plugins
            ;;
        2)
            # Submenu for migration
            echo
            echo -e "${YELLOW}Migration options:${NC}"
            echo "1. Migrate all plugins"
            echo "2. Migrate specific plugins"
            echo "3. Return to main menu"
            read -p "Enter your choice (1-3): " migrate_choice
            
            case $migrate_choice in
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
                        if ! [[ "$num" =~ ^[0-9]+$ ]] || [ "$num" -lt 1 ] || [ "$num" -gt "$total_plugins" ]; then
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
                    # Return to main menu
                    continue
                    ;;
                *)
                    echo -e "${RED}Invalid choice${NC}"
                    ;;
            esac
            ;;
        3)
            # Develop a plugin
            echo -e "${BLUE}Enter the number of the plugin you want to develop:${NC}"
            read -p "Plugin number: " plugin_number
            
            if ! [[ "$plugin_number" =~ ^[0-9]+$ ]] || [ "$plugin_number" -lt 1 ] || [ "$plugin_number" -gt "$total_plugins" ]; then
                echo -e "${RED}Invalid selection: ${plugin_number}${NC}"
                continue
            fi
            
            # Get plugin name by index
            plugin=$(echo "$plugins" | sed -n "${plugin_number}p")
            
            develop_plugin "$plugin"
            ;;
        4)
            echo -e "${BLUE}Exiting...${NC}"
            # Clean up
            rm -rf "$TEMP_DIR"
            exit 0
            ;;
        *)
            echo -e "${RED}Invalid choice${NC}"
            ;;
    esac
done
