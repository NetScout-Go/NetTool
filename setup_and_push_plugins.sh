#!/bin/bash

# Script to clone plugin repositories and push all changes
# This script will:
# 1. Clone all plugin repositories from GitHub
# 2. Copy existing local files to the cloned repos
# 3. Push any changes

PLUGINS_DIR="/home/anoam/NetTool/app/plugins/plugins"
BACKUP_DIR="/home/anoam/NetTool/plugin_backup_$(date +%Y%m%d_%H%M%S)"
LOG_FILE="/home/anoam/NetTool/plugin_setup_push.log"
DATA_JSON="/home/anoam/NetTool/app/plugins/data.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to log messages
log_message() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$LOG_FILE"
}

# Function to check if directory is a git repository
is_git_repo() {
    local dir="$1"
    if [ -d "$dir/.git" ]; then
        return 0
    else
        return 1
    fi
}

# Function to backup existing plugin directory
backup_plugin() {
    local plugin_name="$1"
    local plugin_path="$PLUGINS_DIR/$plugin_name"
    
    if [ -d "$plugin_path" ]; then
        echo -e "${CYAN}Backing up existing $plugin_name...${NC}"
        mkdir -p "$BACKUP_DIR"
        cp -r "$plugin_path" "$BACKUP_DIR/"
        log_message "Backed up $plugin_name to $BACKUP_DIR"
        return 0
    fi
    return 1
}

# Function to clone a plugin repository
clone_plugin() {
    local plugin_name="$1"
    local repo_url="$2"
    local plugin_path="$PLUGINS_DIR/$plugin_name"
    
    echo -e "${BLUE}Cloning $plugin_name from $repo_url...${NC}"
    log_message "Cloning $plugin_name from $repo_url"
    
    # Remove existing directory if it exists and is not a git repo
    if [ -d "$plugin_path" ] && ! is_git_repo "$plugin_path"; then
        echo -e "${YELLOW}Removing existing non-git directory: $plugin_name${NC}"
        rm -rf "$plugin_path"
    fi
    
    # Clone the repository
    if git clone "$repo_url" "$plugin_path"; then
        echo -e "${GREEN}Successfully cloned $plugin_name${NC}"
        log_message "Successfully cloned $plugin_name"
        return 0
    else
        echo -e "${RED}Failed to clone $plugin_name${NC}"
        log_message "Failed to clone $plugin_name"
        return 1
    fi
}

# Function to restore files from backup
restore_from_backup() {
    local plugin_name="$1"
    local plugin_path="$PLUGINS_DIR/$plugin_name"
    local backup_path="$BACKUP_DIR/$plugin_name"
    
    if [ -d "$backup_path" ]; then
        echo -e "${CYAN}Restoring files from backup for $plugin_name...${NC}"
        
        # Copy all files from backup, but don't overwrite .git directory
        rsync -av --exclude='.git' "$backup_path/" "$plugin_path/"
        
        log_message "Restored files from backup for $plugin_name"
        return 0
    else
        echo -e "${YELLOW}No backup found for $plugin_name${NC}"
        return 1
    fi
}

# Function to process a single plugin (commit and push)
process_plugin() {
    local plugin_dir="$1"
    local plugin_name=$(basename "$plugin_dir")
    
    echo -e "${BLUE}Processing plugin: $plugin_name${NC}"
    log_message "Processing plugin: $plugin_name"
    
    # Check if it's a git repository
    if ! is_git_repo "$plugin_dir"; then
        echo -e "${RED}Error: $plugin_name is not a git repository${NC}"
        log_message "Error: $plugin_name is not a git repository"
        return 1
    fi
    
    # Change to plugin directory
    cd "$plugin_dir" || {
        echo -e "${RED}Error: Cannot change to directory $plugin_dir${NC}"
        log_message "Error: Cannot change to directory $plugin_dir"
        return 1
    }
    
    # Check if there are any changes
    if git diff --quiet && git diff --cached --quiet; then
        echo -e "${YELLOW}No changes to commit in $plugin_name${NC}"
        log_message "No changes to commit in $plugin_name"
        return 0
    fi
    
    # Add all changes
    echo "Adding all changes..."
    if ! git add .; then
        echo -e "${RED}Error: Failed to add changes in $plugin_name${NC}"
        log_message "Error: Failed to add changes in $plugin_name"
        return 1
    fi
    
    # Check if there are staged changes
    if git diff --cached --quiet; then
        echo -e "${YELLOW}No staged changes in $plugin_name after git add${NC}"
        log_message "No staged changes in $plugin_name after git add"
        return 0
    fi
    
    # Get current branch
    current_branch=$(git branch --show-current)
    if [ -z "$current_branch" ]; then
        echo -e "${RED}Error: Cannot determine current branch in $plugin_name${NC}"
        log_message "Error: Cannot determine current branch in $plugin_name"
        return 1
    fi
    
    # Commit changes with timestamp
    commit_message="Auto-update: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Committing changes with message: $commit_message"
    if ! git commit -m "$commit_message"; then
        echo -e "${RED}Error: Failed to commit changes in $plugin_name${NC}"
        log_message "Error: Failed to commit changes in $plugin_name"
        return 1
    fi
    
    # Push changes
    echo "Pushing to origin/$current_branch..."
    if git push origin "$current_branch"; then
        echo -e "${GREEN}Successfully pushed $plugin_name${NC}"
        log_message "Successfully pushed $plugin_name"
        return 0
    else
        echo -e "${RED}Error: Failed to push $plugin_name${NC}"
        log_message "Error: Failed to push $plugin_name"
        return 1
    fi
}

