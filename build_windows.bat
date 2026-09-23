@echo off
echo Building DDS238 Modbus Simulator for Windows (AMD64)...
if not exist "bin" mkdir bin
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o bin\dds238-modbus-simulator.exe .\cmd\meter-sim
if %ERRORLEVEL% EQU 0 (
    echo [SUCCESS] Binary built at: bin\dds238-modbus-simulator.exe
) else (
    echo [ERROR] Build failed!
)
