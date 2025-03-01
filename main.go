package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

/*
Each bootloader image contains a 4KB (4096 byte) header followed by a code body.

Known fields:
0x00  0    |  512 =    ???: ???
0x200 512  |  512 =    ???: consistent across images, unique per model (or series?)
0x400 1024 |    4 = uint32: magic
0x404 1028 |    8 =    ???: ???
0x40C 1036 |    4 = uint32: length of bootloader body
0x410 1040 |    4 = uint32: "USB Bootable" bit amongst other bitflags?
0x414 1044 |   12 =    ???: ???
0x420 1056 |   32 =  bytes: signature 1?
0x440 1088 |   32 =  bytes: signature 2?
0x460 1120 | 2976 =    ???: ??? (always empty)
*/

/* TODO:
FBPK:
- Create FBPK package, migrate main of fbpk to unique cmd
- Parse and use FBPKv2 bootloader image via fbpk

OTA:
- Include aota
- Parse and use OTA payload image via aota

DNW:
- Create DNW package, create main for unique cmd
- Create reader and writer threads with read and write queues
- Add cloning support for unique position trackers to allow independent queue seeking
- Create Go types and enums for known fields in a response message to clean up processing
- Support waiting for a queued message with constraints (i.e. ACK/NAK for EUB)
*/

const (
	app = "Tensor-USBDL"
	ver = "v0.0.1"
	god = "JoshuaDoes"
)

var (
	help   = false
	useDNW = false
	bitUSB = false

	src         = "sources"
	factory     = "bootloader.img"
	ota         = "payload.bin"
	ufs         = "ufs.img"
	partition0  = "partition_0.img"
	partition1  = "partition_1.img"
	partition2  = "partition_2.img"
	partition3  = "partition_3.img"
	bl1         = "bl1.img"
	pbl         = "pbl.img"
	bl2         = "bl2.img"
	abl         = "abl.img"
	bl31        = "bl31.img"
	gsa         = "gsa.img"
	tzsw        = "tzsw.img"
	ldfw        = "ldfw.img"
	ufsfwupdate = "ufsfwupdate.img"
	dpm         = ""

	address []byte
	crc     []byte
	header  = 4096
)

func usage() {
	prog := strings.TrimSuffix(filepath.Base(os.Args[0]), filepath.Ext(os.Args[0]))
	text := fmt.Sprintf("\n -> %s %s - %s <-\n\n"+
		" Tensor USB Downloader is a tool to send bootloaders over serial USB to a"+
		" connected Google Pixel device in Exynos USB Boot mode."+
		"\n"+
		" By default, we look for all specified images in a relative folder named '%s'.\n"+
		" If available, the factory bootloader image will be used first, and defaults to '%s'.\n"+
		" In lieu of that, an OTA payload may be used instead, defaulting to '%s'.\n"+
		" Lastly, we try to discover the individual images, named to match their counterparts (as available in"+
		" both a factory images ZIP's embedded images ZIP as well as an OTA payload).\n"+
		"\n"+
		" When specifying a factory bootloader image, you must provide the path to either the raw image itself"+
		" or a factory images ZIP containing the bootloader image.\n"+
		" When specifying an OTA payload, you must provide the path to either the payload image itself or an OTA"+
		" ZIP containing the payload image.\n"+
		"\n"+
		" Usage of %s:\n"+
		" -h, --help    | none   | Prints the help you see now and ignores other arguments\n"+
		"\n"+
		" > Sources\n"+
		" -i, --src     | string | Directory with bootloader images to serve         | %s\n"+
		" -f, --factory | string | FBPK (FastBoot PacK) v2 bootloader image to serve | %s\n"+
		" -o, --ota     | string | OTA payload to serve                              | %s\n"+
		" -u, --ufs     | string | UFS image to serve                                | %s\n"+
		" --partition0  | string | 1st UFS LUN to serve                              | %s\n"+
		" --partition1  | string | 2nd UFS LUN to serve                              | %s\n"+
		" --partition2  | string | 3rd UFS LUN to serve                              | %s\n"+
		" --partition3  | string | 4th UFS LUN to serve                              | %s\n"+
		" -1, --bl1     | string | BL1 image to serve                                | %s\n"+
		" -p, --pbl     | string | PBL image to serve                                | %s\n"+
		" -2, --bl2     | string | BL2 image to serve                                | %s\n"+
		" -a, --abl     | string | ABL image to serve                                | %s\n"+
		" -3, --bl31    | string | BL31 image to serve                               | %s\n"+
		" -g, --gsa     | string | GSA image to serve                                | %s\n"+
		" -t, --tzsw    | string | TZSW (TrustZone SoftWare) image to serve          | %s\n"+
		" -l, --ldfw    | string | LDFW (LoaDable FirmWare) image to serve           | %s\n"+
		" --ufsfwupdate | string | UFS firmware update image to serve                | %s\n"+
		" -d, --dpm     | string | DPM image to serve instead of zeroed 4KB\n"+
		"\n"+
		" > Controls\n"+
		" --address     | hex    | Target download address to write to                          | %X\n"+
		" --header      | number | Number of bytes to interpret as header for splittable images | %d\n"+
		" -c, --crc     | hex    | Overrides the calculated CRC when writing DNW commands\n"+
		" --dnw         | none   | Overrides the download address to %X\n"+
		" --usb         | none   | sets the 1040th byte to 01 if it is 00\n",
		app, ver, god,
		src, factory, ota,
		prog,
		src, factory, ota,
		ufs, partition0, partition1, partition2, partition3,
		bl1, pbl, bl2, abl, bl31, gsa, tzsw, ldfw, ufsfwupdate,
		address, header, opDNW)
	fmt.Fprintf(os.Stderr, "%s\n", text)
}

