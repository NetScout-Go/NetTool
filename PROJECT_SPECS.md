# NetTool Project Specifications

> **Last Updated**: January 2026  
> **Version**: 2.1.0  
> **Status**: Active Development

This document serves as the living specification for the NetTool project. It defines the architecture, standards, and guidelines that all contributors should follow.

## Recent Updates (v2.1.0)

- ✅ Enhanced plugin state management with React Context (`PluginContext.jsx`)
- ✅ Improved API layer with error classes and interceptors (`api/index.js`)
- ✅ Unified plugin type definitions and utilities (`types/plugins.js`)
- ✅ Reusable UI component library (`components/common/index.jsx`)
- ✅ Dynamic plugin loading in Sidebar component
- ✅ Enhanced PluginPage with parameter validation and result rendering
- ✅ Backend plugin execution with progress reporting (`execution.go`)
- ✅ WebSocket streaming for plugin execution (`streaming.go`)
- ✅ Frontend streaming hook (`usePluginStream.js`)

---

## Table of Contents

- [Project Overview](#project-overview)
- [Architecture](#architecture)
- [Technology Stack](#technology-stack)
- [Directory Structure](#directory-structure)
- [Backend Specifications](#backend-specifications)
- [Frontend Specifications](#frontend-specifications)
- [Plugin System](#plugin-system)
- [API Contracts](#api-contracts)
- [Data Flow](#data-flow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Deployment](#deployment)
- [Roadmap](#roadmap)

---

## Project Overview

**NetTool** is a web-based network diagnostic and monitoring console designed for Raspberry Pi and Linux devices. It provides:

- Real-time network monitoring and visualization
- Modular plugin system for extensible diagnostics
- WebSocket-based live telemetry streaming
- REST API for automation and integration
- Responsive dashboard for desktop and mobile

### Core Principles

1. **Modularity**: All features should be implemented as plugins when possible
2. **Real-time**: Use WebSocket for live data; polling only as fallback
3. **Security**: All inputs validated, secure headers, no exposed secrets
4. **Performance**: Optimized for low-power devices like Raspberry Pi
5. **Extensibility**: Easy to add new plugins without modifying core code

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        React Frontend (SPA)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  Dashboard  │  │   Plugin    │  │    Plugin Manager       │  │
│  │   Page      │  │   Pages     │  │       Page              │  │
│  └──────┬──────┘  └──────┬──────┘  └────────────┬────────────┘  │
│         │                │                      │                │
│  ┌──────┴────────────────┴──────────────────────┴────────────┐  │
│  │              Context Providers & Hooks                     │  │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────┐ │  │
│  │  │ PluginContext  │  │ WebSocketHook  │  │  API Layer   │ │  │
│  │  └────────────────┘  └────────────────┘  └──────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                    HTTP/REST │ WebSocket
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Go Backend (Gin)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │  REST API   │  │  WebSocket  │  │    Static Files         │  │
│  │  Handlers   │  │   Server    │  │    (SPA Assets)         │  │
│  └──────┬──────┘  └──────┬──────┘  └─────────────────────────┘  │
│         │                │                                       │
│  ┌──────┴────────────────┴──────────────────────────────────┐   │
│  │                   Plugin Manager                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────┐  │   │
│  │  │   Loader   │  │  Registry  │  │     Installer      │  │   │
│  │  └─────┬──────┘  └─────┬──────┘  └─────────┬──────────┘  │   │
│  └────────┼───────────────┼───────────────────┼─────────────┘   │
│           │               │                   │                  │
│  ┌────────┴───────────────┴───────────────────┴─────────────┐   │
│  │                     Plugin Directory                      │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │   │
│  │  │Plugin 1 │  │ Plugin 2 │  │ Plugin 3 │  │   ...    │  │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │   │
│  └───────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐   │
│  │                    Core Services                          │   │
│  │  ┌────────────────┐  ┌────────────────────────────────┐  │   │
│  │  │  Network Info  │  │    System Utilities            │  │   │
│  │  └────────────────┘  └────────────────────────────────┘  │   │
│  └───────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Technology Stack

### Backend

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| Language | Go | 1.24+ | Core backend language |
| Web Framework | Gin | v1.10+ | HTTP routing and middleware |
| WebSocket | Gorilla WebSocket | v1.5+ | Real-time communication |
| System Info | gopsutil | v3.24+ | System/network metrics |

### Frontend

| Component | Technology | Version | Purpose |
|-----------|------------|---------|---------|
| Framework | React | 18+ | UI framework |
| Build Tool | Vite | 5+ | Development and bundling |
| Styling | TailwindCSS | 3+ | Utility-first CSS |
| Animation | Framer Motion | 10+ | Smooth animations |
| Icons | Lucide React | latest | Icon library |
| HTTP Client | Axios | 1+ | API requests |
| Routing | React Router | 6+ | Client-side routing |

---

## Directory Structure

```
NetTool/
├── main.go                    # Application entry point
├── spa_support.go             # SPA serving logic
├── go.mod                     # Go module definition
├── go.sum                     # Go module checksums
├── PROJECT_SPECS.md           # This file - project specifications
├── README.md                  # Project documentation
├── LICENSE                    # MIT License
│
├── app/                       # Backend application code
│   ├── core/                  # Core functionality
│   │   └── network_info.go    # Network information retrieval
│   │
│   └── plugins/               # Plugin system
│       ├── plugin_manager.go  # Plugin lifecycle management
│       ├── plugin_installer.go# GitHub installation support
│       ├── loader.go          # Dynamic plugin loading
│       ├── config_manager.go  # Configuration management
│       ├── types.go           # Package-level types
│       │
│       ├── types/             # Shared type definitions
│       │   ├── plugin_types.go
│       │   ├── types.go
│       │   └── iterable_plugin.go
│       │
│       ├── cli/               # CLI utilities
│       │   └── iterable_cli.go
│       │
│       └── plugins/           # Individual plugins
│           ├── arp_manager/
│           ├── bandwidth_test/
│           └── device_discovery/
│
├── frontend/                  # React frontend
│   ├── package.json
│   ├── vite.config.js
│   ├── tailwind.config.js
│   │
│   ├── public/                # Static assets
│   │
│   └── src/
│       ├── main.jsx           # Entry point
│       ├── App.jsx            # Root component
│       ├── index.css          # Global styles
│       │
│       ├── api/               # API client layer
│       │   └── index.js
│       │
│       ├── hooks/             # Custom React hooks
│       │   ├── useWebSocket.js
│       │   └── usePlugins.js
│       │
│       ├── context/           # React context providers
│       │   └── PluginContext.jsx
│       │
│       ├── components/        # Reusable components
│       │   ├── common/        # Shared UI components
│       │   ├── dashboard/     # Dashboard widgets
│       │   ├── layout/        # Layout components
│       │   └── plugins/       # Plugin-specific components
│       │
│       └── pages/             # Page components
│           ├── Dashboard.jsx
│           ├── PluginManager.jsx
│           └── PluginPage.jsx
│
└── scripts/                   # Utility scripts
    └── sync_plugin_repos.sh
```

---

## Backend Specifications

### API Endpoints

#### Core Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/network-info` | Get current network information |
| GET | `/ws` | WebSocket connection for real-time updates |

#### Plugin Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/plugins` | List all registered plugins |
| GET | `/api/plugins/:id` | Get specific plugin info |
| POST | `/api/plugins/:id/run` | Execute a plugin |
| POST | `/api/run-plugin` | Generic plugin execution |

#### Plugin Management

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/plugins/manage/list` | List installed plugins |
| GET | `/api/plugins/manage/details/:id` | Get plugin details |
| GET | `/api/plugins/manage/available` | List available plugins |
| POST | `/api/plugins/manage/install` | Install from repository |
| POST | `/api/plugins/manage/bulk-install` | Bulk install plugins |
| POST | `/api/plugins/manage/update/:id` | Update a plugin |
| POST | `/api/plugins/manage/uninstall/:id` | Uninstall a plugin |
| POST | `/api/plugins/manage/refresh-catalog` | Refresh from GitHub |
| POST | `/api/plugins/manage/sync` | Sync version information |

### WebSocket Protocol

#### Message Types

```typescript
// Network Update (Server -> Client)
{
  type: "network_update",
  data: NetworkInfo,
  timestamp: string // ISO 8601
}

// Plugin Progress (Server -> Client)
{
  type: "plugin_progress",
  pluginId: string,
  progress: number, // 0-100
  message: string
}

// Plugin Result (Server -> Client)
{
  type: "plugin_result",
  pluginId: string,
  data: any,
  success: boolean,
  error?: string
}
```

### Network Info Structure

```go
type NetworkInfo struct {
    IPv4Address    string         `json:"ipv4Address"`
    IPv6Address    string         `json:"ipv6Address"`
    SubnetMask     string         `json:"subnetMask"`
    Gateway        string         `json:"gateway"`
    SSID           string         `json:"ssid,omitempty"`
    EthernetInfo   EthernetInfo   `json:"ethernetInfo,omitempty"`
    DNSServers     []string       `json:"dnsServers"`
    DHCPInfo       DHCPInfo       `json:"dhcpInfo"`
    VLANInfo       VLANInfo       `json:"vlanInfo,omitempty"`
    Connection     Connection     `json:"connection"`
    Traffic        Traffic        `json:"traffic"`
    ARPEntries     []ARPEntry     `json:"arpEntries"`
    ServiceLatency ServiceLatency `json:"serviceLatency"`
    Timestamp      time.Time      `json:"timestamp"`
}
```

---

## Frontend Specifications

### Component Structure

#### Page Components
- Handle routing and data fetching
- Compose smaller components
- Manage page-level state

#### Container Components
- Connect to context/hooks
- Handle business logic
- Pass data to presentational components

#### Presentational Components
- Pure UI rendering
- Accept props only
- No direct API calls

### State Management

```jsx
// Context Hierarchy
<App>
  <PluginProvider>        // Plugin state and actions
    <WebSocketProvider>   // Real-time data
      <Router>
        <Layout>
          <Pages />
        </Layout>
      </Router>
    </WebSocketProvider>
  </PluginProvider>
</App>
```

### CSS Guidelines

- Use TailwindCSS utility classes
- Custom components in `index.css`
- Dark theme as default
- Glass morphism design language

```css
/* Standard glass card */
.glass-card {
  @apply bg-dark-900/50 backdrop-blur-xl border border-dark-800/50 rounded-2xl;
}

/* Gradient variants */
.gradient-blue { /* Blue accent gradient */ }
.gradient-cyan { /* Cyan accent gradient */ }
.gradient-purple { /* Purple accent gradient */ }
.gradient-orange { /* Orange accent gradient */ }
```

---

## Plugin System

### Plugin Definition (plugin.json)

```json
{
  "id": "example_plugin",
  "name": "Example Plugin",
  "description": "Description of the plugin",
  "version": "1.0.0",
  "author": "Author Name",
  "license": "MIT",
  "icon": "plug",
  "repository": "https://github.com/org/Plugin_example",
  "requires": ["iperf3", "nmap"],
  "parameters": [
    {
      "id": "host",
      "name": "Target Host",
      "description": "The host to test",
      "type": "string",
      "required": true,
      "default": "localhost"
    },
    {
      "id": "count",
      "name": "Count",
      "type": "number",
      "required": false,
      "default": 5,
      "min": 1,
      "max": 100,
      "canIterate": true
    }
  ]
}
```

### Plugin Implementation (plugin.go)

```go
package example_plugin

import (
    "fmt"
    "time"
)

// Execute is the main entry point for the plugin
func Execute(params map[string]interface{}) (interface{}, error) {
    // Extract parameters with type assertions
    host, _ := params["host"].(string)
    countRaw, ok := params["count"].(float64)
    if !ok {
        countRaw = 5
    }
    count := int(countRaw)

    // Plugin logic here...

    return map[string]interface{}{
        "success":   true,
        "host":      host,
        "count":     count,
        "timestamp": time.Now().Format(time.RFC3339),
    }, nil
}
```

### Plugin Categories

| Category | Icon | Description |
|----------|------|-------------|
| Network Analysis | Activity | Traffic analysis, bandwidth testing |
| Network Discovery | Search | Device/port scanning |
| Connectivity | Network | Ping, traceroute, MTU testing |
| Performance | Gauge | Speed tests, iPerf |
| DNS Tools | Globe | DNS lookup, propagation |
| Security | Shield | SSL checks, vulnerability scanning |

---

## API Contracts

### Request/Response Standards

#### Success Response
```json
{
  "success": true,
  "data": { ... },
  "timestamp": "2026-01-01T12:00:00Z"
}
```

#### Error Response
```json
{
  "success": false,
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": { ... }
}
```

### Plugin Execution Request
```json
{
  "id": "plugin_id",
  "params": {
    "host": "example.com",
    "count": 5
  },
  "config": {
    "iterate": false,
    "maxIterations": 0,
    "iterationDelay": 1000
  }
}
```

---

## Data Flow

### Real-time Network Updates

```
1. Backend: NetworkInfo broadcaster (every 3s)
   │
   ├─ Collect metrics via gopsutil
   ├─ Measure service latencies
   ├─ Get ARP table
   │
   └─ Broadcast to all WebSocket clients
         │
         └─ Frontend: useWebSocket hook
               │
               ├─ Parse message
               ├─ Update networkData state
               │
               └─ Components re-render with new data
```

### Plugin Execution Flow

```
1. User clicks "Run Plugin"
   │
   └─ Frontend: POST /api/plugins/:id/run
         │
         └─ Backend: PluginManager.RunPlugin()
               │
               ├─ Validate parameters
               ├─ Get plugin from registry
               ├─ Execute plugin function
               │
               └─ Return result
                     │
                     └─ Frontend: Display results
```

---

## Coding Standards

### Go Code Style

- Follow official Go style guide
- Use `gofmt` for formatting
- Run `golangci-lint` for linting
- Document exported functions
- Handle all errors explicitly

```go
// Good: Explicit error handling
result, err := doSomething()
if err != nil {
    return nil, fmt.Errorf("failed to do something: %w", err)
}
```

### JavaScript/React Style

- Use functional components with hooks
- Destructure props
- Use TypeScript-style JSDoc comments
- Prefer `const` over `let`

```jsx
/**
 * StatsCard component
 * @param {Object} props
 * @param {string} props.title - Card title
 * @param {React.ReactNode} props.children - Card content
 */
export default function StatsCard({ title, children }) {
  return (
    <div className="glass-card p-6">
      <h3 className="text-lg font-semibold">{title}</h3>
      {children}
    </div>
  )
}
```

---

## Testing Guidelines

### Backend Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./app/plugins/...
```

### Frontend Testing

```bash
# Run tests
npm test

# Run with coverage
npm run test:coverage
```

---

## Deployment

### Build Commands

```bash
# Backend
go build -ldflags "-X main.Version=1.0.0 -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o nettool

# Frontend
cd frontend && npm run build

# Combined
./build.sh
```

### Systemd Service

```ini
[Unit]
Description=NetTool Network Diagnostic Tool
After=network.target

[Service]
ExecStart=/opt/nettool/nettool --port 8080
WorkingDirectory=/opt/nettool
Restart=always
User=nettool

[Install]
WantedBy=multi-user.target
```

---

## Roadmap

### Version 2.0 (Current)
- [x] React frontend migration
- [x] Plugin management UI
- [ ] Enhanced plugin execution with streaming
- [ ] Improved error handling
- [ ] Plugin result visualization

### Version 2.1 (Planned)
- [ ] User authentication
- [ ] Role-based access control
- [ ] Plugin marketplace UI
- [ ] Telemetry export

### Version 2.2 (Future)
- [ ] Multi-device management
- [ ] Historical data storage
- [ ] Custom dashboards
- [ ] Alert system

---

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/name`
3. Follow coding standards
4. Write tests for new features
5. Update this spec document if needed
6. Submit pull request

---

## Changelog

### v2.0.0 (2026-01-01)
- Migrated to React frontend
- Added plugin context for global state
- Improved WebSocket handling
- Enhanced plugin execution system
- Added project specifications document

---

*This document should be updated whenever significant changes are made to the project architecture, APIs, or standards.*
