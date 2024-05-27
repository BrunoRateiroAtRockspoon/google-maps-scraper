package main

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/adapters/writers/jsonwriter"
	"github.com/gosom/scrapemate/scrapemateapp"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/playwright-community/playwright-go"
)

func main() {
	// Check if the environment variable is set to install Playwright
	if os.Getenv("PLAYWRIGHT_INSTALL_ONLY") == "1" {
		if err := installPlaywright(); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	http.HandleFunc("/scrape", scrapeHandler)
	http.ListenAndServe(":8080", nil)
}

func installPlaywright() error {
	return playwright.Install()
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	args := arguments{
		concurrency:              runtime.NumCPU() / 2,
		maxDepth:                 1,
		langCode:                 "en",
		exitOnInactivityDuration: 30 * time.Second,
	}

	queries := r.URL.Query()["query"]
	if len(queries) == 0 {
		http.Error(w, "query parameter is required", http.StatusBadRequest)
		return
	}

	seedJobs, err := createSeedJobs(args.langCode, strings.NewReader(strings.Join(queries, "\n")), args.maxDepth, args.email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	writers := []scrapemate.ResultWriter{
		jsonwriter.NewJSONWriter(w),
	}

	opts := []func(*scrapemateapp.Config) error{
		scrapemateapp.WithConcurrency(args.concurrency),
		scrapemateapp.WithExitOnInactivity(args.exitOnInactivityDuration),
		scrapemateapp.WithJS(scrapemateapp.DisableImages()),
	}

	cfg, err := scrapemateapp.NewConfig(
		writers,
		opts...,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app, err := scrapemateapp.NewScrapeMateApp(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = app.Start(ctx, seedJobs...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func createSeedJobs(langCode string, r io.Reader, maxDepth int, email bool) (jobs []scrapemate.IJob, err error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		query := strings.TrimSpace(scanner.Text())
		if query == "" {
			continue
		}

		var id string

		if before, after, ok := strings.Cut(query, "#!#"); ok {
			query = strings.TrimSpace(before)
			id = strings.TrimSpace(after)
		}

		jobs = append(jobs, gmaps.NewGmapJob(id, langCode, query, maxDepth, email))
	}

	return jobs, scanner.Err()
}

type arguments struct {
	concurrency              int
	maxDepth                 int
	langCode                 string
	exitOnInactivityDuration time.Duration
	email                    bool
}
