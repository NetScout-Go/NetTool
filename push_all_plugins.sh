#!/bin/bash

# Script to push all changes in plugin directories
# Each plugin directory is assumed to be a git repository

PLUGINS_DIR="/home/anoam/NetTool/app/plugins/plugins"
LOG_FILE="/home/anoam/NetTool/plugin_push.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# Function to process a single plugin
process_plugin() {
    local plugin_dir="$1"
    local plugin_name=$(basename "$plugin_dir")
    
    echo -e "${BLUE}Processing plugin: $plugin_name${NC}"
    log_message "Processing plugin: $plugin_name"
    
    # Check if it's a git repository
    if ! is_git_repo "$plugin_dir"; then
        echo -e "${YELLOW}Warning: $plugin_name is not a git repository, skipping...${NC}"
        log_message "Warning: $plugin_name is not a git repository"
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

# Main execution
main() {
    echo -e "${BLUE}Starting plugin push script...${NC}"
    log_message "Starting plugin push script"
    
    # Check if plugins directory exists
    if [ ! -d "$PLUGINS_DIR" ]; then
        echo -e "${RED}Error: Plugins directory $PLUGINS_DIR does not exist${NC}"
        log_message "Error: Plugins directory $PLUGINS_DIR does not exist"
        exit 1
    fi
    
    # Initialize counters
    local total_plugins=0
    local successful_pushes=0
    local failed_pushes=0
    local skipped_plugins=0
    
    # Process each plugin directory
    for plugin_path in "$PLUGINS_DIR"/*; do
        if [ -d "$plugin_path" ]; then
            # Skip if it's just a file (like DEPENDENCIES.md)
            if [ ! -f "$plugin_path" ]; then
                total_plugins=$((total_plugins + 1))
                
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
    echo -e "Total plugins processed: $total_plugins"
    echo -e "${GREEN}Successful pushes: $successful_pushes${NC}"
    echo -e "${RED}Failed pushes: $failed_pushes${NC}"
    echo -e "${YELLOW}Skipped (not git repos): $skipped_plugins${NC}"
    echo -e "Log file: $LOG_FILE"
    
    log_message "Summary - Total: $total_plugins, Success: $successful_pushes, Failed: $failed_pushes, Skipped: $skipped_plugins"
    
    # Return appropriate exit code
    if [ $failed_pushes -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Check if running interactively
if [ -t 1 ]; then
    echo -e "${YELLOW}This script will add and push all changes in plugin directories.${NC}"
    echo -e "${YELLOW}Do you want to continue? (y/N)${NC}"
    read -r response
    if [[ ! "$response" =~ ^[Yy]$ ]]; then
        echo "Aborted by user"
        exit 0
    fi
fi

# Run main function
main
