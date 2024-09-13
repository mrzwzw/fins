package fins

import (
	"encoding/binary"
	"fmt"
	"time"
)

const DEFAULT_RESPONSE_TIMEOUT = 2000 // ms

type Client interface {
	SetByteOrder(binary.ByteOrder)
	SetTimeoutMs(uint)
	Close()
	ReadWordsToUint16(memoryArea byte, address uint16, readCount uint16) ([]uint16, error)
	ReadWordsToUint32(memoryArea byte, address uint16, readCount uint16) ([]uint32, error)
	ReadBytes(memoryArea byte, address uint16, readCount uint16) ([]byte, error)
	ReadString(memoryArea byte, address uint16, readCount uint16) (string, error)
	ReadBits(memoryArea byte, address uint16, bitOffset byte, readCount uint16) ([]bool, error)
	ReadClock() (*time.Time, error)
	WriteWords(memoryArea byte, address uint16, data []uint16) error
	WriteString(memoryArea byte, address uint16, s string) error
	WriteBytes(memoryArea byte, address uint16, b []byte) error
	WriteBits(memoryArea byte, address uint16, bitOffset byte, data []bool) error
	SetBit(memoryArea byte, address uint16, bitOffset byte) error
	ResetBit(memoryArea byte, address uint16, bitOffset byte) error
	ToggleBit(memoryArea byte, address uint16, bitOffset byte) error
	bitTwiddle(memoryArea byte, address uint16, bitOffset byte, value byte) error
	nextHeader() *Header
	incrementSid() byte
	sendCommand(command []byte) (*response, error)
	listenLoop()
}

func NewClient(_type, remoteAddr, remotePort, localAddr, localPort string, plcAddrNetwork, plcAddrNode, plcAddrUnit, localAddrNetwork, localAddrNode, localAddrUnit byte) (Client, error) {
	if _type == "udp" {
		return NewUDPConn(remoteAddr, remotePort, localAddr, localPort, plcAddrNetwork, plcAddrNode, plcAddrUnit, localAddrNetwork, localAddrNode, localAddrUnit)
	}
	if _type == "tcp" {
		return NewTCPConn(remoteAddr, remotePort, plcAddrNetwork, plcAddrNode, plcAddrUnit, localAddrNetwork, localAddrNode, localAddrUnit)
	}
	return nil, fmt.Errorf("网络连接非tcp、udp")
}

func checkResponse(r *response, e error) error {
	if e != nil {
		return e
	}
	if r.endCode != EndCodeNormalCompletion {
		return fmt.Errorf("error reported by destination, end code 0x%x", r.endCode)
	}
	return nil
}

func checkIsWordMemoryArea(memoryArea byte) bool {
	if memoryArea == MemoryAreaDMWord ||
		memoryArea == MemoryAreaARWord ||
		memoryArea == MemoryAreaHRWord ||
		memoryArea == MemoryAreaCIOWord ||
		memoryArea == MemoryAreaWRWord {
		return true
	}
	return false
}

func checkIsBitMemoryArea(memoryArea byte) bool {
	if memoryArea == MemoryAreaDMBit ||
		memoryArea == MemoryAreaARBit ||
		memoryArea == MemoryAreaHRBit ||
		memoryArea == MemoryAreaCIOBit ||
		memoryArea == MemoryAreaWRBit {
		return true
	}
	return false
}
