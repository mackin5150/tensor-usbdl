package tensorutils

import (
	"fmt"
	"time"

	"github.com/google/gousb"
)

const (
	GS101_VID = 0x18d1
	GS101_PID = 0x4f00

	GS101_EP_OUT = 0x02
	GS101_EP_IN  = 0x81
	GS101_EP_INT = 0x83

	GS101_BULK_PKT_SIZE = 512
	GS101_INT_PKT_SIZE  = 10

	GS101_CONFIG = 1
	GS101_IFACE  = 1
	GS101_ALT    = 0

	GS101_TIMEOUT = 5 * time.Second
)

type GS101Device struct {
	ctx    *gousb.Context
	dev    *gousb.Device
	cfg    *gousb.Config
	intf   *gousb.Interface
	outEp  *gousb.OutEndpoint
	inEp   *gousb.InEndpoint
	intEp  *gousb.InEndpoint
	closed bool
	info   string
}

// NewGS101Device initializes the GS101 USB device connection.
func NewGS101Device() (*GS101Device, error) {
	ctx := gousb.NewContext()

	// Open devices matching VID and PID, close others
	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == gousb.ID(GS101_VID) && desc.Product == gousb.ID(GS101_PID)
	})
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("error opening devices: %w", err)
	}
	if len(devs) == 0 {
		ctx.Close()
		return nil, fmt.Errorf("no GS101 device found")
	}
	dev := devs[0]
	for _, d := range devs[1:] {
		d.Close()
	}

	// Ensure device is closed on error to avoid leak
	defer func() {
		if dev != nil {
			dev.Close()
		}
	}()

	// Explicitly set configuration (recommended)
	if err := dev.SetConfig(GS101_CONFIG); err != nil {
		ctx.Close()
		return nil, fmt.Errorf("failed to set configuration %d: %w", GS101_CONFIG, err)
	}

	// Claim interface and alt setting
	if err := dev.ClaimInterface(GS101_IFACE); err != nil {
		ctx.Close()
		return nil, fmt.Errorf("failed to claim interface %d: %w", GS101_IFACE, err)
	}

	intf, err := dev.Config(GS101_CONFIG)
	if err != nil {
		dev.ReleaseInterface(GS101_IFACE)
		ctx.Close()
		return nil, fmt.Errorf("failed to get config %d: %w", GS101_CONFIG, err)
	}

	// Get the interface instance
	iface, err := intf.Interface(GS101_IFACE, GS101_ALT)
	if err != nil {
		intf.Close()
		dev.ReleaseInterface(GS101_IFACE)
		ctx.Close()
		return nil, fmt.Errorf("failed to get interface %d alt %d: %w", GS101_IFACE, GS101_ALT, err)
	}

	// Acquire endpoints by endpoint number (low nibble)
	outEp, err := iface.OutEndpoint(int(GS101_EP_OUT & 0x0f))
	if err != nil {
		iface.Close()
		intf.Close()
		dev.ReleaseInterface(GS101_IFACE)
		ctx.Close()
		return nil, fmt.Errorf("failed to open OUT endpoint 0x%02x: %w", GS101_EP_OUT, err)
	}

	inEp, err := iface.InEndpoint(int(GS101_EP_IN & 0x0f))
	if err != nil {
		iface.Close()
		intf.Close()
		dev.ReleaseInterface(GS101_IFACE)
		ctx.Close()
		return nil, fmt.Errorf("failed to open IN endpoint 0x%02x: %w", GS101_EP_IN, err)
	}

	intEp, err := iface.InEndpoint(int(GS101_EP_INT & 0x0f))
	if err != nil {
		iface.Close()
		intf.Close()
		dev.ReleaseInterface(GS101_IFACE)
		ctx.Close()
		return nil, fmt.Errorf("failed to open interrupt IN endpoint 0x%02x: %w", GS101_EP_INT, err)
	}

	// Now device is successfully opened; cancel defer close after here
	devCopy := dev
	dev = nil

	gs101 := &GS101Device{
		ctx:    ctx,
		dev:    devCopy,
		cfg:    intf.Config(),
		intf:   iface,
		outEp:  outEp,
		inEp:   inEp,
		intEp:  intEp,
		closed: false,
		info:   fmt.Sprintf("GS101 Device - VID:PID=%04X:%04X Serial:%s", GS101_VID, GS101_PID, devCopy.SerialNumber()),
	}

	fmt.Println("✅ GS101 device connected:", gs101.info)
	return gs101, nil
}

// Close releases all USB resources safely.
func (gs101 *GS101Device) Close() error {
	if gs101.closed {
		return nil
	}
	gs101.closed = true
	if gs101.intf != nil {
		gs101.intf.Close()
	}
	if gs101.cfg != nil {
		gs101.cfg.Close()
	}
	if gs101.dev != nil {
		gs101.dev.ReleaseInterface(GS101_IFACE)
		gs101.dev.Close()
	}
	if gs101.ctx != nil {
		gs101.ctx.Close()
	}
	fmt.Println("🔐 GS101 device closed successfully")
	return nil
}

// Write sends data to bulk OUT endpoint with timeout
func (gs101 *GS101Device) Write(data []byte) (int, error) {
	if gs101.closed {
		return 0, fmt.Errorf("device closed")
	}
	n, err := gs101.outEp.Write(data)
	if err != nil {
		return n, fmt.Errorf("write to OUT endpoint failed: %w", err)
	}
	return n, nil
}

// Read reads data from the bulk IN endpoint with timeout
func (gs101 *GS101Device) Read(buf []byte) (int, error) {
	if gs101.closed {
		return 0, fmt.Errorf("device closed")
	}
	n, err := gs101.inEp.Read(buf)
	if err != nil {
		return n, fmt.Errorf("read from IN endpoint failed: %w", err)
	}
	return n, nil
}

// ReadInterrupt reads from interrupt IN endpoint
func (gs101 *GS101Device) ReadInterrupt() ([]byte, error) {
	if gs101.closed {
		return nil, fmt.Errorf("device closed")
	}
	buf := make([]byte, GS101_INT_PKT_SIZE)
	n, err := gs101.intEp.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read interrupt failed: %w", err)
	}
	return buf[:n], nil
}

// WriteBootloader sends bootloader to device in chunks respecting packet size
func (gs101 *GS101Device) WriteBootloader(data []byte) error {
	if gs101.closed {
		return fmt.Errorf("device closed")
	}
	offset := 0
	for offset < len(data) {
		chunkSize := GS101_BULK_PKT_SIZE
		if len(data)-offset < chunkSize {
			chunkSize = len(data) - offset
		}
		n, err := gs101.Write(data[offset : offset+chunkSize])
		if err != nil {
			return fmt.Errorf("bootloader write failed at offset %d: %w", offset, err)
		}
		if n != chunkSize {
			return fmt.Errorf("short write at offset %d: wrote %d of %d bytes", offset, n, chunkSize)
		}
		offset += n
		time.Sleep(50 * time.Millisecond) // optional delay between chunks
	}
	return nil
}

// GetDeviceInfo returns string describing connected device
func (gs101 *GS101Device) GetDeviceInfo() string {
	if gs101.closed {
		return "device closed"
	}
	return gs101.info
}
