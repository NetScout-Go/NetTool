#!/usr/bin/env python3
"""
Example external plugin for NetScout-Pi
This demonstrates how to create an external plugin in Python
"""

import json
import sys
import time

def main():
    # Read parameters from stdin
    params_json = sys.stdin.read()
    params = json.loads(params_json)
    
    # Process parameters
    plugin_type = params.get("plugin_type", "")
    custom_params = json.loads(params.get("custom_params", "{}"))
    timeout = params.get("timeout", 30)
    # Simulate some work
    time.sleep(1)
    
    # Create result
    result = {
        "plugin_name": "Python Example Plugin",
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "received_params": params,
        "custom_params": custom_params,
        "message": "Hello from Python external plugin!"
    }
    
    # Output result as JSON to stdout
    print(json.dumps(result))

if __name__ == "__main__":
    main()
