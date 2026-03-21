# NetTool

[![Go Version](https://img.shields.io/github/go-mod/go-version/NetScout-Go/NetTool?color=00ADD8&label=Go%20Version)](https://github.com/NetScout-Go/NetTool/blob/main/go.mod)
[![License](https://img.shields.io/github/license/NetScout-Go/NetTool.svg?color=green)](LICENSE)
[![Build](https://img.shields.io/badge/build-Go%20modules-blue)](https://github.com/NetScout-Go/NetTool)
[![Status](https://img.shields.io/badge/status-experimental-orange)](#project-status)

> [!IMPORTANT]
> NetTool is a personal, educational project. It is not hardened for production environments and several plugins expect trusted networks or elevated privileges. Review each plugin before using it in critical scenarios.

NetTool is a web-based network diagnostic console for Raspberry Pi and other Linux devices. It centralizes common troubleshooting utilities, live telemetry, and a plugin marketplace into a single responsive dashboard.

> Documentation is maintained with the help of AI tooling. If you spot an omission or error, please open an issue or pull request.

![NetTool UI concept diagram](Resources/)

---

## Table of Contents

- [Highlights](#highlights)
- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Running & Deployment](#running--deployment)
- [Configuration](#configuration)
- [Dashboard at a Glance](#dashboard-at-a-glance)
- [Plugin Ecosystem](#plugin-ecosystem)
- [API & Realtime Access](#api--realtime-access)
- [Troubleshooting](#troubleshooting)
- [Development Workflow](#development-workflow)
- [Project Status](#project-status)
- [License](#license)
- [Open Source Credits](#open-source-credits)
- [Acknowledgments](#acknowledgments)

---

## Highlights

- **Unified dashboard** with real-time interface statistics, DHCP lease state, traffic counters, and connection health.
- **Modular plugin framework** supporting native Go plugins and external scripts (Python/Bash) with dynamic discovery.
- **Rich diagnostics catalog** including bandwidth tests, latency heatmaps, packet capture, iPerf3, traceroute, DNS utilities, SSL checks, and more.
- **REST & WebSocket APIs** for automation, remote orchestration, and live telemetry streaming.
- **Mobile-first UI** built with Bootstrap and Chart.js so the console works on tablets and phones beside your rack.
- **Service-friendly packaging** featuring systemd examples, install scripts, and optional external tool integrations.

## Architecture Overview

- **Backend**: Go 1.24+, Gin web framework, Gorilla WebSocket, gopsutil for system interrogation.
- **Frontend**: Bootstrap 5, Bootstrap Icons, Chart.js, vanilla JS modules for real-time updates.
- **Plugins**: Organized under `app/plugins/plugins/<plugin_id>`, each with metadata (`plugin.json`) and an `Execute` entrypoint.
- **Storage**: Configuration and plugin assets served from the local filesystem; no database dependency.

## Prerequisites

- Raspberry Pi (Zero 2W, 3B+, 4) or any Linux host.
- Go toolchain **1.20+**, tested on 1.24.4.
- GitHub CLI (`gh`) for plugin install automation.
- Optional third-party CLIs based on your plugin selection: `librespeed-cli`, `iperf3`, `nmap`, `tcpdump`, etc.

## Quick Start

```bash
git clone https://github.com/NetScout-Go/NetTool.git
cd NetTool

# Install core dependencies
sudo apt update
sudo apt install gh librespeed-cli
gh auth login

# Fetch plugins (select interactively)
chmod +x install-plugins.sh
./install-plugins.sh

# Build & run
go build -o nettool .
NETTOOL_ALLOW_DEV_BUILD=1 ./nettool --port 8080
```

> **Development note:** source-built binaries do not match the signed release hashes. For local development/testing, opt in explicitly with `NETTOOL_ALLOW_DEV_BUILD=1`. Official release builds still use the normal integrity verification path.
>
> **Pi Zero tip:** Use `env CGO_ENABLED=0 go build -o nettool .` if you hit CGO-related link errors.

Visit `http://<device-ip>:8080` to open the dashboard.

## Running & Deployment

- **Foreground:** `./nettool --port 8080`
- **Elevated plugins:** `sudo ./nettool` for features requiring privileged sockets or traffic shaping.
- **systemd service:**

  ```ini
  [Unit]
  Description=NetTool Network Diagnostic Tool
  After=network.target

  [Service]
  ExecStart=/home/pi/NetTool/nettool --port 8080
  WorkingDirectory=/home/pi/NetTool
  Restart=always
  User=pi

  [Install]
  WantedBy=multi-user.target
  ```

  ```bash
  sudo systemctl enable netscout.service
  sudo systemctl start netscout.service
  sudo systemctl status netscout.service
  ```

## Configuration

`nettool` accepts flags and an optional `config.json` in the project root:

```json
{
  "port": 8888,
  "debug": true,
  "allowCORS": false,
  "refreshInterval": 5
}
```

Flags override file values. Restart the service after editing the config.

## Dashboard at a Glance

- **Connection status** with uptime, link speed, duplex mode, and interface health.
- **IP configuration** showing IPv4/IPv6 addresses, gateways, DNS servers, and DHCP lease metrics.
- **Traffic statistics** updating live via WebSocket (packets, bytes, errors).
- **Network topology hints** including ARP snapshots and discovered devices.
- **Speed test card** backed by `librespeed-cli` (preferred) with fallbacks to other CLIs or simulated results.

## Plugin Ecosystem

- Plugins live under `app/plugins/plugins/` and are categorized (Analysis, Discovery, DNS, Security, etc.).
- Each plugin exposes metadata via `plugin.json` and a Go `Execute` function or external script wrapper.
- Manage plugins with helper scripts:
  - `./install-plugins.sh` – clone/update official plugin repositories.
  - `./list-plugins.sh` – enumerate installed plugins the UI will surface.
- Refer to [app/plugins/DEVELOPMENT.md](app/plugins/DEVELOPMENT.md) for authoring guidelines (parameters, result contracts, logging).

### Spotlight Plugins

| Category | Plugin | Description |
| --- | --- | --- |
| Analysis | `bandwidth_test` | Run LibreSpeed/Speedtest CLIs and surface Mbps, latency, jitter. |
| Analysis | `network_latency_heatmap` | Measure multi-target latency and plot heatmap. |
| Discovery | `port_scanner` | Front-end to `nmap` for quick reconnaissance. |
| DNS | `dns_propagation` | Compare DNS responses across providers. |
| Security | `ssl_checker` | Inspect leaf and chain certificates, expiry, and issuer details. |

## API & Realtime Access

- List plugins: `GET /api/plugins`
- Plugin metadata: `GET /api/plugins/{id}`
- Run plugin: `POST /api/plugins/{id}/run` with JSON payload
- Network snapshot: `GET /api/network-info`

Example (ping):

```bash
curl -X POST http://<device-ip>:8080/api/plugins/ping/run \
  -H "Content-Type: application/json" \
  -d '{"host": "example.com", "count": 4}'
```

### WebSocket Stream

```javascript
const ws = new WebSocket('ws://<device-ip>:8080/ws');
ws.onmessage = event => console.log(JSON.parse(event.data));
```

Messages include traffic counters, interface state changes, DHCP lease updates, and plugin progress events.

## Troubleshooting

- **Compilation errors (Pi Zero):** `env CGO_ENABLED=0 go build`
- **Permission errors:** Run with sudo if the plugin needs raw sockets or tc access.
- **Missing tool:** Install the CLI noted in the plugin card or disable the plugin in config.
- **WebSocket blocked:** Check firewalls or reverse proxies that strip upgrade headers.
- **Logs:** `journalctl -u netscout.service -f`

## Development Workflow

- Format code: `gofmt -w .`
- Lint (optional): integrate `golangci-lint` or `staticcheck` locally.
- Run tests: `go test ./...`
- Hot reload (dev): use `CompileDaemon` or `air` if preferred.
- Contribution steps:
  1. Fork the repo
  2. `git checkout -b feature/<name>`
  3. Commit with clear messages
  4. `git push origin feature/<name>`
  5. Open a Pull Request describing changes and testing

## Project Status

- Focused on hobbyist and hackathon deployments.
- Known rough edges: limited authentication, plugin permission model, i18n gaps.
- Roadmap ideas: role-based access, plugin marketplace UI enhancements, telemetry export.

## License

This project ships under the MIT License. See [LICENSE](LICENSE) for full text.

## Open Source Credits

NetTool exists thanks to these excellent projects:

- [Go](https://go.dev/)
- [Gin](https://gin-gonic.com/) and [gin-contrib/multitemplate](https://github.com/gin-contrib/multitemplate)
- [gopsutil](https://github.com/shirou/gopsutil)
- [gorilla/websocket](https://github.com/gorilla/websocket)
- [Bootstrap](https://getbootstrap.com/), [Bootstrap Icons](https://icons.getbootstrap.com/), and [Chart.js](https://www.chartjs.org/)
- [LibreSpeed CLI](https://github.com/librespeed/speedtest-cli) and [speedtest-cli](https://github.com/sivel/speedtest-cli)
- [iperf3](https://github.com/esnet/iperf), [nmap](https://nmap.org/), and [tcpdump](https://www.tcpdump.org/)

## Acknowledgments

- The Go community for stellar networking libraries and tooling
- Raspberry Pi Foundation for repeatedly punching above its weight
- All contributors, testers, and Hack Club members cheering this project on
