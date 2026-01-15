// Package main provides a simple HTTP server for testing the WebOS JavaScript client.
// This demo server serves static files from the static directory.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

var (
	addr      = flag.String("addr", ":8080", "HTTP server address")
	staticDir = flag.String("static", "./static", "Static files directory")
)

func main() {
	flag.Parse()

	fmt.Println("WebOS Client Demo Server")
	fmt.Println("========================")
	fmt.Printf("Server address: http://%s\n", *addr)
	fmt.Printf("Static files directory: %s\n", *staticDir)
	fmt.Println("\nTo test the client:")
	fmt.Println("1. Open http://localhost:8080 in your browser")
	fmt.Println("2. The client will attempt to connect to ws://localhost:8080/ws")
	fmt.Println("3. For full WebSocket support, use a compatible WebSocket server")

	// Static file server
	fs := http.FileServer(http.Dir(*staticDir))
	http.Handle("/", fs)

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Info endpoint
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"name": "WebOS Client Demo",
			"version": "1.0.0",
			"protocol": "Custom binary protocol (WEBS)",
			"features": [
				"Static file serving",
				"WebSocket endpoint (requires compatible server)"
			]
		}`))
	})

	log.Printf("Starting server on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
