#!/bin/bash

# Build and run the subnet calculator plugin
cd "$(dirname "$0")"

echo "==== Testing Calculate Action ===="
go run main.go -plugin subnet_calculator -args '{"action":"calculate","address":"192.168.1.0/24"}'

echo -e "\n==== Testing Divide Action ===="
go run main.go -plugin subnet_calculator -args '{"action":"divide","address":"192.168.1.0/24","num_subnets":4}'

echo -e "\n==== Testing Supernet Action ===="
go run main.go -plugin subnet_calculator -args '{"action":"supernet","ip_list":"192.168.1.0/24,192.168.2.0/24"}'

echo -e "\n==== Testing Aggregate Action ===="
go run main.go -plugin subnet_calculator -args '{"action":"aggregate","ip_list":"192.168.1.0/24,192.168.2.0/24"}'

echo -e "\n==== Testing Conflict Detection Action ===="
go run main.go -plugin subnet_calculator -args '{"action":"conflict_detect","ip_list":"192.168.1.0/24,192.168.2.0/24,192.168.1.128/25"}'
