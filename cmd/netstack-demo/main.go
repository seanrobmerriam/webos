// netstack-demo demonstrates the network protocol stack implementation.
//
// This demo shows:
// - Ethernet frame parsing and creation
// - ARP protocol for address resolution
// - IPv4/IPv6 packet handling
// - ICMP echo requests (ping)
// - TCP connection management
// - UDP datagram handling
// - Routing table operations
// - Socket API usage
package main

import (
	"fmt"
	network "net"

	"webos/pkg/netstack/ethernet"
	ipv4 "webos/pkg/netstack/ip"
	"webos/pkg/netstack/route"
	"webos/pkg/netstack/socket"
	"webos/pkg/netstack/tcp"
	"webos/pkg/netstack/udp"
)

func main() {
	fmt.Println("=== WebOS Network Protocol Stack Demo ===")
	fmt.Println()

	demoEthernet()
	demoARP()
	demoIPv4()
	demoICMP()
	demoTCP()
	demoUDP()
	demoRouting()
	demoSockets()

	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}

func demoEthernet() {
	fmt.Println("--- Ethernet Layer Demo ---")

	// Create Ethernet frame
	dstMAC := network.HardwareAddr{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	srcMAC := network.HardwareAddr{0x00, 0x0C, 0x29, 0xAB, 0xCD, 0xEF}
	payload := []byte("Hello, Ethernet!")

	frame := ethernet.NewFrame(dstMAC, srcMAC, 0x0800, payload) // 0x0800 = IPv4

	fmt.Printf("Frame: Src=%s, Dst=%s, Type=0x%04X, Payload=%q\n",
		srcMAC, dstMAC, 0x0800, string(payload))

	serialized := frame.Serialize()
	fmt.Printf("Serialized length: %d bytes\n", len(serialized))

	// Parse the frame
	parsed, err := ethernet.ParseFrame(serialized)
	if err != nil {
		fmt.Printf("Error parsing frame: %v\n", err)
		return
	}

	fmt.Printf("Parsed: Src=%s, Dst=%s, Type=0x%04X\n",
		parsed.SrcMAC, parsed.DstMAC, parsed.EtherType)
	fmt.Println()
}

func demoARP() {
	fmt.Println("--- ARP Protocol Demo ---")

	// Create ARP request
	srcMAC := network.HardwareAddr{0x00, 0x0C, 0x29, 0xAB, 0xCD, 0xEF}
	senderIP := network.ParseIP("192.168.1.100")
	targetIP := network.ParseIP("192.168.1.1")

	arpReq := ethernet.NewARPRequest(srcMAC, senderIP, targetIP)
	fmt.Printf("ARP Request: Who has %s? Tell %s\n", targetIP, senderIP)

	serialized := arpReq.Serialize()
	fmt.Printf("Serialized length: %d bytes\n", len(serialized))

	// ARP table
	table := ethernet.NewARPTable()
	table.Set(targetIP, network.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})

	mac, _ := table.Lookup(targetIP)
	fmt.Printf("ARP Cache: %s -> %s\n", targetIP, mac)
	fmt.Println()
}

func demoIPv4() {
	fmt.Println("--- IPv4 Layer Demo ---")

	srcIP := network.ParseIP("192.168.1.100")
	dstIP := network.ParseIP("8.8.8.8")
	payload := []byte("Hello, IP!")

	datagram := ipv4.NewDatagram(srcIP, dstIP, ipv4.ProtocolUDP, payload)
	fmt.Printf("Datagram: %s -> %s, Protocol=%d, Payload=%q\n",
		srcIP, dstIP, ipv4.ProtocolUDP, string(payload))

	serialized := datagram.Serialize()
	fmt.Printf("Serialized length: %d bytes\n", len(serialized))

	// Fragmentation demo
	largePayload := make([]byte, 3000)
	largeDatagram := ipv4.NewDatagram(srcIP, dstIP, ipv4.ProtocolTCP, largePayload)

	fragments, err := ipv4.Fragment(largeDatagram, 1500)
	if err != nil {
		fmt.Printf("Fragmentation error: %v\n", err)
	} else {
		fmt.Printf("Fragmented into %d packets\n", len(fragments))
		for i, frag := range fragments {
			fmt.Printf("  Fragment %d: %d bytes (offset=%d, more=%v)\n",
				i, len(frag.Payload), frag.Header.FragOffset, frag.Header.Flags&0x1 != 0)
		}
	}
	fmt.Println()
}

func demoICMP() {
	fmt.Println("--- ICMP Protocol Demo ---")

	// Create ping request
	ping := ipv4.NewEchoRequest(0x1234, 1, []byte("ping data"))
	fmt.Printf("ICMP Echo Request: ID=0x%04X, Seq=%d, Data=%q\n",
		ping.Header.ID, ping.Header.Seq, string(ping.Payload))

	serialized := ping.Serialize()
	fmt.Printf("Serialized length: %d bytes\n", len(serialized))

	// Parse back
	parsed, err := ipv4.ParseMessage(serialized)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Printf("Parsed: Type=%d, IsEchoRequest=%v\n", parsed.Header.Type, parsed.IsEchoRequest())
	}
	fmt.Println()
}

