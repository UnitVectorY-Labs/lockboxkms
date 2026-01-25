package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	ProjectID string
	Location  string
	KeyRing   string
	Port      string
}

// LoadConfig reads configuration from environment variables
func LoadConfig() Config {
	return Config{
		ProjectID: getEnv("GOOGLE_CLOUD_PROJECT", ""),
		Location:  getEnv("KMS_LOCATION", "us"),
		KeyRing:   getEnv("KMS_KEY_RING", "lockboxkms"),
		Port:      getEnv("PORT", "8080"),
	}
}

// getEnv retrieves environment variables or returns a default value
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
