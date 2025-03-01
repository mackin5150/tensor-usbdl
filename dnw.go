package main

import (
	"fmt"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

type DNW struct {
	mutex sync.Mutex
	port  *enumerator.PortDetails
	sock  serial.Port
}

func NewDNW() (*DNW, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	if len(ports) == 0 {
		return nil, fmt.Errorf("dnw: no ports found")
	}

	var port *enumerator.PortDetails
	for i := 0; i < len(ports); i++ {
		test := ports[i]
		if test.VID == "18D1" && test.PID == "4F00" { //Google Pixel 6 series
			port = test
			break
		}
	}
	if port == nil {
		return nil, fmt.Errorf("dnw: no allowed port found in list")
	}

	mode := &serial.Mode{BaudRate: 115200, Parity: serial.NoParity, DataBits: 8, StopBits: serial.OneStopBit}
	sock, err := serial.Open(port.Name, mode)
	if err != nil {
		return nil, err
	}
	sock.SetReadTimeout(time.Millisecond * 500)

	return &DNW{port: port, sock: sock}, nil
}

func (d *DNW) GetPort() string {
	if d.port == nil {
		return ""
	}
	return d.port.Name
}

func (d *DNW) GetSerial() string {
	if d.port == nil {
		return ""
	}
	return d.port.SerialNumber
}

func (d *DNW) GetID() string {
	vid := d.GetVID()
	pid := d.GetPID()
	if vid == "" || pid == "" {
		return ""
	}
	return vid + ":" + pid
}

func (d *DNW) GetVID() string {
	if d.port == nil {
		return ""
	}
	return d.port.VID
}

func (d *DNW) GetPID() string {
	if d.port == nil {
		return ""
	}
	return d.port.PID
}

func (d *DNW) GetUSB() bool {
	if d.port == nil {
		return false
	}
	return d.port.IsUSB
}

func (d *DNW) ReadMsg() (*Message, error) {
	if d.sock == nil {
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
			break
		}
		if NewMessage(string(p[0])).IsControlBit() {
			if len(bytes) == 0 {
				continue
			}
			break
		}
		bytes = append(bytes, p...)
	}

	if len(bytes) == 0 {
		return nil, nil
	}

	return NewMessage(string(bytes)), nil
}

func (d *DNW) WaitForMsg() (*Message, error) {
	for {
		msg, err := d.ReadMsg()
		if err != nil {
			return nil, err
		}
		if msg != nil {
			return msg, nil
		}
	}
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
	if err := d.sock.ResetInputBuffer(); err != nil {
		return nil
	}
	if err := d.sock.ResetOutputBuffer(); err != nil {
		return nil
	}
	return nil
}

func (d *DNW) Close() error {
	if d.sock != nil {
		if err := d.sock.Close(); err != nil {
			return err
		}
		d.sock = nil
		return nil
	}
	return fmt.Errorf("dnw: already closed")
}

func (d *DNW) Read(p []byte) (n int, err error) {
	if d.sock == nil {
		return 0, fmt.Errorf("dnw: closed")
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if err := d.sock.Drain(); err != nil {
		return 0, err
	}
	return d.sock.Read(p)
}

func (d *DNW) Write(p []byte) (n int, err error) {
	if d.sock == nil {
		return 0, fmt.Errorf("dnw: closed")
	}
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if n, err = d.sock.Write(p); err != nil {
		return n, err
	}
	if err = d.sock.Drain(); err != nil {
		return n, err
	}
	return n, nil
}
