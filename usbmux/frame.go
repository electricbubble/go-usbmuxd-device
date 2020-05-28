package usbmux

var _version = "v0.0.1"
var _name = "go-usbmuxd-device"

type MessageType string

const (
	MessageTypeResult       MessageType = "Result"
	MessageTypeConnect      MessageType = "Connect"
	MessageTypeListen       MessageType = "Listen"
	MessageTypeDeviceAdd    MessageType = "Attached"
	MessageTypeDeviceRemove MessageType = "Detached"
	MessageTypeReadBUID     MessageType = "ReadBUID"
	MessageTypeDeviceList   MessageType = "ListDevices"
)

type ConnectionType string

const (
	ConnectionTypeUSB     = "USB"
	ConnectionTypeNetwork = "Network"
)

type DefaultRequestFrame struct {
	MessageType         MessageType `plist:"MessageType"`
	ClientVersionString string      `plist:"ClientVersionString"`
	ProgName            string      `plist:"ProgName"`
}

type ConnectRequestFrame struct {
	DefaultRequestFrame
	DeviceID int `plist:"DeviceID"`
	Port     int `plist:"PortNumber"`
}

func NewDefaultRequestFrame(msgType MessageType) DefaultRequestFrame {
	return DefaultRequestFrame{
		MessageType:         msgType,
		ProgName:            _name,
		ClientVersionString: _name + "_" + _version,
	}
}

func NewConnectRequestFrame(deviceID, port int) ConnectRequestFrame {
	return ConnectRequestFrame{
		DefaultRequestFrame: NewDefaultRequestFrame(MessageTypeConnect),
		DeviceID:            deviceID,
		Port:                ((port << 8) & 0xFF00) | (port >> 8),
	}
}

type ResultResponseFrame struct {
	MessageType MessageType `plist:"MessageType"`
	ReplyCode   ReplyCode   `plist:"Number"`
}

type DeviceResponseFrame struct {
	MessageType string                            `plist:"MessageType"`
	DeviceID    int                               `plist:"DeviceID"`
	Properties  DeviceAttachedPropertiesDictFrame `plist:"Properties"`
}

type DeviceAttachedPropertiesDictFrame struct {
	ConnectionSpeed int            `plist:"ConnectionSpeed"`
	ConnectionType  ConnectionType `plist:"ConnectionType"`
	DeviceID        int            `plist:"DeviceID"`
	LocationID      int            `plist:"LocationID"`
	ProductID       int            `plist:"ProductID"`
	SerialNumber    string         `plist:"SerialNumber"`
}

type DeviceListResponseFrame struct {
	DeviceList []DeviceResponseFrame `plist:"DeviceList"`
}
