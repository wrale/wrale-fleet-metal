package server

import (
	"encoding/json"
	"net/http"
	"time"
)

// SystemStatus represents the complete system status
type SystemStatus struct {
	DeviceID    string    `json:"device_id"`
	Time        time.Time `json:"time"`
	Health      struct {
		Power struct {
			BatteryLevel float64 `json:"battery_level"`
			Charging     bool    `json:"charging"`
			Voltage     float64 `json:"voltage"`
		} `json:"power"`
		Thermal struct {
			CPUTemp     float64 `json:"cpu_temp"`
			GPUTemp     float64 `json:"gpu_temp"`
			AmbientTemp float64 `json:"ambient_temp"`
			FanSpeed    int     `json:"fan_speed"`
			Throttled   bool    `json:"throttled"`
		} `json:"thermal"`
		Security struct {
			CaseOpen      bool `json:"case_open"`
			MotionDetected bool `json:"motion_detected"`
			VoltageNormal  bool `json:"voltage_normal"`
		} `json:"security"`
	} `json:"health"`
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// handleStatus returns complete system status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mux.RLock()
	defer s.mux.RUnlock()

	// Get current state from subsystems
	powerState := s.power.GetState()
	thermalState := s.thermal.GetState()
	securityState := s.security.GetState()

	// Build response
	status := SystemStatus{
		DeviceID: s.deviceID,
		Time:     time.Now(),
	}

	// Fill power status
	status.Health.Power.BatteryLevel = powerState.BatteryLevel
	status.Health.Power.Charging = powerState.Charging
	status.Health.Power.Voltage = powerState.Voltage

	// Fill thermal status 
	status.Health.Thermal.CPUTemp = thermalState.CPUTemp
	status.Health.Thermal.GPUTemp = thermalState.GPUTemp
	status.Health.Thermal.AmbientTemp = thermalState.AmbientTemp
	status.Health.Thermal.FanSpeed = thermalState.FanSpeed
	status.Health.Thermal.Throttled = thermalState.Throttled

	// Fill security status
	status.Health.Security.CaseOpen = securityState.CaseOpen
	status.Health.Security.MotionDetected = securityState.MotionDetected
	status.Health.Security.VoltageNormal = securityState.VoltageNormal

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}