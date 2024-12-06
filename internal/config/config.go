package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the complete server configuration
type Config struct {
	// Core settings
	DeviceID string
	Location string
	LogLevel string

	// Server settings
	HTTPAddr string
	TLSCert  string
	TLSKey   string

	// Hardware settings
	GPIOConfig    GPIOConfig
	PowerConfig   PowerConfig
	ThermalConfig ThermalConfig
}

// GPIOConfig holds GPIO-related settings
type GPIOConfig struct {
	FanPin        string
	CaseSensor    string
	MotionSensor  string
	VoltageSensor string
}

// PowerConfig holds power management settings
type PowerConfig struct {
	BatteryADCPath string
	VoltageADCPath string
	CurrentADCPath string
	WarnLevel      float64
	CriticalLevel  float64
}

// ThermalConfig holds thermal management settings
type ThermalConfig struct {
	CPUTempPath      string
	GPUTempPath      string
	AmbientTempPath  string
	FanThreshold     float64
	WarnThreshold    float64
	CriticalThreshold float64
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		// Core settings with defaults
		DeviceID: getEnvOrDefault("WRALE_DEVICE_ID", ""),
		Location: getEnvOrDefault("WRALE_LOCATION", ""),
		LogLevel: getEnvOrDefault("WRALE_LOG_LEVEL", "info"),

		// Server settings
		HTTPAddr: getEnvOrDefault("WRALE_HTTP_ADDR", ":8080"),
		TLSCert:  getEnvOrDefault("WRALE_TLS_CERT", ""),
		TLSKey:   getEnvOrDefault("WRALE_TLS_KEY", ""),

		// Hardware settings
		GPIOConfig: GPIOConfig{
			FanPin:        getEnvOrDefault("WRALE_GPIO_FAN_PIN", "GPIO18"),
			CaseSensor:    getEnvOrDefault("WRALE_GPIO_CASE_SENSOR", "GPIO17"),
			MotionSensor:  getEnvOrDefault("WRALE_GPIO_MOTION_SENSOR", "GPIO27"),
			VoltageSensor: getEnvOrDefault("WRALE_GPIO_VOLTAGE_SENSOR", "GPIO22"),
		},

		PowerConfig: PowerConfig{
			BatteryADCPath: getEnvOrDefault("WRALE_POWER_BATTERY_ADC", "/sys/bus/iio/devices/iio:device0"),
			VoltageADCPath: getEnvOrDefault("WRALE_POWER_VOLTAGE_ADC", "/sys/bus/iio/devices/iio:device1"),
			CurrentADCPath: getEnvOrDefault("WRALE_POWER_CURRENT_ADC", "/sys/bus/iio/devices/iio:device2"),
			WarnLevel:      getEnvFloatOrDefault("WRALE_POWER_WARN_LEVEL", 20.0),
			CriticalLevel:  getEnvFloatOrDefault("WRALE_POWER_CRITICAL_LEVEL", 10.0),
		},

		ThermalConfig: ThermalConfig{
			CPUTempPath:       getEnvOrDefault("WRALE_THERMAL_CPU_PATH", "/sys/class/thermal/thermal_zone0/temp"),
			GPUTempPath:       getEnvOrDefault("WRALE_THERMAL_GPU_PATH", "/sys/class/thermal/thermal_zone1/temp"),
			AmbientTempPath:   getEnvOrDefault("WRALE_THERMAL_AMBIENT_PATH", "/sys/class/thermal/thermal_zone2/temp"),
			FanThreshold:      getEnvFloatOrDefault("WRALE_THERMAL_FAN_THRESHOLD", 60.0),
			WarnThreshold:     getEnvFloatOrDefault("WRALE_THERMAL_WARN_THRESHOLD", 70.0),
			CriticalThreshold: getEnvFloatOrDefault("WRALE_THERMAL_CRITICAL_THRESHOLD", 80.0),
		},
	}

	// Validate required fields
	if config.DeviceID == "" {
		return nil, fmt.Errorf("WRALE_DEVICE_ID is required")
	}

	return config, nil
}

// Helper functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloatOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}