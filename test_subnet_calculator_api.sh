#!/bin/bash

# Test the subnet calculator API endpoint with different actions
echo "Testing subnet calculator API with 'calculate' action"
curl -s -X POST http://localhost:8080/api/plugins/subnet_calculator/run \
  -H "Content-Type: application/json" \
  -d '{"action":"calculate","address":"192.168.1.0/24"}'

echo -e "\nTesting subnet calculator API with 'supernet' action"
curl -s -X POST http://localhost:8080/api/plugins/subnet_calculator/run \
  -H "Content-Type: application/json" \
  -d '{"action":"supernet","ip_list":"192.168.1.0/24,192.168.2.0/24"}'

echo -e "\nTesting subnet calculator API with 'aggregate' action"
curl -s -X POST http://localhost:8080/api/plugins/subnet_calculator/run \
  -H "Content-Type: application/json" \
  -d '{"action":"aggregate","ip_list":"192.168.1.0/24,192.168.2.0/24,192.168.3.0/24,192.168.4.0/24"}'

echo -e "\nTesting subnet calculator API with 'conflict_detect' action"
curl -s -X POST http://localhost:8080/api/plugins/subnet_calculator/run \
  -H "Content-Type: application/json" \
  -d '{"action":"conflict_detect","ip_list":"192.168.1.0/24,192.168.1.128/25,192.168.2.0/24"}'
