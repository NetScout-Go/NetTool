#!/bin/bash

# NetTool Optimized Linux Build Script
# Builds minimal, high-performance binaries for Linux

set -e

# Configuration
BUILD_DIR="build"
MAIN_BINARY="nettool"
ITERATE_BINARY="nettool-iterate"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags for maximum optimization
LDFLAGS="-w -s"
LDFLAGS="$LDFLAGS -X main.Version=$VERSION"
LDFLAGS="$LDFLAGS -X main.BuildTime=$BUILD_TIME"
LDFLAGS="$LDFLAGS -X main.GitCommit=$GIT_COMMIT"

# Go build flags
BUILDFLAGS="-a -installsuffix cgo"
GCFLAGS="-B -C"
ASMFLAGS="-B"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔨 NetTool Linux Build Script${NC}"
echo -e "${BLUE}================================${NC}"
echo "Version: $VERSION"
echo "Build Time: $BUILD_TIME"
echo "Git Commit: $GIT_COMMIT"
echo ""

# Clean previous builds
echo -e "${YELLOW}🧹 Cleaning previous builds...${NC}"
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Set environment for Linux builds
export GOOS=linux
export CGO_ENABLED=0

# Build main NetTool server
echo -e "${YELLOW}🔧 Building main NetTool server...${NC}"
go build \
    $BUILDFLAGS \
    -ldflags "$LDFLAGS" \
    -gcflags "$GCFLAGS" \
    -asmflags "$ASMFLAGS" \
    -o $BUILD_DIR/$MAIN_BINARY \
    ./main.go

# Build iterate CLI tool
echo -e "${YELLOW}🔧 Building iterate CLI tool...${NC}"
go build \
    $BUILDFLAGS \
    -ldflags "$LDFLAGS" \
    -gcflags "$GCFLAGS" \
    -asmflags "$ASMFLAGS" \
    -o $BUILD_DIR/$ITERATE_BINARY \
    ./app/cmd/iterate/main.go

# Strip binaries for minimal size (if strip is available)
if command -v strip >/dev/null 2>&1; then
    echo -e "${YELLOW}✂️  Stripping binaries...${NC}"
    strip $BUILD_DIR/$MAIN_BINARY
    strip $BUILD_DIR/$ITERATE_BINARY
fi

# Compress binaries with UPX if available
if command -v upx >/dev/null 2>&1; then
    echo -e "${YELLOW}📦 Compressing binaries with UPX...${NC}"
    upx --best --lzma $BUILD_DIR/$MAIN_BINARY
    upx --best --lzma $BUILD_DIR/$ITERATE_BINARY
else
    echo -e "${YELLOW}⚠️  UPX not found, skipping compression${NC}"
fi

