package socket_test

import (
	network "net"
	"testing"

	"webos/pkg/netstack/route"
	"webos/pkg/netstack/socket"
)

func TestNewTCPSocket(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)

	if s.ID == 0 {
		t.Error("Socket ID should not be zero")
	}
	if s.Protocol != socket.ProtocolTCP {
		t.Errorf("Protocol = %d, want %d (TCP)", s.Protocol, socket.ProtocolTCP)
	}
	if s.Type != socket.SocketStream {
		t.Errorf("Type = %d, want %d (Stream)", s.Type, socket.SocketStream)
	}
	if s.Status != socket.StatusUnconnected {
		t.Errorf("Status = %d, want %d (Unconnected)", s.Status, socket.StatusUnconnected)
	}
}

func TestNewUDPSocket(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewUDPSocket(53, network.ParseIP("192.168.1.100"), rt)

	if s.Protocol != socket.ProtocolUDP {
		t.Errorf("Protocol = %d, want %d (UDP)", s.Protocol, socket.ProtocolUDP)
	}
	if s.Type != socket.SocketDgram {
		t.Errorf("Type = %d, want %d (Dgram)", s.Type, socket.SocketDgram)
	}
	if s.Status != socket.StatusConnected {
		t.Errorf("Status = %d, want %d (Connected)", s.Status, socket.StatusConnected)
	}

	localAddr := s.LocalAddr().(*network.UDPAddr)
	if localAddr.Port != 53 {
		t.Errorf("Local port = %d, want 53", localAddr.Port)
	}
}

func TestListen(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)

	err := s.Listen(10)
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	if s.Status != socket.StatusListening {
		t.Errorf("Status = %d, want %d (Listening)", s.Status, socket.StatusListening)
	}
}

func TestListenOnConnected(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)
	s.Status = socket.StatusConnected

	err := s.Listen(10)
	if err == nil {
		t.Error("Listen on connected socket should fail")
	}
}

func TestAccept(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)
	s.Listen(10)

	conn, err := s.Accept()
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}

	if conn.Status != socket.StatusConnected {
		t.Errorf("Accepted socket status = %d, want %d (Connected)", conn.Status, socket.StatusConnected)
	}
}

func TestAcceptOnUnconnected(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)

	_, err := s.Accept()
	if err == nil {
		t.Error("Accept on unconnected socket should fail")
	}
}

func TestClose(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)

	err := s.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if s.Status != socket.StatusClosed {
		t.Errorf("Status = %d, want %d (Closed)", s.Status, socket.StatusClosed)
	}
}

func TestCloseAlreadyClosed(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)
	s.Close()

	err := s.Close()
	if err == nil {
		t.Error("Close on already closed socket should fail")
	}
}

func TestSendUDP(t *testing.T) {
	rt := route.NewRouteTable()
	s := socket.NewUDPSocket(0, network.ParseIP("0.0.0.0"), rt)

	n, err := s.Send([]byte("test"))
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if n != 4 {
		t.Errorf("Sent = %d, want 4", n)
	}
}

func TestSocketManager(t *testing.T) {
	sm := socket.NewSocketManager()
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)

	sm.Add(s)
	if _, ok := sm.Get(s.ID); !ok {
		t.Error("Socket should be in manager")
	}

	sm.Remove(s.ID)
	if _, ok := sm.Get(s.ID); ok {
		t.Error("Socket should not be in manager after removal")
	}
}

func TestSocketManagerList(t *testing.T) {
	sm := socket.NewSocketManager()
	rt := route.NewRouteTable()

	for i := 0; i < 5; i++ {
		sm.Add(socket.NewTCPSocket(rt))
	}

	ids := sm.List()
	if len(ids) != 5 {
		t.Errorf("List returned %d sockets, want 5", len(ids))
	}
}

func TestSocketManagerGet(t *testing.T) {
	sm := socket.NewSocketManager()
	rt := route.NewRouteTable()
	s := socket.NewTCPSocket(rt)

	sm.Add(s)

	retrieved, ok := sm.Get(s.ID)
	if !ok {
		t.Error("Get should return true for existing socket")
	}
	if retrieved.ID != s.ID {
		t.Errorf("Retrieved socket ID = %d, want %d", retrieved.ID, s.ID)
	}

	_, ok = sm.Get(99999)
	if ok {
		t.Error("Get should return false for non-existing socket")
	}
}
