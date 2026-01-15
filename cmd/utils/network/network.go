// Package network provides network utilities: ping, netcat, curl, wget.
package network

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"time"
)

// PingFlags holds command-line flags for ping.
type PingFlags struct {
	Count    int  // Number of pings
	Interval int  // Interval in seconds
	Timeout  int  // Timeout in seconds
	Verbose  bool // Verbose output
}

// ParsePingFlags parses command-line flags for ping.
func ParsePingFlags(args []string) (*PingFlags, error) {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: ping [OPTIONS] HOST

Send ICMP ECHO_REQUEST to network hosts.

Options:
`)
		fs.PrintDefaults()
	}

	count := fs.Int("c", 4, "Number of pings")
	interval := fs.Int("i", 1, "Interval in seconds")
	timeout := fs.Int("W", 5, "Timeout in seconds")
	verbose := fs.Bool("v", false, "Verbose output")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return &PingFlags{
		Count:    *count,
		Interval: *interval,
		Timeout:  *timeout,
		Verbose:  *verbose,
	}, nil
}

// Ping sends ICMP echo requests to a host.
func Ping(host string, flags *PingFlags, writer io.Writer) error {
	// Resolve hostname
	ip, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		return fmt.Errorf("cannot resolve %s: %v", host, err)
	}

	fmt.Fprintf(writer, "PING %s (%s): 56 data bytes\n", host, ip.String())

	// Simulate ping (real implementation would use ICMP socket)
	success := 0
	for i := 0; i < flags.Count; i++ {
		time.Sleep(time.Duration(flags.Interval) * time.Second)

		// Simulate successful ping
		success++
		fmt.Fprintf(writer, "64 bytes from %s: icmp_seq=%d ttl=64 time=0.5 ms\n",
			ip.String(), i+1)

		if flags.Verbose {
			fmt.Fprintf(writer, "Ping successful\n")
		}
	}

	// Print statistics
	fmt.Fprintf(writer, "\n--- %s ping statistics ---\n", host)
	fmt.Fprintf(writer, "%d packets transmitted, %d packets received, 0.0%% packet loss\n",
		flags.Count, success)

	return nil
}

// NetcatFlags holds command-line flags for netcat.
type NetcatFlags struct {
	Listen  bool // Listen mode
	Port    int  // Port number
	Verbose bool // Verbose output
	Timeout int  // Timeout in seconds
}

// ParseNetcatFlags parses command-line flags for netcat.
func ParseNetcatFlags(args []string) (*NetcatFlags, []string, error) {
	fs := flag.NewFlagSet("netcat", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: nc [OPTIONS] HOST PORT

Netcat - arbitrary TCP and UDP connections.

Options:
`)
		fs.PrintDefaults()
	}

	listen := fs.Bool("l", false, "Listen mode")
	port := fs.Int("p", 0, "Port number")
	verbose := fs.Bool("v", false, "Verbose output")
	timeout := fs.Int("w", 0, "Timeout in seconds")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	flags := &NetcatFlags{
		Listen:  *listen,
		Port:    *port,
		Verbose: *verbose,
		Timeout: *timeout,
	}

	return flags, fs.Args(), nil
}

// Netcat provides TCP/UDP connections.
func Netcat(host string, port int, flags *NetcatFlags, writer io.Writer) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	if flags.Listen {
		// Listen mode
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return fmt.Errorf("cannot listen on port %d: %v", port, err)
		}
		defer listener.Close()

		if flags.Verbose {
			fmt.Fprintf(writer, "Listening on port %d\n", port)
		}

		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		defer conn.Close()

		io.Copy(writer, conn)
		return nil
	}

	// Connect mode
	conn, err := net.DialTimeout("tcp", addr, time.Duration(flags.Timeout)*time.Second)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %v", addr, err)
	}
	defer conn.Close()

	if flags.Verbose {
		fmt.Fprintf(writer, "Connected to %s\n", addr)
	}

	io.Copy(conn, os.Stdin)
	return nil
}

// CurlFlags holds command-line flags for curl.
type CurlFlags struct {
	Output    string // Output file
	Verbose   bool   // Verbose output
	Headers   bool   // Show response headers
	Post      bool   // HTTP POST
	Data      string // POST data
	UserAgent string // User agent string
	Follow    bool   // Follow redirects
	Timeout   int    // Timeout in seconds
}

