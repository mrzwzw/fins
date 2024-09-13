package fins

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"time"
)

// TcpClient Omron FINS client
type TcpClient struct {
	conn *net.TCPConn
	resp []chan response
	sync.Mutex
	dst               finsAddress
	src               finsAddress
	sid               byte
	closed            bool
	responseTimeoutMs time.Duration
	byteOrder         binary.ByteOrder
}

// NewClient creates a new Omron FINS client
func NewTCPConn(ip, port string, plcAddrNetwork, plcAddrNode, plcAddrUnit, localAddrNetwork, localAddrNode, localAddrUnit byte) (*TcpClient, error) {
	c := new(TcpClient)
	c.dst = finsAddress{
		network: localAddrNetwork,
		node:    localAddrNode,
		unit:    localAddrUnit,
	}
	c.src = finsAddress{
		network: plcAddrNetwork,
		node:    plcAddrNode,
		unit:    plcAddrUnit,
	}
	c.responseTimeoutMs = DEFAULT_RESPONSE_TIMEOUT
	c.byteOrder = binary.BigEndian

	tcpAddr, err := net.ResolveTCPAddr("tcp", ip+":"+port)
	if err != nil {
		return nil, err
	}

	// raddr := &net.TCPConn{
	// 	IP:   net.ParseIP(remoteAddr),
	// 	Port: 5010,
	// }

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	if err = conn.SetKeepAlive(true); err != nil {
		return nil, err
	}

	// 发送握手命令
	// 46494E53 0000000C 00000000 00000000 000000C8

	c.conn = conn

	log.Println("plcAddrNode:", plcAddrNode)
	resv, err := c.ShakeHandsCommand(plcAddrNode)
	if err != nil {
		return nil, err
	}
	log.Println("握手信号回复为：", resv)
	c.resp = make([]chan response, 256) // storage for all responses, sid is byte - only 256 values
	go c.listenLoop()
	return c, nil
}

func (c *TcpClient) SetByteOrder(o binary.ByteOrder) {
	c.byteOrder = o
}

func (c *TcpClient) SetTimeoutMs(t uint) {
	c.responseTimeoutMs = time.Duration(t)
}

func (c *TcpClient) Close() {
	c.closed = true
	c.conn.Close()
}

// ReadWordsToUint16 读取plc连续数据(uint16)地址区域
func (c *TcpClient) ReadWordsToUint16(memoryArea byte, address uint16, readCount uint16) ([]uint16, error) {
	if checkIsWordMemoryArea(memoryArea) == false {
		return nil, IncompatibleMemoryAreaError{memoryArea}
	}
	command := readCommand(memAddr(memoryArea, address), readCount)
	r, e := c.sendCommand(command)
	e = checkResponse(r, e)
	if e != nil {
		return nil, e
	}

	data := make([]uint16, readCount, readCount)
	for i := 0; i < int(readCount); i++ {
		data[i] = c.byteOrder.Uint16(r.data[i*2 : i*2+2])
	}

	return data, nil
}

// ReadWordsToUint32 读取plc连续数据(uint32)地址区域
func (c *TcpClient) ReadWordsToUint32(memoryArea byte, address uint16, readCount uint16) ([]uint32, error) {
	if checkIsWordMemoryArea(memoryArea) == false {
		return nil, IncompatibleMemoryAreaError{memoryArea}
	}
	command := readCommand(memAddr(memoryArea, address), readCount)
	r, e := c.sendCommand(command)
	e = checkResponse(r, e)
	if e != nil {
		return nil, e
	}

	data := make([]uint32, readCount, readCount)
	for i := 0; i < int(readCount); i++ {
		data[i] = c.byteOrder.Uint32(r.data[i*4 : i*4+4])
	}

	return data, nil
}

//  读取plc连续数据(byte)地址区域
func (c *TcpClient) ReadBytes(memoryArea byte, address uint16, readCount uint16) ([]byte, error) {
	if checkIsWordMemoryArea(memoryArea) == false {
		return nil, IncompatibleMemoryAreaError{memoryArea}
	}
	command := readCommand(memAddr(memoryArea, address), readCount)
	r, e := c.sendCommand(command)
	e = checkResponse(r, e)
	if e != nil {
		return nil, e
	}

	return r.data, nil
}

// ReadString 读取plc连续数据(string)地址区域
func (c *TcpClient) ReadString(memoryArea byte, address uint16, readCount uint16) (string, error) {
	data, e := c.ReadBytes(memoryArea, address, readCount)
	if e != nil {
		return "", e
	}
	n := bytes.IndexByte(data, 0)
	if n == -1 {
		n = len(data)
	}
	return string(data[:n]), nil
}

