package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/JoshuaDoes/crunchio"
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

func writeFile(dnw *DNW, file string, addr []byte) error {
	bytes, err := readFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes, addr)
}

func writeFileHead(dnw *DNW, file string, addr []byte) error {
	bytes, err := readFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes[:header], addr)
}

func writeFileBody(dnw *DNW, file string, addr []byte) error {
	bytes, err := readFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes[header:], addr)
}

func writeRaw(dnw *DNW, bytes, addr []byte) error {
	if addr != nil {
		if err := dnw.WriteCmd(NewCommand(address, bytes, checksum(bytes))); err != nil {
			fmt.Printf("[!] Failed to write %d bytes to address %X: %v\n", len(bytes), addr, err)
			return err
		}
		fmt.Printf("[*] Wrote %d bytes to address %X\n", len(bytes), addr)
		return nil
	}
	if err := dnw.WriteCmd(NewCommand(nil, bytes, nil)); err != nil {
		fmt.Printf("[!] Failed to write %d bytes: %v\n", len(bytes), err)
		return err
	}
	fmt.Printf("[*] Wrote %d bytes\n", len(bytes))
	return nil
}

func checksum(bytes []byte) []byte {
	buf := crunchio.NewBuffer("crc", make([]byte, 2))
	if crc != nil {
		buf.Buffer().WriteBytesNext(crc)
		fmt.Printf("[#] Using checksum: %X\n", crc)
	} else {
		var sum uint16

		for i := 0; i < len(bytes); i++ {
			sum += uint16(bytes[i])
		}

		buf.Buffer().WriteU16LENext([]uint16{sum})
		fmt.Printf("[#] Calculated checksum: %X\n", sum)
	}
	return buf.Bytes()
}

func isDir(paths ...string) error {
	path := pathJoin(paths...)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Directory '%s' does not exist", path)
		}
		return fmt.Errorf("Error opening directory '%s': %v", path, err)
	}
	return nil
}

func isFile(paths ...string) error {
	path := pathJoin(paths...)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("File '%s' does not exist", path)
		}
		return fmt.Errorf("Error opening file '%s': %v", path, err)
	}
	return nil
}

func pathJoin(paths ...string) string {
	path := strings.Join(paths, "/")
	if runtime.GOOS == "windows" {
		strings.ReplaceAll(path, "/", "\\")
	} else {
		strings.ReplaceAll(path, "\\", "/")
	}
	return path
}
