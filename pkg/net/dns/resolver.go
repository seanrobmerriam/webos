package dns

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// ResolverOption configures a DNS resolver
type ResolverOption func(*Resolver)

// WithServers sets the DNS servers to use
func WithServers(servers []string) ResolverOption {
	return func(r *Resolver) {
		r.servers = servers
	}
}

// WithTimeout sets the query timeout
func WithTimeout(timeout time.Duration) ResolverOption {
	return func(r *Resolver) {
		r.timeout = timeout
	}
}

// WithCache sets the DNS cache to use
func WithCache(cache *Cache) ResolverOption {
	return func(r *Resolver) {
		r.cache = cache
	}
}

// WithSearchList sets the search list domains
func WithSearchList(searchList []string) ResolverOption {
	return func(r *Resolver) {
		r.searchList = searchList
	}
}

// WithTLS sets DNS-over-TLS configuration
func WithTLS(tlsConfig *tls.Config) ResolverOption {
	return func(r *Resolver) {
		r.tlsConfig = tlsConfig
	}
}

// WithHostsFile sets a custom hosts file path
func WithHostsFile(path string) ResolverOption {
	return func(r *Resolver) {
		r.hostsFile = path
	}
}

// Resolver performs DNS lookups
type Resolver struct {
	servers    []string
	cache      *Cache
	timeout    time.Duration
	searchList []string
	tlsConfig  *tls.Config
	hostsFile  string
	hosts      map[string][]net.IP // In-memory hosts file
	hostsMu    bool                // Whether hosts map is populated
	parser     *Parser
}

