// Package server provides the integration layer for wrale-fleet-metal
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/wrale/wrale-fleet-metal-hw/pkg/gpio"
	"github.com/wrale/wrale-fleet-metal-hw/pkg/power"
	"github.com/wrale/wrale-fleet-metal-hw/pkg/thermal"
	"github.com/wrale/wrale-fleet-metal-core/pkg/state"
	"github.com/wrale/wrale-fleet-metal-diag/pkg/diagnostics"
)

// Server represents the main fleet-metal integration layer
type Server struct {
	mux sync.RWMutex

	// Core configuration
	deviceID string
	httpAddr string

	// Hardware layer
	gpio     *gpio.Controller
	power    *power.Manager
	thermal  *thermal.Monitor

	// Core system layer
	state    *state.Manager
	diag     *diagnostics.Manager

	// HTTP server
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

	// Initialize hardware layer
	gpioCtrl, err := gpio.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GPIO: %w", err)
	}

	// Initialize managers
	powerMgr, err := power.New(power.Config{
		GPIO: gpioCtrl,
		DeviceID: cfg.DeviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize power manager: %w", err)
	}

	thermalMgr, err := thermal.New(thermal.Config{
		GPIO: gpioCtrl,
		DeviceID: cfg.DeviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize thermal manager: %w", err)
	}

	// Initialize core system layer
	stateMgr, err := state.New(state.Config{
		DeviceID: cfg.DeviceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Initialize diagnostics
	diagMgr, err := diagnostics.New(diagnostics.Config{
		DeviceID: cfg.DeviceID,
		GPIO: gpioCtrl,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize diagnostics: %w", err)
	}

	// Create server instance
	s := &Server{
		deviceID:   cfg.DeviceID,
		httpAddr:   cfg.HTTPAddr,
		gpio:       gpioCtrl,
		power:      powerMgr,
		thermal:    thermalMgr,
		state:      stateMgr,
		diag:       diagMgr,
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

	// Start subsystems
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
		if err := s.state.Run(ctx); err != nil {
			log.Printf("State manager error: %v", err)
		}
	}()

	go func() {
		if err := s.diag.Run(ctx); err != nil {
			log.Printf("Diagnostics error: %v", err)
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
