# UI & UX Design Standards for DDS238 Modbus Simulator

## Design Philosophy

The user interface must be clean, modern, ultra-responsive, and tailored for embedded firmware engineers, IoT developers, and electrical test benches where clarity, fast feedback, and high visual contrast are essential.

---

## 1. Visual Aesthetics & Theme

1. **Color Palette:**
   - **Primary / Cyber Accent:** Vivid Cyan (`#00f0ff`) & Neon Blue (`#3b82f6`) symbolizing electricity and digital telemetry.
   - **Backgrounds:** Deep Obsidian Slate (`#0b0f19` / `#111827`) with dark mode styling and high contrast.
   - **Status & Register Indicators:**
     - Modbus Active / OK: Emerald Green (`#10b981`)
     - Warning / Jitter: Amber (`#f59e0b`)
     - Fault / Relay Tripped / CRC Error: Crimson Red (`#ef4444`)
     - Modbus Function Codes: Hex Cyan / Violet Chips (`#06b6d4` / `#8b5cf6`)
2. **Modern Styling Elements:**
   - Dark glassmorphism cards with subtle borders (`rgba(255, 255, 255, 0.08)`).
   - Distinct telemetry cards with glowing accent metrics (Voltage, Current, Power, Energy).
   - High-contrast monospace font (`Fira Code`, `JetBrains Mono`, `Consolas`) for Modbus hex packets, byte inspectors, and C code snippets.

---

## 2. Packet Inspector & Wire Sniffer UX

1. **Byte-by-Byte Tagging:**
   - Each raw byte must be clearly mapped to its Modbus protocol meaning (Slave ID, Function Code, Register, Byte Count, Data, CRC).
   - Use distinct chip/pill tags (`.byte-tag`) pairing a highlighted hex pill with a descriptive white label.
2. **Device Origin Identification:**
   - Always tag packets with the originating meter identity (e.g. `[Meter #1 Main Grid]`, `[Meter #2 Solar PV]`) so developers can easily trace multi-drop RS-485 traffic.

---

## 3. Multi-Meter Grid Switching UX

1. **Tabbed Switcher:**
   - Prominent top tabs showing each simulated meter, Server ID, and operational state.
   - One-click `[ + Add Virtual Meter ]` button with customizable Server ID and name presets.
2. **Instant Preset Injection:**
   - Clean action chips for quick physical test states (e.g. Normal Grid, Undervoltage 180V, Heavy Load 45A, Solar Export 3.5kW).