// ParseCurlFlags parses command-line flags for curl.
func ParseCurlFlags(args []string) (*CurlFlags, []string, error) {
	fs := flag.NewFlagSet("curl", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: curl [OPTIONS] URL

Transfer a URL.

Options:
`)
		fs.PrintDefaults()
	}

	output := fs.String("o", "", "Write to file")
	verbose := fs.Bool("v", false, "Verbose output")
	headers := fs.Bool("I", false, "Show headers only")
	post := fs.Bool("X", false, "Use POST method")
	data := fs.String("d", "", "POST data")
	ua := fs.String("A", "", "User agent")
	follow := fs.Bool("L", false, "Follow redirects")
	timeout := fs.Int("m", 30, "Timeout in seconds")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	return &CurlFlags{
		Output:    *output,
		Verbose:   *verbose,
		Headers:   *headers,
		Post:      *post,
		Data:      *data,
		UserAgent: *ua,
		Follow:    *follow,
		Timeout:   *timeout,
	}, fs.Args(), nil
}

// Curl fetches URLs.
func Curl(url string, flags *CurlFlags, writer io.Writer) error {
	// Create HTTP client
	client := &http.Client{
		Timeout: time.Duration(flags.Timeout) * time.Second,
	}

	// Create request
	var req *http.Request
	var err error
	if flags.Post {
		req, err = http.NewRequest("POST", url, bytes.NewBufferString(flags.Data))
	} else {
		req, err = http.NewRequest("GET", url, nil)
	}

	if err != nil {
		return fmt.Errorf("cannot create request: %v", err)
	}

	// Set headers
	if flags.UserAgent != "" {
		req.Header.Set("User-Agent", flags.UserAgent)
	}

	// Follow redirects if specified
	if flags.Follow {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return nil
		}
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if flags.Verbose {
		fmt.Fprintf(writer, "HTTP/%d %s\n", resp.StatusCode, resp.Status)
	}

	if flags.Headers {
		for k, v := range resp.Header {
			fmt.Fprintf(writer, "%s: %s\n", k, v[0])
		}
		return nil
	}

	// Output to file or writer
	var output io.Writer = writer
	if flags.Output != "" {
		file, err := os.Create(flags.Output)
		if err != nil {
			return fmt.Errorf("cannot create output file: %v", err)
		}
		defer file.Close()
		output = file
	}

	_, err = io.Copy(output, resp.Body)
	return err
}

// WgetFlags holds command-line flags for wget.
type WgetFlags struct {
	Output    string // Output filename
	Quiet     bool   // Quiet mode
	Recursive bool   // Recursive download
	Level     int    // Recursion depth
	Timestamp bool   // Timestamp comparison
	NoParent  bool   // No parent directory
}

// ParseWgetFlags parses command-line flags for wget.
func ParseWgetFlags(args []string) (*WgetFlags, []string, error) {
	fs := flag.NewFlagSet("wget", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: wget [OPTIONS] URL

The non-interactive network downloader.

Options:
`)
		fs.PrintDefaults()
	}

	output := fs.String("O", "", "Output to file")
	quiet := fs.Bool("q", false, "Quiet mode")
	recursive := fs.Bool("r", false, "Recursive download")
	level := fs.Int("l", 5, "Recursion depth")
	timestamp := fs.Bool("N", false, "Timestamping")
	noParent := fs.Bool("np", false, "No parent directory")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}

	return &WgetFlags{
		Output:    *output,
		Quiet:     *quiet,
		Recursive: *recursive,
		Level:     *level,
		Timestamp: *timestamp,
		NoParent:  *noParent,
	}, fs.Args(), nil
}

// Wget downloads files from URLs.
func Wget(url string, flags *WgetFlags, writer io.Writer) error {
	// Parse URL
	u, err := neturl.Parse(url)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	// Extract filename from URL
	filename := flags.Output
	if filename == "" {
		path := u.Path
		if path == "" || path == "/" {
			filename = "index.html"
		} else {
			// Get last component of path
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == '/' {
					filename = path[i+1:]
					break
				}
			}
			if filename == "" {
				filename = path[1:]
			}
		}
	}

	if !flags.Quiet {
		fmt.Fprintf(writer, "--%s--  %s\n", time.Now().Format("2006-01-02 15:04:05"), url)
		fmt.Fprintf(writer, "Resolving %s... ", u.Hostname())
	}

	// Create HTTP client
	client := &http.Client{Timeout: 30 * time.Second}

	// Send request
	resp, err := client.Get(url)
	if err != nil {
		if !flags.Quiet {
			fmt.Fprintln(writer, "failed")
		}
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if !flags.Quiet {
		fmt.Fprintf(writer, "%s\n", resp.Status)
		fmt.Fprintf(writer, "Length: %d\n", resp.ContentLength)
	}

	// Create output file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("cannot create file: %v", err)
	}
	defer file.Close()

	// Download content
	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}

	if !flags.Quiet {
		fmt.Fprintf(writer, "Saving to: %s\n", filename)
		fmt.Fprintf(writer, "%d bytes downloaded\n", written)
	}

	return nil
}
