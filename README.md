# Hiking DDS238-2 ZN/S Modbus RTU & TCP Energy Meter Simulator

[![Go Version](https://img.shields.io/github/go-mod/go-version/vishnu-illikkal/dds238-modbus-simulator)](https://golang.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/vishnu-illikkal/dds238-modbus-simulator.svg)](https://pkg.go.dev/github.com/vishnu-illikkal/dds238-modbus-simulator)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-informational)](#-quick-start)
[![Modbus Protocol](https://img.shields.io/badge/Modbus-RTU%20(RS485)%20%7C%20TCP-orange)](https://github.com/vishnu-illikkal/dds238-modbus-simulator)

> **Also known as:** Hiking DDS238-2 ZN/S, DDS238-2 ZN-S, DDS238-2ZN/S, DDS238-2, Hiking Smart DIN-Rail Energy Meter, DDS238 Modbus RTU / TCP Simulator.

A high-performance Modbus RTU / TCP simulator and interactive debugging tool written in Go, designed for developing and testing Modbus client (controller) applications on **STM32**, Arduino, ESP32, Raspberry Pi, or PC.

Includes a **built-in modern Web Dashboard, Client Reader & Packet Inspector** that provides real-time telemetry, live Modbus wire traffic analysis, fault injection, and instant **STM32 C code generation**.

---

## ⚡ Features

- **Multi-Meter Virtual Grid (Multi-Drop RS-485 Simulation)**:
  - Simulate multiple independent energy meters simultaneously on the same Serial COM port and TCP bridge (e.g. **Meter #1 Main Grid**, **Meter #2 Solar PV Inverter**, **Meter #3 EV Charger**).
  - Add, remove, and switch between virtual meters on the fly with real-time UI tabs.
- **Full DDS238-2 ZN/S Register Support**:
  - Registers `0x0000` through `0x001A` (Voltage, Current, Active/Reactive Power, Power Factor, Frequency, Total/Export/Import Energy, Station Address/Baud, Relay Control).
  - Handles **Function Code 0x03** (Read Holding Registers) & **Function Code 0x10** (Write Multiple Registers).
  - Accurately rejects **Function Code 0x06** with Exception `0x01` (Illegal Function) as per the meter specification.
  - Zeroing total energy counter resets all energy accumulators.
- **Bi-Directional Solar & Grid Energy Accumulation**:
  - Simulates both forward power consumption (**Import Energy**) and reverse solar generation (**Export Energy**).
- **Physical UART / Serial & Network Transports**:
  - Direct connection to STM32 via USB-to-UART converter (COM port on Windows / `/dev/ttyUSB` on Linux).
  - Modbus TCP bridge (`:8502`, customizable in UI) for hardware-free simulation and virtual testing.
- **Web Dashboard & Client Reader (`http://localhost:8238`)**:
  - **Live Gauges & Telemetry**: Voltage, Current, Powers, Frequency, PF, Import/Export/Total Energy, Relay.
  - **Live Wire Packet Inspector**: Real-time stream of incoming (RX from STM32) and outgoing (TX) Modbus frames with CRC checks, byte-by-byte breakdowns, and device origin tagging.
  - **Interactive Reader Sandbox**: Test preset or custom Modbus queries for any meter ID.
  - **STM32 C Code Generator**: Generates copy-paste ready C arrays and parsing code for STM32 HAL UART.
  - **Fault Injection**: Sliders for grid voltage (undervoltage/overvoltage), load current, noise/jitter, and relay trips.

---

## 🚀 Quick Start

### 1. Launch with One-Click Scripts

```powershell
# Windows (PowerShell / Command Prompt):
.\run_windows.bat

# Linux:
./run_linux.sh

# macOS:
./run_mac.sh
```

*(These scripts automatically verify and compile the binary if missing before launching).*

---

### 2. Direct Binary Usage (`bin/`)

You can run the pre-built standalone binaries directly from the `bin/` folder without needing Go installed:

#### Windows (PowerShell / Command Prompt):
```powershell
# Run with default settings (Web UI on :8238, TCP bridge on :8502):
.\bin\dds238-modbus-simulator.exe

# Run connected to STM32 on COM3 at 9600 baud with Server ID 1:
.\bin\dds238-modbus-simulator.exe -port COM3 -baud 9600 -server 1

# List all available serial COM ports on the system:
.\bin\dds238-modbus-simulator.exe -list-ports

# View all CLI options and flags:
.\bin\dds238-modbus-simulator.exe -help
```

#### Linux & Raspberry Pi:
```bash
# x86_64:
./bin/dds238-modbus-simulator-linux-amd64 -port /dev/ttyUSB0 -server 1

# ARM64 (Raspberry Pi / Embedded Linux):
./bin/dds238-modbus-simulator-linux-arm64 -port /dev/ttyUSB0 -server 1
```

#### macOS (Apple Silicon & Intel):
```bash
# Apple Silicon (M1/M2/M3):
./bin/dds238-modbus-simulator-darwin-arm64 -port /dev/tty.usbserial-10 -server 1

# Intel Mac:
./bin/dds238-modbus-simulator-darwin-amd64 -port /dev/tty.usbserial-10 -server 1
```

---

### ⚙️ Command-Line Flags Reference

| Flag | Default | Description | Example |
|---|---|---|---|
| `-port` | `""` | Serial / USB-to-UART COM port | `-port COM3` or `-port /dev/ttyUSB0` |
| `-baud` | `9600` | Serial baud rate (`9600`, `4800`, `2400`, `1200`) | `-baud 9600` |
| `-server` | `1` | Modbus Server / Unit ID address (`1` to `247`) | `-server 1` |
| `-http` | `:8238` | Web Dashboard and REST API listen address | `-http :8238` or `-http :9000` |
| `-tcp` | `:8502` | Modbus TCP bridge listen address | `-tcp :8502` or `-tcp :502` |
| `-list-ports`| `false` | Detect and list all active serial COM ports, then exit | `-list-ports` |
| `-help` | — | Display help message and list of all flags | `-help` |

---

### 3. Open the Web Dashboard

Open your browser and navigate to:
👉 **[http://localhost:8238](http://localhost:8238)**

---

## 🔌 Hardware Connection (STM32 ↔ PC)

```
+---------------------------+             +-------------------------------+
|     STM32 Microcontroller |             |    USB-to-UART Adapter (PC)   |
|     (e.g., STM32F4 / F1)  |             |    (FTDI / CP2102 / CH340)    |
|                           |             |                               |
|  USART2_TX (PA2)          |------------>|  RX Pin                       |
|  USART2_RX (PA3)          |<------------|  TX Pin                       |
|  GND                      |-------------|  GND                          |
+---------------------------+             +-------------------------------+
```

> **Note**: For industrial RS-485 setups, connect a **MAX3485 (3.3V)** transceiver between the STM32 UART and a USB-RS485 adapter on your PC.

---

## 💡 Importance of Connecting to a COM Port & Why It's Needed

While the simulator includes a Modbus TCP bridge (`:8502`) for pure software testing, **connecting the simulator to a real Serial COM Port (via USB-to-UART or USB-to-RS485) is essential for embedded development**:

### 1. 🛡️ Safe Hardware-in-the-Loop (HIL) Testing Without High Voltage (230V AC)
* A physical DDS238 energy meter typically operates on dangerous **230V AC mains electricity**.
* Connecting the simulator to your PC's COM port allows your STM32/microcontroller to communicate with a realistic server device over genuine UART/RS-485 wires **without exposing your desk, probes, or microcontroller to high-voltage AC**.

### 2. 🔍 True Physical-Layer Validation (Baud, Framing & Timing)
* Modbus RTU relies on strict serial framing (**8 data bits, 1 stop bit, No parity**, 9600 default baud) and inter-frame silent intervals ($> 3.5$ character times).
* Testing over an actual COM port validates your STM32's **hardware peripherals, baud rate clock dividers, UART interrupts / DMA channels, ring buffers, and timeout logic** under realistic physical constraints.

### 3. 🧪 Live Modbus RTU CRC-16 & Frame Verification
* Modbus RTU appends a 2-byte CRC-16 checksum to every request and response.
* By passing frames through the COM port, the simulator's **Live Packet Inspector** validates whether your STM32 calculates CRC-16 (Modbus polynomial `0xA001`) accurately and alerts you if any frame is corrupted, truncated, or malformed.

### 4. 🔄 Instant Simulation vs. Production Parity
* When your STM32 code works seamlessly with the simulator over the USB-to-UART COM port, you can swap the USB adapter with a real DDS238 meter on an RS-485 bus with **zero firmware modifications**.

| Mode | Transport | Best Used For |
|---|---|---|
| **COM Port (Modbus RTU)** | USB-to-UART / RS-485 | **STM32, ESP32, Arduino hardware firmware validation**, HAL UART driver debugging, DMA & ISR testing |
| **TCP Bridge (`:8502`)** | Network Socket | SCADA software, Node-RED, Python Modbus scripts, or virtual testing when no hardware dongle is connected |

---

## 📊 Register Map Reference

| Register(s) | Parameter | Units / Scale | Type | Byte Order | Function Codes |
|---|---|---|---|---|---|
| `0x0000 - 0x0001` | Total Energy | `0.01 kWh` | `uint32` (2 regs) | Big-Endian | 0x03, 0x10 (write 0 to reset) |
| `0x0008 - 0x0009` | Export Energy | `0.01 kWh` | `uint32` (2 regs) | Big-Endian | 0x03 |
| `0x000A - 0x000B` | Import Energy | `0.01 kWh` | `uint32` (2 regs) | Big-Endian | 0x03 |
| `0x000C` | Voltage | `0.1 V` | `uint16` (1 reg) | Big-Endian | 0x03 |
| `0x000D` | Current | `0.01 A` | `uint16` (1 reg) | Big-Endian | 0x03 |
| `0x000E` | Active Power | `1 W` | `int16` (1 reg) | Big-Endian | 0x03 (Signed) |
| `0x000F` | Reactive Power | `1 VAr` | `uint16` (1 reg) | Big-Endian | 0x03 |
| `0x0010` | Power Factor | `0.001` | `uint16` (1 reg) | Big-Endian | 0x03 |
| `0x0011` | Frequency | `0.01 Hz` | `uint16` (1 reg) | Big-Endian | 0x03 |
| `0x0015` | Addr / Baud | High: Addr (1-247), Low: Baud (1-4) | `uint16` | Big-Endian | 0x03, 0x10 |
| `0x001A` | Relay Control | `0 = Off, 1 = On` | `uint16` | Big-Endian | 0x03, 0x10 |

---

## 💻 STM32 HAL C Code Examples

### 1. Read Voltage (`0x000C`, 1 Register)

```c
// Modbus RTU Read Voltage Request:
// [Server/Unit=0x01] [Func=0x03] [Addr=0x00, 0x0C] [Count=0x00, 0x01] [CRC=0x44, 0x09]
uint8_t req_voltage[] = { 0x01, 0x03, 0x00, 0x0C, 0x00, 0x01, 0x44, 0x09 };
uint8_t rx_buf[7];

// Transmit request
HAL_UART_Transmit(&huart2, req_voltage, sizeof(req_voltage), 100);

// Receive 7-byte response: [01][03][02][HighByte][LowByte][CRC_L][CRC_H]
if (HAL_UART_Receive(&huart2, rx_buf, sizeof(rx_buf), 200) == HAL_OK) {
    uint16_t raw_v = (rx_buf[3] << 8) | rx_buf[4];
    float voltage = raw_v / 10.0f; // Scale is 0.1 V
    printf("Voltage: %.1f V\r\n", voltage);
}
```

### 2. Read All Power Parameters (`0x000C` to `0x0011`, 6 Registers)

```c
// Request: Read 6 registers starting at 0x000C
// [Server/Unit=0x01][Func=0x03][Addr=0x00, 0x0C][Count=0x00, 0x06][CRC=0x84, 0x0A]
uint8_t req_all[] = { 0x01, 0x03, 0x00, 0x0C, 0x00, 0x06, 0x84, 0x0A };
uint8_t rx_buf[17]; // 1 + 1 + 1 + 12 + 2 = 17 bytes

HAL_UART_Transmit(&huart2, req_all, sizeof(req_all), 100);

if (HAL_UART_Receive(&huart2, rx_buf, sizeof(rx_buf), 300) == HAL_OK) {
    float voltage = ((rx_buf[3] << 8) | rx_buf[4]) / 10.0f;
    float current = ((rx_buf[5] << 8) | rx_buf[6]) / 100.0f;
    int16_t active_power = (int16_t)((rx_buf[7] << 8) | rx_buf[8]);
    uint16_t reactive_power = (rx_buf[9] << 8) | rx_buf[10];
    float power_factor = ((rx_buf[11] << 8) | rx_buf[12]) / 1000.0f;
    float frequency = ((rx_buf[13] << 8) | rx_buf[14]) / 100.0f;

    printf("V: %.1fV | I: %.2fA | P: %dW | Q: %dVAr | PF: %.3f | F: %.2fHz\r\n",
           voltage, current, active_power, reactive_power, power_factor, frequency);
}
```

### 3. Read Total Energy (`0x0000` - `0x0001`, 2 Registers / 32-bit DWord)

```c
uint8_t req_energy[] = { 0x01, 0x03, 0x00, 0x00, 0x00, 0x02, 0xC4, 0x0B };
uint8_t rx_buf[9]; // 1 + 1 + 1 + 4 + 2 = 9 bytes

HAL_UART_Transmit(&huart2, req_energy, sizeof(req_energy), 100);

if (HAL_UART_Receive(&huart2, rx_buf, sizeof(rx_buf), 200) == HAL_OK) {
    uint32_t raw_energy = ((uint32_t)rx_buf[3] << 24) |
                          ((uint32_t)rx_buf[4] << 16) |
                          ((uint32_t)rx_buf[5] << 8)  |
                          ((uint32_t)rx_buf[6]);
    float total_kwh = raw_energy / 100.0f; // Scale is 0.01 kWh
    printf("Total Energy: %.2f kWh\r\n", total_kwh);
}
```

---

## 🧪 Testing and Verification

Run all unit tests:
```bash
go test -v ./...
```

---

## 📚 References & External Links

- [DDS238-2 ZN-S Modbus Protocol Register Documentation (fawno/Modbus)](https://github.com/fawno/Modbus/blob/master/DDS238-2%20ZN-S%20Modbus.md)
- [DDS238-2 ZN-S Modbus RTU Reference Gist (alphp)](https://gist.github.com/alphp/95e1efe916c0dd6df7156f43dd521d53)

---

## 🏷️ Keywords & Search Tags

`Hiking DDS238-2 ZN/S` · `Hiking DDS238-2 ZN-S` · `DDS238-2ZN/S` · `DDS238-2` · `DDS238` · `Modbus RTU Simulator` · `Modbus TCP Simulator` · `RS-485 Energy Meter` · `Smart Meter Simulation` · `STM32 Modbus Master` · `ESP32 Modbus Master` · `Arduino Modbus RTU` · `DIN-Rail Power Meter` · `Virtual Grid Telemetry` · `Power Factor Meter` · `Bi-directional Solar Meter` · `Go Modbus Server`
