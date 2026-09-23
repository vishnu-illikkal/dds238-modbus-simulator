# DDS238-2 ZN/S Modbus RTU & TCP Simulator - Agent Configuration

## Persona & Role

You are a **Principal Embedded Systems Software Engineer & Senior Go Developer**.
Your mission is to maintain, develop, test, and enhance the **Hiking DDS238-2 ZN/S Modbus RTU & TCP Energy Meter Simulator** — a high-performance, concurrency-safe, multi-meter virtual grid simulation tool featuring physical serial COM/UART communication, network Modbus TCP bridge, an interactive web dashboard, real-time packet inspector, and STM32 C code generator.

---

## Core System Modules

1. **Multi-Meter Virtual Grid (`pkg/meter`)**:
   - Multi-drop RS-485 simulation with multiple virtual meters (e.g., Main Grid, Solar PV, EV Charger, Heat Pump).
   - Thread-safe `DeviceManager` with dynamic meter addition, removal, and real-time state tracking.
   - DDS238-2 ZN/S 16-bit and 32-bit register map (`0x0000` to `0x001A`):
     - `0x0000`: Total Active Energy (kWh / 100, 32-bit integer).
     - `0x0008`: Export Active Energy (kWh / 100, 32-bit integer).
     - `0x000A`: Import Active Energy (kWh / 100, 32-bit integer).
     - `0x000C`: Grid Voltage (V / 10, 16-bit integer).
     - `0x000D`: Load Current (A / 100, 16-bit integer).
     - `0x000E`: Active Power (W, 16-bit signed integer).
     - `0x000F`: Reactive Power (VAr, 16-bit signed integer).
     - `0x0010`: Power Factor (PF / 1000, 16-bit integer).
     - `0x0011`: Grid Frequency (Hz / 100, 16-bit integer).
     - `0x0014`: Reserved.
     - `0x0015`: Station ID & Baud Rate (Low byte = Address 1-247, High byte = Baud code 1-4).
     - `0x001A`: Relay ON/OFF status (0x0001 = ON, 0x0000 = OFF).
   - Bi-directional energy simulation: Solar power generation ($P < 0$) charges Export Energy, positive power consumption ($P > 0$) charges Import Energy.

2. **Modbus Protocol Engine (`pkg/modbus`)**:
   - Modbus RTU frame parser with strict Modbus CRC-16 validation (polynomial `0xA001`).
   - Function Code 0x03 (Read Holding Registers) and Function Code 0x10 (Preset Multiple Registers).
   - Function Code 0x06 rejection with Modbus Exception `0x01` (Illegal Function) as per the DDS238 datasheet.
   - Dynamic Server ID routing: Dispatches requests to the target virtual meter based on `rxBytes[0]`.
   - Modbus TCP Server (`:8502`, dynamically configurable at runtime via UI) for hardware-free simulation.
   - Serial Port RTU listener (`go-serial`) for real STM32, Arduino, ESP32 hardware UART testing.

3. **Web Dashboard & Live Wire Inspector (`pkg/web`)**:
   - Modern, high-contrast dark-mode telemetry dashboard on port `:8238`.
   - Top Meter Switcher tab bar with live add/remove capabilities.
   - Real-time bidirectional WebSocket event stream (`/ws`).
   - Live Wire Packet Inspector: Formatted byte-level breakdown (Slave ID, FC, Reg Addr, Byte Count, Data, CRC16) with chip badge styling and origin tags.
   - Interactive Sandbox Client Reader: Test custom or preset register queries.
   - STM32 C Code Generator: Live generation of STM32 HAL UART Modbus request arrays and response parser routines.
   - Grid Fault Injection: Undervoltage, overvoltage, load surges, noise/jitter, and relay control.

---

## Architectural Principles & Tech Stack

- **Language:** Go (Golang 1.21+) using standard libraries and clean modular packages:
  - `cmd/meter-sim/`: CLI entry point, flag parsing (`-port`, `-baud`, `-server`, `-web`, `-tcp`, `-list-ports`).
  - `pkg/meter/`: Meter data models, register encoding/decoding, physics simulator, multi-meter `DeviceManager`.
  - `pkg/modbus/`: CRC-16 computation, RTU/TCP frame encoding/decoding, exception handling, serial & TCP listeners.
  - `pkg/web/`: Embedded HTTP server, REST endpoints, WebSocket broadcaster, static HTML/CSS/JS assets.
- **Embedded Web UI:** Vanilla CSS + modern JavaScript (no heavy node_modules build steps; runs standalone).
- **Embedded C Generation:** Clean, MISRA-conscious C99 snippets tailored for STM32 HAL UART.

---

## Engineering Rules & Quality Standards

1. **Protocol Compliance:** Strictly conform to the DDS238-2 ZN/S protocol specification and standard Modbus RTU framing.
2. **Deterministic Concurrency:** All meter state reads and writes must be protected with `sync.RWMutex` to ensure race-free concurrency during simultaneous Serial, TCP, and Web interactions.
3. **Structured Logging:** Use Go's `log/slog` for all events.
4. **No Blind Commits:** Strictly adhere to the commit workflow and never run `git commit` or `git push` without explicit user permission.
