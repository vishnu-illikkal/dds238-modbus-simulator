---
description: Starts the DDS238 Modbus Simulator development server and checks application health.
---

# Run Development Server Workflow

Use this workflow to launch and verify the local development environment for the DDS238 Modbus simulator.

---

## Step 1: Verify Dependencies
Ensure all Go modules and static assets are installed:
```bash
go mod tidy
```

## Step 2: Launch the Simulator
Run the simulator with the Web Dashboard and Modbus TCP server:
```powershell
go run cmd/meter-sim/main.go
```

Or connect to an STM32 on a serial COM port:
```powershell
go run cmd/meter-sim/main.go -port COM3 -baud 9600 -server 1
```

## Step 3: Health & Web UI Check
Verify that the simulator starts properly:
- Web Dashboard: Open `http://localhost:8238` in your browser.
- Modbus TCP Bridge: Connect your Modbus TCP client to `localhost:8502`.
