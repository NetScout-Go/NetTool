# NetTool Binary Security

This document describes the security measures implemented in NetTool binaries to protect against tampering, reverse engineering, and unauthorized modifications.

## 🔐 Security Features

### 1. Binary Integrity Verification

NetTool verifies its own integrity at startup by comparing its hash against known-good values.

**How it works:**
1. At startup, NetTool calculates a SHA256 hash of the running binary
2. This hash is compared against:
   - An embedded hash (set at build time)
   - Or fetched from the official GitHub releases
3. If hashes don't match, a warning is displayed

**Usage:**
```bash
# Check integrity status and exit
nettool --integrity

# Skip integrity check (not recommended)
nettool --skip-integrity

# Normal startup (integrity check enabled by default)
nettool --port 8080
```

**API Endpoint:**
```bash
# Get integrity status via API
curl http://localhost:8080/api/integrity
```

### 2. Build Hardening

Official releases are built with maximum hardening:

| Feature | Description | Flag |
|---------|-------------|------|
| Symbol Stripping | Removes debug symbols | `-w -s` |
| Path Removal | Removes build paths from binary | `-trimpath` |
| Build ID Removal | Removes unique build identifier | `-buildid=` |
| Static Linking | No external dependencies | `CGO_ENABLED=0` |
| Optimized Tags | Static network/user resolution | `netgo,osusergo` |

### 3. Optional Obfuscation (Garble)

For maximum protection against reverse engineering, builds can use [garble](https://github.com/burrowers/garble):

```bash
# Install garble
go install mvdan.cc/garble@latest

# Build with obfuscation
USE_GARBLE=true ./build-secure.sh
```

Garble provides:
- Function/variable name obfuscation
- String literal obfuscation
- Package path obfuscation
- Control flow obfuscation

### 4. UPX Compression

Binaries can be compressed with [UPX](https://github.com/upx/upx):

```bash
# Install UPX (Ubuntu/Debian)
sudo apt install upx

# Install UPX (macOS)
brew install upx

# Build with UPX compression
USE_UPX=true ./build-secure.sh
```

Benefits:
- Reduces binary size by 50-70%
- Adds layer of obfuscation
- Self-extracting at runtime

### 5. Anti-Tampering Checks

At runtime, NetTool performs additional checks:
- Runtime environment validation
- Memory integrity verification
- Stack corruption detection

## 🛠️ Build Scripts

### Linux/macOS: `build-secure.sh`

```bash
# Default secure build
./build-secure.sh

# With all protections
USE_GARBLE=true USE_UPX=true ./build-secure.sh

# For specific platform
TARGET_OS=linux TARGET_ARCH=arm64 ./build-secure.sh
```

### Windows: `build-secure.ps1`

```powershell
# Default secure build
.\build-secure.ps1

# With obfuscation
.\build-secure.ps1 -UseGarble

# Without UPX
.\build-secure.ps1 -UseUPX:$false
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `USE_GARBLE` | `false` | Enable garble obfuscation |
| `USE_UPX` | `true` | Enable UPX compression |
| `ENABLE_INTEGRITY` | `true` | Enable integrity verification |
| `TARGET_OS` | `linux` | Target operating system |
| `TARGET_ARCH` | `amd64` | Target architecture |

## 📋 Verification

### Verify Downloaded Binary

```bash
# Download checksums
curl -sL https://github.com/NetScout-Go/NetTool/releases/latest/download/SHA256SUMS -o SHA256SUMS

# Verify
sha256sum -c SHA256SUMS
```

### Verify Running Binary

```bash
# Show detailed integrity status
nettool --integrity
```

### API Verification

```bash
curl -s http://localhost:8080/api/integrity | jq
```

Response:
```json
{
  "verified": true,
  "expected_hash": "abc123...",
  "actual_hash": "abc123...",
  "source": "github",
  "checked_at": "2026-01-02T12:00:00Z",
  "binary_path": "/opt/nettool/nettool",
  "binary_size": 15728640,
  "runtime_os": "linux",
  "runtime_arch": "amd64",
  "tamper_detected": false
}
```

## 🔒 Best Practices

### For Users

1. **Always verify downloads** using provided checksums
2. **Don't disable integrity checks** unless necessary for development
3. **Keep binaries updated** to get latest security patches
4. **Run as non-root user** via systemd service
5. **Use firewall rules** to restrict access

### For Developers

1. **Never commit secrets** to the repository
2. **Use secure build scripts** for releases
3. **Sign releases** with GPG when possible
4. **Keep dependencies updated** for security patches

## 📊 Binary Size Optimization

| Optimization | Size Reduction | Notes |
|--------------|----------------|-------|
| `-w -s` flags | ~30% | Strips symbols |
| `-trimpath` | ~5% | Removes paths |
| UPX --best | ~50% | LZMA compression |
| garble -tiny | ~10% | Removes extra data |

**Example sizes (Linux amd64):**
- Unoptimized: ~25 MB
- With `-w -s`: ~18 MB
- With UPX: ~8 MB
- With garble + UPX: ~7 MB

## ⚠️ Limitations

1. **Obfuscation is not encryption** - Determined attackers can still reverse engineer
2. **UPX can be unpacked** - Tools exist to decompress UPX binaries
3. **Hash verification requires network** - GitHub-based verification needs internet
4. **Embedded hash has bootstrap problem** - Can't hash itself perfectly

## 🔗 Resources

- [Go Build Modes](https://golang.org/cmd/go/#hdr-Build_modes)
- [Garble Documentation](https://github.com/burrowers/garble)
- [UPX Documentation](https://upx.github.io/)
- [Go Security Best Practices](https://golang.org/doc/security)
