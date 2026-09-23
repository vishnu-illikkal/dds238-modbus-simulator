---
description: Runs the automated Go test suite with data race detection for the Modbus simulator.
---

# Test Simulator & Concurrency Workflow

Use this workflow to run unit, protocol decoding, and concurrency race detection tests across the DDS238 Modbus simulator.

---

## Step 1: Run Unit Tests with Race Detector
Execute Go tests across all simulator packages:
```powershell
go test -v -race ./...
```

## Step 2: Validate Modbus Protocol Frame & CRC Tests
Ensure Modbus RTU CRC16, frame decoding, and register mapping pass:
```powershell
go test -v -race ./pkg/modbus/...
go test -v -race ./pkg/meter/...
```

## Step 3: Check Code Formatting & Lints
Verify `go vet` and formatting:
```powershell
go vet ./...
gofmt -l .
```
