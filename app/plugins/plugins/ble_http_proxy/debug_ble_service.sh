#!/bin/bash

# Script to debug BLE HTTP Proxy service
# This will run the Python script with exception catching

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}BLE HTTP Proxy Debug Script${NC}"
echo

# Create a wrapper Python script
cat > /tmp/ble_debug_wrapper.py << 'EOL'
#!/usr/bin/env python3
"""
Debug wrapper for BLE HTTP Proxy service
"""
import sys
import traceback

if __name__ == "__main__":
    try:
        # Import the actual script
        script_path = sys.argv[1]
        with open(script_path) as f:
            code = compile(f.read(), script_path, 'exec')
            # Create a new module namespace
            module_namespace = {}
            # Execute the script in this namespace
            exec(code, module_namespace)
    except Exception as e:
        print(f"\n\nERROR: {e}")
        print("\nFull traceback:")
        traceback.print_exc()
        sys.exit(1)
EOL

# Make wrapper executable
chmod +x /tmp/ble_debug_wrapper.py

# Run the script through the wrapper
echo -e "${BLUE}Running BLE HTTP Proxy service with debug wrapper...${NC}"
python3 /tmp/ble_debug_wrapper.py "$(pwd)/pi_zero_ble_service.py" "$@"
