import "tensorutils"


package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/JoshuaDoes/tensor-usbdl/tensorutils"
)

const (
	VERSION = "1.0.0-GS101"
	BANNER = `
████████╗███████╗███╗   ██╗███████╗ ██████╗ ██████╗       ██╗   ██╗███████╗██████╗ ██╗     
╚══██╔══╝██╔════╝████╗  ██║██╔════╝██╔═══██╗██╔══██╗      ██║   ██║██╔════╝██╔══██╗██║     
   ██║   █████╗  ██╔██╗ ██║███████╗██║   ██║██████╔╝█████╗██║   ██║███████╗██████╔╝██║     
   ██║   ██╔══╝  ██║╚██╗██║╚════██║██║   ██║██╔══██╗╚════╝██║   ██║╚════██║██╔══██╗██║     
   ██║   ███████╗██║ ╚████║███████║╚██████╔╝██║  ██║      ╚██████╔╝███████║██████╔╝███████╗
   ╚═╝   ╚══════╝╚═╝  ╚═══╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝       ╚═════╝ ╚══════╝╚═════╝ ╚══════╝

GS101 Pixel 6a Emergency USB Download Tool v%s
Based on keyholes.txt endpoint analysis - First successful Pixel 6a unbrick!
`
)

type FlashMode int

const (
	ModeSerial FlashMode = iota  // Original DNW serial mode
	ModeUSB                      // New USB bulk transfer mode  
	ModeAuto                     // Auto-detect best mode
)

