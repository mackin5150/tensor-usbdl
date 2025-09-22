package tensorutils

import "strings"

type Message struct {
	bytes []byte

	cmd string //C, eub, exynos_usb_booting, bl1 header fail, a literal carriage return
	sub string //req, irom_booting_failure
	dev string //09845001cddf16d00bd4, 09845001
	arg string //DPM, EPBL, bl1, ABL, ABLB, ...
}

func NewMessage(bytes []byte) *Message {
	if len(bytes) == 0 {
		return nil
	}

	msg := new(Message)
	msg.bytes = bytes

	split := strings.Split(string(bytes), ":")
	msg.cmd = split[0]
	if len(split) > 1 {
		msg.sub = split[1]
		if len(split) > 2 {
			msg.dev = split[2]
			if len(split) > 3 {
				msg.arg = split[3]
			}
		}
	}

	if len(msg.cmd) > 12 && msg.cmd[len(msg.cmd)-12:] == " header fail" {
		msg.arg = msg.cmd[:len(msg.cmd)-12]
		msg.cmd = "error"
		msg.sub = "header fail"
	}

	return msg
}

func (msg *Message) Command() string {
	return msg.cmd
}
func (msg *Message) SubCommand() string {
	return msg.sub
}
func (msg *Message) Device() string {
	return msg.dev
}
func (msg *Message) Argument() string {
	return msg.arg
}

func (msg *Message) Bytes() []byte {
	return msg.bytes
}

func (msg *Message) String() string {
	return string(msg.bytes)
}
