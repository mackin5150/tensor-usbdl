package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/JoshuaDoes/crunchio"
	"github.com/spf13/pflag"
)

var (
	header = 4096
	crc    []byte
	src    = "sources"
	bl1    = "bl1.img"
	pbl    = "pbl.img"
	bl2    = "bl2.img"
	abl    = "abl.img"
	bl31   = "bl31.img"
	gsa    = "gsa.img"
	tzsw   = "tzsw.img"
	ldfw   = "ldfw.img"
	dpm    = ""
)

func main() {
	pflag.IntVarP(&header, "header", "h", header, "Number of bytes to send as header before body for split images")
	pflag.BytesHexVarP(&crc, "crc", "c", crc, "CRC to use for DNW image commands instead of calculating one")
	pflag.StringVarP(&src, "src", "i", src, "Directory with bootloader images to serve")
	pflag.StringVarP(&bl1, "bl1", "1", bl1, "bl1 image to serve")
	pflag.StringVarP(&pbl, "pbl", "p", pbl, "PBL image to serve")
	pflag.StringVarP(&bl2, "bl2", "2", bl2, "BL2 image to serve")
	pflag.StringVarP(&abl, "abl", "a", abl, "ABL image to serve")
	pflag.StringVarP(&bl31, "bl31", "3", bl31, "BL31 image to serve")
	pflag.StringVarP(&gsa, "gsa", "g", gsa, "GSA image to serve")
	pflag.StringVarP(&tzsw, "tzsw", "t", tzsw, "TZSW image to serve")
	pflag.StringVarP(&ldfw, "ldfw", "l", ldfw, "LDFW image to serve")
	pflag.StringVarP(&dpm, "dpm", "d", dpm, "DPM image to serve instead of empty image")
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
			case "\x1B", string('\x00'):
				//Ignore, possibly marks end of request?
			case "C":
				fmt.Println("Received C byte")
			case "exynos_usb_booting":
				if msg.Device() != "" && !toldLive {
					toldLive = true
					fmt.Println("Pixel ROM Recovery identified as", msg.Device())
				}
			case "eub":
				if !toldLive {
					fmt.Println("Received message but not yet alive:", msg)
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

						time.Sleep(time.Second * 1)
					} else {
						dnw.WriteCmd(cmdStop)
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
		return dnw.WriteCmd(NewCommand("\x1BDNW", bytes, checksum))
	}
	return dnw.WriteCmd(NewCommand("", bytes, nil))
}
