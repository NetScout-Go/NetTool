#!/bin/bash

# NetTool Cross-Platform Build Script
# Builds optimized binaries for multiple Linux architectures

set -e

# Configuration
BUILD_DIR="dist"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Target architectures
TARGETS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "linux/386"
)

# Build flags for maximum optimization
LDFLAGS="-w -s"
LDFLAGS="$LDFLAGS -X main.Version=$VERSION"
LDFLAGS="$LDFLAGS -X main.BuildTime=$BUILD_TIME"
LDFLAGS="$LDFLAGS -X main.GitCommit=$GIT_COMMIT"

BUILDFLAGS="-a -installsuffix cgo"
GCFLAGS="-B -C"
ASMFLAGS="-B"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔨 NetTool Multi-Architecture Build${NC}"
echo -e "${BLUE}=====================================${NC}"
echo "Version: $VERSION"
echo ""

# Clean previous builds
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# Set common environment
export CGO_ENABLED=0

# Build for each target
for target in "${TARGETS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$target"
    
    echo -e "${YELLOW}🔧 Building for $GOOS/$GOARCH...${NC}"
    
    # Create target directory
    TARGET_DIR="$BUILD_DIR/nettool-$VERSION-$GOOS-$GOARCH"
    mkdir -p "$TARGET_DIR"
    
    # Set environment
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    
    # Build main binary
    go build \
        $BUILDFLAGS \
        -ldflags "$LDFLAGS" \
        -gcflags "$GCFLAGS" \
        -asmflags "$ASMFLAGS" \
        -o "$TARGET_DIR/nettool" \
        ./main.go
    
    # Build CLI tool
    go build \
        $BUILDFLAGS \
        -ldflags "$LDFLAGS" \
        -gcflags "$GCFLAGS" \
        -asmflags "$ASMFLAGS" \
        -o "$TARGET_DIR/nettool-iterate" \
        ./app/cmd/iterate/main.go
    
    # Copy React frontend
    mkdir -p "$TARGET_DIR/frontend/dist"
    cp -r frontend/dist/* "$TARGET_DIR/frontend/dist/" 2>/dev/null || echo "Frontend not built"
    mkdir -p "$TARGET_DIR/app/plugins/plugins"
    cp app/plugins/config.json.example "$TARGET_DIR/app/plugins/"
    cp README.md "$TARGET_DIR/" 2>/dev/null || echo "README.md not found"
    
    # Create archive
    echo -e "${YELLOW}📦 Creating archive for $GOOS/$GOARCH...${NC}"
    cd "$BUILD_DIR"
    tar -czf "nettool-$VERSION-$GOOS-$GOARCH.tar.gz" "nettool-$VERSION-$GOOS-$GOARCH/"
    cd ..
    
    # Show size
    SIZE=$(du -sh "$TARGET_DIR" | cut -f1)
    ARCHIVE_SIZE=$(du -sh "$BUILD_DIR/nettool-$VERSION-$GOOS-$GOARCH.tar.gz" | cut -f1)
    echo "  📊 Build size: $SIZE, Archive: $ARCHIVE_SIZE"
done

echo ""
echo -e "${GREEN}✅ Multi-architecture build complete!${NC}"
echo -e "${GREEN}Packages created in $BUILD_DIR/${NC}"