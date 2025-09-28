package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/99souls/ariadne/engine"
)

// simpleJSONConfig represents a minimal subset of engine.Config loadable from a JSON file.
// Experimental: Placeholder until layered config system is implemented.
type simpleJSONConfig struct {
	DiscoveryWorkers  *int           `json:"discovery_workers"`
	ExtractionWorkers *int           `json:"extraction_workers"`
	ProcessingWorkers *int           `json:"processing_workers"`
	OutputWorkers     *int           `json:"output_workers"`
	BufferSize        *int           `json:"buffer_size"`
	RetryBaseDelay    *time.Duration `json:"retry_base_delay"`
	RetryMaxDelay     *time.Duration `json:"retry_max_delay"`
	RetryMaxAttempts  *int           `json:"retry_max_attempts"`
}

func applySimpleConfig(base engine.Config, sc *simpleJSONConfig) engine.Config {
	if sc == nil {
		return base
	}
	if sc.DiscoveryWorkers != nil {
		base.DiscoveryWorkers = *sc.DiscoveryWorkers
	}
	if sc.ExtractionWorkers != nil {
		base.ExtractionWorkers = *sc.ExtractionWorkers
	}
	if sc.ProcessingWorkers != nil {
		base.ProcessingWorkers = *sc.ProcessingWorkers
	}
	if sc.OutputWorkers != nil {
		base.OutputWorkers = *sc.OutputWorkers
	}
	if sc.BufferSize != nil {
		base.BufferSize = *sc.BufferSize
	}
	if sc.RetryBaseDelay != nil {
		base.RetryBaseDelay = *sc.RetryBaseDelay
	}
	if sc.RetryMaxDelay != nil {
		base.RetryMaxDelay = *sc.RetryMaxDelay
	}
	if sc.RetryMaxAttempts != nil {
		base.RetryMaxAttempts = *sc.RetryMaxAttempts
	}
	return base
}

