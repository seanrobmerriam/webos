package route_test

import (
	network "net"
	"testing"

	"webos/pkg/netstack/route"
)

func TestAddRoute(t *testing.T) {
	rt := route.NewRouteTable()

	_, net1, _ := network.ParseCIDR("192.168.1.0/24")
	err := rt.AddRoute(route.Route{
		Dest:      *net1,
		Gateway:   network.ParseIP("192.168.1.1"),
		Interface: "eth0",
		Metric:    10,
		Valid:     true,
	})

	if err != nil {
		t.Fatalf("AddRoute failed: %v", err)
	}

	stats := rt.Stats()
	if stats.TotalRoutes != 1 {
		t.Errorf("TotalRoutes = %d, want 1", stats.TotalRoutes)
	}
	if stats.ValidRoutes != 1 {
		t.Errorf("ValidRoutes = %d, want 1", stats.ValidRoutes)
	}
}

func TestAddDuplicateRoute(t *testing.T) {
	rt := route.NewRouteTable()

	_, net1, _ := network.ParseCIDR("192.168.1.0/24")
	gateway := network.ParseIP("192.168.1.1")

	err := rt.AddRoute(route.Route{
		Dest:      *net1,
		Gateway:   gateway,
		Interface: "eth0",
	})
	if err != nil {
		t.Fatalf("First AddRoute failed: %v", err)
	}

	// Try to add the same route again
	err = rt.AddRoute(route.Route{
		Dest:      *net1,
		Gateway:   gateway,
		Interface: "eth0",
	})
	if err == nil {
		t.Error("Should have failed to add duplicate route")
	}
}

func TestRemoveRoute(t *testing.T) {
	rt := route.NewRouteTable()

	_, net1, _ := network.ParseCIDR("192.168.1.0/24")
	rt.AddRoute(route.Route{
		Dest:      *net1,
		Gateway:   network.ParseIP("192.168.1.1"),
		Interface: "eth0",
	})

	err := rt.RemoveRoute(*net1)
	if err != nil {
		t.Fatalf("RemoveRoute failed: %v", err)
	}

	stats := rt.Stats()
	if stats.TotalRoutes != 0 {
		t.Errorf("TotalRoutes = %d, want 0", stats.TotalRoutes)
	}
}

func TestRemoveNonExistentRoute(t *testing.T) {
	rt := route.NewRouteTable()

	_, net1, _ := network.ParseCIDR("192.168.1.0/24")
	err := rt.RemoveRoute(*net1)
	if err == nil {
		t.Error("Should have failed to remove non-existent route")
	}
}

func TestLookup(t *testing.T) {
	rt := route.NewRouteTable()

	// Add local network route
	_, net1, _ := network.ParseCIDR("192.168.1.0/24")
	rt.AddRoute(route.Route{
		Dest:      *net1,
		Gateway:   nil, // Local
		Interface: "eth0",
		Valid:     true,
	})

	// Add default route
	_, defaultNet, _ := network.ParseCIDR("0.0.0.0/0")
	rt.AddRoute(route.Route{
		Dest:      *defaultNet,
		Gateway:   network.ParseIP("192.168.1.1"),
		Interface: "eth0",
		Valid:     true,
	})

	// Lookup local IP should return local route
	r := rt.Lookup(network.ParseIP("192.168.1.100"))
	if r == nil {
		t.Fatal("Lookup returned nil for local IP")
	}
	if r.Gateway != nil {
		t.Errorf("Expected local route (no gateway), got gateway %v", r.Gateway)
	}

	// Lookup remote IP should return default route
	r = rt.Lookup(network.ParseIP("8.8.8.8"))
	if r == nil {
		t.Fatal("Lookup returned nil for remote IP")
	}
	if r.Gateway == nil {
		t.Error("Expected default route with gateway")
	}
}

func TestLookupLongestPrefix(t *testing.T) {
	rt := route.NewRouteTable()

	// Add /16 route
	_, net16, _ := network.ParseCIDR("192.168.0.0/16")
	rt.AddRoute(route.Route{
		Dest:      *net16,
		Gateway:   network.ParseIP("192.168.1.1"),
		Interface: "eth0",
		Valid:     true,
	})

	// Add /24 route (more specific)
	_, net24, _ := network.ParseCIDR("192.168.1.0/24")
	rt.AddRoute(route.Route{
		Dest:      *net24,
		Gateway:   network.ParseIP("192.168.1.2"),
		Interface: "eth0",
		Valid:     true,
	})

	// Lookup should return /24 route (longer prefix)
	r := rt.Lookup(network.ParseIP("192.168.1.100"))
	if r == nil {
		t.Fatal("Lookup returned nil")
	}
	if r.Gateway.String() != "192.168.1.2" {
		t.Errorf("Expected /24 route gateway, got %v", r.Gateway)
	}
}

func TestGetAllRoutes(t *testing.T) {
	rt := route.NewRouteTable()

	_, net1, _ := network.ParseCIDR("192.168.1.0/24")
	rt.AddRoute(route.Route{Dest: *net1})

	_, net2, _ := network.ParseCIDR("10.0.0.0/8")
	rt.AddRoute(route.Route{Dest: *net2})

	routes := rt.GetAllRoutes()
	if len(routes) != 2 {
		t.Errorf("GetAllRoutes returned %d routes, want 2", len(routes))
	}
}

func TestSetDefaultRoute(t *testing.T) {
	rt := route.NewRouteTable()

	err := rt.SetDefaultRoute(network.ParseIP("192.168.1.1"), "eth0")
	if err != nil {
		t.Fatalf("SetDefaultRoute failed: %v", err)
	}

	stats := rt.Stats()
	if stats.DefaultRoutes != 1 {
		t.Errorf("DefaultRoutes = %d, want 1", stats.DefaultRoutes)
	}

	r := rt.Lookup(network.ParseIP("8.8.8.8"))
	if r == nil {
		t.Fatal("Default route lookup failed")
	}
}

func TestAddLocalRoute(t *testing.T) {
	rt := route.NewRouteTable()

	localIP := network.ParseIP("192.168.1.100")
	err := rt.AddLocalRoute(localIP, "eth0")
	if err != nil {
		t.Fatalf("AddLocalRoute failed: %v", err)
	}

	r := rt.Lookup(network.ParseIP("192.168.1.50"))
	if r == nil {
		t.Fatal("Local route lookup failed")
	}
	if r.Gateway != nil {
		t.Error("Local route should have no gateway")
	}
}

func TestInvalidRoute(t *testing.T) {
	rt := route.NewRouteTable()

	err := rt.AddRoute(route.Route{
		Dest:      network.IPNet{},
		Interface: "eth0",
	})
	if err == nil {
		t.Error("Should have failed to add invalid route")
	}
}