// NewResolver creates a new DNS resolver
func NewResolver(opts ...ResolverOption) *Resolver {
	r := &Resolver{
		servers:    []string{"8.8.8.8", "8.8.4.4"}, // Default to Google DNS
		cache:      NewCache(),
		timeout:    DefaultTimeout,
		searchList: []string{},
		hostsFile:  "/etc/hosts",
		hosts:      make(map[string][]net.IP),
		parser:     NewParser(),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// LookupHost performs a DNS lookup for a hostname and returns its IP addresses
func (r *Resolver) LookupHost(name string) ([]net.IP, error) {
	return r.LookupHostContext(context.Background(), name)
}

// LookupHostContext performs a DNS lookup with context
func (r *Resolver) LookupHostContext(ctx context.Context, name string) ([]net.IP, error) {
	// Normalize the name
	name = normalizeName(name)
	if name == "" {
		return nil, fmt.Errorf("empty hostname")
	}

	// Check hosts file first
	if ips := r.lookupHosts(name); ips != nil {
		return ips, nil
	}

	// Check cache
	if r.cache != nil {
		if cached, ok := r.cache.Get(name, RecordTypeA); ok {
			ips := make([]net.IP, 0, len(cached))
			for _, rr := range cached {
				if ip := rr.IP(); ip != nil {
					ips = append(ips, ip)
				}
			}
			if len(ips) > 0 {
				return ips, nil
			}
		}
	}

	// Try each search domain
	names := []string{name}
	if len(r.searchList) > 0 && !strings.HasSuffix(name, ".") {
		for _, domain := range r.searchList {
			if strings.HasSuffix(name, domain) {
				continue
			}
			names = append(names, name+"."+domain)
		}
	}

	var lastErr error
	for _, n := range names {
		ips, err := r.lookupIP(ctx, n)
		if err != nil {
			lastErr = err
			continue
		}
		if len(ips) > 0 {
			// Cache successful result
			if r.cache != nil {
				rrs := make([]ResourceRecord, len(ips))
				for i, ip := range ips {
					rrs[i] = ResourceRecord{
						Name:  name,
						Type:  RecordTypeA,
						Class: ClassIN,
						TTL:   DefaultCacheTTL,
						RData: ip,
					}
					rrs[i].Expiration = time.Now().Add(DefaultCacheTTL)
				}
				r.cache.Set(name, RecordTypeA, rrs)
			}
			return ips, nil
		}
	}

	return nil, lastErr
}

// lookupIP performs a DNS A record lookup
func (r *Resolver) lookupIP(ctx context.Context, name string) ([]net.IP, error) {
	// Build query
	query, err := BuildQuery(name, RecordTypeA)
	if err != nil {
		return nil, err
	}

	// Send query
	resp, err := r.sendQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Check response
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("DNS lookup failed: %s", resp.RCODE)
	}

	// Extract IPs from answers
	ips := make([]net.IP, 0, len(resp.Answers))
	for _, rr := range resp.Answers {
		if ip := rr.IP(); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// LookupAAAA performs a DNS AAAA (IPv6) lookup
func (r *Resolver) LookupAAAA(name string) ([]net.IP, error) {
	return r.LookupAAAAContext(context.Background(), name)
}

// LookupAAAAContext performs a DNS AAAA lookup with context
func (r *Resolver) LookupAAAAContext(ctx context.Context, name string) ([]net.IP, error) {
	name = normalizeName(name)

	// Check hosts file first
	if ips := r.lookupHosts(name); ips != nil {
		ipv6s := make([]net.IP, 0)
		for _, ip := range ips {
			if ip.To4() == nil {
				ipv6s = append(ipv6s, ip)
			}
		}
		if len(ipv6s) > 0 {
			return ipv6s, nil
		}
	}

	// Build query
	query, err := BuildQuery(name, RecordTypeAAAA)
	if err != nil {
		return nil, err
	}

	// Send query
	resp, err := r.sendQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("DNS lookup failed: %s", resp.RCODE)
	}

	// Extract IPs
	ips := make([]net.IP, 0, len(resp.Answers))
	for _, rr := range resp.Answers {
		if ip := rr.IP(); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// LookupMX performs a DNS MX record lookup
func (r *Resolver) LookupMX(name string) ([]*MXRecord, error) {
	name = normalizeName(name)

	query, err := BuildQuery(name, RecordTypeMX)
	if err != nil {
		return nil, err
	}

	resp, err := r.sendQuery(context.Background(), query)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("DNS lookup failed: %s", resp.RCODE)
	}

	mx := make([]*MXRecord, 0, len(resp.Answers))
	for _, rr := range resp.Answers {
		mx = append(mx, &MXRecord{
			Host:     rr.MXHost(),
			Priority: rr.MXPriority(),
		})
	}

	return mx, nil
}

// MXRecord represents a DNS MX record
type MXRecord struct {
	Host     string
	Priority uint16
}

// LookupTXT performs a DNS TXT record lookup
func (r *Resolver) LookupTXT(name string) ([]string, error) {
	name = normalizeName(name)

	query, err := BuildQuery(name, RecordTypeTXT)
	if err != nil {
		return nil, err
	}

	resp, err := r.sendQuery(context.Background(), query)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("DNS lookup failed: %s", resp.RCODE)
	}

	txt := make([]string, 0, len(resp.Answers))
	for _, rr := range resp.Answers {
		txt = append(txt, rr.TXT())
	}

	return txt, nil
}

// LookupNS performs a DNS NS record lookup
func (r *Resolver) LookupNS(name string) ([]string, error) {
	name = normalizeName(name)

	query, err := BuildQuery(name, RecordTypeNS)
	if err != nil {
		return nil, err
	}

	resp, err := r.sendQuery(context.Background(), query)
	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("DNS lookup failed: %s", resp.RCODE)
	}

	ns := make([]string, 0, len(resp.Answers))
	for _, rr := range resp.Answers {
		ns = append(ns, rr.NS())
	}

	return ns, nil
}

// LookupCNAME performs a DNS CNAME record lookup
func (r *Resolver) LookupCNAME(name string) (string, error) {
	name = normalizeName(name)

	query, err := BuildQuery(name, RecordTypeCNAME)
	if err != nil {
		return "", err
	}

	resp, err := r.sendQuery(context.Background(), query)
	if err != nil {
		return "", err
	}

	if !resp.IsSuccess() {
		return "", fmt.Errorf("DNS lookup failed: %s", resp.RCODE)
	}

	for _, rr := range resp.Answers {
		if cname := rr.CNAME(); cname != "" {
			return cname, nil
		}
	}

	return "", fmt.Errorf("no CNAME record found")
}

// LookupAllRecords performs a DNS lookup for all record types
func (r *Resolver) LookupAllRecords(name string) (*Message, error) {
	name = normalizeName(name)

	query, err := BuildQuery(name, RecordTypeANY)
	if err != nil {
		return nil, err
	}

	return r.sendQuery(context.Background(), query)
}

// sendQuery sends a DNS query and returns the response
func (r *Resolver) sendQuery(ctx context.Context, query *Message) (*Message, error) {
	// Build query message
	data, err := r.parser.BuildMessage(query)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Try each server
	var lastErr error
	for _, server := range r.servers {
		var resp *Message
		var err error

		if r.tlsConfig != nil {
			resp, err = r.sendQueryTLS(ctx, server, data)
		} else {
			resp, err = r.sendQueryUDP(ctx, server, data)
		}

		if err != nil {
			lastErr = err
			continue
		}

		// If response is truncated, retry with TCP
		if resp != nil && resp.TC {
			resp, err = r.sendQueryTCP(ctx, server, data)
			if err != nil {
				lastErr = err
				continue
			}
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all DNS servers failed: %w", lastErr)
}

// sendQueryUDP sends a DNS query over UDP
func (r *Resolver) sendQueryUDP(ctx context.Context, server string, query []byte) (*Message, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(server, "53"))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Set timeout
	conn.SetDeadline(time.Now().Add(r.timeout))

	// Send query
	_, err = conn.Write(query)
	if err != nil {
		return nil, err
	}

	// Read response
	buf := make([]byte, 512) // Standard DNS UDP limit
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return r.parser.ParseMessage(buf[:n])
}

// sendQueryTCP sends a DNS query over TCP
func (r *Resolver) sendQueryTCP(ctx context.Context, server string, query []byte) (*Message, error) {
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(server, "53"))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Set timeout
	conn.SetDeadline(time.Now().Add(r.timeout))

	// Send length prefix
	var lengthBuf [2]byte
	binary.BigEndian.PutUint16(lengthBuf[:], uint16(len(query)))
	_, err = conn.Write(lengthBuf[:])
	if err != nil {
		return nil, err
	}

	// Send query
	_, err = conn.Write(query)
	if err != nil {
		return nil, err
	}

	// Read length prefix
	_, err = conn.Read(lengthBuf[:])
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(lengthBuf[:])
	buf := make([]byte, length)
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return r.parser.ParseMessage(buf)
}

// sendQueryTLS sends a DNS query over DNS-over-TLS
func (r *Resolver) sendQueryTLS(ctx context.Context, server string, query []byte) (*Message, error) {
	conn, err := tls.Dial("tcp", net.JoinHostPort(server, "853"), r.tlsConfig)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Set timeout
	conn.SetDeadline(time.Now().Add(r.timeout))

	// Send length prefix
	var lengthBuf [2]byte
	binary.BigEndian.PutUint16(lengthBuf[:], uint16(len(query)))
	_, err = conn.Write(lengthBuf[:])
	if err != nil {
		return nil, err
	}

	// Send query
	_, err = conn.Write(query)
	if err != nil {
		return nil, err
	}

	// Read length prefix
	_, err = conn.Read(lengthBuf[:])
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(lengthBuf[:])
	buf := make([]byte, length)
	_, err = conn.Read(buf)
	if err != nil {
		return nil, err
	}

	return r.parser.ParseMessage(buf)
}

// lookupHosts looks up a name in the hosts file
func (r *Resolver) lookupHosts(name string) []net.IP {
	// Load hosts file if not loaded
	if !r.hostsMu {
		r.loadHostsFile()
		r.hostsMu = true
	}

	// Normalize name
	name = strings.ToLower(name)

	// Direct lookup
	if ips, ok := r.hosts[name]; ok {
		return ips
	}

	// Also check with domain parts
	parts := strings.Split(name, ".")
	for i := range parts {
		candidate := strings.Join(parts[i:], ".")
		if ips, ok := r.hosts[candidate]; ok {
			return ips
		}
	}

	return nil
}

// loadHostsFile loads the hosts file into memory
func (r *Resolver) loadHostsFile() {
	file, err := os.Open(r.hostsFile)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}

		for _, name := range fields[1:] {
			// Skip comments
			if strings.HasPrefix(name, "#") {
				break
			}
			name = strings.ToLower(name)
			if existing, ok := r.hosts[name]; ok {
				r.hosts[name] = append(existing, ip)
			} else {
				r.hosts[name] = []net.IP{ip}
			}
		}
	}
}