# Copy essential files
echo -e "${YELLOW}📂 Copying essential files...${NC}"
mkdir -p $BUILD_DIR/frontend
cp -r frontend/dist/* $BUILD_DIR/frontend/ 2>/dev/null || echo "Frontend not built, run 'npm run build' in frontend/ first"
mkdir -p $BUILD_DIR/app/plugins/plugins
cp app/plugins/config.json.example $BUILD_DIR/app/plugins/
cp app/plugins/config.json $BUILD_DIR/app/plugins/ 2>/dev/null || echo "config.json not found, using example"

# Create systemd service file
echo -e "${YELLOW}⚙️  Creating systemd service file...${NC}"
cat > $BUILD_DIR/nettool.service << EOF
[Unit]
Description=NetTool Network Analysis Server
After=network.target
Wants=network.target

[Service]
Type=simple
User=nettool
Group=nettool
WorkingDirectory=/opt/nettool
ExecStart=/opt/nettool/nettool --port=8080
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nettool
KillMode=mixed
KillSignal=SIGINT
TimeoutStopSec=5

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/nettool
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
EOF

# Create installation script
echo -e "${YELLOW}📜 Creating installation script...${NC}"
cat > $BUILD_DIR/install.sh << 'EOF'
#!/bin/bash

# NetTool Installation Script for Linux

set -e

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

INSTALL_DIR="/opt/nettool"
SERVICE_FILE="/etc/systemd/system/nettool.service"
USER="nettool"
GROUP="nettool"

echo "🚀 Installing NetTool..."
echo "📂 Script location: $SCRIPT_DIR"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root (use sudo)"
    exit 1
fi

# Create user and group
if ! getent group $GROUP >/dev/null; then
    echo "👥 Creating group $GROUP..."
    groupadd --system $GROUP
fi

if ! getent passwd $USER >/dev/null; then
    echo "👤 Creating user $USER..."
    useradd --system --gid $GROUP --shell /bin/false --home-dir $INSTALL_DIR $USER
fi

# Create installation directory
echo "📂 Creating installation directory..."
mkdir -p $INSTALL_DIR
chown $USER:$GROUP $INSTALL_DIR

# Copy files
echo "📦 Installing files..."
# Change to script directory
cd "$SCRIPT_DIR"

# Check if binaries exist
if [ ! -f "nettool" ]; then
    echo "❌ Error: nettool binary not found in $SCRIPT_DIR"
    echo "Contents of $SCRIPT_DIR:"
    ls -la
    exit 1
fi

# Copy binaries
cp nettool $INSTALL_DIR/
cp nettool-iterate $INSTALL_DIR/

# Copy directories and other files
cp -r app $INSTALL_DIR/ 2>/dev/null || echo "No app directory found"
cp README.md $INSTALL_DIR/ 2>/dev/null || echo "No README.md found"

# Set ownership and permissions
echo "🔧 Setting permissions..."
chown -R $USER:$GROUP $INSTALL_DIR

# Make binaries executable (check if they exist first)
if [ -f "$INSTALL_DIR/nettool" ]; then
    chmod +x $INSTALL_DIR/nettool
    echo "✅ Main binary installed"
else
    echo "❌ Error: nettool binary not found in $INSTALL_DIR"
    echo "Contents of $INSTALL_DIR:"
    ls -la $INSTALL_DIR
    exit 1
fi

if [ -f "$INSTALL_DIR/nettool-iterate" ]; then
    chmod +x $INSTALL_DIR/nettool-iterate
    echo "✅ CLI tool installed"
else
    echo "❌ Error: nettool-iterate binary not found in $INSTALL_DIR"
    exit 1
fi

# Install systemd service
echo "⚙️  Installing systemd service..."
if [ -f "$SCRIPT_DIR/nettool.service" ]; then
    cp "$SCRIPT_DIR/nettool.service" $SERVICE_FILE
    systemctl daemon-reload
    systemctl enable nettool
    echo "✅ Systemd service installed"
else
    echo "⚠️  Warning: nettool.service not found, skipping service installation"
fi

# Create symlinks for global access
echo "🔗 Creating symbolic links..."
ln -sf $INSTALL_DIR/nettool /usr/local/bin/nettool
ln -sf $INSTALL_DIR/nettool-iterate /usr/local/bin/nettool-iterate

echo "✅ Installation complete!"
echo ""
echo "Usage:"
echo "  Start service: sudo systemctl start nettool"
echo "  Stop service:  sudo systemctl stop nettool"
echo "  View logs:     sudo journalctl -u nettool -f"
echo "  Web interface: http://localhost:8080"
echo "  CLI tool:      nettool-iterate --help"
echo ""
echo "⚠️  Don't forget to configure GitHub token in $INSTALL_DIR/app/plugins/config.json"
EOF

chmod +x $BUILD_DIR/install.sh

# Create uninstall script
echo -e "${YELLOW}🗑️  Creating uninstall script...${NC}"
cat > $BUILD_DIR/uninstall.sh << 'EOF'
#!/bin/bash

# NetTool Uninstall Script

set -e

INSTALL_DIR="/opt/nettool"
SERVICE_FILE="/etc/systemd/system/nettool.service"
USER="nettool"
GROUP="nettool"

echo "🗑️  Uninstalling NetTool..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run as root (use sudo)"
    exit 1
fi

# Stop and disable service
if systemctl is-active --quiet nettool; then
    echo "⏹️  Stopping NetTool service..."
    systemctl stop nettool
fi

if systemctl is-enabled --quiet nettool; then
    echo "❌ Disabling NetTool service..."
    systemctl disable nettool
fi

# Remove service file
if [ -f "$SERVICE_FILE" ]; then
    echo "🗑️  Removing service file..."
    rm -f $SERVICE_FILE
    systemctl daemon-reload
fi

# Remove symlinks
echo "🔗 Removing symbolic links..."
rm -f /usr/local/bin/nettool
rm -f /usr/local/bin/nettool-iterate

# Remove installation directory
if [ -d "$INSTALL_DIR" ]; then
    echo "📂 Removing installation directory..."
    rm -rf $INSTALL_DIR
fi

# Remove user and group
if getent passwd $USER >/dev/null; then
    echo "👤 Removing user $USER..."
    userdel $USER
fi

if getent group $GROUP >/dev/null; then
    echo "👥 Removing group $GROUP..."
    groupdel $GROUP
fi

echo "✅ NetTool uninstalled successfully!"
EOF

chmod +x $BUILD_DIR/uninstall.sh

# Create README for deployment
echo -e "${YELLOW}📋 Creating deployment README...${NC}"
cat > $BUILD_DIR/README.md << EOF
# NetTool Linux Distribution

This package contains optimized Linux binaries for NetTool.

## Contents

- \`nettool\` - Main NetTool web server
- \`nettool-iterate\` - CLI tool for automated plugin execution
- \`frontend/dist/\` - React web UI (built)
- \`app/plugins/\` - Plugin configuration
- \`nettool.service\` - Systemd service file
- \`install.sh\` - Automatic installation script
- \`uninstall.sh\` - Removal script

## Quick Install

\`\`\`bash
sudo ./install.sh
sudo systemctl start nettool
\`\`\`

Access the web interface at: http://localhost:8080

## Manual Installation

1. Copy files to your preferred location
2. Make binaries executable: \`chmod +x nettool nettool-iterate\`
3. Run: \`./nettool --port=8080\`

## Configuration

Edit \`app/plugins/config.json\` to add your GitHub token for plugin management.

## CLI Usage

\`\`\`bash
# Run a plugin directly
./nettool-iterate -plugin ping -paramsJson '{"host":"8.8.8.8","count":5}'

# Run with iteration
./nettool-iterate -plugin ping -paramsJson '{"host":"8.8.8.8"}' -iterate -max 10 -delay 5
\`\`\`

## Build Info

- Version: $VERSION
- Build Time: $BUILD_TIME
- Git Commit: $GIT_COMMIT
- Target: Linux (CGO disabled)
- Optimization: Full (-w -s flags, stripped, UPX compressed if available)
EOF

# Display build results
echo ""
echo -e "${GREEN}✅ Build completed successfully!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""

# Show binary sizes
echo -e "${BLUE}📊 Binary Information:${NC}"
echo "Main server:"
ls -lh $BUILD_DIR/$MAIN_BINARY | awk '{print "  Size: " $5 "  File: " $9}'
file $BUILD_DIR/$MAIN_BINARY | sed 's/^/  /'

echo ""
echo "CLI tool:"
ls -lh $BUILD_DIR/$ITERATE_BINARY | awk '{print "  Size: " $5 "  File: " $9}'
file $BUILD_DIR/$ITERATE_BINARY | sed 's/^/  /'

echo ""
echo -e "${BLUE}📦 Package Contents:${NC}"
du -sh $BUILD_DIR | awk '{print "  Total size: " $1}'
echo "  Contents:"
find $BUILD_DIR -type f | head -10 | sed 's/^/    /'
if [ $(find $BUILD_DIR -type f | wc -l) -gt 10 ]; then
    echo "    ... and $(($( find $BUILD_DIR -type f | wc -l) - 10)) more files"
fi

echo ""
echo -e "${GREEN}🚀 Ready for deployment!${NC}"
echo "Run: cd $BUILD_DIR && sudo ./install.sh"