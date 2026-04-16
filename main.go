// tglys — TG Traffic Light Volume Control
//
// A cross-platform background application that polls a remote endpoint for a
// traffic-light state ("red", "yellow", "green") and adjusts the system
// master volume accordingly.
//
// The app runs with a native GUI: a system-tray icon shows the current state,
// and the Settings window lets you view and edit all configuration values.
//
// Configuration is read from an .env file in the working directory and can
// also be updated at runtime via the Settings window.
//
// Usage:
//
//	tglys
package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/VidarTessem/tglys/checker"
	"github.com/VidarTessem/tglys/config"
	"github.com/VidarTessem/tglys/gui"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("")

	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load .env: %v (using defaults)", err)
	}

	log.Println("=== tglys — TG Traffic Light Volume Control ===")
	if cfg.URL == "" {
		log.Println("note: TGLYS_URL is not set — configure it via the Settings window or .env")
	} else {
		log.Printf("polling URL : %s", cfg.URL)
	}
	log.Printf("poll interval: %ds | red=%d%% yellow=%d%% green=%d%%",
		cfg.PollInterval, cfg.RedVolume, cfg.YellowVolume, cfg.GreenVolume)

	// Shared, guarded config.
	var mu sync.RWMutex
	currentCfg := cfg

	getCfg := func() config.Config {
		mu.RLock()
		defer mu.RUnlock()
		return currentCfg
	}

	ch := checker.New(cfg)

	setCfg := func(newCfg config.Config) error {
		if err := config.Save(newCfg); err != nil {
			return fmt.Errorf("writing .env: %w", err)
		}
		mu.Lock()
		currentCfg = newCfg
		mu.Unlock()
		ch.UpdateConfig(newCfg)
		return nil
	}

	// Start background polling in a goroutine.
	// The GUI (Fyne) takes over the main goroutine via gui.Run().
	go ch.Run(context.Background())

	// gui.Run blocks until the window is closed / Quit is selected.
	gui.Run(ch, getCfg, setCfg)
}

