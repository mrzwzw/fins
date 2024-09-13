package fins

import "net"

// finsAddress A FINS device address
type finsAddress struct {
	network byte
	node    byte
	unit    byte
}

type PlcConn interface {
	// SendCmd(message McMessage, retSize int, debug bool) ([]byte, error)
	// GetCPUInfo() (string, error)
	// options() *plcOptions
	// Close() error
}

type tcpConn struct {
	net.Conn
	// option *plcOptions
}

type udpConn struct {
	*net.UDPConn
}

// Address A full device address
type Address struct {
	finsAddress finsAddress
	udpAddress  *net.UDPAddr
}
// type Address struct {
// 	finsAddress finsAddress
// 	udpAddress  *net.TCPConn
// }

// func NewAddress(ip string, port int, network, node, unit byte) Address {
// 	return Address{
// 		// udpAddress: &net.UDPAddr{
// 		// 	IP:   net.ParseIP(ip),
// 		// 	Port: port,
// 		// },
// 		finsAddress: finsAddress{
// 			network: network,
// 			node:    node,
// 			unit:    unit,
// 		},
// 	}
// }
