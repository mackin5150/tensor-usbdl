package main

import (
	"fmt"
	"time"

	"go.bug.st/serial"
)

type DNW struct {
	port serial.Port
}

func NewDNW() (*DNW, error) {
	mode := &serial.Mode{BaudRate: 9600, Parity: serial.NoParity, DataBits: 8, StopBits: serial.OneStopBit}
	ports := []string{"/dev/ttyACM0", "COM4"}
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
		if c := string(p); c == "\n" || c == "\r" {
			if len(bytes) == 0 {
				continue
			}
			break
		}
		bytes = append(bytes, p...)
	}

	return NewMessage(string(bytes)), nil
}

func (d *DNW) WriteCommand(c *Command) error {
	cmd := c.Bytes()
	if len(cmd) == 0 {
		return nil
	}

	wrote := 0
	for {
		n, err := d.Write(cmd[wrote:])
		if err != nil {
			return err
		}
		wrote += n
		if wrote == len(cmd) {
			break
		}
	}
	if wrote != len(cmd) {
		return fmt.Errorf("dnw: incomplete write")
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
	if err := d.port.Drain(); err != nil {
		return 0, err
	}
	return d.port.Read(p)
}

func (d *DNW) Write(p []byte) (n int, err error) {
	if d.port == nil {
		return 0, fmt.Errorf("dnw: closed")
	}
	if n, err = d.port.Write(p); err != nil {
		return n, err
	}
	if err = d.port.Drain(); err != nil {
		return n, err
	}
	return n, nil
}
