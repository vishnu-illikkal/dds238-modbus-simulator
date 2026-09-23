@echo off
if not exist "bin\dds238-modbus-simulator.exe" (
    echo Binary not found in bin\. Building first...
    call build_windows.bat
)
echo Starting DDS238 Modbus Simulator...
echo Web UI will be available at: http://localhost:8238
bin\dds238-modbus-simulator.exe %*
