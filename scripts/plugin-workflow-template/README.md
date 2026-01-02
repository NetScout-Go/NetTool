# NetTool Plugin Workflow Templates

GitHub Actions workflow templates for building and releasing NetTool plugins.

## Files

| File | Purpose |
|------|---------|
| `build.yml` | Creates releases when version tags are pushed (e.g., `v1.0.0`) |
| `beta.yml` | Creates beta releases on every push to main branch |

## Quick Setup

### Option 1: Automatic Setup (Recommended)

Use the setup script to deploy workflows to plugin repositories:

```bash
# Set up core plugins only (ping, traceroute, dns_lookup, etc.)
./scripts/setup_plugin_workflows.sh --core-only

# Set up all plugins
./scripts/setup_plugin_workflows.sh

# Set up a specific plugin
./scripts/setup_plugin_workflows.sh --plugin ping

# Preview changes without making them
./scripts/setup_plugin_workflows.sh --dry-run
```

### Option 2: Manual Setup

```bash
cd Plugin_your_plugin
mkdir -p .github/workflows
cp /path/to/templates/build.yml .github/workflows/
cp /path/to/templates/beta.yml .github/workflows/
git add .github/workflows/
git commit -m "Add GitHub Actions workflows"
git push
```

## How It Works

### Release Builds (`build.yml`)

Triggered when you push a version tag:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Creates:
- Binaries for 5 platforms (Linux x64, x32, ARM6, ARM7, ARM64)
- UPX-compressed executables
- SHA256 checksums for verification
- A GitHub Release with all assets

### Beta Builds (`beta.yml`)

Triggered automatically on every push to `main` or `master` (excluding docs).

Creates:
- Binaries for 3 key platforms (Linux x64, x32, ARM6)
- A pre-release tagged as `beta` (replaces previous beta)

## Supported Platforms

| Platform | Architecture | Target Device |
|----------|--------------|---------------|
| Linux x64 | amd64 | Desktop/server |
| Linux x32 | 386 | Older 32-bit systems |
| Linux ARM6 | arm/v6 | Raspberry Pi Zero/1 |
| Linux ARM7 | arm/v7 | Raspberry Pi 3/4 (32-bit) |
| Linux ARM64 | arm64 | Raspberry Pi 4+ (64-bit) |

## Plugin Requirements

Your plugin repository should have:

```
Plugin_your_plugin/
├── plugin.json       # Plugin metadata (required)
├── plugin.go         # Main plugin code (required)
├── go.mod            # Go module file (required for build)
├── data.json         # Store listing metadata (optional)
├── README.md         # Documentation (optional)
└── static/           # Static assets (optional)
    ├── styles.css
    └── ui.html
```

### plugin.json Example

```json
{
  "id": "your_plugin",
  "name": "Your Plugin",
  "description": "Plugin description",
  "version": "1.0.0",
  "author": "Your Name",
  "license": "MIT",
  "icon": "puzzle-piece",
  "requires": ["curl"],
  "parameters": [
    {
      "id": "target",
      "name": "Target",
      "type": "string",
      "required": true
    }
  ]
}
```

## Build Optimizations

### Hardened Build Flags

```go
-ldflags "-w -s -extldflags '-static'"
-trimpath
-buildid=
-tags "netgo,osusergo"
CGO_ENABLED=0
```

### UPX Compression

- x86/x64: `--best --lzma` (maximum compression)
- ARM: `--best` (LZMA can have issues on ARM)

## Verifying Downloads

```bash
# Download release
curl -sLO https://github.com/NetScout-Go/Plugin_ping/releases/latest/download/ping-linux-amd64-v1.0.0.tar.gz
curl -sLO https://github.com/NetScout-Go/Plugin_ping/releases/latest/download/SHA256SUMS

# Verify checksum
sha256sum -c SHA256SUMS
```

## Release Assets

Each release includes:

| File | Description |
|------|-------------|
| `{plugin}-{platform}-{version}.tar.gz` | Compressed plugin archive |
| `SHA256SUMS` | Checksums for all archives |
| `checksums.json` | JSON format checksums for API use |

### checksums.json Format

```json
{
  "plugin_id": "ping",
  "version": "v1.0.0",
  "build_time": "2026-01-02T12:00:00Z",
  "platforms": {
    "linux-amd64": "abc123...",
    "linux-386": "def456...",
    "linux-arm6": "789ghi..."
  }
}
```

## Core Plugins

Essential plugins that should always have working builds:

| Plugin | Description |
|--------|-------------|
| `ping` | Network ping utility |
| `traceroute` | Network path tracing |
| `dns_lookup` | DNS resolution |
| `port_scanner` | Port scanning |
| `network_info` | Network interface info |
| `bandwidth_test` | Speed testing |
| `network_latency_heatmap` | Latency visualization |
| `subnet_calculator` | IP/subnet calculations |
| `device_discovery` | Network device discovery |
| `arp_manager` | ARP table management |

## Troubleshooting

### Build Fails

1. Ensure `go.mod` exists in the plugin root
2. Check that all imports are resolvable
3. Verify `plugin.json` has valid JSON syntax

### UPX Compression Fails

Some binaries may fail UPX compression. The workflow handles this gracefully and continues with the uncompressed binary.

### Release Not Created

Check that:
1. You pushed a tag starting with `v` (e.g., `v1.0.0`)
2. The workflow has write permissions (`permissions: contents: write`)
3. No syntax errors in workflow files

### Rate Limiting

If you're setting up many plugins, GitHub may rate-limit API requests. Use a personal access token:

```bash
export GITHUB_TOKEN=your_token_here
./scripts/setup_plugin_workflows.sh
```
