// Package server provides the main server implementation for fleet-metal
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/wrale/wrale-fleet-metal-hw/pkg/gpio"
	"github.com/wrale/wrale-fleet-metal-hw/pkg/power"
	"github.com/wrale/wrale-fleet-metal-hw/pkg/secure"
	"github.com/wrale/wrale-fleet-metal-hw/pkg/thermal"
)

// Server represents the main fleet-metal server
type Server struct {
	mux sync.RWMutex

	// Hardware subsystems
	gpio     *gpio.Controller
	power    *power.Manager
	security *secure.Manager
	thermal  *thermal.Monitor

	// Server configuration
	deviceID   string
	httpServer *http.Server
	started    bool
}

// Config holds the server configuration
type Config struct {
	DeviceID string
	HTTPAddr string
}

// New creates a new server instance
func New(cfg Config) (*Server, error) {
	if cfg.DeviceID == "" {
		return nil, fmt.Errorf("device ID is required")
	}

	// Initialize GPIO controller first
	gpioCtrl, err := gpio.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GPIO: %w", err)
	}

	// Initialize power management
	powerMgr, err := power.New(power.Config{
		GPIO: gpioCtrl,
		DeviceID: cfg.DeviceID,
		OnPowerCritical: func(state power.PowerState) {
			log.Printf("CRITICAL: Power state critical - battery: %.1f%%, voltage: %.1fV",
				state.BatteryLevel, state.Voltage)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize power management: %w", err)
	}

	// Initialize thermal management
	thermalMgr, err := thermal.New(thermal.Config{
		GPIO: gpioCtrl,
		DeviceID: cfg.DeviceID,
		OnWarning: func(state thermal.ThermalState) {
			log.Printf("WARNING: High temperature - CPU: %.1f°C, GPU: %.1f°C",
				state.CPUTemp, state.GPUTemp)
		},
		OnCritical: func(state thermal.ThermalState) {
			log.Printf("CRITICAL: Temperature critical - CPU: %.1f°C, GPU: %.1f°C",
				state.CPUTemp, state.GPUTemp)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize thermal management: %w", err)
	}

	// Initialize security management
	securityMgr, err := secure.New(secure.Config{
		GPIO: gpioCtrl,
		DeviceID: cfg.DeviceID,
		OnTamper: func(state secure.TamperState) {
			log.Printf("ALERT: Security tamper detected - case: %v, motion: %v",
				state.CaseOpen, state.MotionDetected)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security management: %w", err)
	}

	// Create server instance
	s := &Server{
		deviceID: cfg.DeviceID,
		gpio:     gpioCtrl,
		power:    powerMgr,
		thermal:  thermalMgr,
		security: securityMgr,
	}

	// Initialize HTTP server
	s.httpServer = &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: s.routes(),
	}

	return s, nil
}

// Run starts the server and blocks until context is canceled
func (s *Server) Run(ctx context.Context) error {
	s.mux.Lock()
	if s.started {
		s.mux.Unlock()
		return fmt.Errorf("server already started")
	}
	s.started = true
	s.mux.Unlock()

	// Start subsystem monitoring
	go func() {
		if err := s.power.Monitor(ctx); err != nil {
			log.Printf("Power monitoring error: %v", err)
		}
	}()

	go func() {
		if err := s.thermal.Monitor(ctx); err != nil {
			log.Printf("Thermal monitoring error: %v", err)
		}
	}()

	go func() {
		if err := s.security.Monitor(ctx); err != nil {
			log.Printf("Security monitoring error: %v", err)
		}
	}()

	// Start HTTP server
	go func() {
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shutting down HTTP server: %w", err)
	}

	return nil
}

// routes sets up the HTTP routing
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// API routes
	mux.HandleFunc("/api/v1/status", s.handleStatus)

	return mux
}