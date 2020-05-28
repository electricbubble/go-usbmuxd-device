package usbmux

import (
	"errors"
	"fmt"
	"howett.net/plist"
	"strings"
	"testing"
	"time"
)

func TestNewProtocol(t *testing.T) {
	Debug = true

	protocol, err := NewProtocol(NewDefaultRequestFrame(MessageTypeListen), PacketProtocolPlist, PacketTypePlistPayload)
	checkErr(t, err, "NewProtocol")

	err = protocol.SendPacket()
	checkErr(t, err, "protocol.SendPacket")

	over := make(chan error)

	go func() {
		for {
			respPacket, err := protocol.RecvPacket()
			if err != nil {
				over <- fmt.Errorf("protocol.RecvPacket: %w", err)
			}

			switch respPacket.MsgType {
			case MessageTypeResult:
				var result ResultResponseFrame
				_, err := plist.Unmarshal(respPacket.Packet, &result)
				if err != nil {
					over <- fmt.Errorf("plist.Unmarshal: %w", err)
				}

				if result.ReplyCode != ReplyCodeOK {
					over <- errors.New(result.ReplyCode.String())
				}
			case MessageTypeDeviceAdd:
				// 新设备接入
				var newDevice DeviceResponseFrame
				_, err := plist.Unmarshal(respPacket.Packet, &newDevice)
				if err != nil {
					over <- fmt.Errorf("plist.Unmarshal: %w", err)
				}
				fmt.Println("设备接入:")
				fmt.Println("DeviceID:", newDevice.DeviceID)
				fmt.Println("ProductID:", newDevice.Properties.ProductID)
				fmt.Println("SerialNumber:", newDevice.Properties.SerialNumber)
			case MessageTypeDeviceRemove:
				// 设备被拔出
				var dev DeviceResponseFrame
				_, err := plist.Unmarshal(respPacket.Packet, &dev)
				if err != nil {
					over <- fmt.Errorf("plist.Unmarshal: %w", err)
				}
				fmt.Println("设备被拔出:")
				fmt.Println("DeviceID:", dev.DeviceID)
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 5)

		newProtocol, err2 := NewProtocol(NewConnectRequestFrame(12, 8100), PacketProtocolPlist, PacketTypePlistPayload)
		if err2 != nil {
			over <- fmt.Errorf("newProtocol.SendNewProtocol: %w", err2)
		}
		err2 = newProtocol.SendPacket()
		if err2 != nil {
			over <- fmt.Errorf("newProtocol.SendNewProtocol: %w", err2)
		}

		fmt.Println("已发送连接设备请求", strings.Repeat("\n", 2))

		_, err2 = newProtocol.RecvPacket()
		if err2 != nil {
			over <- fmt.Errorf("newProtocol.RecvPacket: %w", err2)
		}

		// fmt.Println("接收 connect 响应:", string(respPacket.Packet))
	}()

	for {
		// time.Sleep(time.Millisecond * 100)
		t.Fatal(<-over)
	}
}

func checkErr(t *testing.T, err error, msg ...string) {
	if err != nil {
		if len(msg) == 0 {
			t.Fatal(err)
		} else {
			t.Fatal(msg, err)
		}
	}
}
