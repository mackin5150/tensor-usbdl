package main

import (
	"github.com/JoshuaDoes/crunchio"
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

	bytes.Write([]byte{0x1B})  //ESC
	bytes.Write([]byte(c.cmd)) //Usually 3 bytes

	length := int32(4 + int(bytes.ByteCapacity()) + len(c.data))
	if len(c.crc) > 0 {
		length += int32(len(c.crc))
	}
	bytes.WriteAbstract(length)

	bytes.Write(c.data)
	if len(c.crc) > 0 {
		bytes.Write(c.crc)
	}

	return bytes.Bytes()
}
