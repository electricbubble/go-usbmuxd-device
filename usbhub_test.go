package go_usbmuxd_device

import (
	"fmt"
	"testing"
)

func TestUSBHub_DeviceList(t *testing.T) {
	usbHub := NewUSBHub()

	Debug()

	devices, err := usbHub.DeviceList()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(devices)

	conn, err := usbHub.CreateConnect(devices[0].DeviceID, 8100)
	if err != nil {
		t.Fatal(err)
	}

	_ = conn
}
