package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"modbus-energy-meter-sim/pkg/meter"
	"modbus-energy-meter-sim/pkg/modbus"
	"modbus-energy-meter-sim/pkg/web"
)

func main() {
	portFlag := flag.String("port", "", "Optional initial COM port (e.g. COM3, COM4)")
	baudFlag := flag.Int("baud", 9600, "Serial Baud Rate (9600, 4800, 2400, 1200)")
	serverFlag := flag.Int("server", 1, "Modbus Server / Device Address (1-247)")
	slaveFlag := flag.Int("slave", 0, "Legacy alias for -server")
	httpAddr := flag.String("http", ":8238", "HTTP Web Dashboard & API address")
	tcpAddr := flag.String("tcp", ":8502", "Modbus TCP listen address (use :502 or :8502)")
	listPortsFlag := flag.Bool("list-ports", false, "List available serial COM ports and exit")
	flag.Parse()

	// Handle server address resolution
	serverAddr := uint8(*serverFlag)
	if *slaveFlag > 0 {
		serverAddr = uint8(*slaveFlag)
	}

	// List COM ports if requested
	ports, _ := modbus.ListAvailablePorts()
	if *listPortsFlag {
		fmt.Println("Available Serial COM Ports:")
		if len(ports) == 0 {
			fmt.Println("  (No serial COM ports detected)")
		} else {
			for _, p := range ports {
				fmt.Printf("  - %s\n", p)
			}
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("\n[System] Shutting down simulation gracefully...")
		cancel()
	}()

	fmt.Println("==================================================================")
	fmt.Println("    Hiking DDS238-2 ZN/S Modbus RTU Energy Meter Simulator        ")
	fmt.Println("==================================================================")
	fmt.Printf(" Modbus Server Address : %d\n", serverAddr)
	fmt.Printf(" Web UI & Reader      : http://localhost%s\n", *httpAddr)
	fmt.Printf(" Modbus TCP Bridge    : localhost%s\n", *tcpAddr)
	fmt.Println(" Detected COM Ports   :")
	if len(ports) == 0 {
		fmt.Println("   (No COM ports found yet — plug in USB UART and refresh in UI)")
	} else {
		for _, p := range ports {
			fmt.Printf("   -> %s\n", p)
		}
	}
	fmt.Println("==================================================================")

	// 1. Initialize Multi-Meter Device Manager
	dm := meter.NewDeviceManager(serverAddr)

	// 2. Initialize Real-Time Simulation Engine (250ms update cycle for all active meters)
	simEngine := meter.NewSimulationEngine(dm)
	simEngine.Start(ctx, 250*time.Millisecond)

	// 3. Initialize Modbus RTU Server (supports dynamic connect/disconnect via Web UI)
	rtuServer := modbus.NewRTUServer(dm)
	if *portFlag != "" {
		if err := rtuServer.Connect(*portFlag, *baudFlag); err != nil {
			log.Printf("[RTU] Could not auto-connect to %s: %v", *portFlag, err)
		}
	} else {
		log.Printf("[RTU] Note: You can select and connect your COM port directly from the Web UI at http://localhost%s", *httpAddr)
	}

	// 4. Initialize Modbus TCP Server
	tcpServer := modbus.NewTCPServer(*tcpAddr, dm, rtuServer)
	go func() {
		if err := tcpServer.Start(ctx); err != nil {
			log.Printf("[TCP] Modbus TCP server error: %v", err)
		}
	}()

	// 5. Initialize Web Dashboard & Reader Server
	webServer := web.NewServer(*httpAddr, dm, rtuServer, tcpServer)
	go func() {
		if err := webServer.Start(ctx); err != nil {
			log.Printf("[Web] Web server error: %v", err)
		}
	}()

	<-ctx.Done()
	_ = rtuServer.Disconnect()
	time.Sleep(300 * time.Millisecond)
	log.Println("[System] Simulation terminated.")
}
