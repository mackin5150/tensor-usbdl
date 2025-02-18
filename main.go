package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/JoshuaDoes/crunchio"
	"github.com/spf13/pflag"
)

var (
	header = 4096
	crc    []byte
	src    = "sources"
	dpm    = ""
	bl1    = "bl1.img"
	pbl    = "pbl.img"
	abl    = "abl.img"
)

func main() {
	pflag.IntVarP(&header, "header", "h", header, "Number of bytes to send as header before body for split images")
	pflag.BytesHexVarP(&crc, "crc", "c", crc, "CRC to use for DNW image commands instead of calculating one")
	pflag.StringVarP(&src, "src", "i", src, "Directory with bootloader images to serve")
	pflag.StringVarP(&dpm, "dpm", "d", dpm, "DPM image to serve instead of empty image")
	pflag.StringVarP(&bl1, "bl1", "1", bl1, "bl1 image to serve")
	pflag.StringVarP(&pbl, "epbl", "p", pbl, "EPBL image to serve")
	pflag.StringVarP(&abl, "abl", "a", abl, "ABL image to serve")
	pflag.Parse()

	lastSent := ""
	for {
		var err error
		var dnw *DNW
		toldLive := false
		justSent := ""
		var lastTrace []string

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
			case "C", "\x1B", string('\x00'):
				//Ignore, possibly marks end of request?
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
					fmt.Println("<-", msg.Argument())

					switch msg.Argument() {
					case "DPM":
						if dpm != "" {
							err = writeFile(dnw, src+"/"+dpm, true)
						} else {
							err = writeRaw(dnw, make([]byte, 4096), true)
						}
						if err != nil {
							fmt.Println("Error writing DPM:", err)
						}
					case "EPBL":
						err = writeFile(dnw, src+"/"+pbl, false)
						if err != nil {
							fmt.Println("Error writing EPBL:", err)
						}
					case "bl1":
						err = writeFile(dnw, src+"/"+bl1, false)
						if err != nil {
							fmt.Println("Error writing bl1:", err)
						}
					case "ABL":
						err = writeFileHead(dnw, src+"/"+abl, false)
						if err != nil {
							fmt.Println("Error writing ABL header:", err)
						}
					case "ABLB":
						err = writeFileBody(dnw, src+"/"+abl, false)
						if err != nil {
							fmt.Println("Error writing ABL body:", err)
						}
					default:
						fmt.Println("DEBUG:", msg)
					}

					lastSent = msg.Argument()
					if err == nil {
						justSent = lastSent
						fmt.Println("->", justSent)
					}
				}
			case "irom_booting_failure":
				trace := make([]string, 15)
				var codeMsg *Message
				for i := 0; i < len(trace); i++ {
					codeMsg, err = dnw.ReadMsg()
					if err != nil {
						break
					}
					trace[i] = codeMsg.Type()
				}
				if err != nil {
					fmt.Println("Error reading BootROM boot failure trace:", err)
				} else {
					if lastTrace != nil {
						diff := false
						for i := 0; i < len(trace); i++ {
							if trace[i] != lastTrace[i] {
								diff = true
								break
							}
						}
						if !diff {
							break
						}
					}
					lastTrace = trace

					prefix := "\n> "
					fmt.Printf("BootROM error booting")
					if lastSent != "" {
						fmt.Printf(" %s", lastSent)
					}
					fmt.Printf(":%s%s\n", prefix, strings.Join(trace, prefix))
				}
			case "error":
				fmt.Printf("%s: %s\n", msg.Command(), msg.Argument())
			default:
				fmt.Println("Unknown message type:", msg.Type(), fmt.Sprintf("(%0x)", []byte(msg.Type())))
			}

			if err != nil {
				break
			}
		}

		fmt.Println("Disconnecting from Pixel ROM Recovery...")
		if err := dnw.Close(); err != nil {
			fmt.Println("Error closing connection:", err)
		}
	}
}

func writeFile(dnw *DNW, file string, asCmd bool) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes, asCmd)
}

func writeFileHead(dnw *DNW, file string, asCmd bool) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes[:header], asCmd)
}

func writeFileBody(dnw *DNW, file string, asCmd bool) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return writeRaw(dnw, bytes[header:], asCmd)
}

func writeRaw(dnw *DNW, bytes []byte, asCmd bool) error {
	if asCmd {
		checksum := crc
		if checksum == nil {
			sum := uint16(0)
			for i := 0; i < len(bytes); i++ {
				sum += uint16(bytes[i])
			}
			sumBytes := crunchio.NewBuffer("crc", make([]byte, 2))
			sumBytes.Buffer().WriteU16LENext([]uint16{sum})
			checksum = sumBytes.Bytes()
		}
		return dnw.WriteCommand(NewCommand("\x1BDNW", bytes, checksum))
	}
	return dnw.WriteCommand(NewCommand("", bytes, nil))
}