// ReloadHosts reloads the hosts file
func (r *Resolver) ReloadHosts() {
	r.hostsMu = false
	r.hosts = make(map[string][]net.IP)
	r.loadHostsFile()
	r.hostsMu = true
}

// normalizeName normalizes a DNS name
func normalizeName(name string) string {
	name = strings.ToLower(name)
	// Remove trailing dot if present
	name = strings.TrimSuffix(name, ".")
	// Remove any leading/trailing whitespace
	name = strings.TrimSpace(name)
	return name
}

// Exchange performs a raw DNS message exchange
func (r *Resolver) Exchange(msg *Message) (*Message, error) {
	return r.sendQuery(context.Background(), msg)
}

// ExchangeContext performs a raw DNS message exchange with context
func (r *Resolver) ExchangeContext(ctx context.Context, msg *Message) (*Message, error) {
	return r.sendQuery(ctx, msg)
}

// DialUDP creates a UDP connection to a DNS server
func DialUDP(server string) (*net.UDPConn, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(server, "53"))
	if err != nil {
		return nil, err
	}
	return net.DialUDP("udp", nil, addr)
}

// DialTCP creates a TCP connection to a DNS server
func DialTCP(server string) (*net.TCPConn, error) {
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(server, "53"))
	if err != nil {
		return nil, err
	}
	return net.DialTCP("tcp", nil, addr)
}