func demoTCP() {
	fmt.Println("--- TCP Layer Demo ---")

	srcPort := uint16(12345)
	dstPort := uint16(80)
	srcIP := network.ParseIP("192.168.1.100")
	dstIP := network.ParseIP("8.8.8.8")

	// Create TCP connection
	connID := tcp.ConnectionID{
		SrcIP:   srcIP,
		SrcPort: srcPort,
		DstIP:   dstIP,
		DstPort: dstPort,
	}

	conn := tcp.NewConnection(connID, nil, nil)
	fmt.Printf("Connection: %s -> %s:%d\n", srcIP, dstIP, dstPort)
	fmt.Printf("Initial Sequence Number: %d\n", conn.ISS)
	fmt.Printf("State: %d (Closed=%d, Established=%d)\n",
		conn.State, tcp.StateClosed, tcp.StateEstablished)

	// Create SYN segment
	synSeg := tcp.NewSegment(srcPort, dstPort, srcIP, dstIP,
		tcp.FlagSYN, conn.ISS, 0, nil)
	fmt.Printf("SYN Segment: Seq=%d, Flags=SYN\n", synSeg.Header.SeqNum)

	// Simulate connection establishment
	conn.State = tcp.StateSynSent
	conn.SND = conn.ISS + 1
	conn.SNDUNA = conn.ISS + 1
	fmt.Printf("Connection state: %d (SynSent)\n", conn.State)
	fmt.Println()
}

func demoUDP() {
	fmt.Println("--- UDP Layer Demo ---")

	srcIP := network.ParseIP("192.168.1.100")
	dstIP := network.ParseIP("8.8.8.8")

	// Create UDP datagram
	dg := udp.NewDatagram(12345, 53, srcIP, dstIP, []byte("DNS query"))
	fmt.Printf("UDP Datagram: %s:%d -> %s:%d, Payload=%q\n",
		srcIP, 12345, dstIP, 53, string(dg.Payload))

	serialized := dg.Serialize()
	fmt.Printf("Serialized length: %d bytes\n", len(serialized))

	// Create UDP socket
	udpSock := udp.NewSocket(8080, srcIP)
	fmt.Printf("UDP Socket: listening on %s:%d\n", srcIP, 8080)

	// Send/receive demo
	dg2 := udp.NewDatagram(12345, 8080, srcIP, dstIP, []byte("response"))
	udpSock.Send(dg2)

	received, _ := udpSock.Receive()
	fmt.Printf("Received: %q\n", string(received.Payload))
	fmt.Println()
}

func demoRouting() {
	fmt.Println("--- Routing Table Demo ---")

	rt := route.NewRouteTable()

	// Add local route
	rt.AddLocalRoute(network.ParseIP("192.168.1.100"), "eth0")

	// Add specific route
	_, net1, _ := network.ParseCIDR("10.0.0.0/8")
	rt.AddRoute(route.Route{
		Dest:      *net1,
		Gateway:   network.ParseIP("192.168.1.1"),
		Interface: "eth0",
		Valid:     true,
	})

	// Set default route
	rt.SetDefaultRoute(network.ParseIP("192.168.1.1"), "eth0")

	stats := rt.Stats()
	fmt.Printf("Routing Table: %d routes (%d valid, %d default)\n",
		stats.TotalRoutes, stats.ValidRoutes, stats.DefaultRoutes)

	// Route lookups
	lookups := []string{"192.168.1.1", "10.0.0.1", "8.8.8.8", "172.16.0.1"}
	for _, ipStr := range lookups {
		ip := network.ParseIP(ipStr)
		route := rt.Lookup(ip)
		if route != nil {
			gw := "direct"
			if route.Gateway != nil {
				gw = route.Gateway.String()
			}
			fmt.Printf("  %s -> %s (interface=%s)\n", ipStr, gw, route.Interface)
		}
	}
	fmt.Println()
}

func demoSockets() {
	fmt.Println("--- Socket API Demo ---")

	rt := route.NewRouteTable()
	sm := socket.NewSocketManager()

	// Create TCP socket
	tcpSock := socket.NewTCPSocket(rt)
	fmt.Printf("TCP Socket: ID=%d, Protocol=TCP, Status=%d\n",
		tcpSock.ID, tcpSock.Status)

	// Listen
	tcpSock.Listen(10)
	fmt.Printf("TCP Socket listening, Status=%d\n", tcpSock.Status)

	// Create UDP socket
	udpSock := socket.NewUDPSocket(8080, network.ParseIP("192.168.1.100"), rt)
	fmt.Printf("UDP Socket: ID=%d, Local=%s\n",
		udpSock.ID, udpSock.LocalAddr())

	// Manage sockets
	sm.Add(tcpSock)
	sm.Add(udpSock)
	fmt.Printf("Socket Manager: %d sockets\n", len(sm.List()))

	// Close sockets
	tcpSock.Close()
	udpSock.Close()
	fmt.Println("Sockets closed")
}
