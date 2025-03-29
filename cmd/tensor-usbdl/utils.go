package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/JoshuaDoes/crunchio"
	tensorutils "github.com/JoshuaDoes/tensor-usbdl"
)

func readFile(file string) ([]byte, error) {
	bytes, err := os.ReadFile(src + "/" + file)
	if err != nil {
		return os.ReadFile(file)
	}

	//Set USB boot flag(?) in header of bootloader
	if bitUSB && len(bytes) > 1040 && bytes[1040] == 0 {
		bytes[1040] = 1
	}

	return bytes, err
}

func writeFile(dnw *tensorutils.DNW, file string, addr []byte) error {
	bytes, err := readFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes, addr)
}

func writeFileHead(dnw *tensorutils.DNW, file string, addr []byte) error {
	bytes, err := readFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes[:header], addr)
}

func writeFileBody(dnw *tensorutils.DNW, file string, addr []byte) error {
	bytes, err := readFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes[header:], addr)
}

func writeRaw(dnw *tensorutils.DNW, bytes, addr []byte) error {
	if addr != nil {
		if err := dnw.WriteCmd(tensorutils.NewCommand(address, bytes, checksum(bytes))); err != nil {
			log.Errorf("Failed to write %d bytes to address %X: %v", len(bytes), addr, err)
			return err
		}
		log.Debugf("Wrote %d bytes to address %X", len(bytes), addr)
		return nil
	}
	if err := dnw.WriteCmd(tensorutils.NewCommand(nil, bytes, nil)); err != nil {
		log.Errorf("Failed to write %d bytes: %v", len(bytes), err)
		return err
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

func stringChunk(s string, size int) []string {
	chunks := make([]string, 0)
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}
