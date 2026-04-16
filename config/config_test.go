package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClamp(t *testing.T) {
	if clamp(-5, 0, 100) != 0 {
		t.Error("clamp below min")
	}
	if clamp(105, 0, 100) != 100 {
		t.Error("clamp above max")
	}
	if clamp(50, 0, 100) != 50 {
		t.Error("clamp in range")
	}
}

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.RedVolume != 0 {
		t.Errorf("default red = %d, want 0", cfg.RedVolume)
	}
	if cfg.YellowVolume != 50 {
		t.Errorf("default yellow = %d, want 50", cfg.YellowVolume)
	}
	if cfg.GreenVolume != 100 {
		t.Errorf("default green = %d, want 100", cfg.GreenVolume)
	}
	if cfg.PollInterval != 30 {
		t.Errorf("default poll = %d, want 30", cfg.PollInterval)
	}
}

func TestLoadFromEnvVars(t *testing.T) {
	// Set env vars (no .env file needed).
	t.Setenv("TGLYS_URL", "http://test.example/api")
	t.Setenv("TGLYS_POLL_INTERVAL", "60")
	t.Setenv("TGLYS_RED_VOLUME", "5")
	t.Setenv("TGLYS_YELLOW_VOLUME", "55")
	t.Setenv("TGLYS_GREEN_VOLUME", "99")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.URL != "http://test.example/api" {
		t.Errorf("URL = %q", cfg.URL)
	}
	if cfg.PollInterval != 60 {
		t.Errorf("PollInterval = %d", cfg.PollInterval)
	}
	if cfg.RedVolume != 5 {
		t.Errorf("RedVolume = %d", cfg.RedVolume)
	}
	if cfg.YellowVolume != 55 {
		t.Errorf("YellowVolume = %d", cfg.YellowVolume)
	}
	if cfg.GreenVolume != 99 {
		t.Errorf("GreenVolume = %d", cfg.GreenVolume)
	}
}

func TestLoadClampVolumes(t *testing.T) {
	t.Setenv("TGLYS_RED_VOLUME", "-10")
	t.Setenv("TGLYS_GREEN_VOLUME", "200")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.RedVolume != 0 {
		t.Errorf("expected clamp to 0, got %d", cfg.RedVolume)
	}
	if cfg.GreenVolume != 100 {
		t.Errorf("expected clamp to 100, got %d", cfg.GreenVolume)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Write and read back from a temporary directory.
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir) //nolint:errcheck

	// Confirm .env file was written to the temp dir.
	envPath := filepath.Join(dir, ".env")

	want := Config{
		URL:          "http://save.example/api",
		PollInterval: 15,
		RedVolume:    10,
		YellowVolume: 60,
		GreenVolume:  90,
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Fatal(".env file was not created")
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error: %v", err)
	}

	if got.URL != want.URL {
		t.Errorf("URL: got %q want %q", got.URL, want.URL)
	}
	if got.PollInterval != want.PollInterval {
		t.Errorf("PollInterval: got %d want %d", got.PollInterval, want.PollInterval)
	}
	if got.RedVolume != want.RedVolume {
		t.Errorf("RedVolume: got %d want %d", got.RedVolume, want.RedVolume)
	}
	if got.YellowVolume != want.YellowVolume {
		t.Errorf("YellowVolume: got %d want %d", got.YellowVolume, want.YellowVolume)
	}
	if got.GreenVolume != want.GreenVolume {
		t.Errorf("GreenVolume: got %d want %d", got.GreenVolume, want.GreenVolume)
	}
}
