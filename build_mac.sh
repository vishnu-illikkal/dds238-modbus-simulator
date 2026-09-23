#!/bin/bash
set -e

echo "Building DDS238 Modbus Simulator for macOS (Apple Silicon M1/M2/M3 & Intel)..."
mkdir -p bin

# macOS Apple Silicon (ARM64)
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-darwin-arm64 ./cmd/meter-sim
# macOS Intel (AMD64)
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-darwin-amd64 ./cmd/meter-sim

# Universal / default selection based on current host architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    cp bin/dds238-modbus-simulator-darwin-arm64 bin/dds238-modbus-simulator
else
    cp bin/dds238-modbus-simulator-darwin-amd64 bin/dds238-modbus-simulator
fi

chmod +x bin/dds238-modbus-simulator*

echo "[SUCCESS] macOS binaries built in bin/:"
echo "  - bin/dds238-modbus-simulator (Host Native)"
echo "  - bin/dds238-modbus-simulator-darwin-arm64 (Apple Silicon)"
echo "  - bin/dds238-modbus-simulator-darwin-amd64 (Intel Mac)"
