package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// main - точка входа. Читает параметры командной строки, создает Downloader и запускает процесс.
func main() {
	// CLI flags
	depth := flag.Int("depth", 2, "recursion depth")
	out := flag.String("out", "./downloaded", "output directory")
	workers := flag.Int("workers", 5, "max concurrent downloads")
	sameDomain := flag.Bool("same-domain", true, "only download same domain")
	timeout := flag.Int("timeout", 30, "http client timeout seconds")
	flag.Parse()

	// Проверяем, что URL передан
	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <url>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		os.Exit(2)
	}
	startURL := flag.Arg(0)

	// Создаем выходную директорию
	if err := ensureDir(*out); err != nil {
		log.Fatalf("failed to create output dir: %v", err)
	}

	cfg := Config{
		StartURL:   startURL,
		Depth:      *depth,
		OutputDir:  *out,
		MaxWorkers: *workers,
		SameDomain: *sameDomain,
		TimeoutSec: *timeout,
	}

	// Инициализация Downloader
	d, err := NewDownloader(cfg)
	if err != nil {
		log.Fatalf("init downloader: %v", err)
	}

	log.Printf("Start mirror: %s (depth=%d) -> %s\n", cfg.StartURL, cfg.Depth, cfg.OutputDir)
	if err := d.Start(); err != nil {
		log.Fatalf("download error: %v", err)
	}
	log.Printf("Done. visited: %d\n", d.visited.Size())
}