func main() {
	pflag.Usage = usage
	pflag.CommandLine.SortFlags = false
	pflag.BoolVarP(&help, "help", "h", false, "")
	pflag.BoolVar(&useDNW, "dnw", false, "")
	pflag.BoolVar(&bitUSB, "usb", false, "")
	pflag.StringVarP(&src, "src", "i", src, "")
	pflag.StringVarP(&factory, "factory", "f", factory, "")
	pflag.StringVarP(&ota, "ota", "o", ota, "")
	pflag.StringVarP(&ufs, "ufs", "u", ufs, "")
	pflag.StringVar(&partition0, "partition0", partition0, "")
	pflag.StringVar(&partition1, "partition1", partition1, "")
	pflag.StringVar(&partition2, "partition2", partition2, "")
	pflag.StringVar(&partition3, "partition3", partition3, "")
	pflag.StringVarP(&bl1, "bl1", "1", bl1, "")
	pflag.StringVarP(&pbl, "pbl", "p", pbl, "")
	pflag.StringVarP(&bl2, "bl2", "2", bl2, "")
	pflag.StringVarP(&abl, "abl", "a", abl, "")
	pflag.StringVarP(&bl31, "bl31", "3", bl31, "")
	pflag.StringVarP(&gsa, "gsa", "g", gsa, "")
	pflag.StringVarP(&tzsw, "tzsw", "t", tzsw, "")
	pflag.StringVarP(&ldfw, "ldfw", "l", ldfw, "")
	pflag.StringVar(&ufsfwupdate, "ufsfwupdate", ufsfwupdate, "")
	pflag.StringVarP(&dpm, "dpm", "d", dpm, "")
	pflag.BytesHexVar(&address, "address", address, "")
	pflag.IntVar(&header, "header", header, "")
	pflag.BytesHexVarP(&crc, "crc", "c", crc, "")
	pflag.Parse()

	if help {
		usage()
		return
	}

	if header <= 0 {
		fmt.Println("[!] Header size must be positive number!")
		return
	}

	if src == "" {
		src = "sources"
	}
	if err := isDir(src); err != nil {
		fmt.Printf("[!] Error opening directory '%s': %v\n", src, err)
		return
	}

	if err := isFile(src, factory); err == nil {
		fmt.Println("[*] Processing FBPKv2")
	} else if err := isFile(src, ota); err == nil {
		fmt.Println("[*] Processing OTA")
	} else {
		fmt.Println("[*] Processing raw")
	}

	//TODO: Actually use the FBPKv2 or OTA when specified
	//----------------------

	lastSent := ""
	for {
		var err error
		var dnw *DNW
		justSent := ""
		canWrite := true
		toldLive := false
		timeLive := time.Now()
		var lastTrace []string

		fmt.Println("\n[*] Scanning for device...")
		for {
			scanStart := time.Now()
			dnw, err = NewDNW()
			if err == nil {
				scanSince := time.Since(scanStart)
				timeLive = scanStart
				fmt.Printf("[*] Found device after %dms since connection\n", scanSince.Milliseconds())
				break
			}
		}
		fmt.Println("[*] Connected to device!")
		fmt.Println("- Port:  ", dnw.GetPort())
		fmt.Println("- ID:    ", dnw.GetID())
		fmt.Println("- Serial:", dnw.GetSerial())
		fmt.Println("- USB:   ", dnw.GetUSB())

		for {
			var msg *Message
			msg, err = dnw.ReadMsg()
			if err != nil {
				fmt.Printf("[!] Error reading from device: %v\n", err)
				err = nil
				break
			}
			if msg == nil {
				continue
			}

			if msg.IsControlBit() {
				//Ignore, possibly marks end of request?
				fmt.Println("[*] Received control bit:", msg.String())
				continue
			}

			switch msg.Type() {
			case "exynos_usb_booting":
				if msg.Device() != "" && !toldLive {
					toldLive = true
					fmt.Println("[*] Device identified as", msg.Device())
				}
			case "eub":
				if !toldLive {
					fmt.Println("[!] Unknown sender:", msg)
					continue
				}

				switch msg.Command() {
				case "req":
					if !canWrite || justSent == msg.Argument() {
						continue
					}
					fmt.Println("[*] Received request for", msg.Argument())

					lastSent = msg.Argument()
					addr := address
					if useDNW {
						addr = opDNW
					}

					switch msg.Argument() {
					case "DPM":
						//addr = []byte{'D', 'P', 'M', '1'}
						if dpm != "" {
							err = writeFile(dnw, dpm, addr)
						} else {
							err = writeRaw(dnw, make([]byte, 4096), addr)
						}
					case "EPBL":
						//addr = []byte{'E', 'P', 'B', 'L'}
						err = writeFile(dnw, pbl, addr)
					case "bl1":
						//addr = []byte{'A', 'P', 'B', 'L'}
						err = writeFile(dnw, bl1, addr)
					case "ABL":
						err = writeFileHead(dnw, abl, addr)
					case "ABLB":
						err = writeFileBody(dnw, abl, addr)
					default:
						fmt.Println("[!] Unhandled EUB request:", msg.Argument())
					}

					if err == nil {
						fmt.Printf("[*] Successfully wrote %s\n", lastSent)
						justSent = lastSent
						canWrite = false
						//time.Sleep(time.Millisecond * 500)
					} else {
						fmt.Printf("[!] Failed to write %s, sending stop\n", lastSent)
						dnw.WriteCmd(cmdStop)
					}
				case "ack": //Acknowledged
					fmt.Println("[*] Acknowledged:", msg.Argument())
					canWrite = true
				case "nak": //Not acknowledged
					fmt.Println("[!] Not acknowledged:", msg.Argument())
					dnw.WriteCmd(cmdStop)
				default:
					fmt.Println("[!] Unhandled EUB message:", msg)
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
					fmt.Println("[!] Error reading BootROM boot failure trace:", err)
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
							fmt.Println("[*] Received duplicate failure trace")
							continue
						}
					}
					lastTrace = trace

					prefix := "\n> "
					fmt.Printf("[!] BootROM error booting")
					if lastSent != "" {
						fmt.Printf(" %s", lastSent)
					}
					fmt.Printf(":%s%s\n", prefix, strings.Join(trace, prefix))
				}
			case "error":
				fmt.Printf("[!] Processed error: %s: %s\n", msg.Command(), msg.Argument())
			default:
				fmt.Printf("[!] Unhandled message: %s (%0X)\n", msg, msg)
			}

			if err != nil {
				break
			}
		}

		if err != nil {
			fmt.Println("[!] Fallback error:", err)
		}

		fmt.Printf("[*] Disconnecting from Pixel ROM Recovery...\n")
		if err := dnw.Close(); err != nil {
			fmt.Println("[!] Error closing connection:", err)
		}

		since := time.Since(timeLive)
		fmt.Printf("[*] Connection lasted %s\n", since.String())
	}
}
