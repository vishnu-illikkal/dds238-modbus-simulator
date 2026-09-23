#!/bin/bash
if [ ! -f "bin/dds238-modbus-simulator" ]; then
    echo "Binary not found in bin/. Building first..."
    ./build_mac.sh
fi

echo "Starting DDS238 Modbus Simulator..."
echo "Web UI will be available at: http://localhost:8238"
./bin/dds238-modbus-simulator "$@"
