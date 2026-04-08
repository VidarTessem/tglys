// Package config handles reading and writing the application configuration
// from/to an .env file on disk.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const defaultEnvFile = ".env"

// Config holds all runtime settings for tglys.
type Config struct {
	// URL to poll for traffic-light JSON.
	URL string
	// PollInterval is the number of seconds between polls.
	PollInterval int
	// RedVolume is the system volume (0–100) to set when trafikklys == "red".
	RedVolume int
	// YellowVolume is the system volume (0–100) to set when trafikklys == "yellow".
	YellowVolume int
	// GreenVolume is the system volume (0–100) to set when trafikklys == "green".
	GreenVolume int
	// AdminPort is the TCP port on which the web admin panel listens.
	AdminPort int
}

// Default returns a Config pre-filled with sensible defaults.
func Default() Config {
	return Config{
		URL:          "",
		PollInterval: 30,
		RedVolume:    0,
		YellowVolume: 50,
		GreenVolume:  100,
		AdminPort:    8765,
	}
}

// Load reads the .env file and merges values into a Config, falling back to
// defaults for any key that is missing or empty.
func Load() (Config, error) {
	cfg := Default()

	// Attempt to load the env file; it's OK if the file doesn't exist yet.
	if err := godotenv.Load(defaultEnvFile); err != nil && !os.IsNotExist(err) {
		return cfg, fmt.Errorf("loading .env: %w", err)
	}

	if v := os.Getenv("TGLYS_URL"); v != "" {
		cfg.URL = v
	}
	if v := os.Getenv("TGLYS_POLL_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.PollInterval = n
		}
	}
	if v := os.Getenv("TGLYS_RED_VOLUME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RedVolume = clamp(n, 0, 100)
		}
	}
	if v := os.Getenv("TGLYS_YELLOW_VOLUME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.YellowVolume = clamp(n, 0, 100)
		}
	}
	if v := os.Getenv("TGLYS_GREEN_VOLUME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GreenVolume = clamp(n, 0, 100)
		}
	}
	if v := os.Getenv("TGLYS_ADMIN_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.AdminPort = n
		}
	}

	return cfg, nil
}

// Save writes cfg back to the .env file, overwriting the previous content.
func Save(cfg Config) error {
	lines := []string{
		fmt.Sprintf("TGLYS_URL=%s", cfg.URL),
		fmt.Sprintf("TGLYS_POLL_INTERVAL=%d", cfg.PollInterval),
		fmt.Sprintf("TGLYS_RED_VOLUME=%d", cfg.RedVolume),
		fmt.Sprintf("TGLYS_YELLOW_VOLUME=%d", cfg.YellowVolume),
		fmt.Sprintf("TGLYS_GREEN_VOLUME=%d", cfg.GreenVolume),
		fmt.Sprintf("TGLYS_ADMIN_PORT=%d", cfg.AdminPort),
	}
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(defaultEnvFile, []byte(content), 0600)
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
