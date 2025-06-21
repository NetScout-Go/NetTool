#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Banner
echo -e "${BLUE}========================================================${NC}"
echo -e "${BLUE}         NetScout-Go Plugin Version Updater             ${NC}"
echo -e "${BLUE}========================================================${NC}"

# Configuration
PLUGINS_DIR="app/plugins/plugins"

# Check if plugins directory exists
if [ ! -d "$PLUGINS_DIR" ]; then
    echo -e "${RED}Error: Plugins directory not found: $PLUGINS_DIR${NC}"
    exit 1
fi

# Get list of plugin directories
plugin_dirs=$(find "$PLUGINS_DIR" -maxdepth 1 -type d | grep -v "^$PLUGINS_DIR$")

# Function to update plugin.json file
update_plugin_json() {
    local plugin_dir=$1
    local plugin_name=$(basename "$plugin_dir")
    local plugin_json="$plugin_dir/plugin.json"
    
    echo -e "${BLUE}Processing ${plugin_name}...${NC}"
    
    # Check if plugin.json exists
    if [ ! -f "$plugin_json" ]; then
        echo -e "${YELLOW}Warning: plugin.json not found in $plugin_dir${NC}"
        return
    fi
    
    # Check if version field already exists
    if grep -q '"version"' "$plugin_json"; then
        echo -e "${GREEN}Version field already exists in $plugin_json${NC}"
        return
    fi
    
    # Add version, author, and license fields
    echo -e "${GREEN}Adding version, author, and license to $plugin_json${NC}"
    
    # Create a temporary file
    temp_file=$(mktemp)
    
    # Process the JSON file to add the missing fields after the icon field
    awk -v version="1.0.0" -v author="NetScout-Go" -v license="MIT" '
    {
        # Print the current line
        print $0
        
        # If the line has "icon" field, add version, author, and license after it
        if ($0 ~ /"icon": "[^"]+"/) {
            gsub(/,$/, "");
            print "  \"icon\": " substr($0, index($0, ":")+1) ","
            print "  \"version\": \"" version "\","
            print "  \"author\": \"" author "\","
            print "  \"license\": \"" license "\","
            
            # Skip the line in the next iteration since we already processed it
            getline
        }
    }
    ' "$plugin_json" > "$temp_file"
    
    # Check if awk command succeeded
    if [ $? -ne 0 ]; then
        echo -e "${RED}Error: Failed to update $plugin_json${NC}"
        rm "$temp_file"
        return
    fi
    
    # Replace the original file
    mv "$temp_file" "$plugin_json"
    
    echo -e "${GREEN}Successfully updated $plugin_json${NC}"
}

# Process each plugin directory
for plugin_dir in $plugin_dirs; do
    update_plugin_json "$plugin_dir"
done

echo -e "${BLUE}========================================================${NC}"
echo -e "${GREEN}Plugin version update completed${NC}"
echo -e "${BLUE}========================================================${NC}"
