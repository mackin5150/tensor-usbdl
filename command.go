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
	if c.CmdLen() > 0 {
		bytes.Write(c.Cmd()) //Usually 4 bytes, i.e. {ESC}DNW
		bytes.WriteAbstract(int32(4 + c.CmdLen() + c.DataLen() + c.CRCLen()))
	}
	if c.DataLen() > 0 {
		bytes.Write(c.Data())
	}
	if c.CRCLen() > 0 {
		bytes.Write(c.CRC())
	}
	return bytes.Bytes()
}

func (c *Command) Cmd() []byte {
	return []byte(c.cmd)
}

func (c *Command) CRC() []byte {
	return c.crc
}

func (c *Command) Data() []byte {
	return c.data
}

func (c *Command) Len() int {
	return len(c.Bytes())
}

func (c *Command) CmdLen() int {
	return len(c.Cmd())
}

func (c *Command) CRCLen() int {
	return len(c.CRC())
}

func (c *Command) DataLen() int {
	return len(c.Data())
}
