package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"ariadne/engine"
)

func main() {
	// Flags (minimal P5 CLI)
	var (
		seedList       string
		seedFile       string
		resume         bool
		checkpointPath string
		snapshotEvery  time.Duration
		showVersion    bool
	)

	flag.StringVar(&seedList, "seeds", "", "Comma separated list of seed URLs")
	flag.StringVar(&seedFile, "seed-file", "", "Path to file containing one seed URL per line")
	flag.BoolVar(&resume, "resume", false, "Resume from existing checkpoint (skip already processed URLs)")
	flag.StringVar(&checkpointPath, "checkpoint", "checkpoint.log", "Path to checkpoint log file")
	flag.DurationVar(&snapshotEvery, "snapshot-interval", 10*time.Second, "Interval between progress snapshots (0=disabled)")
	flag.BoolVar(&showVersion, "version", false, "Show version / build info")
	flag.Parse()

	if showVersion {
		fmt.Println("ariadne engine CLI (facade mode) – phase-3 experimental")
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

	eng, err := engine.New(cfg)
	if err != nil {
		log.Fatalf("create engine: %v", err)
	}
	defer func() { _ = eng.Stop() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		log.Println("signal received; initiating graceful shutdown...")
		cancel()
		// second signal forces exit
		<-sigCh
		log.Println("second signal received; forcing exit")
		os.Exit(1)
	}()

	results, err := eng.Start(ctx, seeds)
	if err != nil {
		log.Fatalf("start engine: %v", err)
	}

	// Snapshot ticker
	var ticker *time.Ticker
	if snapshotEvery > 0 {
		ticker = time.NewTicker(snapshotEvery)
		defer ticker.Stop()
	}

	// Result consumption
	done := make(chan struct{})
	go func() {
		enc := json.NewEncoder(os.Stdout)
		for r := range results {
			// Stream results as JSON lines
			if err := enc.Encode(r); err != nil {
				log.Printf("encode result: %v", err)
			}
		}
		close(done)
	}()

	// Snapshot loop
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
	// Final snapshot (best-effort)
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
		defer f.Close()
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
	// de-duplicate while preserving order
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
