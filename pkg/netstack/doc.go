// Package netstack provides a complete TCP/IP network stack implementation
// including Ethernet, IP, TCP, UDP, ICMP, and socket abstractions.
//
// This implementation follows RFC specifications and provides a BSD-socket-like
// interface for network communication.
//
// Layer Structure:
//   - Layer 2 (Link): Ethernet frames, ARP
//   - Layer 3 (Network): IPv4, IPv6, ICMP, fragmentation
//   - Layer 4 (Transport): TCP, UDP
//   - Socket API: BSD-socket-like interface
//
// Example usage:
//
//	iface := &netstack.Interface{
//	    Name: "eth0",
//	    MAC:  net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
//	    IP:   net.ParseIP("192.168.1.100"),
//	    MTU:  1500,
//	}
//	tcpConn, err := tcp.Dial("192.168.1.1", 80)
package netstack
