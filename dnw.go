package main

import (
	"fmt"
	"sync"
	"time"

	"go.bug.st/serial"
)

type DNW struct {
	mutex sync.Mutex
	port  serial.Port
}

func NewDNW() (*DNW, error) {
	mode := &serial.Mode{BaudRate: 9600, Parity: serial.NoParity, DataBits: 8, StopBits: serial.OneStopBit}
	ports := []string{
		"/dev/ttyACM0", "COM4", //Google Tensor
		"COM3", //Samsung Exynos
	}
	for i := 0; i < len(ports); i++ {
		ser, err := serial.Open(ports[i], mode)
		if err != nil {
			continue
		}
		ser.SetReadTimeout(time.Millisecond * 500)
		return &DNW{port: ser}, nil
	}
	return nil, fmt.Errorf("dnw: no ports found")
}

func (d *DNW) ReadMsg() (*Message, error) {
	if d.port == nil {
		return nil, fmt.Errorf("dnw: closed")
	}

	//d.Write([]byte("\n")) //Triggers a faster response for the next read

	bytes := make([]byte, 0)
	for {
		p := make([]byte, 1)
		n, err := d.Read(p)
		if err != nil {
			return nil, err
		}
		if n != 1 {
			continue
		}
		if p[0] == '\n' || p[0] == '\r' || p[0] == '\x00' {
			if len(bytes) == 0 {
				continue
			}
			break
		}
		bytes = append(bytes, p...)
	}

	return NewMessage(string(bytes)), nil
}

func (d *DNW) WriteCmd(c *Command) error {
	cmd := c.Bytes()
	if len(cmd) == 0 {
		return nil
	}

	toWrite := 512
	wrote := 0
	for {
		if wrote+toWrite >= len(cmd) {
			toWrite -= (wrote + toWrite) - len(cmd)
		}
		n, err := d.Write(cmd[wrote : wrote+toWrite])
		if err != nil {
			return fmt.Errorf("dnw: incomplete write (%d/%d bytes): %v", wrote, len(cmd), err)
		}
		wrote += n
		if wrote >= len(cmd) {
			break
		}
	}
	if wrote != len(cmd) {
		return fmt.Errorf("dnw: incomplete write (%d/%d bytes)", wrote, len(cmd))
	}

	return nil
}

func (d *DNW) Reset() error {
	if err := d.port.ResetInputBuffer(); err != nil {
		return nil
	}
	if err := d.port.ResetOutputBuffer(); err != nil {
		return nil
	}
	return nil
}

func (d *DNW) Close() error {
	if d.port != nil {
		if err := d.port.Close(); err != nil {
			return err
		}
		d.port = nil
		return nil
	}
	return fmt.Errorf("dnw: already closed")
}

func (d *DNW) Read(p []byte) (n int, err error) {
	if d.port == nil {
		return 0, fmt.Errorf("dnw: closed")
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if err := d.port.Drain(); err != nil {
		return 0, err
	}
	return d.port.Read(p)
}

func (d *DNW) Write(p []byte) (n int, err error) {
	if d.port == nil {
		return 0, fmt.Errorf("dnw: closed")
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if n, err = d.port.Write(p); err != nil {
		return n, err
	}
	if err = d.port.Drain(); err != nil {
		return n, err
	}
	return n, nil
}
