#!/bin/bash

# Script to update individual plugin data.json files from the main data.json
# This script extracts only the specific plugin data for each plugin directory

MAIN_DATA_JSON="/home/anoam/NetTool/app/plugins/data.json"
PLUGINS_DIR="/home/anoam/NetTool/app/plugins/plugins"

# Check if main data.json exists
if [ ! -f "$MAIN_DATA_JSON" ]; then
    echo "Error: Main data.json file not found at $MAIN_DATA_JSON"
    exit 1
fi

# Check if plugins directory exists
if [ ! -d "$PLUGINS_DIR" ]; then
    echo "Error: Plugins directory not found at $PLUGINS_DIR"
    exit 1
fi

echo "Starting plugin data.json update process..."

# Function to extract plugin data and create individual data.json
update_plugin_data() {
    local plugin_id="$1"
    local plugin_dir="$2"
    
    echo "Processing plugin: $plugin_id"
    
    # Extract the specific plugin data using jq
    plugin_data=$(jq --arg id "$plugin_id" '.plugins[] | select(.id == $id)' "$MAIN_DATA_JSON")
    
    if [ -z "$plugin_data" ] || [ "$plugin_data" = "null" ]; then
        echo "Warning: No data found for plugin '$plugin_id' in main data.json"
        return 1
    fi
    
    # Create the individual data.json with proper structure
    echo "{" > "$plugin_dir/data.json"
    echo "  \"plugin\": $plugin_data" >> "$plugin_dir/data.json"
    echo "}" >> "$plugin_dir/data.json"
    
    echo "  ✓ Updated $plugin_dir/data.json"
}

# Iterate through all plugin directories
for plugin_dir in "$PLUGINS_DIR"/*/; do
    if [ -d "$plugin_dir" ]; then
        # Extract plugin name from directory path
        plugin_name=$(basename "$plugin_dir")
        
        # Skip directories that shouldn't have plugin data
        if [ "$plugin_name" = "example" ] || [ "$plugin_name" = "external_plugin" ]; then
            echo "Skipping $plugin_name (special directory)"
            continue
        fi
        
        # Create backup of existing data.json if it exists
        if [ -f "$plugin_dir/data.json" ]; then
            backup_file="$plugin_dir/data.json.backup.$(date +%Y%m%d_%H%M%S)"
            cp "$plugin_dir/data.json" "$backup_file"
            echo "  Backed up existing data.json to $(basename "$backup_file")"
        fi
        
        # Update the plugin data
        update_plugin_data "$plugin_name" "$plugin_dir"
    fi
done

echo ""
echo "Plugin data.json update process completed!"
echo "Note: Backup files were created for existing data.json files"
