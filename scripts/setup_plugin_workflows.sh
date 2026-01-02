#!/usr/bin/env bash
set -euo pipefail

# =============================================================================
# NetTool Plugin Workflow Setup Script
# Deploys GitHub Actions workflows to plugin repositories
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="${SCRIPT_DIR}/plugin-workflow-template"
TEMP_DIR=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Core plugins that should always have working builds
CORE_PLUGINS=(
    "ping"
    "traceroute"
    "dns_lookup"
    "port_scanner"
    "network_info"
    "bandwidth_test"
    "network_latency_heatmap"
    "subnet_calculator"
    "device_discovery"
    "arp_manager"
)

# All available plugins
ALL_PLUGINS=(
    "arp_manager"
    "bandwidth_test"
    "ble_http_proxy"
    "device_discovery"
    "dns_lookup"
    "dns_propagation"
    "example"
    "external_plugin"
    "iperf3"
    "iperf3_server"
    "mtu_tester"
    "network_info"
    "network_latency_heatmap"
    "network_quality"
    "packet_capture"
    "ping"
    "port_scanner"
    "reverse_dns_lookup"
    "ssl_checker"
    "subnet_calculator"
    "tc_controller"
    "traceroute"
    "wifi_device_locator"
    "wifi_device_proximity"
    "wifi_scanner"
)

# GitHub organization
GITHUB_ORG="NetScout-Go"

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        log_info "Cleaning up temporary directory..."
        rm -rf "$TEMP_DIR"
    fi
}

trap cleanup EXIT

show_help() {
    cat << EOF
NetTool Plugin Workflow Setup Script

Usage: $(basename "$0") [OPTIONS]

Options:
    -h, --help          Show this help message
    -c, --core-only     Only set up workflows for core plugins
    -p, --plugin NAME   Set up workflow for a specific plugin
    -l, --list          List all available plugins
    -d, --dry-run       Show what would be done without making changes
    -f, --force         Force update even if workflows exist
    --no-push           Clone and update but don't push changes

Examples:
    $(basename "$0")                    # Set up all plugins
    $(basename "$0") --core-only        # Set up only core plugins
    $(basename "$0") --plugin ping      # Set up only the ping plugin
    $(basename "$0") --dry-run          # Preview changes
EOF
}

list_plugins() {
    echo "Core Plugins:"
    for plugin in "${CORE_PLUGINS[@]}"; do
        echo "  - $plugin"
    done
    echo ""
    echo "All Plugins:"
    for plugin in "${ALL_PLUGINS[@]}"; do
        echo "  - $plugin"
    done
}

# =============================================================================
# Main Logic
# =============================================================================