// DialTLS creates a TLS connection to a DNS server for DoT
func DialTLS(server string, tlsConfig *tls.Config) (*tls.Conn, error) {
	return tls.Dial("tcp", net.JoinHostPort(server, "853"), tlsConfig)
}

// IsNotFound returns true if the error indicates a domain was not found
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "NXDOMAIN") ||
		strings.Contains(err.Error(), "no such host")
}

// IsTimeout returns true if the error indicates a timeout
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "i/o timeout")
}

// ParseResolvConf parses a resolv.conf file and returns DNS servers and search domains
func ParseResolvConf(path string) (servers []string, searchList []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "nameserver":
			servers = append(servers, fields[1])
		case "search":
			for i := 1; i < len(fields); i++ {
				searchList = append(searchList, fields[i])
			}
		case "domain":
			if len(searchList) == 0 {
				searchList = append(searchList, fields[1])
			}
		}
	}

	return servers, searchList, scanner.Err()
}

// DefaultResolvConf returns the default DNS configuration from /etc/resolv.conf
func DefaultResolvConf() ([]string, []string, error) {
	return ParseResolvConf("/etc/resolv.conf")
}

// PackDomainName packs a domain name into DNS wire format
func PackDomainName(name string) ([]byte, error) {
	var buf bytes.Buffer

	labels := strings.Split(name, ".")
	for _, label := range labels {
		if err := buf.WriteByte(byte(len(label))); err != nil {
			return nil, err
		}
		if _, err := buf.WriteString(label); err != nil {
			return nil, err
		}
	}

	if err := buf.WriteByte(0); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnpackDomainName unpacks a domain name from DNS wire format
func UnpackDomainName(data []byte, offset int) (string, int, error) {
	var name bytes.Buffer
	pos := offset
	loop := 0

	for loop < 256 {
		if pos >= len(data) {
			return "", 0, fmt.Errorf("buffer too short")
		}

		length := int(data[pos])
		pos++

		// Check for compression pointer
		if (length & 0xC0) == 0xC0 {
			if pos >= len(data) {
				return "", 0, fmt.Errorf("invalid compression pointer")
			}
			pointer := (length & 0x3F) << 8
			pointer |= int(data[pos])
			pos++

			pointed, _, err := UnpackDomainName(data, pointer)
			if err != nil {
				return "", 0, err
			}

			if name.Len() > 0 {
				name.WriteByte('.')
			}
			name.WriteString(pointed)
			break
		}

		// End of name
		if length == 0 {
			break
		}

		if pos+length > len(data) {
			return "", 0, fmt.Errorf("label extends beyond buffer")
		}

		if name.Len() > 0 {
			name.WriteByte('.')
		}
		name.Write(data[pos : pos+length])
		pos += length

		loop++
	}

	return name.String(), pos, nil
}
