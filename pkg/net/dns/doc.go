// Package dns provides a complete DNS client implementation from scratch.
// It includes DNS message parsing, recursive resolution, caching, and hosts file support.
//
// This implementation uses only Go standard library and supports:
//   - DNS query/response parsing and generation
//   - Common record types: A, AAAA, CNAME, MX, TXT, NS
//   - TTL-based caching with automatic expiration
//   - Recursive resolution through upstream DNS servers
//   - Hosts file support for local name resolution
//   - DNS-over-TLS for secure queries
//
// Example usage:
//
//	cache := dns.NewCache()
//	resolver := dns.NewResolver(dns.WithCache(cache))
//	ip, err := resolver LookupHost("example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("IP address:", ip)
package dns