func main() {
	var (
		seedList       string
		seedFile       string
		resume         bool
		checkpointPath string
		snapshotEvery  time.Duration
		showVersion    bool
		metricsAddr    string
		healthAddr     string
		configPath     string
		metricsBackend string
		enableMetrics  bool
	)
	flag.StringVar(&seedList, "seeds", "", "Comma separated list of seed URLs")
	flag.StringVar(&seedFile, "seed-file", "", "Path to file containing one seed URL per line")
	flag.BoolVar(&resume, "resume", false, "Resume from existing checkpoint (skip already processed URLs)")
	flag.StringVar(&checkpointPath, "checkpoint", "checkpoint.log", "Path to checkpoint log file")
	flag.DurationVar(&snapshotEvery, "snapshot-interval", 10*time.Second, "Interval between progress snapshots (0=disabled)")
	flag.BoolVar(&showVersion, "version", false, "Show version / build info")
	flag.StringVar(&metricsAddr, "metrics", "", "Expose metrics on address (e.g. :9090)")
	flag.StringVar(&healthAddr, "health", "", "Expose health endpoint on address (e.g. :9091)")
	flag.StringVar(&configPath, "config", "", "Optional JSON config file (temporary minimal format)")
	flag.StringVar(&metricsBackend, "metrics-backend", "prom", "Metrics backend: prom|otel|noop (effective only if -metrics set and enabled)")
	flag.BoolVar(&enableMetrics, "enable-metrics", false, "Enable metrics provider (required to serve metrics)")
	flag.Parse()

	if showVersion {
		fmt.Println("ariadne CLI – engine module hard-cut edition")
		return
	}

	seeds, err := gatherSeeds(seedList, seedFile)
	if err != nil {
		log.Fatalf("collect seeds: %v", err)
	}
	if len(seeds) == 0 {
		fmt.Println("No seeds provided. Use -seeds or -seed-file. Example: -seeds https://example.com,https://example.org")
		os.Exit(1)
	}

	cfg := engine.Defaults()
	cfg.Resume = resume
	cfg.CheckpointPath = checkpointPath

	// Merge simple config file if provided
	if configPath != "" {
		f, err := os.Open(configPath)
		if err != nil { log.Fatalf("open config: %v", err) }
		var sc simpleJSONConfig
		if err := json.NewDecoder(f).Decode(&sc); err != nil { log.Fatalf("decode config: %v", err) }
		_ = f.Close()
		cfg = applySimpleConfig(cfg, &sc)
	}

	if enableMetrics {
		cfg.MetricsEnabled = true
		cfg.MetricsBackend = metricsBackend
	}
	cfg.CheckpointPath = checkpointPath

	eng, err := engine.New(cfg)
	if err != nil {
		log.Fatalf("create engine: %v", err)
	}
	defer func() { _ = eng.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		log.Println("signal received; initiating graceful shutdown...")
		cancel()
		<-sigCh
		log.Println("second signal received; forcing exit")
		os.Exit(1)
	}()

	results, err := eng.Start(ctx, seeds)
	if err != nil {
		log.Fatalf("start engine: %v", err)
	}

	// Metrics endpoint (basic stub until adapter wiring); only if enabled & address provided.
	if metricsAddr != "" && cfg.MetricsEnabled {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// Minimal placeholder exposition; real impl will integrate provider registry.
			_, _ = w.Write([]byte("# HELP ariadne_build_info Build info metric placeholder\n# TYPE ariadne_build_info gauge\nariadne_build_info 1\n"))
		})
		go func() {
			srv := &http.Server{Addr: metricsAddr, Handler: mux}
			<-ctx.Done()
			_ = srv.Shutdown(context.Background())
		}()
		go func() {
			log.Printf("metrics listening on %s (backend=%s)", metricsAddr, cfg.MetricsBackend)
			_ = http.ListenAndServe(metricsAddr, mux)
		}()
	}

	if healthAddr != "" {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			hs := eng.HealthSnapshot(r.Context())
			_ = json.NewEncoder(w).Encode(map[string]any{"status": hs.Overall, "probes": hs.Probes, "generated": hs.Generated, "ttl": hs.TTL.Seconds()})
		})
		go func() {
			srv := &http.Server{Addr: healthAddr, Handler: mux}
			<-ctx.Done()
			_ = srv.Shutdown(context.Background())
		}()
		go func() {
			log.Printf("health endpoint listening on %s", healthAddr)
			_ = http.ListenAndServe(healthAddr, mux)
		}()
	}

	var ticker *time.Ticker
	if snapshotEvery > 0 {
		ticker = time.NewTicker(snapshotEvery)
		defer ticker.Stop()
	}

	done := make(chan struct{})
	go func() {
		enc := json.NewEncoder(os.Stdout)
		for r := range results {
			if err := enc.Encode(r); err != nil {
				log.Printf("encode result: %v", err)
			}
		}
		close(done)
	}()

	if ticker != nil {
		go func() {
			for {
				select {
				case <-ticker.C:
					snap := eng.Snapshot()
					b, _ := json.MarshalIndent(snap, "", "  ")
					fmt.Fprintf(os.Stderr, "\n=== SNAPSHOT %s ===\n%s\n", time.Now().Format(time.RFC3339), string(b))
				case <-done:
					return
				}
			}
		}()
	}

	<-done
	final := eng.Snapshot()
	b, _ := json.MarshalIndent(final, "", "  ")
	fmt.Fprintf(os.Stderr, "\n=== FINAL SNAPSHOT %s ===\n%s\n", time.Now().Format(time.RFC3339), string(b))
}

func gatherSeeds(seedList, seedFile string) ([]string, error) {
	seeds := []string{}
	if seedList != "" {
		for _, s := range strings.Split(seedList, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				seeds = append(seeds, s)
			}
		}
	}
	if seedFile != "" {
		f, err := os.Open(seedFile)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				seeds = append(seeds, line)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	seen := make(map[string]struct{}, len(seeds))
	out := make([]string, 0, len(seeds))
	for _, s := range seeds {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}
