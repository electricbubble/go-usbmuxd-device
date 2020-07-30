package usbmux

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"runtime"
	"strconv"

	"howett.net/plist"
)

type PacketType uint32

const (
	PacketTypeResult PacketType = 1 + iota
	PacketTypeConnect
	PacketTypeListen
	PacketTypeDeviceAdd
	PacketTypeDeviceRemove
	_ // ignore `6`
	_ // ignore `7`
	PacketTypePlistPayload
)

type PacketProtocol uint32

const (
	PacketProtocolBinary PacketProtocol = iota
	PacketProtocolPlist
)

type ReplyCode uint32

const (
	ReplyCodeOK ReplyCode = iota
	ReplyCodeBadCommand
	ReplyCodeBadDevice
	ReplyCodeConnectionRefused
	_ // ignore `4`
	_ // ignore `5`
	ReplyCodeBadVersion
)

func (rc ReplyCode) String() string {
	switch rc {
	case ReplyCodeOK:
		return "ok"
	case ReplyCodeBadCommand:
		return "bad command"
	case ReplyCodeBadDevice:
		return "bad device"
	case ReplyCodeConnectionRefused:
		return "connection refused"
	case ReplyCodeBadVersion:
		return "bad version"
	default:
		return "unknown reply code: " + strconv.Itoa(int(rc))
	}
}

var ErrConnBroken = errors.New("socket connection broken")

var Debug = false

type ResponsePacket struct {
	MsgType      MessageType
	ProtoVersion PacketProtocol
	ProtoType    PacketType
	ProtoTag     uint32
	Packet       []byte
}

type Protocol struct {
	sock         net.Conn
	frameBuffer  *bytes.Buffer
	protoVersion PacketProtocol
	protoType    PacketType
	tag          uint32
}

func NewProtocol(frame interface{}, protoVersion PacketProtocol, protoType PacketType) (*Protocol, error) {
	buffer := new(bytes.Buffer)
	encoder := plist.NewEncoder(buffer)
	if err := encoder.Encode(frame); err != nil {
		return nil, err
	}

	p := &Protocol{
		frameBuffer:  buffer,
		protoVersion: protoVersion,
		protoType:    protoType,
		tag:          1,
	}

	var conn net.Conn
	var err error
	if runtime.GOOS == "windows" {
		conn, err = net.Dial("tcp", "127.0.0.1:27015")
	} else {
		conn, err = net.Dial("unix", "/var/run/usbmuxd")
	}
	if err != nil {
		return nil, err
	}
	p.sock = conn
	return p, nil
}

func (p *Protocol) Conn() net.Conn {
	return p.sock
}

func (p *Protocol) SendPacket() error {
	packet := p._pack(p.tag)
	p.tag++
	return p._send(packet)
}

func (p *Protocol) _pack(protoTag uint32) []byte {
	b := new(bytes.Buffer)
	_ = binary.Write(b, binary.LittleEndian, uint32(p.frameBuffer.Len()+16))
	_ = binary.Write(b, binary.LittleEndian, p.protoVersion)
	_ = binary.Write(b, binary.LittleEndian, p.protoType)
	_ = binary.Write(b, binary.LittleEndian, protoTag)
	b.Write(p.frameBuffer.Bytes())

	if Debug {
		log.Printf("[DEBUG]↩︎\n"+
			"请求报文总长度: %d\t协议版本: %d\t协议类型: %d\t请求 Tag: %d\n"+
			"请求报文: %s\n\n", p.frameBuffer.Len()+16, p.protoVersion, p.protoType, protoTag, p.frameBuffer.String())
	}

	return b.Bytes()
}

func (p *Protocol) RecvPacket() (respPacket *ResponsePacket, err error) {
	var recvMsg []byte
	if recvMsg, err = p._recv(4); err != nil {
		return nil, err
	}
	uSize := binary.LittleEndian.Uint32(recvMsg)

	return p._unpack(p._recv(int(uSize) - 4))
}

func (p *Protocol) _unpack(recvMsg []byte, errs ...error) (respPacket *ResponsePacket, err error) {
	if len(errs) != 0 && errs[0] != nil {
		return nil, errs[0]
	}
	respPacket = new(ResponsePacket)
	reader := bytes.NewReader(recvMsg)

	if err = binary.Read(reader, binary.LittleEndian, &respPacket.ProtoVersion); err != nil {
		return nil, err
	}
	if err = binary.Read(reader, binary.LittleEndian, &respPacket.ProtoType); err != nil {
		return nil, err
	}
	if err = binary.Read(reader, binary.LittleEndian, &respPacket.ProtoTag); err != nil {
		return nil, err
	}

	type tmpMessageType struct {
		MessageType MessageType `plist:"MessageType"`
	}

	var msgType tmpMessageType
	if _, err = plist.Unmarshal(recvMsg[12:], &msgType); err != nil {
		return nil, err
	}
	respPacket.MsgType = msgType.MessageType

	respPacket.Packet = recvMsg[12:]

	if Debug {
		log.Printf("[DEBUG]↩︎\n"+
			"响应报文总长度: %d\t协议版本: %d\t协议类型: %d\t响应 Tag: %d\n"+
			"响应报文: %s\n\n", len(recvMsg)+4, respPacket.ProtoVersion, respPacket.ProtoType, respPacket.ProtoTag, string(respPacket.Packet))
	}

	return
}

func (p *Protocol) _send(msg []byte) (err error) {
	for totalSent := 0; totalSent < len(msg); {
		var sent int
		if sent, err = p.sock.Write(msg[totalSent:]); err != nil {
			return err
		}
		if sent == 0 {
			return ErrConnBroken
		}
		totalSent += sent
	}
	return
}

func (p *Protocol) _recv(size int) (msg []byte, err error) {
	msg = make([]byte, 0, size)
	for len(msg) < size {
		buf := make([]byte, size-len(msg))
		var n int
		if n, err = p.sock.Read(buf); err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, ErrConnBroken
		}
		msg = append(msg, buf...)
	}
	return
}
