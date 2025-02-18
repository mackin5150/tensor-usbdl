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
		dnwTargetAddr: []byte{0x1b, 'D', 'N', 'W'},
		dnwCRC:        []byte{0xFF, 0xFF},
	}
}

func (d *Dnw) Open() error {
	if d.ser != nil {
		return nil
	}
	mode := &serial.Mode{BaudRate: 9600, Parity: serial.NoParity, DataBits: 8, StopBits: serial.OneStopBit}
	ser, err := serial.Open(d.port, mode)
	if err != nil {
		return err
	}
	ser.SetReadTimeout(time.Millisecond * 500)
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
	stopData := []byte{0x1b, 'D', 'N', 'W', 0x00, 0x00, 0x00, 0x00, 0x01, 0x00}
	d.ser.Write(stopData)
	d.ser.Drain()
}

func (d *Dnw) GetProgress() int {
	return d.progressPercent
}

func (d *Dnw) GetReqFromDevice() (string, string, error) {
	d.Putc([]byte{'\n'}) //Triggers a quick response

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
			return d.GetReqFromDevice()
		case string(0x0D): //CR
			return d.GetReqFromDevice()
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
		case "bl1 header fail":
			return "", "", fmt.Errorf("bl1 header fail")
		default:
			return "", "", fmt.Errorf("unknown request: %s", str)
		}
	}
	return d.GetReqFromDevice()
}

func (d *Dnw) SendOverDnw(apModel, req string, fileContent []byte) error {
	d.progressPercent = 0

	dnwSize := len(fileContent) + 8
	dnwSign := dnwSize + 2

	dnwForm := make([]byte, 0)
	dnwForm = append(dnwForm, 0x1B, 'D', 'N', 'W')
	dnwForm = append(dnwForm, byte(dnwSign), byte(dnwSign>>8), byte(dnwSign>>16), byte(dnwSign>>24))
	//dnwForm = append(dnwForm, byte(dnwSize), byte(dnwSize>>8), byte(dnwSize>>16), byte(dnwSize>>24))
	dnwForm = append(dnwForm, fileContent...)
	dnwForm = append(dnwForm, 0xFF, 0xFF)

	unitSize := 512
	switch runtime.GOOS {
	case "windows":
		unitSize = 10240
	case "darwin":
		unitSize = 384
	}

	loops := (dnwSign / unitSize)
	if dnwSign%unitSize != 0 {
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

	recvd, err := d.Getc(800)
	if err != nil {
		return fmt.Errorf("sendOverDnw: response error: %v", err)
	}

	re := regexp.MustCompile(`(?s).*eub:([acn]+k):` + apModel + `:` + req + `\n(.*)`)
	matches := re.FindSubmatch(recvd)
	if matches == nil {
		recvd2, err := d.Getc(400)
		if err != nil {
			return fmt.Errorf("sendOverDnw: wrong response: %s\ndebug error: %v", string(recvd), err)
		}
		return fmt.Errorf("sendOverDnw: wrong response: %s\ndebug: %s", string(recvd), string(recvd2))
	}

	resp := string(matches[1])
	if resp == "ack" {
		return nil
	}
	return fmt.Errorf("sendOverDnw: got %s instead of ack", resp)
}
