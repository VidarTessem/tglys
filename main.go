// tglys — TG Traffic Light Volume Control
//
// A cross-platform background service that polls a remote endpoint for a
// traffic-light state ("red", "yellow", "green") and adjusts the system
// master volume accordingly.
//
// Configuration is read from an .env file in the working directory, and can
// also be updated at runtime via the built-in web admin panel.
//
// Usage:
//
//	tglys [--admin-port PORT]
//
// The admin panel is served on http://localhost:<TGLYS_ADMIN_PORT> (default 8765).
// Stop the process with Ctrl-C (SIGINT/SIGTERM).
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/VidarTessem/tglys/admin"
	"github.com/VidarTessem/tglys/checker"
	"github.com/VidarTessem/tglys/config"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("")

	cfg, err := config.Load()
	if err != nil {
		log.Printf("warning: could not load .env: %v (using defaults)", err)
	}

	// Allow --admin-port flag override.
	args := os.Args[1:]
	for i, arg := range args {
		if (arg == "--admin-port" || arg == "-admin-port") && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				cfg.AdminPort = n
			}
		}
	}

	log.Println("=== tglys — TG Traffic Light Volume Control ===")
	if cfg.URL == "" {
		log.Println("note: TGLYS_URL is not set — configure it via the admin panel or .env")
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start background polling.
	go ch.Run(ctx)

	// Start admin web panel.
	addr := fmt.Sprintf(":%d", getCfg().AdminPort)
	h := admin.Handler(ch, getCfg, setCfg)
	go func() {
		if err := admin.Serve(addr, h); err != nil {
			log.Printf("[admin] server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down…")
}
