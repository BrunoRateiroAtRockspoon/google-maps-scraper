package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
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
	"github.com/sirupsen/logrus"
)

func main() {
	if os.Getenv("PLAYWRIGHT_INSTALL_ONLY") == "1" {
		if err := playwright.Install(); err != nil {
			logrus.Fatal(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	go func() {
		http.ListenAndServe(":6060", nil)
	}()

	http.HandleFunc("/scrape", scrapeHandler)
	logrus.Info("Starting server on :8080")
	http.ListenAndServe(":8080", nil)
}

func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	// Create a channel to signal when the request is done
	done := make(chan struct{})

	go func() {
		defer close(done) // Ensure the channel is closed when the request is done

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel() // Ensure the context is canceled when the request is done

		query := r.URL.Query().Get("query")
		if query == "" {
			http.Error(w, "query parameter is required", http.StatusBadRequest)
			return
		}

		args := getScrapeArgs()
		job := createJob(args.langCode, strings.TrimSpace(query), args.maxDepth, args.email)

		writers := []scrapemate.ResultWriter{
			jsonwriter.NewJSONWriter(w),
		}
		w.Header().Set("Content-Type", "application/json")

		app, err := newScrapeApp(args, writers)
		if err != nil {
			httpError(w, err)
			return
		}

		// Start the job and ensure cleanup
		logrus.Infof("Starting job for query: %s", query)
		if err := app.Start(ctx, job); err != nil {
			httpError(w, err)
		} else {
			logrus.Infof("Finished job for query: %s", query)
		}

		// Close the ScrapemateApp instance and associated resources
		if err := app.Close(); err != nil {
			logrus.Errorf("Failed to close ScrapemateApp: %v", err)
		}

		logrus.Infof("Number of Go Routines: %d", runtime.NumGoroutine())
	}()

	// Wait for the goroutine to finish before returning
	<-done
}

func getScrapeArgs() arguments {
	return arguments{
		concurrency:              1,
		maxDepth:                 1,
		langCode:                 "en",
		exitOnInactivityDuration: 20 * time.Second, // Increase inactivity duration
	}
}

func createJob(langCode, query string, maxDepth int, email bool) scrapemate.IJob {
	var id string
	if before, after, ok := strings.Cut(query, "#!#"); ok {
		query = strings.TrimSpace(before)
		id = strings.TrimSpace(after)
	}
	return gmaps.NewGmapJob(id, langCode, query, maxDepth, email)
}

func newScrapeApp(args arguments, writers []scrapemate.ResultWriter) (*scrapemateapp.ScrapemateApp, error) {
	opts := []func(*scrapemateapp.Config) error{
		scrapemateapp.WithConcurrency(args.concurrency),
		scrapemateapp.WithExitOnInactivity(args.exitOnInactivityDuration),
		scrapemateapp.WithJS(scrapemateapp.DisableImages()),
	}

	cfg, err := scrapemateapp.NewConfig(writers, opts...)
	if err != nil {
		return nil, err
	}

	return scrapemateapp.NewScrapeMateApp(cfg)
}

func httpError(w http.ResponseWriter, err error) {
	logrus.Error(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

type arguments struct {
	concurrency              int
	maxDepth                 int
	langCode                 string
	exitOnInactivityDuration time.Duration
	email                    bool
}
