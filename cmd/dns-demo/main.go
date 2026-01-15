// DNS Demo - demonstrates the DNS resolver package
//
// This demo shows how to use the DNS resolver to perform lookups,
// cache DNS records, and work with hosts files.
//
// Usage:
//
//	dns-demo -host example.com              # Look up A records
//	dns-demo -host example.com -ipv6        # Look up AAAA records
//	dns-demo -host example.com -mx          # Look up MX records
//	dns-demo -host example.com -all         # Look up all record types
//	dns-demo -host example.com -v           # Verbose output with cache stats
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"webos/pkg/net/dns"
)

var (
	host     = flag.String("host", "", "Hostname to look up")
	ipv6     = flag.Bool("ipv6", false, "Look up AAAA (IPv6) records")
	mx       = flag.Bool("mx", false, "Look up MX records")
	txt      = flag.Bool("txt", false, "Look up TXT records")
	ns       = flag.Bool("ns", false, "Look up NS records")
	cname    = flag.Bool("cname", false, "Look up CNAME record")
	all      = flag.Bool("all", false, "Look up all record types")
	server   = flag.String("server", "", "DNS server to use (default: system default)")
	cacheTTL = flag.Duration("cache-ttl", 5*time.Minute, "Cache TTL")
	verbose  = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	if *host == "" {
		fmt.Println("Usage: dns-demo -host <hostname> [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Create cache and resolver with options
	cache := dns.NewCacheWithTTL(*cacheTTL)
	opts := []dns.ResolverOption{
		dns.WithCache(cache),
	}

	if *server != "" {
		opts = append(opts, dns.WithServers([]string{*server}))
	}

	resolver := dns.NewResolver(opts...)

	ctx := context.Background()

	if *verbose {
		fmt.Printf("Looking up: %s\n", *host)
		fmt.Printf("Cache TTL: %v\n", *cacheTTL)
	}

	// Perform lookups based on flags
	if *all {
		if err := lookupAll(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *ipv6 {
		if err := lookupAAAA(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *mx {
		if err := lookupMX(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *txt {
		if err := lookupTXT(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *ns {
		if err := lookupNS(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if *cname {
		if err := lookupCNAME(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := lookupA(ctx, resolver, *host); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	// Show cache stats
	if *verbose {
		stats := cache.Stats()
		fmt.Printf("\nCache stats:\n")
		fmt.Printf("  Total entries: %d\n", stats.Total)
		fmt.Printf("  A records: %d\n", stats.A)
		fmt.Printf("  AAAA records: %d\n", stats.AAAA)
	}
}

// lookupA performs an A record lookup
func lookupA(ctx context.Context, r *dns.Resolver, host string) error {
	ips, err := r.LookupHostContext(ctx, host)
	if err != nil {
		return fmt.Errorf("A lookup failed: %w", err)
	}

	fmt.Printf("A records for %s:\n", host)
	for _, ip := range ips {
		fmt.Printf("  %s\n", ip.String())
	}
	return nil
}

// lookupAAAA performs an AAAA record lookup
func lookupAAAA(ctx context.Context, r *dns.Resolver, host string) error {
	ips, err := r.LookupAAAAContext(ctx, host)
	if err != nil {
		return fmt.Errorf("AAAA lookup failed: %w", err)
	}

	fmt.Printf("AAAA records for %s:\n", host)
	for _, ip := range ips {
		fmt.Printf("  %s\n", ip.String())
	}
	return nil
}

// lookupMX performs an MX record lookup
func lookupMX(ctx context.Context, r *dns.Resolver, host string) error {
	mxRecords, err := r.LookupMX(host)
	if err != nil {
		return fmt.Errorf("MX lookup failed: %w", err)
	}

	fmt.Printf("MX records for %s:\n", host)
	for _, mx := range mxRecords {
		fmt.Printf("  %d %s\n", mx.Priority, mx.Host)
	}
	return nil
}

// lookupTXT performs a TXT record lookup
func lookupTXT(ctx context.Context, r *dns.Resolver, host string) error {
	txtRecords, err := r.LookupTXT(host)
	if err != nil {
		return fmt.Errorf("TXT lookup failed: %w", err)
	}

	fmt.Printf("TXT records for %s:\n", host)
	for _, txt := range txtRecords {
		fmt.Printf("  \"%s\"\n", txt)
	}
	return nil
}

// lookupNS performs an NS record lookup
func lookupNS(ctx context.Context, r *dns.Resolver, host string) error {
	nsRecords, err := r.LookupNS(host)
	if err != nil {
		return fmt.Errorf("NS lookup failed: %w", err)
	}

	fmt.Printf("NS records for %s:\n", host)
	for _, ns := range nsRecords {
		fmt.Printf("  %s\n", ns)
	}
	return nil
}

// lookupCNAME performs a CNAME record lookup
func lookupCNAME(ctx context.Context, r *dns.Resolver, host string) error {
	cname, err := r.LookupCNAME(host)
	if err != nil {
		return fmt.Errorf("CNAME lookup failed: %w", err)
	}

	fmt.Printf("CNAME for %s:\n", host)
	fmt.Printf("  %s\n", cname)
	return nil
}

// lookupAll performs all lookups
func lookupAll(ctx context.Context, r *dns.Resolver, host string) error {
	// Show canonical name first
	cname, err := r.LookupCNAME(host)
	if err == nil && cname != "" {
		fmt.Printf("Canonical name: %s\n", cname)
	}

	// A records
	ips, err := r.LookupHostContext(ctx, host)
	if err == nil && len(ips) > 0 {
		fmt.Printf("A records:\n")
		for _, ip := range ips {
			fmt.Printf("  %s\n", ip.String())
		}
	}

	// AAAA records
	ips6, err := r.LookupAAAAContext(ctx, host)
	if err == nil && len(ips6) > 0 {
		fmt.Printf("AAAA records:\n")
		for _, ip := range ips6 {
			fmt.Printf("  %s\n", ip.String())
		}
	}

	// MX records
	mx, err := r.LookupMX(host)
	if err == nil && len(mx) > 0 {
		fmt.Printf("MX records:\n")
		for _, m := range mx {
			fmt.Printf("  %d %s\n", m.Priority, m.Host)
		}
	}

	// TXT records
	txt, err := r.LookupTXT(host)
	if err == nil && len(txt) > 0 {
		fmt.Printf("TXT records:\n")
		for _, t := range txt {
			fmt.Printf("  \"%s\"\n", t)
		}
	}

	// NS records
	ns, err := r.LookupNS(host)
	if err == nil && len(ns) > 0 {
		fmt.Printf("NS records:\n")
		for _, n := range ns {
			fmt.Printf("  %s\n", n)
		}
	}

	return nil
}
