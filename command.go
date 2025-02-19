package main

import (
	"github.com/JoshuaDoes/crunchio"
)

var (
	cmdStop = NewCommand("\x1BDNW", nil, []byte("\x01\x00"))
)

type Command struct {
	cmd       string
	crc, data []byte
}

func NewCommand(cmd string, data, crc []byte) *Command {
	return &Command{
		cmd:  cmd,
		crc:  crc,
		data: data,
	}
}

func (c *Command) Bytes() []byte {
	bytes := crunchio.NewBuffer(c.cmd)

	if c.cmd != "" {
		bytes.Write([]byte(c.cmd)) //Usually 4 bytes, i.e. {ESC}DNW

		length := int32(4 + int(bytes.ByteCapacity()) + len(c.data))
		if len(c.crc) > 0 {
			length += int32(len(c.crc))
		}
		bytes.WriteAbstract(length)
	}

	if len(c.data) > 0 {
		bytes.Write(c.data)
	}

	if c.cmd != "" && len(c.crc) > 0 {
		bytes.Write(c.crc)
	}

	return bytes.Bytes()
}
