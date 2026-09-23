#!/bin/bash
set -e

echo "=================================================================="
echo "   Building DDS238 Modbus Simulator for ALL Platforms"
echo "=================================================================="
mkdir -p bin

echo "[1/5] Building Windows (x86_64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator.exe ./cmd/meter-sim

echo "[2/5] Building Linux (x86_64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-linux-amd64 ./cmd/meter-sim

echo "[3/5] Building Linux (ARM64 / Raspberry Pi)..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-linux-arm64 ./cmd/meter-sim

echo "[4/5] Building macOS (Apple Silicon M1/M2/M3)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-darwin-arm64 ./cmd/meter-sim

echo "[5/5] Building macOS (Intel x86_64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/dds238-modbus-simulator-darwin-amd64 ./cmd/meter-sim

chmod +x bin/dds238-modbus-simulator-* 2>/dev/null || true

echo ""
echo "=================================================================="
echo "[SUCCESS] All binaries compiled into bin/:"
echo "  - bin/dds238-modbus-simulator.exe          (Windows x86_64)"
echo "  - bin/dds238-modbus-simulator-linux-amd64  (Linux x86_64)"
echo "  - bin/dds238-modbus-simulator-linux-arm64  (Linux ARM64 / Pi)"
echo "  - bin/dds238-modbus-simulator-darwin-arm64 (macOS Apple Silicon)"
echo "  - bin/dds238-modbus-simulator-darwin-amd64 (macOS Intel)"
echo "=================================================================="
