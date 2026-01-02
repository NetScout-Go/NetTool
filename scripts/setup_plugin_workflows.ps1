# =============================================================================
# NetTool Plugin Workflow Setup Script (PowerShell)
# Deploys GitHub Actions workflows to plugin repositories
# =============================================================================

param(
    [switch]$CoreOnly,
    [string]$Plugin,
    [switch]$List,
    [switch]$DryRun,
    [switch]$Force,
    [switch]$NoPush,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$TemplateDir = Join-Path $ScriptDir "plugin-workflow-template"

# GitHub organization
$GitHubOrg = "NetScout-Go"

# Core plugins
$CorePlugins = @(
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

# All plugins
$AllPlugins = @(
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

function Show-Help {
    Write-Host @"
NetTool Plugin Workflow Setup Script (PowerShell)

Usage: .\setup_plugin_workflows.ps1 [OPTIONS]

Options:
    -Help           Show this help message
    -CoreOnly       Only set up workflows for core plugins
    -Plugin NAME    Set up workflow for a specific plugin
    -List           List all available plugins
    -DryRun         Show what would be done without making changes
    -Force          Force update even if workflows exist
    -NoPush         Clone and update but don't push changes

Examples:
    .\setup_plugin_workflows.ps1                    # Set up all plugins
    .\setup_plugin_workflows.ps1 -CoreOnly          # Set up only core plugins
    .\setup_plugin_workflows.ps1 -Plugin ping       # Set up only the ping plugin
    .\setup_plugin_workflows.ps1 -DryRun            # Preview changes
"@
}

function Show-Plugins {
    Write-Host "Core Plugins:" -ForegroundColor Cyan
    foreach ($p in $CorePlugins) {
        Write-Host "  - $p"
    }
    Write-Host ""
    Write-Host "All Plugins:" -ForegroundColor Cyan
    foreach ($p in $AllPlugins) {
        Write-Host "  - $p"
    }
}

function Setup-PluginWorkflow {
    param(
        [string]$PluginId,
        [bool]$IsDryRun,
        [bool]$IsNoPush,
        [bool]$IsForce
    )

    $RepoName = "Plugin_$PluginId"
    $RepoUrl = "https://github.com/$GitHubOrg/$RepoName.git"
    $TempDir = Join-Path $env:TEMP "nettool-plugin-$PluginId-$(Get-Random)"

    Write-Host "[INFO] Processing $PluginId..." -ForegroundColor Blue

    if ($IsDryRun) {
        Write-Host "[DRY-RUN] Would clone $RepoUrl" -ForegroundColor Yellow
        Write-Host "[DRY-RUN] Would add .github/workflows/build.yml" -ForegroundColor Yellow
        Write-Host "[DRY-RUN] Would add .github/workflows/beta.yml" -ForegroundColor Yellow
        Write-Host "[DRY-RUN] Would commit and push changes" -ForegroundColor Yellow
        return $true
    }

    try {
        # Clone the repository
        $output = git clone --depth 1 $RepoUrl $TempDir 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Host "[ERROR] Failed to clone $RepoName. Repository may not exist." -ForegroundColor Red
            return $false
        }

        # Check if workflows already exist
        $WorkflowDir = Join-Path $TempDir ".github\workflows"
        $BuildYml = Join-Path $WorkflowDir "build.yml"
        $BetaYml = Join-Path $WorkflowDir "beta.yml"

        if ((Test-Path $BuildYml) -and (Test-Path $BetaYml) -and -not $IsForce) {
            Write-Host "[WARNING] Workflows already exist for $PluginId. Use -Force to overwrite." -ForegroundColor Yellow
            return $true
        }

        # Create workflow directory
        if (-not (Test-Path $WorkflowDir)) {
            New-Item -ItemType Directory -Path $WorkflowDir -Force | Out-Null
        }

        # Copy workflow templates
        Copy-Item (Join-Path $TemplateDir "build.yml") $BuildYml -Force
        Copy-Item (Join-Path $TemplateDir "beta.yml") $BetaYml -Force

        Write-Host "[SUCCESS] Added workflow files to $PluginId" -ForegroundColor Green

        # Commit changes
        Push-Location $TempDir
        try {
            git add ".github/workflows/" 2>&1 | Out-Null

            $status = git diff --staged --quiet 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host "[INFO] No changes to commit for $PluginId" -ForegroundColor Blue
                return $true
            }

            git commit -m "Add GitHub Actions workflows for automated builds

- build.yml: Creates releases on version tags (v*)
- beta.yml: Creates beta releases on push to main

Platforms: Linux x64, x32, ARM6, ARM7, ARM64
Features: UPX compression, SHA256 checksums" 2>&1 | Out-Null

            if ($IsNoPush) {
                Write-Host "[NO-PUSH] Changes committed but not pushed for $PluginId" -ForegroundColor Yellow
                return $true
            }

            # Push changes
            $pushOutput = git push origin HEAD 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-Host "[SUCCESS] Pushed workflow changes for $PluginId" -ForegroundColor Green
            } else {
                Write-Host "[ERROR] Failed to push changes for $PluginId. Check permissions." -ForegroundColor Red
                Write-Host $pushOutput -ForegroundColor Red
                return $false
            }
        }
        finally {
            Pop-Location
        }

        return $true
    }
    catch {
        Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }
    finally {
        # Cleanup
        if (Test-Path $TempDir) {
            Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
        }
    }
}

# =============================================================================
# Main Execution
# =============================================================================

if ($Help) {
    Show-Help
    exit 0
}

if ($List) {
    Show-Plugins
    exit 0
}

# Check for required files
$BuildTemplate = Join-Path $TemplateDir "build.yml"
$BetaTemplate = Join-Path $TemplateDir "beta.yml"

if (-not (Test-Path $BuildTemplate) -or -not (Test-Path $BetaTemplate)) {
    Write-Host "[ERROR] Workflow templates not found in $TemplateDir" -ForegroundColor Red
    Write-Host "[ERROR] Please ensure build.yml and beta.yml exist in the template directory" -ForegroundColor Red
    exit 1
}

# Check for git
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Write-Host "[ERROR] git is required but not installed" -ForegroundColor Red
    exit 1
}

# Determine which plugins to process
if ($Plugin) {
    $PluginsToProcess = @($Plugin)
} elseif ($CoreOnly) {
    $PluginsToProcess = $CorePlugins
} else {
    $PluginsToProcess = $AllPlugins
}

# Process plugins
$SuccessCount = 0
$FailCount = 0

Write-Host "[INFO] Processing $($PluginsToProcess.Count) plugin(s)..." -ForegroundColor Blue
Write-Host ""

foreach ($p in $PluginsToProcess) {
    $result = Setup-PluginWorkflow -PluginId $p -IsDryRun $DryRun -IsNoPush $NoPush -IsForce $Force
    if ($result) {
        $SuccessCount++
    } else {
        $FailCount++
    }
    Write-Host ""
}

# Summary
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Summary:"
Write-Host "  Total:    $($PluginsToProcess.Count)"
Write-Host "  Success:  $SuccessCount" -ForegroundColor Green
Write-Host "  Failed:   $FailCount" -ForegroundColor $(if ($FailCount -gt 0) { "Red" } else { "Green" })
Write-Host "==========================================" -ForegroundColor Cyan

if ($DryRun) {
    Write-Host "[INFO] This was a dry run. No changes were made." -ForegroundColor Yellow
}

if ($FailCount -gt 0) {
    exit 1
}

exit 0