setup_plugin_workflow() {
    local plugin_id="$1"
    local dry_run="${2:-false}"
    local no_push="${3:-false}"
    local force="${4:-false}"
    
    local repo_name="Plugin_${plugin_id}"
    local repo_url="https://github.com/${GITHUB_ORG}/${repo_name}.git"
    local plugin_dir="${TEMP_DIR}/${repo_name}"
    
    log_info "Processing ${plugin_id}..."
    
    if [ "$dry_run" = "true" ]; then
        log_info "[DRY-RUN] Would clone ${repo_url}"
        log_info "[DRY-RUN] Would add .github/workflows/build.yml"
        log_info "[DRY-RUN] Would add .github/workflows/beta.yml"
        log_info "[DRY-RUN] Would commit and push changes"
        return 0
    fi
    
    # Clone the repository
    if ! git clone --depth 1 "${repo_url}" "${plugin_dir}" 2>/dev/null; then
        log_error "Failed to clone ${repo_name}. Repository may not exist."
        return 1
    fi
    
    # Check if workflows already exist
    local workflow_dir="${plugin_dir}/.github/workflows"
    if [ -d "$workflow_dir" ] && [ "$force" != "true" ]; then
        if [ -f "${workflow_dir}/build.yml" ] && [ -f "${workflow_dir}/beta.yml" ]; then
            log_warning "Workflows already exist for ${plugin_id}. Use --force to overwrite."
            return 0
        fi
    fi
    
    # Create workflow directory
    mkdir -p "${workflow_dir}"
    
    # Copy workflow templates
    cp "${TEMPLATE_DIR}/build.yml" "${workflow_dir}/build.yml"
    cp "${TEMPLATE_DIR}/beta.yml" "${workflow_dir}/beta.yml"
    
    log_success "Added workflow files to ${plugin_id}"
    
    # Commit changes
    cd "${plugin_dir}"
    git add .github/workflows/
    
    if git diff --staged --quiet; then
        log_info "No changes to commit for ${plugin_id}"
        return 0
    fi
    
    git commit -m "Add GitHub Actions workflows for automated builds

- build.yml: Creates releases on version tags (v*)
- beta.yml: Creates beta releases on push to main

Platforms: Linux x64, x32, ARM6, ARM7, ARM64
Features: UPX compression, SHA256 checksums"
    
    if [ "$no_push" = "true" ]; then
        log_info "[NO-PUSH] Changes committed but not pushed for ${plugin_id}"
        return 0
    fi
    
    # Push changes
    if git push origin HEAD 2>/dev/null; then
        log_success "Pushed workflow changes for ${plugin_id}"
    else
        log_error "Failed to push changes for ${plugin_id}. Check permissions."
        return 1
    fi
    
    cd - > /dev/null
}

# =============================================================================
# Parse Arguments
# =============================================================================

CORE_ONLY=false
SPECIFIC_PLUGIN=""
DRY_RUN=false
NO_PUSH=false
FORCE=false
LIST_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -c|--core-only)
            CORE_ONLY=true
            shift
            ;;
        -p|--plugin)
            SPECIFIC_PLUGIN="$2"
            shift 2
            ;;
        -l|--list)
            LIST_ONLY=true
            shift
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        --no-push)
            NO_PUSH=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# =============================================================================
# Main Execution
# =============================================================================

if [ "$LIST_ONLY" = "true" ]; then
    list_plugins
    exit 0
fi

# Check for required files
if [ ! -f "${TEMPLATE_DIR}/build.yml" ] || [ ! -f "${TEMPLATE_DIR}/beta.yml" ]; then
    log_error "Workflow templates not found in ${TEMPLATE_DIR}"
    log_error "Please ensure build.yml and beta.yml exist in the template directory"
    exit 1
fi

# Check for git
if ! command -v git &> /dev/null; then
    log_error "git is required but not installed"
    exit 1
fi

# Create temporary directory
TEMP_DIR=$(mktemp -d)
log_info "Using temporary directory: ${TEMP_DIR}"

# Determine which plugins to process
if [ -n "$SPECIFIC_PLUGIN" ]; then
    PLUGINS_TO_PROCESS=("$SPECIFIC_PLUGIN")
elif [ "$CORE_ONLY" = "true" ]; then
    PLUGINS_TO_PROCESS=("${CORE_PLUGINS[@]}")
else
    PLUGINS_TO_PROCESS=("${ALL_PLUGINS[@]}")
fi

# Process plugins
SUCCESS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

log_info "Processing ${#PLUGINS_TO_PROCESS[@]} plugin(s)..."
echo ""

for plugin in "${PLUGINS_TO_PROCESS[@]}"; do
    if setup_plugin_workflow "$plugin" "$DRY_RUN" "$NO_PUSH" "$FORCE"; then
        ((SUCCESS_COUNT++))
    else
        ((FAIL_COUNT++))
    fi
    echo ""
done

# Summary
echo "=========================================="
echo "Summary:"
echo "  Total:    ${#PLUGINS_TO_PROCESS[@]}"
echo "  Success:  ${SUCCESS_COUNT}"
echo "  Failed:   ${FAIL_COUNT}"
echo "=========================================="

if [ "$DRY_RUN" = "true" ]; then
    log_info "This was a dry run. No changes were made."
fi

if [ $FAIL_COUNT -gt 0 ]; then
    exit 1
fi

exit 0
