package gmaps

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/gosom/scrapemate"
	"github.com/playwright-community/playwright-go"
)

type GmapJob struct {
	scrapemate.Job

	MaxDepth     int
	LangCode     string
	ExtractEmail bool
}

func NewGmapJob(id, langCode, query string, maxDepth int, extractEmail bool) *GmapJob {
	query = url.QueryEscape(query)

	const (
		maxRetries = 3
		prio       = scrapemate.PriorityLow
	)

	if id == "" {
		id = uuid.New().String()
	}

	job := GmapJob{
		Job: scrapemate.Job{
			ID:         id,
			Method:     http.MethodGet,
			URL:        "https://www.google.com/maps/search/" + query,
			URLParams:  map[string]string{"hl": langCode},
			MaxRetries: maxRetries,
			Priority:   prio,
		},
		MaxDepth:     maxDepth,
		LangCode:     langCode,
		ExtractEmail: extractEmail,
	}

	return &job
}

func (j *GmapJob) UseInResults() bool {
	return false
}

func (j *GmapJob) Process(ctx context.Context, resp *scrapemate.Response) (any, []scrapemate.IJob, error) {
	defer func() {
		resp.Document = nil
		resp.Body = nil
	}()

	log := scrapemate.GetLoggerFromContext(ctx)

	doc, ok := resp.Document.(*goquery.Document)
	if !ok {
		return nil, nil, fmt.Errorf("could not convert to goquery document")
	}

	var next []scrapemate.IJob

	if strings.Contains(resp.URL, "/maps/place/") {
		placeJob := NewPlaceJob(j.ID, j.LangCode, resp.URL, j.ExtractEmail)
		next = append(next, placeJob)
	} else {
		doc.Find(`div[role=feed] div[jsaction]>a`).Each(func(i int, s *goquery.Selection) {
			if href := s.AttrOr("href", ""); href != "" {
				nextJob := NewPlaceJob(j.ID, j.LangCode, href, j.ExtractEmail)
				next = append(next, nextJob)
			}
		})
	}

	log.Info(fmt.Sprintf("%d places found", len(next)))

	return nil, next, nil
}

func (j *GmapJob) BrowserActions(ctx context.Context, page playwright.Page) scrapemate.Response {
	var resp scrapemate.Response

	// Navigate to the initial search URL
	pageResponse, err := page.Goto(j.GetFullURL(), playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})

	if err != nil {
		resp.Error = err
		return resp
	}

	// Handle the cookie consent if required
	if err = clickRejectCookiesIfRequired(page); err != nil {
		resp.Error = err
		return resp
	}

	// Wait for the load state to ensure redirection is complete
	const defaultTimeout = 10000 // Increase timeout to handle redirection
	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		Timeout: playwright.Float(defaultTimeout),
	})

	if err != nil {
		resp.Error = err
		return resp
	}

	// Retry mechanism to check for redirection
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		finalURL := page.URL()
		if strings.Contains(finalURL, "/maps/place/") {
			resp.URL = finalURL
			resp.StatusCode = pageResponse.Status()
			resp.Headers = make(http.Header, len(pageResponse.Headers()))
			for k, v := range pageResponse.Headers() {
				resp.Headers.Add(k, v)
			}

			// Extract the page content for the place
			body, err := page.Content()
			if err != nil {
				resp.Error = err
				return resp
			}

			resp.Body = []byte(body)
			return resp
		}

		// Wait for a while before checking again
		page.WaitForTimeout(2000) // Wait for 2 seconds before retrying
	}

	resp.Error = fmt.Errorf("unexpected URL after navigation: %s", page.URL())
	return resp
}

func clickRejectCookiesIfRequired(page playwright.Page) error {
	sel := `form[action="https://consent.google.com/save"]:first-of-type button:first-of-type`

	const timeout = 500

	el, err := page.WaitForSelector(sel, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(timeout),
	})

	if err != nil {
		return nil
	}

	if el == nil {
		return nil
	}

	return el.Click()
}

func scroll(ctx context.Context, page playwright.Page, maxDepth int) (int, error) {
	scrollSelector := `div[role='feed']`
	expr := `async () => {
		const el = document.querySelector("` + scrollSelector + `");
		el.scrollTop = el.scrollHeight;

		return new Promise((resolve, reject) => {
  			setTimeout(() => {
    		resolve(el.scrollHeight);
  			}, %d);
		});
	}`

	var currentScrollHeight int
	waitTime := 100.
	cnt := 0

	const (
		timeout  = 500
		maxWait2 = 2000
	)

	for i := 0; i < maxDepth; i++ {
		cnt++
		waitTime2 := timeout * cnt

		if waitTime2 > timeout {
			waitTime2 = maxWait2
		}

		scrollHeight, err := page.Evaluate(fmt.Sprintf(expr, waitTime2))
		if err != nil {
			return cnt, err
		}

		height, ok := scrollHeight.(int)
		if !ok {
			return cnt, fmt.Errorf("scrollHeight is not an int")
		}

		if height == currentScrollHeight {
			break
		}

		currentScrollHeight = height

		select {
		case <-ctx.Done():
			return currentScrollHeight, nil
		default:
		}

		waitTime *= 1.5

		if waitTime > maxWait2 {
			waitTime = maxWait2
		}

		page.WaitForTimeout(waitTime)
	}

	return cnt, nil
}
