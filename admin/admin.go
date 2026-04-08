// Package admin provides a small HTTP server that serves a web-based
// administration panel for configuring and monitoring tglys at runtime.
package admin

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/VidarTessem/tglys/checker"
	"github.com/VidarTessem/tglys/config"
)

//go:embed panel.html
var panelHTML []byte

// statusResponse is the JSON shape returned by GET /api/status.
type statusResponse struct {
	State       string `json:"state"`
	LastError   string `json:"last_error,omitempty"`
	URL         string `json:"url"`
	PollInterval int   `json:"poll_interval"`
	RedVolume   int    `json:"red_volume"`
	YellowVolume int   `json:"yellow_volume"`
	GreenVolume int    `json:"green_volume"`
	AdminPort   int    `json:"admin_port"`
}

// Handler builds an http.ServeMux that exposes the admin panel.
//
//   GET  /               — web panel HTML
//   GET  /api/status     — JSON status + current config
//   POST /api/config     — update config (form-encoded)
func Handler(ch *checker.Checker, getCfg func() config.Config, setCfg func(config.Config) error) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(panelHTML) //nolint:errcheck
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		cfg := getCfg()
		var errStr string
		if err := ch.LastError(); err != nil {
			errStr = err.Error()
		}
		resp := statusResponse{
			State:        string(ch.Current()),
			LastError:    errStr,
			URL:          cfg.URL,
			PollInterval: cfg.PollInterval,
			RedVolume:    cfg.RedVolume,
			YellowVolume: cfg.YellowVolume,
			GreenVolume:  cfg.GreenVolume,
			AdminPort:    cfg.AdminPort,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form data", http.StatusBadRequest)
			return
		}

		cfg := getCfg()

		if v := r.FormValue("url"); v != "" {
			cfg.URL = v
		} else {
			cfg.URL = ""
		}
		if v := r.FormValue("poll_interval"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.PollInterval = n
			}
		}
		if v := r.FormValue("red_volume"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				cfg.RedVolume = clamp(n, 0, 100)
			}
		}
		if v := r.FormValue("yellow_volume"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				cfg.YellowVolume = clamp(n, 0, 100)
			}
		}
		if v := r.FormValue("green_volume"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				cfg.GreenVolume = clamp(n, 0, 100)
			}
		}
		if v := r.FormValue("admin_port"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				cfg.AdminPort = n
			}
		}

		if err := setCfg(cfg); err != nil {
			http.Error(w, fmt.Sprintf("saving config: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	})

	return mux
}

// Serve starts the HTTP server on the given address and blocks until it exits.
func Serve(addr string, h http.Handler) error {
	log.Printf("[admin] web panel available at http://localhost%s", addr)
	return http.ListenAndServe(addr, h)
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
