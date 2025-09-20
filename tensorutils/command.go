package tensorutils

import "github.com/JoshuaDoes/crunchio"

var (
	OpDNW   = []byte("\x1BDNW")
	CmdDNW  = NewCommand(OpDNW, nil, nil, nil)
	CmdStop = NewCommand(OpDNW, make([]byte, 4), nil, []byte("\x01\x00"))
)

type Command struct {
	cmd, arg, data, crc []byte
}

func NewCommand(cmd, arg, data, crc []byte) *Command {
	return &Command{
		cmd:  cmd,
		arg:  arg,
		data: data,
		crc:  crc,
	}
}

func (c *Command) Bytes() []byte {
	bytes := crunchio.NewBuffer(string(c.cmd))
	if c.CmdLen() > 0 {
		bytes.Write(c.Cmd()) //Usually 4 bytes, i.e. {ESC}DNW
		if c.ArgLen() > 0 {
			bytes.Write(c.Arg())
		} else {
			bytes.WriteAbstract(int32(4 + c.CmdLen() + c.CRCLen() + c.DataLen())) //Assume the argument is the command's byte length, including this
		}
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

func (c *Command) Arg() []byte {
	return c.arg
}

func (c *Command) Data() []byte {
	return c.data
}

func (c *Command) CRC() []byte {
	return c.crc
}

func (c *Command) CmdLen() int {
	return len(c.Cmd())
}

func (c *Command) ArgLen() int {
	return len(c.Arg())
}

func (c *Command) DataLen() int {
	return len(c.Data())
}

func (c *Command) CRCLen() int {
	return len(c.CRC())
}

func (c *Command) Len() int {
	return len(c.Bytes())
}
