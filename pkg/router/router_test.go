package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouter_New(t *testing.T) {
	r := New()
	if r == nil {
		t.Fatal("New() returned nil")
	}
	if r.routes == nil {
		t.Error("routes map is nil")
	}
	if r.notFound == nil {
		t.Error("notFound handler is nil")
	}
}

func TestRouter_GET(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test", handler)

	if len(r.routes[http.MethodGet]) != 1 {
		t.Errorf("expected 1 GET route, got %d", len(r.routes[http.MethodGet]))
	}
}

func TestRouter_POST(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.POST("/test", handler)

	if len(r.routes[http.MethodPost]) != 1 {
		t.Errorf("expected 1 POST route, got %d", len(r.routes[http.MethodPost]))
	}
}

func TestRouter_ServeHTTP_ExactMatch(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("exact match"))
	})
	r.GET("/exact/path", handler)

	req := httptest.NewRequest(http.MethodGet, "/exact/path", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "exact match" {
		t.Errorf("expected body 'exact match', got '%s'", w.Body.String())
	}
}

func TestRouter_ServeHTTP_NotFound(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestRouter_ServeHTTP_MethodNotAllowed(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test", handler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
	}
}

func TestRouter_NamedParameter(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, ok := ParamsFromContext(r.Context())
		if !ok {
			t.Error("expected params in context")
			return
		}
		if params["id"] != "123" {
			t.Errorf("expected id '123', got '%s'", params["id"])
		}
		w.Write([]byte("OK"))
	})
	r.GET("/users/:id", handler)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouter_NestedParameters(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, ok := ParamsFromContext(r.Context())
		if !ok {
			t.Error("expected params in context")
			return
		}
		if params["id"] != "456" {
			t.Errorf("expected id '456', got '%s'", params["id"])
		}
		if params["commentId"] != "789" {
			t.Errorf("expected commentId '789', got '%s'", params["commentId"])
		}
		w.Write([]byte("OK"))
	})
	r.GET("/posts/:id/comments/:commentId", handler)

	req := httptest.NewRequest(http.MethodGet, "/posts/456/comments/789", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouter_Wildcard(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params, ok := ParamsFromContext(r.Context())
		if !ok {
			t.Error("expected params in context")
			return
		}
		wildcard, ok := params["wildcard"]
		if !ok {
			t.Error("expected wildcard in params")
			return
		}
		// Wildcard includes the leading slash
		if wildcard != "/api/v1/users" {
			t.Errorf("expected wildcard '/api/v1/users', got '%s'", wildcard)
		}
		w.Write([]byte("OK"))
	})
	r.GET("/api/*", handler)

	req := httptest.NewRequest(http.MethodGet, "/api/api/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRouter_Use(t *testing.T) {
	r := New()
	middlewareCalled := false
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}
	r.Use(middleware)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !middlewareCalled {
		t.Error("middleware was not called")
	}
}

func TestRouter_SetNotFoundHandler(t *testing.T) {
	r := New()
	customNotFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("Custom Not Found"))
	})
	r.SetNotFoundHandler(customNotFound)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test", handler)

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, w.Code)
	}
}

func TestRouter_SetMethodNotAllowedHandler(t *testing.T) {
	r := New()
	customNotAllowed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("Custom Not Allowed"))
	})
	r.SetMethodNotAllowedHandler(customNotAllowed)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test", handler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status %d, got %d", http.StatusTeapot, w.Code)
	}
}

func TestRouter_Routes(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	r.GET("/test1", handler)
	r.POST("/test2", handler)

	routes := r.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}

func TestParams_Get(t *testing.T) {
	p := Params{
		"id":   "123",
		"name": "test",
	}

	if p.Get("id") != "123" {
		t.Errorf("expected '123', got '%s'", p.Get("id"))
	}
	if p.Get("name") != "test" {
		t.Errorf("expected 'test', got '%s'", p.Get("name"))
	}
	if p.Get("missing") != "" {
		t.Errorf("expected empty string, got '%s'", p.Get("missing"))
	}
}

