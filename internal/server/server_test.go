package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	// Create server instance
	srv, err := New(Config{
		DeviceID: "test-device-001",
		HTTPAddr: ":0",
	})
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() {
		if err := srv.Run(ctx); err != nil && err != context.DeadlineExceeded {
			t.Errorf("Server error: %v", err)
		}
	}()

	// Allow server to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	t.Run("Health Check", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health", nil)

		srv.routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["status"] != "ok" {
			t.Error("Expected status 'ok'")
		}
	})

	// Test method not allowed
	t.Run("Method Not Allowed", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/health", nil)

		srv.routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
		}
	})

	// Test status endpoint
	t.Run("Status Check", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/v1/status", nil)

		srv.routes().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp SystemStatus
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Basic validation
		if resp.DeviceID != "test-device-001" {
			t.Error("Expected device ID test-device-001")
		}
		if resp.Time.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})
}