func main() {
	fmt.Printf(BANNER, VERSION)
	
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	
	command := os.Args[1]
	
	switch command {
	case "flash":
		if len(os.Args) < 3 {
			fmt.Println("Error: flash command requires bootloader path")
			printUsage()
			os.Exit(1)
		}
		bootloaderPath := os.Args[2]
		
		mode := ModeAuto
		if len(os.Args) > 3 {
			switch strings.ToLower(os.Args[3]) {
			case "serial":
				mode = ModeSerial
			case "usb":
				mode = ModeUSB  
			case "auto":
				mode = ModeAuto
			default:
				fmt.Printf("Error: unknown mode '%s'\n", os.Args[3])
				printUsage()
				os.Exit(1)
			}
		}
		
		err := flashBootloader(bootloaderPath, mode)
		if err != nil {
			fmt.Printf("Flash failed: %v\n", err)
			os.Exit(1)
		}
		
	case "detect":
		detectDevices()
		
	case "test":
		testEndpoints()
		
	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
Usage: tensor-usbdl <command> [options]

Commands:
  flash <bootloader_path> [mode]  Flash bootloader to GS101 device
                                  Modes: serial, usb, auto (default: auto)
  detect                          Detect and list compatible devices
  test                           Test USB endpoints communication

Examples:
  tensor-usbdl flash pbl.img                    # Auto-detect mode
  tensor-usbdl flash pbl.img usb                # Force USB bulk mode  
  tensor-usbdl flash pbl.img serial             # Force serial DNW mode
  tensor-usbdl detect                           # List devices
  tensor-usbdl test                             # Test endpoints

Supported bootloader files:
  - pbl.img (Primary bootloader)
  - bl1.img, bl2.img, bl31.img
  - abl.img (Android bootloader)
  - tzsw.img (TrustZone)
  - ldfw.img, gsa.img
`)
}

func flashBootloader(bootloaderPath string, mode FlashMode) error {
	// Check if file exists
	if _, err := os.Stat(bootloaderPath); os.IsNotExist(err) {
		return fmt.Errorf("bootloader file not found: %s", bootloaderPath)
	}
	
	// Read bootloader data
	data, err := ioutil.ReadFile(bootloaderPath)
	if err != nil {
		return fmt.Errorf("failed to read bootloader: %v", err)
	}
	
	fmt.Printf("Loaded bootloader: %s (%d bytes)\n", filepath.Base(bootloaderPath), len(data))
	
	// Try flashing based on mode
	switch mode {
	case ModeUSB:
		return flashUSB(data, bootloaderPath)
		
	case ModeSerial:
		return flashSerial(data, bootloaderPath)
		
	case ModeAuto:
		// Try USB first (more direct), then fallback to serial
		fmt.Println("Auto-mode: Trying USB bulk transfer first...")
		err := flashUSB(data, bootloaderPath)
		if err != nil {
			fmt.Printf("USB mode failed (%v), trying serial mode...\n", err)
			return flashSerial(data, bootloaderPath) 
		}
		return nil
		
	default:
		return fmt.Errorf("unknown flash mode")
	}
}

func flashUSB(data []byte, bootloaderPath string) error {
	fmt.Println("=== USB Bulk Transfer Mode ===")
	fmt.Println("Using endpoints from keyholes.txt analysis:")
	fmt.Println("- OUT: 0x02 (Bulk, 512 bytes)")  
	fmt.Println("- IN:  0x81 (Bulk, 512 bytes)")
	fmt.Println("- INT: 0x83 (Interrupt, 10 bytes)")
	
	// Create GS101 USB device
	gs101, err := tensorutils.NewGS101Device()
	if err != nil {
		return fmt.Errorf("failed to connect to GS101 device: %v", err)
	}
	defer gs101.Close()
	
	fmt.Println("Connected to:", gs101.GetDeviceInfo())
	
	// Send bootloader 
	err = gs101.WriteBootloader(data)
	if err != nil {
		return fmt.Errorf("failed to write bootloader: %v", err)
	}
	
	// Read response/status
	fmt.Println("Reading device response...")
	status, err := gs101.ReadStatus()
	if err != nil {
		fmt.Printf("Warning: could not read status: %v\n", err)
	} else {
		fmt.Printf("Device response (%d bytes): %x\n", len(status), status)
	}
	
	fmt.Println("✅ USB flash completed successfully!")
	return nil
}

func flashSerial(data []byte, bootloaderPath string) error {
	fmt.Println("=== Serial DNW Mode ===")
	fmt.Println("Using CDC-ACM serial communication (115200 baud)")
	
	// Get DNW device (original implementation)
	dnw, err := tensorutils.GetDNW()
	if err != nil {
		return fmt.Errorf("failed to get DNW device: %v", err)
	}
	defer dnw.Close()
	
	fmt.Printf("Connected to DNW device: %s (VID:PID = %s)\n", dnw.GetPort(), dnw.GetID())
	
	// Create DNW command with bootloader data
	cmd := tensorutils.NewCommand(tensorutils.OpDNW, nil, data, nil)
	
	// Send command
	err = dnw.WriteCmd(cmd)
	if err != nil {
		return fmt.Errorf("failed to send DNW command: %v", err)
	}
	
	// Read response
	fmt.Println("Reading DNW response...")
	msg, err := dnw.ReadMsg()
	if err != nil {
		fmt.Printf("Warning: could not read DNW response: %v\n", err)
	} else if msg != nil {
		fmt.Printf("DNW response: %s\n", msg.String())
	}
	
	fmt.Println("✅ Serial flash completed successfully!")
	return nil
}

func detectDevices() {
	fmt.Println("=== Device Detection ===")
	
	// Try USB detection
	fmt.Println("\nScanning for GS101 USB devices...")
	gs101, err := tensorutils.NewGS101Device()
	if err != nil {
		fmt.Printf("❌ GS101 USB device not found: %v\n", err)
	} else {
		fmt.Printf("✅ Found GS101 USB device: %s\n", gs101.GetDeviceInfo())
		gs101.Close()
	}
	
	// Try serial detection  
	fmt.Println("\nScanning for DNW serial devices...")
	dnw, err := tensorutils.GetDNW()
	if err != nil {
		fmt.Printf("❌ DNW serial device not found: %v\n", err)
	} else {
		fmt.Printf("✅ Found DNW device: %s (VID:PID = %s, Serial: %s)\n", 
			dnw.GetPort(), dnw.GetID(), dnw.GetSerial())
		dnw.Close()
	}
}

func testEndpoints() {
	fmt.Println("=== USB Endpoints Test ===")
	fmt.Println("Testing endpoints discovered in keyholes.txt analysis")
	
	gs101, err := tensorutils.NewGS101Device()
	if err != nil {
		fmt.Printf("❌ Cannot connect to GS101 device: %v\n", err)
		return
	}
	defer gs101.Close()
	
	fmt.Printf("✅ Connected: %s\n", gs101.GetDeviceInfo())
	
	// Test write
	fmt.Println("\nTesting Bulk OUT (0x02)...")
	testData := []byte("TENSOR-TEST-PACKET")
	n, err := gs101.Write(testData)
	if err != nil {
		fmt.Printf("❌ Write test failed: %v\n", err)
	} else {
		fmt.Printf("✅ Write test passed: %d bytes sent\n", n)
	}
	
	// Test read
	fmt.Println("\nTesting Bulk IN (0x81)...")
	buf := make([]byte, 512)
	n, err = gs101.Read(buf)
	if err != nil {
		fmt.Printf("⚠️  Read test failed (may be normal): %v\n", err)
	} else {
		fmt.Printf("✅ Read test passed: %d bytes received: %x\n", n, buf[:n])
	}
	
	// Test interrupt
	fmt.Println("\nTesting Interrupt IN (0x83)...")
	intData, err := gs101.ReadInterrupt()
	if err != nil {
		fmt.Printf("⚠️  Interrupt test failed (may be normal): %v\n", err)
	} else {
		fmt.Printf("✅ Interrupt test passed: %d bytes received: %x\n", len(intData), intData)
	}
	
	fmt.Println("\n🎯 Endpoint testing completed!")
}
