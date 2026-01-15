package route

import (
	"fmt"
	network "net"
	"sync"
)

// Route represents a network route.
type Route struct {
	Dest      network.IPNet // Destination network
	Gateway   network.IP    // Next hop gateway (nil for direct)
	Interface string        // Output interface name
	Metric    int           // Route metric
	Valid     bool          // Route is valid
	Preferred bool          // Route is preferred
}

// RouteTable manages network routes.
type RouteTable struct {
	mu     sync.RWMutex
	routes []*Route
}

// NewRouteTable creates a new routing table.
func NewRouteTable() *RouteTable {
	return &RouteTable{
		routes: make([]*Route, 0),
	}
}

// AddRoute adds a route to the routing table.
func (rt *RouteTable) AddRoute(route Route) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Validate route
	if route.Dest.IP == nil {
		return fmt.Errorf("invalid destination")
	}

	// Check for duplicate routes
	for _, r := range rt.routes {
		if r.Dest.String() == route.Dest.String() {
			return fmt.Errorf("route already exists")
		}
	}

	rt.routes = append(rt.routes, &route)
	return nil
}

// RemoveRoute removes a route from the routing table.
func (rt *RouteTable) RemoveRoute(dest network.IPNet) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for i, r := range rt.routes {
		if r.Dest.String() == dest.String() {
			rt.routes = append(rt.routes[:i], rt.routes[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("route not found")
}

// Lookup finds the best route for a destination IP.
func (rt *RouteTable) Lookup(dest network.IP) *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var best *Route
	bestMaskLen := -1

	for _, r := range rt.routes {
		if !r.Valid {
			continue
		}

		// Check if destination is in this route's network
		if !r.Dest.Contains(dest) {
			continue
		}

		// Calculate prefix length
		maskLen, _ := r.Dest.Mask.Size()
		if maskLen > bestMaskLen {
			best = r
			bestMaskLen = maskLen
		}
	}

	return best
}

// GetAllRoutes returns all routes in the table.
func (rt *RouteTable) GetAllRoutes() []Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	routes := make([]Route, len(rt.routes))
	for i, r := range rt.routes {
		routes[i] = *r
	}
	return routes
}

// SetDefaultRoute sets the default route (0.0.0.0/0).
func (rt *RouteTable) SetDefaultRoute(gateway network.IP, iface string) error {
	_, defaultRoute, _ := network.ParseCIDR("0.0.0.0/0")

	return rt.AddRoute(Route{
		Dest:      *defaultRoute,
		Gateway:   gateway,
		Interface: iface,
		Metric:    0,
		Valid:     true,
		Preferred: true,
	})
}

// AddLocalRoute adds a route for a local network.
func (rt *RouteTable) AddLocalRoute(localIP network.IP, iface string) error {
	// Get the network address
	mask := network.CIDRMask(24, 32)
	localNet := network.IPNet{
		IP:   localIP.Mask(mask),
		Mask: mask,
	}

	return rt.AddRoute(Route{
		Dest:      localNet,
		Gateway:   nil,
		Interface: iface,
		Metric:    0,
		Valid:     true,
		Preferred: true,
	})
}

// RouteMetrics provides routing metrics.
type RouteMetrics struct {
	RoutesAdded   int
	RoutesRemoved int
	Lookups       int
	CacheHits     int
}

// Stats returns routing table statistics.
type Stats struct {
	TotalRoutes   int
	ValidRoutes   int
	DefaultRoutes int
}

// Stats returns routing table statistics.
func (rt *RouteTable) Stats() Stats {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	stats := Stats{
		TotalRoutes: len(rt.routes),
	}

	for _, r := range rt.routes {
		if r.Valid {
			stats.ValidRoutes++
		}
		maskLen, _ := r.Dest.Mask.Size()
		if maskLen == 0 {
			stats.DefaultRoutes++
		}
	}

	return stats
}
