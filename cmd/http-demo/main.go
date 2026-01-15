package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"webos/pkg/http"
)

func main() {
	fmt.Println("=== HTTP Package Demo ===")
	fmt.Println()

	// Demo 1: HTTP types and constants
	fmt.Println("1. HTTP Types and Constants:")
	fmt.Printf("   Status 200: %s\n", http.StatusText(200))
	fmt.Printf("   Status 404: %s\n", http.StatusText(404))
	fmt.Printf("   Method GET: %s\n", http.MethodGet)
	fmt.Printf("   Method POST: %s\n", http.MethodPost)
	fmt.Printf("   Protocol HTTP/1.1: %s\n", http.ProtocolHTTP11)
	fmt.Println()

	// Demo 2: Header operations
	fmt.Println("2. Header Operations:")
	h := make(http.Header)
	h.Set("Content-Type", "text/html")
	h.Add("Accept", "application/json")
	h.Add("Accept", "text/plain")
	fmt.Printf("   Content-Type: %s\n", h.Get("Content-Type"))
	fmt.Printf("   Accept (case-insensitive): %s\n", h.Get("accept"))
	fmt.Println()

	// Demo 3: URL encoding/decoding
	fmt.Println("3. URL Encoding/Decoding:")
	data := map[string]string{
		"name":  "John Doe",
		"query": "hello world",
	}
	encoded := http.URLEncode(data)
	fmt.Printf("   Encoded: %s\n", encoded)
	decoded, _ := http.URLDecode(encoded)
	fmt.Printf("   Decoded name: %s\n", decoded["name"])
	fmt.Println()

	// Demo 4: Request creation
	fmt.Println("4. Request Creation:")
	req, err := http.NewRequest(http.MethodGet, "http://example.com/path?query=value", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "DemoClient/1.0")
	fmt.Printf("   Method: %s\n", req.Method)
	fmt.Printf("   URL: %s\n", req.URL.String())
	fmt.Printf("   Host: %s\n", req.Host)
	fmt.Printf("   User-Agent: %s\n", req.Header.Get("User-Agent"))
	fmt.Println()

	// Demo 5: Response creation
	fmt.Println("5. Response Creation:")
	resp := http.NewResponse(200, strings.NewReader("Hello, World!"))
	resp.Header.Set("Content-Type", "text/plain")
	resp.Header.Set("Content-Length", "13")
	fmt.Printf("   Status: %s\n", resp.Status)
	fmt.Printf("   StatusCode: %d\n", resp.StatusCode)
	fmt.Println()

	// Demo 6: Write HTTP messages
	fmt.Println("6. HTTP Message Writing:")
	var buf bytes.Buffer
	req.WriteTo(&buf)
	fmt.Printf("   Request:\n%s", buf.String())

	buf.Reset()
	resp.WriteTo(&buf)
	fmt.Printf("   Response:\n%s", buf.String())
	fmt.Println()

	// Demo 7: ServeMux routing
	fmt.Println("7. ServeMux Routing:")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.WriteText(w, 200, "Home Page")
	})
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		http.WriteJSON(w, 200, []byte(`{"message":"Hello, World!"}`))
	})
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		http.WriteJSON(w, 200, []byte(`[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`))
	})
	fmt.Println("   Registered routes:")
	fmt.Println("   - GET /")
	fmt.Println("   - GET /api/hello")
	fmt.Println("   - GET /api/users")
	fmt.Println()

	// Demo 8: HTTP/2 info
	fmt.Println("8. HTTP/2 Support:")
	fmt.Println("   h2 package provides:")
	fmt.Println("   - Frame types: DATA, HEADERS, SETTINGS, PING, WINDOW_UPDATE")
	fmt.Println("   - Stream management with priority")
	fmt.Println("   - HPACK header compression")
	fmt.Println()

	// Demo 9: Start a test server
	fmt.Println("9. Starting HTTP Server on :8080...")
	go startTestServer()
	time.Sleep(100 * time.Millisecond)
	fmt.Println("    Server started at http://localhost:8080")
	fmt.Println()
	fmt.Println("=== Demo Complete ===")
}

func startTestServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><body><h1>HTTP Demo Server</h1><p>Path: %s</p></body></html>", r.URL.Path)
	})
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"message":"Hello from WebOS HTTP Server!"}`))
	})
	mux.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"time":"` + time.Now().Format(time.RFC3339) + `"}`))
	})
	mux.HandleFunc("/api/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		w.Write(body)
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	log.Fatal(server.ListenAndServe())
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		http.WriteText(conn, 400, "Bad Request: "+err.Error())
		return
	}

	var responseBody string
	switch req.URL.Path {
	case "/":
		responseBody = "Hello from the HTTP demo!"
	case "/time":
		responseBody = "Current time: " + time.Now().Format(time.RFC3339)
	default:
		responseBody = "Not Found: " + req.URL.Path
	}

	http.WriteText(conn, 200, responseBody)
}

func init() {
	log.SetOutput(os.Stderr)
}
