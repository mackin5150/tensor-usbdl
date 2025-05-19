package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/JoshuaDoes/crunchio"
	tensorutils "github.com/JoshuaDoes/tensor-usbdl"
)

func readFile(file string, offset, length int) (bytes []byte, err error) {
	bytes, err = rf(src+"/"+file, offset, length)
	if err != nil {
		bytes, err = rf(file, offset, length)
	}
	return
}

func rf(file string, offset, length int) ([]byte, error) {
	if offset == 0 && length == 0 {
		return os.ReadFile(file)
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err = f.Seek(int64(offset), 0); err != nil {
		return nil, err
	}

	var bytes []byte
	if length > 0 {
		bytes = make([]byte, length)
	} else {
		fi, err := f.Stat()
		if err != nil {
			return nil, err
		}
		length = int(fi.Size()) - offset
		if length <= 0 {
			return nil, fmt.Errorf("file too small for offset:%d length:%d", offset, length)
		}
	}

	n, err := f.Read(bytes)
	if err != nil {
		return nil, err
	}
	if n != length {
		return nil, fmt.Errorf("read %d bytes, expected %d", n, length)
	}

	//Set USB-bootable bit (first bit) in header of bootloader
	if bitUSB && offset <= 1040 && len(bytes) > (1040-offset) {
		bytes[1040-offset] |= 1
	}

	return bytes, err
}

func writeFile(dnw *tensorutils.DNW, cmd, arg []byte, file string) error {
	bytes, err := readFile(file, 0, 0)
	if err != nil {
		return err
	}
	return writeRaw(dnw, cmd, arg, bytes)
}

func writeFileHead(dnw *tensorutils.DNW, cmd, arg []byte, file string) error {
	bytes, err := readFile(file, 0, header)
	if err != nil {
		return err
	}
	return writeRaw(dnw, cmd, arg, bytes)
}

func writeFileBody(dnw *tensorutils.DNW, cmd, arg []byte, file string) error {
	bytes, err := readFile(file, header, 0)
	if err != nil {
		return err
	}
	return writeRaw(dnw, cmd, arg, bytes)
}

func writeRaw(dnw *tensorutils.DNW, cmd, arg, bytes []byte) error {
	if cmd != nil {
		if err := dnw.WriteCmd(tensorutils.NewCommand(cmd, arg, bytes, checksum(bytes))); err != nil {
			return fmt.Errorf("failed to write %d bytes to address %X: %v", len(bytes), cmd, err)
		}
		log.Debugf("Wrote %d bytes to address %X", len(bytes), cmd)
		return nil
	}
	if err := dnw.WriteCmd(tensorutils.NewCommand(nil, nil, bytes, nil)); err != nil {
		return fmt.Errorf("failed to write %d bytes: %v", len(bytes), err)
	}
	log.Debugf("Wrote %d bytes", len(bytes))
	return nil
}

func checksum(bytes []byte) []byte {
	buf := crunchio.NewBuffer("crc", make([]byte, 2))
	if crc != nil {
		buf.Buffer().WriteBytesNext(crc)
		log.Tracef("Used checksum: %X", crc)
	} else {
		var sum uint16

		for i := 0; i < len(bytes); i++ {
			sum += uint16(bytes[i])
		}

		buf.Buffer().WriteU16LENext([]uint16{sum})
		log.Tracef("Calculated checksum: %X", sum)
	}
	return buf.Bytes()
}

func bumpUint32(b []byte) []byte {
	u32 := crunchio.NewBuffer("u32", b)
	v := u32.Buffer().ReadU32LE(0, 1)[0]
	v += 1
	u32.Buffer().WriteU32LE(0, []uint32{v})
	return u32.Bytes()
}

func bumpUint16(b []byte) []byte {
	u16 := crunchio.NewBuffer("u16", b)
	v := u16.Buffer().ReadU16LE(0, 1)[0]
	v += 1
	u16.Buffer().WriteU16LE(0, []uint16{v})
	return u16.Bytes()
}

func isDir(paths ...string) error {
	path := pathJoin(paths...)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory '%s' does not exist", path)
		}
		return fmt.Errorf("error opening directory '%s': %v", path, err)
	}
	return nil
}

func isFile(paths ...string) error {
	path := pathJoin(paths...)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file '%s' does not exist", path)
		}
		return fmt.Errorf("error opening file '%s': %v", path, err)
	}
	return nil
}

func pathJoin(paths ...string) string {
	path := strings.Join(paths, "/")
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "/", "\\")
	} else {
		path = strings.ReplaceAll(path, "\\", "/")
	}
	return path
}