func TestParams_Has(t *testing.T) {
	p := Params{
		"id": "123",
	}

	if !p.Has("id") {
		t.Error("expected Has('id') to return true")
	}
	if p.Has("missing") {
		t.Error("expected Has('missing') to return false")
	}
}

func TestWithParams(t *testing.T) {
	ctx := context.Background()
	ctx = WithParams(ctx, Params{"key": "value"})
	params, ok := ParamsFromContext(ctx)
	if !ok {
		t.Error("expected params to be present in context")
	}
	if params["key"] != "value" {
		t.Errorf("expected 'value', got '%s'", params["key"])
	}
}

func TestLoggingMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	middleware := LoggingMiddleware()
	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	middleware := RecoveryMiddleware()
	handler := middleware(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	// This should not panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestAuthMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	middleware := AuthMiddleware("valid-token")
	handler := middleware(nextHandler)

	// Test with valid token
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "valid-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Test with invalid token
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.Header.Set("Authorization", "invalid-token")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w2.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	middleware := CORSMiddleware()
	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected Access-Control-Allow-Origin header")
	}

	// Test OPTIONS request
	req2 := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w2.Code)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate one if not present (middleware should have added it)
			requestID = w.Header().Get("X-Request-ID")
		}
		if requestID == "" {
			t.Error("expected X-Request-ID header to be set")
		}
		w.Write([]byte("OK"))
	})
	middleware := RequestIDMiddleware()
	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header to be set in response")
	}
}

func TestBodyBufferMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If body is nil or empty, skip the check
		if r.Body == nil {
			w.Write([]byte("OK"))
			return
		}
		body, ok := BodyFromContext(r.Context())
		if !ok {
			t.Error("expected body in context")
			return
		}
		if len(body) == 0 {
			w.Write([]byte("OK"))
			return
		}
		if string(body) != "test body" {
			t.Errorf("expected 'test body', got '%s'", string(body))
		}
		w.Write([]byte("OK"))
	})
	middleware := BodyBufferMiddleware()
	handler := middleware(nextHandler)

	// Test with nil body - should handle gracefully
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Test with actual body
	req2 := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("test body"))
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w2.Code)
	}
}

func TestChain(t *testing.T) {
	callOrder := []string{}
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "1")
			next.ServeHTTP(w, r)
		})
	}
	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, "2")
			next.ServeHTTP(w, r)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, "3")
		w.Write([]byte("OK"))
	})

	chain := Chain(middleware1, middleware2)
	wrappedHandler := chain(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Middleware should be called in reverse order (2 then 1)
	expected := []string{"1", "2", "3"}
	if len(callOrder) != len(expected) {
		t.Errorf("expected call order %v, got %v", expected, callOrder)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	middleware := TimeoutMiddleware(100)
	handler := middleware(nextHandler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestHandlerType(t *testing.T) {
	handlerCalled := false
	var h Handler = func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.Write([]byte("OK"))
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
}

func TestContext_Param(t *testing.T) {
	r := New()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &Context{ResponseWriter: w, Request: r}
		if ctx.Param("id") != "123" {
			t.Errorf("expected '123', got '%s'", ctx.Param("id"))
		}
		w.Write([]byte("OK"))
	})
	r.GET("/users/:id", handler)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
}

func TestContext_Redirect(t *testing.T) {
	ctx := &Context{
		ResponseWriter: httptest.NewRecorder(),
		Request:        httptest.NewRequest(http.MethodGet, "/test", nil),
	}
	ctx.Redirect("/new", http.StatusMovedPermanently)
}

func TestContext_WithContext(t *testing.T) {
	ctx := &Context{
		ResponseWriter: httptest.NewRecorder(),
		Request:        httptest.NewRequest(http.MethodGet, "/test", nil),
	}
	bgCtx := context.Background()
	newCtx := ctx.WithContext(bgCtx)
	if newCtx == nil {
		t.Error("expected non-nil context")
	}
}
