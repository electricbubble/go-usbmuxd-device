package go_usbmuxd_device

import (
	"errors"
	"fmt"
	"github.com/electricbubble/go-usbmuxd-device/usbmux"
	"howett.net/plist"
	"net"
)

var ErrNoFindUSBDevice = errors.New("no find match device")

type USBDevice struct {
	DeviceID        int
	LocationID      int
	ProductID       int
	SerialNumber    string
	ConnectionSpeed int
	ConnectionType  usbmux.ConnectionType
}

type USBHub struct {
	// proto usbmux.Protocol
}

func NewUSBHub() *USBHub {
	return &USBHub{}
}

func (c *USBHub) DeviceList() (usbDevices []USBDevice, err error) {
	var proto *usbmux.Protocol
	if proto, err = usbmux.NewProtocol(usbmux.NewDefaultRequestFrame(usbmux.MessageTypeDeviceList), usbmux.PacketProtocolPlist, usbmux.PacketTypePlistPayload); err != nil {
		return nil, err
	}

	if err = proto.SendPacket(); err != nil {
		return nil, err
	}

	var respPacket *usbmux.ResponsePacket
	if respPacket, err = proto.RecvPacket(); err != nil {
		return nil, err
	}

	var devList usbmux.DeviceListResponseFrame
	if _, err = plist.Unmarshal(respPacket.Packet, &devList); err != nil {
		return nil, err
	}

	usbDevices = make([]USBDevice, 0, len(devList.DeviceList))
	for i := range devList.DeviceList {
		var dev USBDevice
		if devList.DeviceList[i].Properties.ConnectionType == usbmux.ConnectionTypeUSB {
			dev.DeviceID = devList.DeviceList[i].Properties.DeviceID
			dev.LocationID = devList.DeviceList[i].Properties.LocationID
			dev.ProductID = devList.DeviceList[i].Properties.ProductID
			dev.SerialNumber = devList.DeviceList[i].Properties.SerialNumber
			dev.ConnectionSpeed = devList.DeviceList[i].Properties.ConnectionSpeed
			dev.ConnectionType = devList.DeviceList[i].Properties.ConnectionType
			usbDevices = append(usbDevices, dev)
		}
	}

	if len(usbDevices) == 0 {
		return nil, ErrNoFindUSBDevice
	}

	return
}

func (c *USBHub) CreateConnect(devID int, port int) (conn net.Conn, err error) {
	var proto *usbmux.Protocol
	if proto, err = usbmux.NewProtocol(usbmux.NewConnectRequestFrame(devID, port), usbmux.PacketProtocolPlist, usbmux.PacketTypePlistPayload); err != nil {
		return nil, err
	}

	if err = proto.SendPacket(); err != nil {
		return nil, err
	}

	var respPacket *usbmux.ResponsePacket
	if respPacket, err = proto.RecvPacket(); err != nil {
		return nil, err
	}

	if respPacket.MsgType != usbmux.MessageTypeResult {
		return nil, fmt.Errorf("message type mismatch: expected '%s', got '%s'", usbmux.MessageTypeResult, respPacket.MsgType)
	}

	var result usbmux.ResultResponseFrame
	if _, err = plist.Unmarshal(respPacket.Packet, &result); err != nil {
		return nil, err
	}

	if result.ReplyCode != usbmux.ReplyCodeOK {
		return nil, fmt.Errorf("connect: %s", result.ReplyCode)
	}

	conn = proto.Conn()

	return
}

func Debug(b ...bool) {
	if len(b) == 0 {
		b = []bool{true}
	}
	usbmux.Debug = b[0]
}
