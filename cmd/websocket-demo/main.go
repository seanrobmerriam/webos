// Package main provides a WebSocket server demonstration program.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webos/pkg/protocol"
	"webos/pkg/websocket"
)

func main() {
	fmt.Println("WebSocket Server Demo")
	fmt.Println("=====================")

	// 1. Create WebSocket server
	fmt.Println("\n1. Starting server on :8080...")
	server := websocket.NewServer(&websocket.ServerConfig{
		Addr:          ":8080",
		PoolConfig:    websocket.DefaultPoolConfig(),
		SessionConfig: websocket.DefaultSessionConfig(),
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  10 * time.Second,
		PingInterval:  25 * time.Second,
	})

	// Set message handler
	server.SetHandler(func(conn *websocket.Connection, msg *protocol.Message) error {
		fmt.Printf("   Received message from %s: opcode=%s, payload=%s\n",
			conn.ID, msg.Opcode, string(msg.Payload))
		return nil
	})

	// Set callbacks
	server.OnAccept(func(conn *websocket.Connection) {
		fmt.Printf("   Client connected: %s\n", conn.ID)
	})

	server.OnUpgrade(func(conn *websocket.Connection) {
		// Create session
		session, err := server.SessionManager().Create(conn.ID, "user-"+conn.ID[:4])
		if err != nil {
			fmt.Printf("   Failed to create session: %v\n", err)
			return
		}
		conn.SetSession(session)
		fmt.Printf("   Session created: %s for connection %s\n", session.ID, conn.ID)
	})

	// Start server in goroutine
	go func() {
		fmt.Println("   Server started successfully")
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// 2. Simulate client connections
	fmt.Println("\n2. Simulating client connections...")
	pool := server.Pool()

	// Add some simulated connections
	for i := 1; i <= 3; i++ {
		connID := fmt.Sprintf("client-%d", i)
		conn := websocket.NewConnection(connID, nil)
		pool.Add(conn)
		fmt.Printf("   Client %d connected: %s\n", i, connID)
	}

	// 3. Display pool statistics
	fmt.Println("\n3. Pool statistics...")
	stats := pool.Stats()
	fmt.Printf("   Active connections: %d\n", stats.ActiveConnections)
	fmt.Printf("   Total accepted: %d\n", stats.TotalAccepted)
	fmt.Printf("   Total closed: %d\n", stats.TotalClosed)
	fmt.Printf("   Total failed: %d\n", stats.TotalFailed)

	// 4. Session management demo
	fmt.Println("\n4. Session management...")
	sessionManager := server.SessionManager()

	// Create sessions
	for i := 1; i <= 3; i++ {
		session, err := sessionManager.Create(fmt.Sprintf("session-%d", i), fmt.Sprintf("user-%d", i))
		if err != nil {
			fmt.Printf("   Failed to create session: %v\n", err)
			continue
		}
		fmt.Printf("   Session created: %s for user %s\n", session.ID, session.UserID)
	}

	fmt.Printf("   Active sessions: %d\n", sessionManager.Count())

	// 5. Broadcast message demo
	fmt.Println("\n5. Broadcasting message to all connections...")
	broadcastMsg := protocol.NewMessage(protocol.OpcodeDisplay, []byte("Hello from server!"))
	pool.BroadcastMessage(broadcastMsg)
	fmt.Println("   Message broadcast sent")

	// 6. Demonstrate frame operations
	fmt.Println("\n6. Frame operations demo...")
	var buf bytes.Buffer
	writer := websocket.NewFrameWriter(&buf)

	// Write various frame types
	writer.WriteText([]byte("Hello, WebSocket!"))
	writer.WriteBinary([]byte{0x00, 0x01, 0x02, 0x03})
	writer.WritePing([]byte("ping"))
	writer.WritePong([]byte("pong"))
	writer.WriteClose(1000, "goodbye")

	fmt.Printf("   Frames written: text, binary, ping, pong, close\n")
	fmt.Printf("   Total bytes written: %d\n", buf.Len())

	// 7. Clean up
	fmt.Println("\n7. Cleaning up...")
	for _, conn := range pool.All() {
		pool.Remove(conn)
	}
	fmt.Printf("   All connections removed. Pool size: %d\n", pool.Count())

	sessionManager.DestroyAll()
	fmt.Printf("   All sessions destroyed. Session count: %d\n", sessionManager.Count())

	// Wait for interrupt signal
	fmt.Println("\n8. Waiting for shutdown signal (Ctrl+C)...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Stop server
	fmt.Println("\n9. Stopping server...")
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}
	fmt.Println("   Server stopped")

	fmt.Println("\nDemo completed successfully!")
}
