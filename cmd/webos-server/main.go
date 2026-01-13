// webos-server is the main HTTP server for the webos operating system.
// It provides HTTP/1.1 and HTTP/2 support, static file serving, and routing.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webos/pkg/router"
	"webos/pkg/server"
)

func main() {
	// Parse command line flags
	addr := ":8080"
	tlsEnabled := false
	staticDir := "./static"

	if envAddr := os.Getenv("WEBOS_ADDR"); envAddr != "" {
		addr = envAddr
	}
	if envStatic := os.Getenv("WEBOS_STATIC"); envStatic != "" {
		staticDir = envStatic
	}
	if os.Getenv("WEBOS_TLS") == "true" {
		tlsEnabled = true
	}

	// Create a new router
	r := router.New()

	// Add middleware
	r.Use(router.RecoveryMiddleware())
	r.Use(router.LoggingMiddleware())
	r.Use(router.CORSMiddleware())

	// Register routes
	setupRoutes(r)

	// Create server with router as handler
	srv := server.New(server.Config{
		Addr:      addr,
		Handler:   r,
		StaticDir: staticDir,
	})

	// Enable TLS if configured
	if tlsEnabled {
		certFile := "server.crt"
		keyFile := "server.key"
		if err := srv.EnableTLS(certFile, keyFile); err != nil {
			log.Fatalf("Failed to enable TLS: %v", err)
		}
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", addr)
		if tlsEnabled {
			if err := srv.ListenAndServeTLS(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// setupRoutes registers all HTTP routes.
func setupRoutes(r *router.Router) {
	// Health check endpoints
	r.GET("/health", server.HealthHandler())
	r.GET("/ready", server.ReadyHandler())
	r.GET("/metrics", server.MetricsHandler())

	// API routes
	r.GET("/api/v1/users/:id", UserHandler())
	r.GET("/api/v1/users", UsersHandler())
	r.POST("/api/v1/users", CreateUserHandler())
	r.GET("/api/v1/posts/:id", PostHandler())
	r.GET("/api/v1/posts", PostsHandler())

	// Static files are handled by the server's static file handler
	// The server will serve files from the static directory for any path not matched above
}

// UserHandler returns a handler for user requests.
func UserHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, ok := router.ParamsFromContext(r.Context())
		if !ok {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		userID := params.Get("id")
		if userID == "" {
			http.Error(w, "User ID required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": %q, "name": "Test User"}`, userID)
	})
}

// UsersHandler returns a handler for listing users.
func UsersHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": "1", "name": "User 1"}, {"id": "2", "name": "User 2"}]`))
	})
}

// CreateUserHandler returns a handler for creating users.
func CreateUserHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": "3", "name": "New User"}`))
	})
}

// PostHandler returns a handler for post requests.
func PostHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, ok := router.ParamsFromContext(r.Context())
		if !ok {
			http.Error(w, "Post not found", http.StatusNotFound)
			return
		}

		postID := params.Get("id")
		if postID == "" {
			http.Error(w, "Post ID required", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"id": %q, "title": "Test Post"}`, postID)
	})
}

// PostsHandler returns a handler for listing posts.
func PostsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": "1", "title": "Post 1"}, {"id": "2", "title": "Post 2"}]`))
	})
}
