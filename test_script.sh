#!/bin/bash

echo "Script started"
PLUGINS_DIR="/home/anoam/NetTool/app/plugins/plugins"
echo "Plugins directory: $PLUGINS_DIR"

echo "Listing directories:"
ls -la "$PLUGINS_DIR"

echo "Testing glob pattern:"
for plugin_dir in "$PLUGINS_DIR"/*/; do
    echo "Found directory: $plugin_dir"
    if [ -d "$plugin_dir" ]; then
        plugin_name=$(basename "$plugin_dir")
        echo "Plugin name: $plugin_name"
    fi
done

echo "Script finished"
