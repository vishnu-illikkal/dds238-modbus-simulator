#!/bin/bash
set -e

echo "Building DDS238 Modbus Simulator for Linux (AMD64 & ARM64)..."
mkdir -p bin

# Linux x86_64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-linux-amd64 ./cmd/meter-sim
# Linux ARM64 (Raspberry Pi / Embedded Linux)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-linux-arm64 ./cmd/meter-sim
# Default symlink/copy for direct execution
cp bin/dds238-modbus-simulator-linux-amd64 bin/dds238-modbus-simulator
chmod +x bin/dds238-modbus-simulator*

echo "[SUCCESS] Linux binaries built in bin/:"
echo "  - bin/dds238-modbus-simulator (AMD64)"
echo "  - bin/dds238-modbus-simulator-linux-arm64"
