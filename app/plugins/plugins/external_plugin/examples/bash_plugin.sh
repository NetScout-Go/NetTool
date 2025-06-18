#!/usr/bin/env bash
# Example external plugin for NetScout-Pi in Bash

# Read JSON from stdin
input=$(cat)

# Extract parameters (requires jq to be installed)
plugin_type=$(echo $input | jq -r '.plugin_type')
custom_params=$(echo $input | jq -r '.custom_params')
timeout=$(echo $input | jq -r '.timeout')

# Simulate work
sleep 1

# Create timestamp
timestamp=$(date +"%Y-%m-%d %H:%M:%S")

# Output JSON result
cat << EOF
{
  "plugin_name": "Bash Example Plugin",
  "timestamp": "$timestamp",
  "received_params": {
    "plugin_type": "$plugin_type",
    "timeout": $timeout,
    "custom_params": $custom_params
  },
  "message": "Hello from Bash external plugin!"
}
EOF
