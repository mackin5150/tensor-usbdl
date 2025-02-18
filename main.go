package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

var (
	crc []byte
	src = "sources"
	dpm = ""
	bl1 = "bl1.img"
	pbl = "pbl.img"
)

func main() {
	pflag.BytesHexVarP(&crc, "crc", "c", crc, "CRC to use for DNW image commands instead of calculating one")
	pflag.StringVarP(&src, "src", "i", src, "Directory with bootloader images to serve")
	pflag.StringVarP(&dpm, "dpm", "d", dpm, "DPM image to serve instead of empty image")
	pflag.StringVarP(&bl1, "bl1", "1", bl1, "bl1 image to serve")
	pflag.StringVarP(&pbl, "epbl", "p", pbl, "EPBL image to serve")
	pflag.Parse()

	for {
		var err error
		var dnw *DNW
		toldLive := false
		justSent := ""

		fmt.Println("")
		fmt.Println("Scanning for Pixel ROM Recovery...")
		for {
			dnw, err = NewDNW()
			if err == nil {
				break
			}
		}
		fmt.Println("Connected to Pixel ROM Recovery!")

		for {
			msg, err := dnw.ReadMsg()
			if err != nil {
				break
			}

			switch msg.Type() {
			default:
				//fmt.Println("Unknown message type:", msg.Type())
			case "C", string('\x1B'): //Ignore, possibly marks end of request?
			case "exynos_usb_booting":
				if msg.Device() != "" && !toldLive {
					toldLive = true
					fmt.Println("Pixel ROM Recovery identified as", msg.Device())
				}
			case "eub":
				if !toldLive {
					continue
				}

				switch msg.Command() {
				case "req":
					if justSent == msg.Argument() {
						continue
					}

					switch msg.Argument() {
					case "DPM":
						if dpm != "" {
							err = writeFile(dnw, src+"/"+dpm)
						} else {
							err = writeRaw(dnw, make([]byte, 4080))
						}
						if err != nil {
							fmt.Println("Error writing DPM:", err)
						}
					case "EPBL":
						err = writeFile(dnw, src+"/"+pbl)
						if err != nil {
							fmt.Println("Error writing EPBL:", err)
						}
					case "bl1":
						err = writeFile(dnw, src+"/"+bl1)
						if err != nil {
							fmt.Println("Error writing bl1:", err)
						}
					}

					if err == nil {
						justSent = msg.Argument()
						fmt.Println("-", justSent)
					}
				}
			case "irom_booting_failure":
				fmt.Println("iROM boot failure:", msg.Device())
			case "error":
				fmt.Printf("%s: %s\n", msg.Command(), msg.Argument())
			}

			if err != nil {
				break
			}
		}

		fmt.Println("Disconnecting from Pixel ROM Recovery...")
		if err := dnw.Close(); err != nil {
			fmt.Println("Error closing connection:", err)
		}
		dnw = nil
		err = nil
	}
}

func writeFile(dnw *DNW, file string) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes)
}

func writeRaw(dnw *DNW, bytes []byte) error {
	checksum := crc
	if checksum == nil {
		sum := uint16(0)
		for i := 0; i < len(bytes); i++ {
			sum += uint16(bytes[i])
		}
		checksum = []byte{byte(sum & 0xFF), byte(sum >> 8)}
	}
	return dnw.WriteCommand(NewCommand("DNW", bytes, checksum))
}
