package main

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"go.bug.st/serial"
)

type Dnw struct {
	port            string
	ser             serial.Port
	dnwTargetAddr   []byte
	dnwCRC          []byte
	progressPercent int
}

func NewDnw(port string) *Dnw {
	return &Dnw{
		port:          port,
		dnwTargetAddr: []byte{0x1b, 0x44, 0x4E, 0x57},
		dnwCRC:        []byte{0xFF, 0xFF},
	}
}

func (d *Dnw) Open() error {
	if d.ser != nil {
		return nil
	}
	mode := &serial.Mode{BaudRate: 115200, Parity: serial.NoParity, DataBits: 8, StopBits: serial.OneStopBit}
	ser, err := serial.Open(d.port, mode)
	if err != nil {
		return err
	}
	ser.SetReadTimeout(time.Second * 3)
	d.ser = ser
	return nil
}

func (d *Dnw) Close() {
	if d.ser != nil {
		d.ser.Close()
		d.ser = nil
	}
}

func (d *Dnw) Getc(size int) ([]byte, error) {
	if d.ser == nil {
		return nil, fmt.Errorf("serial port not open")
	}
	buf := make([]byte, size)
	for {
		n, err := d.ser.Read(buf)
		if err != nil {
			return nil, err
		}
		if n > 0 {
			buf = buf[:n]
			break
		}
	}
	return buf, nil
}

func (d *Dnw) ReadAllInBuffer() ([]byte, error) {
	buf := make([]byte, 4096)
	n, err := d.ser.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (d *Dnw) Putc(data []byte) error {
	_, err := d.ser.Write(data)
	if err != nil {
		return err
	}
	d.ser.Drain()
	return nil
}

func (d *Dnw) SendStopPattern() {
	stopData := []byte{0x1b, 0x44, 0x4e, 0x57, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00}
	d.ser.Write(stopData)
	d.ser.Drain()
}

func (d *Dnw) GetProgress() int {
	return d.progressPercent
}

func (d *Dnw) GetReqFromDevice() (string, string, error) {
	str := ""
	for {
		char, err := d.Getc(1)
		if err != nil {
			return "", "", fmt.Errorf("failed to get char: %v", err)
		}
		if string(char) == "\n" || string(char) == string(0x0D) {
			if str == "" {
				continue
			}
			break
		}
		str += string(char)
	}
	if str != "" {
		arr := strings.Split(str, ":")
		switch arr[0] {
		case "C": //Ignore, possibly marks end of request?
			return "", "", nil
		case string(0x0D): //CR
			return "", "", nil
		case "eub":
			switch arr[1] {
			case "req":
				return arr[2], arr[3], nil
			default:
				return "", "", fmt.Errorf("unknown eub request: %s", str)
			}
		case "exynos_usb_booting":
			return arr[2], "", nil
		case "irom_booting_failure":
			return "", "", fmt.Errorf("irom boot failure: %s", arr[2])
		default:
			return "", "", fmt.Errorf("unknown request: %s\n(%x)", str, []byte(str))
		}
	}
	return "", "", fmt.Errorf("failed to get req from device")
}

func (d *Dnw) SendOverDnw(apModel, req string, fileContent []byte) error {
	d.progressPercent = 0

	dnwSize := len(fileContent) + 4 + 4 + 2
	dnwForm := append(d.dnwTargetAddr, byte(dnwSize), byte(dnwSize>>8), byte(dnwSize>>16), byte(dnwSize>>24))
	dnwForm = append(dnwForm, fileContent...)
	dnwForm = append(dnwForm, d.dnwCRC...)

	var unitSize int
	switch runtime.GOOS {
	case "windows":
		unitSize = 10240
	case "darwin":
		unitSize = 384
	default:
		unitSize = 512
	}

	loops := (dnwSize / unitSize)
	if dnwSize%unitSize != 0 {
		loops++
	}

	for i := 0; i < loops; i++ {
		start := i * unitSize
		end := start + unitSize
		if end > len(dnwForm) {
			end = len(dnwForm)
		}
		if err := d.Putc(dnwForm[start:end]); err != nil {
			return fmt.Errorf("Exception on loop %d/%d: %v", i+1, loops, err)
		}
		d.progressPercent = (i + 1) * 100 / loops
		fmt.Printf("sent: %d/%d bytes\n", start+(end-start), len(dnwForm))
	}

	recvd, err := d.Getc(400)
	if err != nil {
		return fmt.Errorf("sendOverDnw: response error: %v", err)
	}

	re := regexp.MustCompile(`(?s).*eub:([acn]+k):` + apModel + `:` + req + `\n(.*)`)
	matches := re.FindSubmatch(recvd)
	if matches == nil {
		return fmt.Errorf("sendOverDnw: wrong response: %s", string(recvd))
	}

	resp := string(matches[1])
	if resp == "ack" {
		return nil
	}
	return fmt.Errorf("sendOverDnw: got %s instead of ack", resp)
}