// ReadBits 读取plc连续数据(bool)地址区域
func (c *TcpClient) ReadBits(memoryArea byte, address uint16, bitOffset byte, readCount uint16) ([]bool, error) {
	if checkIsBitMemoryArea(memoryArea) == false {
		return nil, IncompatibleMemoryAreaError{memoryArea}
	}
	command := readCommand(memAddrWithBitOffset(memoryArea, address, bitOffset), readCount)
	r, e := c.sendCommand(command)
	e = checkResponse(r, e)
	if e != nil {
		return nil, e
	}

	data := make([]bool, readCount, readCount)
	for i := 0; i < int(readCount); i++ {
		data[i] = r.data[i]&0x01 > 0
	}

	return data, nil
}

// ReadClock Reads the PLC clock
func (c *TcpClient) ReadClock() (*time.Time, error) {
	r, e := c.sendCommand(clockReadCommand())
	e = checkResponse(r, e)
	if e != nil {
		return nil, e
	}
	year, _ := decodeBCD(r.data[0:1])
	if year < 50 {
		year += 2000
	} else {
		year += 1900
	}
	month, _ := decodeBCD(r.data[1:2])
	day, _ := decodeBCD(r.data[2:3])
	hour, _ := decodeBCD(r.data[3:4])
	minute, _ := decodeBCD(r.data[4:5])
	second, _ := decodeBCD(r.data[5:6])

	t := time.Date(
		int(year), time.Month(month), int(day), int(hour), int(minute), int(second),
		0, // nanosecond
		time.Local,
	)
	return &t, nil
}

// WriteWords Writes words to the PLC data area
func (c *TcpClient) WriteWords(memoryArea byte, address uint16, data []uint16) error {
	if checkIsWordMemoryArea(memoryArea) == false {
		return IncompatibleMemoryAreaError{memoryArea}
	}
	l := uint16(len(data))
	bts := make([]byte, 2*l, 2*l)
	for i := 0; i < int(l); i++ {
		c.byteOrder.PutUint16(bts[i*2:i*2+2], data[i])
	}
	command := writeCommand(memAddr(memoryArea, address), l, bts)

	return checkResponse(c.sendCommand(command))
}

// WriteString Writes a string to the PLC data area
func (c *TcpClient) WriteString(memoryArea byte, address uint16, s string) error {
	if checkIsWordMemoryArea(memoryArea) == false {
		return IncompatibleMemoryAreaError{memoryArea}
	}
	bts := make([]byte, 2*len(s), 2*len(s))
	copy(bts, s)

	command := writeCommand(memAddr(memoryArea, address), uint16((len(s)+1)/2), bts) // TODO: test on real PLC

	return checkResponse(c.sendCommand(command))
}

// WriteBytes Writes bytes array to the PLC data area
func (c *TcpClient) WriteBytes(memoryArea byte, address uint16, b []byte) error {
	if checkIsWordMemoryArea(memoryArea) == false {
		return IncompatibleMemoryAreaError{memoryArea}
	}
	command := writeCommand(memAddr(memoryArea, address), uint16(len(b)), b)
	return checkResponse(c.sendCommand(command))
}

// WriteBits Writes bits to the PLC data area
func (c *TcpClient) WriteBits(memoryArea byte, address uint16, bitOffset byte, data []bool) error {
	if checkIsBitMemoryArea(memoryArea) == false {
		return IncompatibleMemoryAreaError{memoryArea}
	}
	l := uint16(len(data))
	bts := make([]byte, 0, l)
	var d byte
	for i := 0; i < int(l); i++ {
		if data[i] {
			d = 0x01
		} else {
			d = 0x00
		}
		bts = append(bts, d)
	}
	command := writeCommand(memAddrWithBitOffset(memoryArea, address, bitOffset), l, bts)

	return checkResponse(c.sendCommand(command))
}

// SetBit Sets a bit in the PLC data area
func (c *TcpClient) SetBit(memoryArea byte, address uint16, bitOffset byte) error {
	return c.bitTwiddle(memoryArea, address, bitOffset, 0x01)
}

// ResetBit Resets a bit in the PLC data area
func (c *TcpClient) ResetBit(memoryArea byte, address uint16, bitOffset byte) error {
	return c.bitTwiddle(memoryArea, address, bitOffset, 0x00)
}

// ToggleBit Toggles a bit in the PLC data area
func (c *TcpClient) ToggleBit(memoryArea byte, address uint16, bitOffset byte) error {
	b, e := c.ReadBits(memoryArea, address, bitOffset, 1)
	if e != nil {
		return e
	}
	var t byte
	if b[0] {
		t = 0x00
	} else {
		t = 0x01
	}
	return c.bitTwiddle(memoryArea, address, bitOffset, t)
}

