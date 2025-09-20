# Tensor-USBDL GS101 Integration

**First Pixel 6a Unbrick Tool with USB Endpoint Analysis**

This enhanced version of tensor-usbdl incorporates USB endpoint analysis from `keyholes.txt` to enable direct bulk transfer communication with GS101 (Pixel 6/6a/6 Pro) devices in emergency download mode.

## Features

### 🔥 **Dual Communication Methods**
- **USB Bulk Transfer** (New): Direct endpoint communication using analyzed endpoints
- **Serial DNW** (Original): CDC-ACM serial communication fallback

### 🎯 **Endpoint Configuration** 
Based on `keyholes.txt` USB capture analysis:
```
Endpoint 0x02 (OUT) - Bulk transfer, 512 bytes
Endpoint 0x81 (IN)  - Bulk transfer, 512 bytes  
Endpoint 0x83 (IN)  - Interrupt transfer, 10 bytes
Interface 1 (CDC Data) - Configuration 1, Alt Setting 0
```

### 📱 **Device Support**
- **Primary**: Google Pixel 6a (bluejay) - GS101 chip
- **Compatible**: Pixel 6 (oriole), Pixel 6 Pro (raven) - GS101 chip
- **VID:PID**: 18D1:4F00 (Google emergency download mode)

## Installation

### Prerequisites
- Windows 10/11
- Go 1.19+ 
- USB drivers for Google devices
- GS101 bootloader images

### Build
```cmd
cd tensor-usbdl
build_gs101.bat
```

This creates:
- `tensor-usbdl-gs101.exe` (Release build)
- `tensor-usbdl-gs101-debug.exe` (Debug build)

## Usage

### Device Detection
```cmd
tensor-usbdl-gs101.exe detect
```
Scans for both USB and serial GS101 devices.

### Endpoint Testing
```cmd
tensor-usbdl-gs101.exe test
```
Tests USB endpoints communication (based on keyholes.txt analysis).

### Bootloader Flashing

**Auto Mode** (Recommended):
```cmd
tensor-usbdl-gs101.exe flash pbl.img
```
Tries USB first, falls back to serial if needed.

**Force USB Mode**:
```cmd
tensor-usbdl-gs101.exe flash pbl.img usb
```
Uses direct USB bulk transfer only.

**Force Serial Mode**:
```cmd
tensor-usbdl-gs101.exe flash pbl.img serial
```
Uses original DNW serial communication only.

## Bootloader Files

### GS101 Bootloader Components
Located in `../gs101/` directory:
```
pbl.img     - Primary bootloader (first stage)
bl1.img     - Bootloader stage 1  
bl2.img     - Bootloader stage 2
bl31.img    - ARM Trusted Firmware (EL3)
abl.img     - Android bootloader
tzsw.img    - TrustZone secure world
ldfw.img    - Low-level firmware
gsa.img     - Google Security Assistant
```

### Flash Order (Emergency Recovery)
1. **pbl.img** (Primary bootloader - critical first)
2. **bl1.img, bl2.img** (Secondary bootloaders)
3. **bl31.img** (ARM Trusted Firmware) 
4. **tzsw.img** (TrustZone)
5. **abl.img** (Android bootloader)

## Emergency Recovery Process

### Pixel 6a Complete Unbrick Procedure

1. **Enter Emergency Download Mode**:
   - Power off device completely
   - Hold Volume Down + Power for 10+ seconds
   - Connect USB cable while holding buttons
   - Device should appear as 18D1:4F00

2. **Detect Device**:
   ```cmd
   tensor-usbdl-gs101.exe detect
   ```

3. **Test Communication**:
   ```cmd
   tensor-usbdl-gs101.exe test
   ```

4. **Flash Primary Bootloader**:
   ```cmd
   tensor-usbdl-gs101.exe flash ../gs101/pbl.img usb
   ```

5. **Flash Remaining Bootloaders** (in order):
   ```cmd
   tensor-usbdl-gs101.exe flash ../gs101/bl1.img usb
   tensor-usbdl-gs101.exe flash ../gs101/bl2.img usb
   tensor-usbdl-gs101.exe flash ../gs101/bl31.img usb
   tensor-usbdl-gs101.exe flash ../gs101/tzsw.img usb
   tensor-usbdl-gs101.exe flash ../gs101/abl.img usb
   ```

6. **Reboot Device**:
   - Device should now boot to fastboot mode
   - Use `fastboot devices` to verify
   - Flash factory images if needed

## Technical Details

### USB Endpoints Analysis
The endpoint configuration was reverse-engineered from `keyholes.txt` USB traffic capture:

```
Configuration Descriptor:
  bNumInterfaces: 2
  Interface 0: CDC Communication Class
  Interface 1: CDC Data Class
    Endpoint 0x02: OUT, Bulk, 512 bytes  ← Main data output
    Endpoint 0x81: IN,  Bulk, 512 bytes  ← Main data input  
    Endpoint 0x83: IN,  Interrupt, 10 bytes ← Status/control
```

### Communication Protocol
1. **USB Enumeration**: Device presents as CDC composite device
2. **Interface Claim**: Claim Interface 1 (data interface)
3. **Bulk Transfer**: Send bootloader data in 512-byte chunks via EP 0x02
4. **Status Read**: Monitor EP 0x81 for responses/acknowledgments
5. **Interrupt Monitor**: EP 0x83 for device status (optional)

### Error Handling
- **USB Timeout**: 5-second timeout for transfers
- **Retry Logic**: Auto-fallback from USB to serial mode
- **Chunk Verification**: Per-chunk error checking
- **Device State**: Connection monitoring and recovery

## Troubleshooting

### Common Issues

**Device Not Found (18D1:4F00)**:
- Ensure device is in emergency download mode
- Try different USB cable/port
- Check Windows Device Manager for driver issues

**USB Transfer Failed**:
- Switch to serial mode: `tensor-usbdl-gs101.exe flash pbl.img serial`
- Check for USB 3.0 compatibility issues
- Try USB 2.0 port

**Permission Denied**:
- Run as Administrator
- Check antivirus software blocking USB access
- Verify USB drivers are installed

### Debug Mode
Use debug build for verbose output:
```cmd
tensor-usbdl-gs101-debug.exe test
```

## Development

### Code Structure
```
main.go         - CLI interface and main logic
gs101_usb.go    - USB bulk transfer implementation  
devices.go      - Device discovery (original)
dnw.go          - Serial DNW communication (original)
command.go      - Command protocol (original)
message.go      - Message handling (original)
```

### Dependencies
- `github.com/google/gousb` - USB device communication
- `go.bug.st/serial` - Serial communication
- `github.com/JoshuaDoes/crunchio` - Data buffering

## Credits

- **Keyholes Analysis**: USB endpoint reverse engineering
- **Original tensor-usbdl**: JoshuaDoes
- **GS101 Integration**: Enhanced for Pixel 6a unbrick
- **Endpoint Discovery**: Based on real USB traffic capture

---

**⚠️ WARNING**: This tool directly manipulates bootloader components. Use only for recovery of bricked devices. Flashing incorrect bootloaders can permanently damage your device.

**🎯 GOAL**: First successful Pixel 6a unbrick using direct USB endpoint communication!
