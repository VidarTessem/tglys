package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/VidarTessem/tglys/checker"
	"github.com/VidarTessem/tglys/config"
)

func TestAdminPanel_Status(t *testing.T) {
	cfg := config.Config{
		URL:          "http://example.com/api",
		PollInterval: 30,
		RedVolume:    0,
		YellowVolume: 50,
		GreenVolume:  100,
		AdminPort:    8765,
	}
	ch := checker.New(cfg)

	h := Handler(ch, func() config.Config { return cfg }, func(c config.Config) error {
		cfg = c
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp statusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.URL != "http://example.com/api" {
		t.Errorf("URL = %q", resp.URL)
	}
	if resp.PollInterval != 30 {
		t.Errorf("PollInterval = %d", resp.PollInterval)
	}
	if resp.GreenVolume != 100 {
		t.Errorf("GreenVolume = %d", resp.GreenVolume)
	}
}

func TestAdminPanel_Index(t *testing.T) {
	cfg := config.Default()
	ch := checker.New(cfg)
	h := Handler(ch, func() config.Config { return cfg }, func(c config.Config) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html, got %q", ct)
	}
}

func TestAdminPanel_NotFound(t *testing.T) {
	cfg := config.Default()
	ch := checker.New(cfg)
	h := Handler(ch, func() config.Config { return cfg }, func(c config.Config) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/notexist", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAdminPanel_SaveConfig(t *testing.T) {
	cfg := config.Default()
	ch := checker.New(cfg)

	var savedCfg config.Config
	h := Handler(ch, func() config.Config { return cfg }, func(c config.Config) error {
		savedCfg = c
		cfg = c
		return nil
	})

	form := url.Values{
		"url":          {"http://new.example/api"},
		"poll_interval": {"15"},
		"red_volume":   {"5"},
		"yellow_volume": {"45"},
		"green_volume":  {"95"},
		"admin_port":   {"9000"},
	}
	req := httptest.NewRequest(http.MethodPost, "/api/config",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	if savedCfg.URL != "http://new.example/api" {
		t.Errorf("URL = %q", savedCfg.URL)
	}
	if savedCfg.PollInterval != 15 {
		t.Errorf("PollInterval = %d", savedCfg.PollInterval)
	}
	if savedCfg.RedVolume != 5 {
		t.Errorf("RedVolume = %d", savedCfg.RedVolume)
	}
	if savedCfg.YellowVolume != 45 {
		t.Errorf("YellowVolume = %d", savedCfg.YellowVolume)
	}
	if savedCfg.GreenVolume != 95 {
		t.Errorf("GreenVolume = %d", savedCfg.GreenVolume)
	}
	if savedCfg.AdminPort != 9000 {
		t.Errorf("AdminPort = %d", savedCfg.AdminPort)
	}
}

func TestAdminPanel_ConfigMethodNotAllowed(t *testing.T) {
	cfg := config.Default()
	ch := checker.New(cfg)
	h := Handler(ch, func() config.Config { return cfg }, func(c config.Config) error { return nil })

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}
