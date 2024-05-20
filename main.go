package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io" // Add the io package
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/scrapemate"
	"github.com/gosom/scrapemate/adapters/writers/jsonwriter"
	"github.com/gosom/scrapemate/scrapemateapp"
)

type arguments struct {
	concurrency              int
	cacheDir                 string
	maxDepth                 int
	resultsFile              string
	json                     bool
	langCode                 string
	debug                    bool
	dsn                      string
	produceOnly              bool
	exitOnInactivityDuration time.Duration
	email                    bool
}

// Result represents the structure of the scraping result
type Result struct {
	Data interface{} `json:"data"`
}

// scrapeHandler is the HTTP handler for the scraping API
func scrapeHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "Missing query parameter", http.StatusBadRequest)
		return
	}

	// Create a context with a longer timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Define your scraping arguments
	args := arguments{
		concurrency:              4,
		maxDepth:                 10,
		json:                     true,
		langCode:                 "en",
		exitOnInactivityDuration: 40 * time.Second,
	}

	// Create a channel to receive the result
	resultChan := make(chan Result)
	errorChan := make(chan error)

	// Run the scraping in a separate goroutine
	go func() {
		result, err := runFromString(ctx, &args, query)
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	// Wait for the result and send the response as soon as it's ready
	select {
	case result := <-resultChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	case err := <-errorChan:
		http.Error(w, fmt.Sprintf("Failed to scrape: %v", err), http.StatusInternalServerError)
	case <-ctx.Done():
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	}
}

// runFromString scrapes data based on the input query string
func runFromString(ctx context.Context, args *arguments, inputString string) (Result, error) {
	input := strings.NewReader(inputString)

	// Create a results writer
	resultsWriter := &strings.Builder{}

	jsonWriter := jsonwriter.NewJSONWriter(resultsWriter)

	writers := []scrapemate.ResultWriter{jsonWriter}

	opts := []func(*scrapemateapp.Config) error{
		scrapemateapp.WithConcurrency(args.concurrency),
		scrapemateapp.WithExitOnInactivity(args.exitOnInactivityDuration),
		scrapemateapp.WithJS(scrapemateapp.DisableImages()),
	}

	cfg, err := scrapemateapp.NewConfig(writers, opts...)
	if err != nil {
		return Result{}, err
	}

	app, err := scrapemateapp.NewScrapeMateApp(cfg)
	if err != nil {
		return Result{}, err
	}

	seedJobs, err := createSeedJobs(args.langCode, input, args.maxDepth, args.email)
	if err != nil {
		return Result{}, err
	}

	err = app.Start(ctx, seedJobs...)
	if err != nil {
		return Result{}, err
	}

	return Result{Data: resultsWriter.String()}, nil
}

// createSeedJobs creates seed jobs from the input query
func createSeedJobs(langCode string, r io.Reader, maxDepth int, email bool) ([]scrapemate.IJob, error) {
	scanner := bufio.NewScanner(r)
	var jobs []scrapemate.IJob

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

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/scrape", scrapeHandler)

	http.ListenAndServe(":8080", r)
}