# Function to extract plugin info from data.json
get_plugin_repositories() {
    if [ ! -f "$DATA_JSON" ]; then
        echo -e "${RED}Error: data.json not found at $DATA_JSON${NC}"
        return 1
    fi
    
    # Extract plugin repositories using jq (or fallback to grep/sed if jq not available)
    if command -v jq >/dev/null 2>&1; then
        jq -r '.plugins[] | "\(.id)|\(.repository)"' "$DATA_JSON"
    else
        # Fallback method using grep and sed
        grep -o '"id":[^,]*' "$DATA_JSON" | sed 's/"id":"\([^"]*\)"/\1/' > /tmp/plugin_ids.tmp
        grep -o '"repository":[^,]*' "$DATA_JSON" | sed 's/"repository":"\([^"]*\)"/\1/' > /tmp/plugin_repos.tmp
        paste -d'|' /tmp/plugin_ids.tmp /tmp/plugin_repos.tmp
        rm -f /tmp/plugin_ids.tmp /tmp/plugin_repos.tmp
    fi
}

# Main execution
main() {
    echo -e "${BLUE}Starting plugin setup and push script...${NC}"
    log_message "Starting plugin setup and push script"
    
    # Check if plugins directory exists
    if [ ! -d "$PLUGINS_DIR" ]; then
        echo -e "${RED}Error: Plugins directory $PLUGINS_DIR does not exist${NC}"
        log_message "Error: Plugins directory $PLUGINS_DIR does not exist"
        exit 1
    fi
    
    # Initialize counters
    local total_plugins=0
    local cloned_plugins=0
    local failed_clones=0
    local successful_pushes=0
    local failed_pushes=0
    local skipped_plugins=0
    
    echo -e "${CYAN}=== PHASE 1: CLONING REPOSITORIES ===${NC}"
    
    # Get plugin repositories from data.json
    while IFS='|' read -r plugin_id repo_url; do
        if [ -n "$plugin_id" ] && [ -n "$repo_url" ] && [[ "$repo_url" =~ ^https://github.com/ ]]; then
            total_plugins=$((total_plugins + 1))
            
            # Backup existing plugin if it exists
            backup_plugin "$plugin_id"
            
            # Clone the repository
            if clone_plugin "$plugin_id" "$repo_url"; then
                cloned_plugins=$((cloned_plugins + 1))
                
                # Restore files from backup if available
                restore_from_backup "$plugin_id"
            else
                failed_clones=$((failed_clones + 1))
            fi
            
            echo "" # Add blank line between plugins
        fi
    done < <(get_plugin_repositories)
    
    echo -e "${CYAN}=== PHASE 2: PUSHING CHANGES ===${NC}"
    
    # Process each plugin directory for pushing
    for plugin_path in "$PLUGINS_DIR"/*; do
        if [ -d "$plugin_path" ]; then
            # Skip if it's just a file (like DEPENDENCIES.md)
            if [ ! -f "$plugin_path" ]; then
                if process_plugin "$plugin_path"; then
                    successful_pushes=$((successful_pushes + 1))
                else
                    # Check if it was skipped or failed
                    if is_git_repo "$plugin_path"; then
                        failed_pushes=$((failed_pushes + 1))
                    else
                        skipped_plugins=$((skipped_plugins + 1))
                    fi
                fi
                
                echo "" # Add blank line between plugins
            fi
        fi
    done
    
    # Summary
    echo -e "${BLUE}=== SUMMARY ===${NC}"
    echo -e "Total plugins in data.json: $total_plugins"
    echo -e "${GREEN}Successfully cloned: $cloned_plugins${NC}"
    echo -e "${RED}Failed to clone: $failed_clones${NC}"
    echo -e "${GREEN}Successful pushes: $successful_pushes${NC}"
    echo -e "${RED}Failed pushes: $failed_pushes${NC}"
    echo -e "${YELLOW}Skipped (not git repos): $skipped_plugins${NC}"
    echo -e "Log file: $LOG_FILE"
    if [ -d "$BACKUP_DIR" ]; then
        echo -e "Backup directory: $BACKUP_DIR"
    fi
    
    log_message "Summary - Total: $total_plugins, Cloned: $cloned_plugins, Failed clones: $failed_clones, Push success: $successful_pushes, Push failed: $failed_pushes, Skipped: $skipped_plugins"
    
    # Return appropriate exit code
    if [ $failed_clones -gt 0 ] || [ $failed_pushes -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Check if running interactively
if [ -t 1 ]; then
    echo -e "${YELLOW}This script will:${NC}"
    echo -e "${YELLOW}1. Backup existing plugin directories${NC}"
    echo -e "${YELLOW}2. Clone all plugin repositories from GitHub${NC}"
    echo -e "${YELLOW}3. Restore your local files to the cloned repos${NC}"
    echo -e "${YELLOW}4. Add, commit, and push all changes${NC}"
    echo -e "${YELLOW}Do you want to continue? (y/N)${NC}"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        echo "Aborted by user"
        exit 0
    fi
fi

# Run main function
main