func (c *TcpClient) bitTwiddle(memoryArea byte, address uint16, bitOffset byte, value byte) error {
	if checkIsBitMemoryArea(memoryArea) == false {
		return IncompatibleMemoryAreaError{memoryArea}
	}
	mem := memoryAddress{memoryArea, address, bitOffset}
	command := writeCommand(mem, 1, []byte{value})

	return checkResponse(c.sendCommand(command))
}

func (c *TcpClient) nextHeader() *Header {
	sid := c.incrementSid()
	header := defaultCommandHeader(c.src, c.dst, sid)
	return &header
}

func (c *TcpClient) incrementSid() byte {
	c.Lock() // thread-safe sid incrementation
	c.sid++
	sid := c.sid
	c.Unlock()
	c.resp[sid] = make(chan response) // clearing cell of storage for new response
	return sid
}

func (c *TcpClient) sendCommand(command []byte) (*response, error) {
	header := c.nextHeader()
	bts := encodeHeader(*header)
	bts = append(bts, command...)

	l := len(bts) + 8
	send := []byte{0x46, 0x49, 0x4E, 0x53}
	len := append([]byte{0x00, 0x00, 0x00}, byte(l))
	_command := []byte{0x00, 0x00, 0x00, 0x02}
	errorCode := []byte{0x00, 0x00, 0x00, 0x00}

	send = append(send, len...)
	send = append(send, _command...)
	send = append(send, errorCode...)
	send = append(send, bts...)

	log.Println("send", send)
	_, err := (*c.conn).Write(send)
	fmt.Println(bts)
	if err != nil {
		return nil, err
	}
	header.serviceID = 0
	log.Println("send serviceID:", header.serviceID)

	// if response timeout is zero, block indefinitely
	if c.responseTimeoutMs > 0 {
		select {
		case resp := <-c.resp[0]:
			log.Println("send serviceID:", header.serviceID)
			return &resp, nil
		case <-time.After(c.responseTimeoutMs * time.Millisecond):
			return nil, ResponseTimeoutError{c.responseTimeoutMs}
		}
	} else {
		log.Println("send serviceID:", header.serviceID)
		resp := <-c.resp[header.serviceID]
		return &resp, nil
	}
}

func (c *TcpClient) listenLoop() {
	for {
		buf := make([]byte, 2048)
		n, err := bufio.NewReader(c.conn).Read(buf)
		if err != nil {
			// do not complain when connection is closed by user
			if !c.closed {
				log.Fatal(err)
			}
			break
		}

		if n > 0 {
			ans := decodeResponse(buf[:n])
			log.Println("buf:", buf)
			log.Println("rec serviceID:", ans.header.serviceID)
			log.Println("ans:", ans)
			c.resp[ans.header.serviceID] <- ans
		} else {
			log.Println("cannot read response: ", buf)
		}
	}
}

func (c *TcpClient) ShakeHandsCommand(node byte) ([]byte, error) {
	header := []byte{0x46, 0x49, 0x4E, 0x53}
	len := []byte{0x00, 0x00, 0x00, 0x0C}
	command := []byte{0x00, 0x00, 0x00, 0x00}
	errorCode := []byte{0x00, 0x00, 0x00, 0x00}
	_node := append([]byte{0x00, 0x00, 0x00}, node)

	header = append(header, len...)
	header = append(header, command...)
	header = append(header, errorCode...)
	header = append(header, _node...)

	_, err := c.conn.Write(header)
	if err != nil {
		return nil, err
	}

	// 长度24，握手信号回复数据长度固定
	buff := make([]byte, 24)

	_, err = io.ReadFull(c.conn, buff)
	if err != nil {
		return nil, fmt.Errorf("got % x, %w", buff, err)
	}

	if reflect.DeepEqual(buff[4:8], []byte{0x00, 0x00, 0x00, 0x00}) {
		log.Println("握手信号的error code为：", buff[4:8])
	}

	return buff[0:24], nil
}

func (c *TcpClient) encodeHeader(h Header) []byte {
	var icf byte
	icf = 1 << icfBridgesBit
	if h.responseRequired == false {
		icf |= 1 << icfResponseRequiredBit
	}
	if h.messageType == MessageTypeResponse {
		icf |= 1 << icfMessageTypeBit
	}
	bytes := []byte{
		icf, 0x00, h.gatewayCount,
		h.dst.network, h.dst.node, h.dst.unit,
		h.src.network, h.src.node, h.src.unit,
		0,
	}
	return bytes
}
