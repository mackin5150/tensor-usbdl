package tensorutils

import (
	"sync"

	"go.bug.st/serial/enumerator"
)

var (
	mutexDevices sync.Mutex
	knownDevices []*enumerator.PortDetails
)

func refreshDevices() {
	mutexDevices.Lock()
	defer mutexDevices.Unlock()

	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return
	}
	knownDevices = ports
}

func getDevice(vid, pid string) *enumerator.PortDetails {
	mutexDevices.Lock()
	defer mutexDevices.Unlock()

	for i := 0; i < len(knownDevices); i++ {
		dev := knownDevices[i]
		if dev.VID != vid || dev.PID != pid {
			continue
		}
		return dev
	}
	return nil
}
