package main

import (
	"strings"
)

// exynos_usb_booting::09845001cddf16d00bd4\n
// eub:req:09845001:DPM\n
// C\n
type Message struct {
	typ string //C, eub, exynos_usb_booting, bl1 header fail, a literal carriage return
	cmd string //req, irom_booting_failure
	dev string //09845001cddf16d00bd4, 09845001
	arg string //DPM, EPBL, bl1, ABL, ABLB, ...
}

func NewMessage(line string) *Message {
	if line == "" {
		return nil
	}
	split := strings.Split(line, ":")

	m := new(Message)
	m.typ = split[0]
	if len(split) > 1 {
		m.cmd = split[1]
		if len(split) > 2 {
			m.dev = split[2]
			if len(split) > 3 {
				m.arg = split[3]
			}
		}
	}

	if len(m.typ) > 12 && m.typ[len(m.typ)-12:] == " header fail" {
		m.arg = m.typ[:len(m.typ)-12]
		m.typ = "error"
		m.cmd = "header fail"
	}

	return m
}

func (m *Message) IsControlBit() bool {
	switch m.Type() {
	case "C", "\x1B", "\x00", "\x06", "\x0F", "\x2B", "\x15", "ACK", "NAK":
		return true
	}
	return false
}

func (m *Message) String() string {
	str := m.typ
	if m.cmd != "" && m.arg != "" {
		str += ":" + m.cmd + ":" + m.dev + ":" + m.arg
	} else if m.dev != "" {
		str += ":" + m.cmd + ":" + m.dev
	} else if m.cmd != "" {
		str += ":" + m.cmd
	}
	return str
	//return fmt.Sprintf("%s (%x)", str, []byte(str))
}

func (m *Message) Type() string {
	return m.typ
}

func (m *Message) Command() string {
	return m.cmd
}

func (m *Message) Device() string {
	return m.dev
}

func (m *Message) Argument() string {
	return m.arg
}
