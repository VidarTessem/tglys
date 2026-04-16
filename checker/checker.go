// Package checker implements the background polling loop that fetches
// traffic-light state from a remote URL and adjusts system volume accordingly.
package checker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/VidarTessem/tglys/config"
	"github.com/VidarTessem/tglys/volume"
)

// State represents a traffic-light colour.
type State string

const (
	StateRed    State = "red"
	StateYellow State = "yellow"
	StateGreen  State = "green"
	StateUnknown State = "unknown"
)

// trafficResponse is the JSON shape returned by the remote endpoint.
// The array may contain multiple objects; we look for the first one that
// has a "trafikklys" key.
type trafficResponse struct {
	Trafikklys string `json:"trafikklys"`
}

// Checker runs the background polling loop.
type Checker struct {
	mu       sync.RWMutex
	cfg      config.Config
	current  State
	lastErr  error
}

// New creates a new Checker using the supplied configuration.
func New(cfg config.Config) *Checker {
	return &Checker{
		cfg:     cfg,
		current: StateUnknown,
	}
}

// UpdateConfig replaces the running configuration without restarting the loop.
func (c *Checker) UpdateConfig(cfg config.Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cfg = cfg
}

// Current returns the most recently observed traffic-light state.
func (c *Checker) Current() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

// LastError returns the error from the most recent poll attempt, or nil.
func (c *Checker) LastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastErr
}

// Run starts the polling loop and blocks until ctx is cancelled.
func (c *Checker) Run(ctx context.Context) {
	// Poll immediately, then on each tick.
	c.poll()

	c.mu.RLock()
	interval := time.Duration(c.cfg.PollInterval) * time.Second
	c.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.poll()

			// Update ticker if interval changed.
			c.mu.RLock()
			newInterval := time.Duration(c.cfg.PollInterval) * time.Second
			c.mu.RUnlock()
			if newInterval != interval {
				interval = newInterval
				ticker.Reset(interval)
			}
		}
	}
}

// poll performs a single fetch + volume-adjust cycle.
func (c *Checker) poll() {
	c.mu.RLock()
	cfg := c.cfg
	c.mu.RUnlock()

	if cfg.URL == "" {
		c.mu.Lock()
		c.lastErr = fmt.Errorf("TGLYS_URL is not configured")
		c.mu.Unlock()
		log.Println("[checker] skipping poll: URL not configured")
		return
	}

	state, err := fetchState(cfg.URL)
	if err != nil {
		c.mu.Lock()
		c.lastErr = err
		c.mu.Unlock()
		log.Printf("[checker] poll error: %v", err)
		return
	}

	targetVolume := volumeForState(state, cfg)
	if err := volume.Set(targetVolume); err != nil {
		c.mu.Lock()
		c.lastErr = fmt.Errorf("setting volume to %d%%: %w", targetVolume, err)
		c.mu.Unlock()
		log.Printf("[checker] volume error: %v", c.lastErr)
		return
	}

	c.mu.Lock()
	c.current = state
	c.lastErr = nil
	c.mu.Unlock()

	log.Printf("[checker] trafikklys=%s → volume=%d%%", state, targetVolume)
}

// fetchState performs the HTTP GET and extracts the trafikklys value.
// The endpoint may return a JSON object or a JSON array of objects.
func fetchState(url string) (State, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return StateUnknown, fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StateUnknown, fmt.Errorf("unexpected HTTP status %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return StateUnknown, fmt.Errorf("reading response body: %w", err)
	}

	return parseState(body)
}

// parseState extracts the traffic-light state from raw JSON.
// Supports both a single JSON object and a JSON array of objects.
func parseState(body []byte) (State, error) {
	// Try array first.
	var arr []trafficResponse
	if err := json.Unmarshal(body, &arr); err == nil {
		for _, item := range arr {
			if item.Trafikklys != "" {
				return normalise(item.Trafikklys), nil
			}
		}
		return StateUnknown, fmt.Errorf("no trafikklys key found in JSON array")
	}

	// Try single object.
	var obj trafficResponse
	if err := json.Unmarshal(body, &obj); err != nil {
		return StateUnknown, fmt.Errorf("could not parse JSON response: %w", err)
	}
	if obj.Trafikklys == "" {
		return StateUnknown, fmt.Errorf("trafikklys key is missing or empty in JSON object")
	}
	return normalise(obj.Trafikklys), nil
}

func normalise(s string) State {
	switch State(s) {
	case StateRed:
		return StateRed
	case StateYellow:
		return StateYellow
	case StateGreen:
		return StateGreen
	default:
		return StateUnknown
	}
}

func volumeForState(s State, cfg config.Config) int {
	switch s {
	case StateRed:
		return cfg.RedVolume
	case StateYellow:
		return cfg.YellowVolume
	case StateGreen:
		return cfg.GreenVolume
	default:
		return cfg.GreenVolume
	}
}
