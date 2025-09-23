package tensorutils

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/gousb"
)

// ErrStall is a custom error returned when a severe endpoint stall is detected
// that cannot be cleared by a simple control transfer.
var ErrStall = fmt.Errorf("severe stall")

const (
	GS101_VID = 0x18d1
	GS101_PID = 0x4f00

	GS101_EP_OUT = 0x02
	GS101_EP_IN = 0x81
	GS101_EP_INT = 0x83

	GS101_BULK_PKT_SIZE = 512
	GS101_INT_PKT_SIZE = 10

	GS101_CONFIG = 1
	GS101_BULK_IFACE = 1
	GS101_INT_IFACE = 0
	GS101_ALT = 0

	GS101_TIMEOUT = 5 * time.Second

	// USB Control Request values for ClearFeature
	LIBUSB_REQUEST_TYPE_STANDARD = 0x00
	LIBUSB_RECIPIENT_ENDPOINT    = 0x02
	LIBUSB_REQUEST_CLEAR_FEATURE = 0x01
	LIBUSB_ENDPOINT_HALT         = 0x00
)

type GS101Device struct {
	ctx      *gousb.Context
	dev      *gousb.Device
	cfg      *gousb.Config
	bulkIntf *gousb.Interface
	intIntf  *gousb.Interface
	outEp    *gousb.OutEndpoint
	inEp     *gousb.InEndpoint
	intEp    *gousb.InEndpoint
	closed   bool
	info     string
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

	// Open the configuration
	cfg, err := dev.Config(GS101_CONFIG)
	if err != nil {
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to open configuration %d: %w", GS101_CONFIG, err)
	}

	// Open bulk data interface
	bulkIntf, err := cfg.Interface(GS101_BULK_IFACE, GS101_ALT)
	if err != nil {
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to open bulk interface %d alt %d: %w", GS101_BULK_IFACE, GS101_ALT, err)
	}

	// Open interrupt data interface
	intIntf, err := cfg.Interface(GS101_INT_IFACE, GS101_ALT)
	if err != nil {
		bulkIntf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to open interrupt interface %d alt %d: %w", GS101_INT_IFACE, GS101_ALT, err)
	}

	// Acquire bulk endpoints from the bulk interface
	outEp, err := bulkIntf.OutEndpoint(int(GS101_EP_OUT & 0x0f))
	if err != nil {
		intIntf.Close()
		bulkIntf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to open OUT endpoint 0x%02x: %w", GS101_EP_OUT, err)
	}

	inEp, err := bulkIntf.InEndpoint(int(GS101_EP_IN & 0x0f))
	if err != nil {
		intIntf.Close()
		bulkIntf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to open IN endpoint 0x%02x: %w", GS101_EP_IN, err)
	}

	// Acquire interrupt endpoint from the interrupt interface
	intEp, err := intIntf.InEndpoint(int(GS101_EP_INT & 0x0f))
	if err != nil {
		intIntf.Close()
		bulkIntf.Close()
		cfg.Close()
		dev.Close()
		ctx.Close()
		return nil, fmt.Errorf("failed to open interrupt IN endpoint 0x%02x: %w", GS101_EP_INT, err)
	}

	serial, err := dev.SerialNumber()
	if err != nil {
		serial = "unknown"
	}

	gs101 := &GS101Device{
		ctx:      ctx,
		dev:      dev,
		cfg:      cfg,
		bulkIntf: bulkIntf,
		intIntf:  intIntf,
		outEp:    outEp,
		inEp:     inEp,
		intEp:    intEp,
		closed:   false,
		info:     fmt.Sprintf("GS101 Device - VID:PID=%04X:%04X Serial:%s", GS101_VID, GS101_PID, serial),
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
	if gs101.intIntf != nil {
		gs101.intIntf.Close()
	}
	if gs101.bulkIntf != nil {
		gs101.bulkIntf.Close()
	}
	if gs101.cfg != nil {
		gs101.cfg.Close()
	}
	if gs101.dev != nil {
		gs101.dev.Close()
	}
	if gs101.ctx != nil {
		gs101.ctx.Close()
	}
	fmt.Println("🔐 GS101 device closed successfully")
	return nil
}

// clearStall sends a control request to clear the stall condition on an endpoint.
func (gs101 *GS101Device) clearStall(endpointAddress uint8) error {
	if gs101.closed {
		return fmt.Errorf("device closed")
	}

	// This is a standard USB control transfer to clear the HALT feature on an endpoint.
	// bmRequestType: Standard (0x00), Recipient Endpoint (0x02) -> 0x02
	// bRequest: ClearFeature (0x01)
	// wValue: Endpoint Halt (0x00)
	// wIndex: Endpoint Address (e.g., 0x02 for OUT, 0x81 for IN)
	_, err := gs101.dev.Control(
		(LIBUSB_REQUEST_TYPE_STANDARD | LIBUSB_RECIPIENT_ENDPOINT),
		LIBUSB_REQUEST_CLEAR_FEATURE,
		LIBUSB_ENDPOINT_HALT,
		uint16(endpointAddress),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to clear stall on endpoint 0x%02x: %w", endpointAddress, err)
	}
	return nil
}

// Write sends data to bulk OUT endpoint, with a retry after stall.
func (gs101 *GS101Device) Write(data []byte) (int, error) {
	if gs101.closed {
		return 0, fmt.Errorf("device closed")
	}
	n, err := gs101.outEp.Write(data)
	if err != nil {
		if strings.Contains(err.Error(), "endpoint stalled") {
			fmt.Printf("⚠️ Endpoint 0x%02x stalled. Attempting to clear stall...\n", gs101.outEp.Desc.Address)
			if clearErr := gs101.clearStall(uint8(gs101.outEp.Desc.Address)); clearErr != nil {
				return 0, ErrStall // Return our custom error
			}
			fmt.Println("✅ Stall cleared. Retrying write...")
			// Retry the write after clearing the stall
			n, err = gs101.outEp.Write(data)
			if err != nil {
				return n, fmt.Errorf("write to OUT endpoint failed after stall clear: %w", err)
			}
		} else {
			return n, fmt.Errorf("write to OUT endpoint failed: %w", err)
		}
	}
	return n, nil
}

// Read reads data from the bulk IN endpoint, with a retry after stall.
func (gs101 *GS101Device) Read(buf []byte) (int, error) {
	if gs101.closed {
		return 0, fmt.Errorf("device closed")
	}
	n, err := gs101.inEp.Read(buf)
	if err != nil {
		if strings.Contains(err.Error(), "endpoint stalled") {
			fmt.Printf("⚠️ Endpoint 0x%02x stalled. Attempting to clear stall...\n", gs101.inEp.Desc.Address)
			if clearErr := gs101.clearStall(uint8(gs101.inEp.Desc.Address)); clearErr != nil {
				return 0, ErrStall // Return our custom error
			}
			fmt.Println("✅ Stall cleared. Retrying read...")
			// Retry the read after clearing the stall
			n, err = gs101.inEp.Read(buf)
			if err != nil {
				return n, fmt.Errorf("read from IN endpoint failed after stall clear: %w", err)
			}
		} else {
			return n, fmt.Errorf("read from IN endpoint failed: %w", err)
		}
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
