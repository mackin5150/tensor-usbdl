package main

import (
	"fmt"
	"os"

	"go.bug.st/serial"
)

var (
	device = "gs101"
	header = 4080
)

func main() {
	fmt.Println("Scanning for COM4")
	foundCOM4 := ""
	for {
		ports, err := serial.GetPortsList()
		if err != nil {
			fmt.Println("Error getting serial ports:", err)
			continue
		}
		for i := 0; i < len(ports); i++ {
			if ports[i] == "/dev/ttyACM0" || ports[i] == "COM4" {
				foundCOM4 = ports[i]
				break
			}
		}
		if foundCOM4 != "" {
			break
		}
	}

	dnw := NewDnw(foundCOM4)
	if err := dnw.Open(); err != nil {
		fmt.Println("Error opening COM4:", err)
		return
	}
	fmt.Println("Connected to COM4")

	hasBL1 := false
	toldLost := false
	toldAlive := false
	for {
		apModel, reqBin, err := dnw.GetReqFromDevice()
		if err != nil {
			hasBL1 = false
			toldAlive = false
			for {
				dnw.Close()
				if !toldLost {
					fmt.Printf("Lost connection to COM4: %v\n\n", err)
					toldLost = true
				}
				if err := dnw.Open(); err != nil {
					continue
				}
				fmt.Println("Connected to COM4")
				toldLost = false
				apModel, reqBin, err = dnw.GetReqFromDevice()
				if err == nil {
					break
				}
			}
		}

		if apModel != "" && !toldAlive {
			toldAlive = true
			fmt.Printf("Device %s is alive!\n", apModel)
		}

		if reqBin == "" {
			continue
		}

		switch reqBin {
		case "bl1":
			fmt.Println("Sending bl1")
			bl1, err := os.ReadFile(device + "/bl1.img")
			if err != nil {
				fmt.Println("Error reading bl1.img:", err)
				dnw.Close()
				continue
			}
			if err := dnw.SendOverDnw(apModel, "bl1", bl1); err != nil {
				fmt.Println("Failed to send bl1:", err)
				dnw.Close()
				continue
			}
			hasBL1 = true
		case "EPBL":
			fmt.Println("Sending EPBL")
			epbl, err := os.ReadFile(device + "/pbl.img")
			if err != nil {
				fmt.Println("Error reading pbl.img:", err)
				dnw.Close()
				continue
			}
			if err := dnw.SendOverDnw(apModel, "EPBL", epbl); err != nil {
				fmt.Println("Failed to send EPBL:", err)
				dnw.Close()
				continue
			}
		case "BL2":
			fmt.Println("Sending BL2 header")
			bl2, err := os.ReadFile(device + "/bl2.img")
			if err != nil {
				fmt.Println("Error reading bl2.img:", err)
				dnw.Close()
				continue
			}
			bl2 = bl2[:header]
			if err := dnw.SendOverDnw(apModel, "BL2", bl2); err != nil {
				fmt.Println("Failed to send BL2 header:", err)
				dnw.Close()
				continue
			}
		case "BL2B":
			fmt.Println("Sending BL2 body")
			bl2, err := os.ReadFile(device + "/bl2.img")
			if err != nil {
				fmt.Println("Error reading bl2.img:", err)
				dnw.Close()
				continue
			}
			bl2 = bl2[header:]
			if err := dnw.SendOverDnw(apModel, "BL2B", bl2); err != nil {
				fmt.Println("Failed to send BL2 body:", err)
				dnw.Close()
				continue
			}
		case "ABL":
			fmt.Println("Sending ABL header")
			abl, err := os.ReadFile(device + "/abl.img")
			if err != nil {
				fmt.Println("Error reading abl.img:", err)
				dnw.Close()
				continue
			}
			abl = abl[:header]
			if err := dnw.SendOverDnw(apModel, "ABL", abl); err != nil {
				fmt.Println("Failed to send ABL header:", err)
				dnw.Close()
				continue
			}
		case "ABLB":
			fmt.Println("Sending ABL body")
			abl, err := os.ReadFile(device + "/abl.img")
			if err != nil {
				fmt.Println("Error reading abl.img:", err)
				dnw.Close()
				continue
			}
			abl = abl[header:]
			if err := dnw.SendOverDnw(apModel, "ABLB", abl); err != nil {
				fmt.Println("Failed to send ABL body:", err)
				dnw.Close()
				continue
			}
		case "BL31":
			fmt.Println("Sending BL31 header")
			bl31, err := os.ReadFile(device + "/bl31.img")
			if err != nil {
				fmt.Println("Error reading bl31.img:", err)
				dnw.Close()
				continue
			}
			bl31 = bl31[:header]
			if err := dnw.SendOverDnw(apModel, "BL31", bl31); err != nil {
				fmt.Println("Failed to send BL31 header:", err)
				dnw.Close()
				continue
			}
		case "BL3B":
			fmt.Println("Sending BL31 body")
			bl31, err := os.ReadFile(device + "/bl31.img")
			if err != nil {
				fmt.Println("Error reading bl31.img:", err)
				dnw.Close()
				continue
			}
			bl31 = bl31[header:]
			if err := dnw.SendOverDnw(apModel, "BL3B", bl31); err != nil {
				fmt.Println("Failed to send BL31 body:", err)
				dnw.Close()
				continue
			}
		case "GSA1":
			fmt.Println("Sending GSA1")
			gsa, err := os.ReadFile(device + "/gsa.img")
			if err != nil {
				fmt.Println("Error reading gsa.img:", err)
				dnw.Close()
				continue
			}
			if err := dnw.SendOverDnw(apModel, "GSA1", gsa); err != nil {
				fmt.Println("Failed to send GSA1:", err)
				dnw.Close()
				continue
			}
		case "TZSW":
			fmt.Println("Sending TZSW header")
			trusty, err := os.ReadFile(device + "/trusty.img")
			if err != nil {
				fmt.Println("Error reading trusty.img:", err)
				dnw.Close()
				continue
			}
			trusty = trusty[:header]
			if err := dnw.SendOverDnw(apModel, "TZSW", trusty); err != nil {
				fmt.Println("Failed to send TZSW header:", err)
				dnw.Close()
				continue
			}
		case "TZSB":
			fmt.Println("Sending TZSW body")
			trusty, err := os.ReadFile(device + "/trusty.img")
			if err != nil {
				fmt.Println("Error reading trusty.img:", err)
				dnw.Close()
				continue
			}
			trusty = trusty[header:]
			if err := dnw.SendOverDnw(apModel, "TZSB", trusty); err != nil {
				fmt.Println("Failed to send TZSW body:", err)
				dnw.Close()
				continue
			}
		case "LDFW":
			fmt.Println("Sending LDFW header")
			ldfw, err := os.ReadFile(device + "/ldfw.img")
			if err != nil {
				fmt.Println("Error reading ldfw.img:", err)
				dnw.Close()
				continue
			}
			ldfw = ldfw[:header]
			if err := dnw.SendOverDnw(apModel, "LDFW", ldfw); err != nil {
				fmt.Println("Failed to send LDFW header:", err)
				dnw.Close()
				continue
			}
		case "LDFB":
			fmt.Println("Sending LDFW body")
			ldfw, err := os.ReadFile(device + "/ldfw.img")
			if err != nil {
				fmt.Println("Error reading ldfw.img:", err)
				dnw.Close()
				continue
			}
			ldfw = ldfw[header:]
			if err := dnw.SendOverDnw(apModel, "LDFB", ldfw); err != nil {
				fmt.Println("Failed to send LDFW body:", err)
				dnw.Close()
				continue
			}
		case "DPM":
			if !hasBL1 {
				continue
			}
			fmt.Println("Sending DPM")
			dpm, err := os.ReadFile(device + "/dpm-userdebug-sbdp-bypass.bin")
			if err != nil {
				fmt.Println("Error reading dpm.img:", err)
				dnw.Close()
				continue
			}
			if err := dnw.SendOverDnw(apModel, "DPM", dpm); err != nil {
				fmt.Println("Failed to send DPM:", err)
				dnw.Close()
				continue
			}
		default:
			fmt.Printf("TODO: IMPLEMENT %s!!!!!!!!!!\n", reqBin)
		}
	}
}
