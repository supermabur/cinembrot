package scraper

import (
	"log"
	"net/http"
	"time"

	"cinembrot/config"
	"github.com/gocolly/colly/v2"
)

// Engine manages colly collectors and scraper lifecycle
type Engine struct {
	Config *config.Config
	Repo   *Repository
}

// NewEngine creates a new scraper engine instance
func NewEngine(cfg *config.Config, repo *Repository) *Engine {
	return &Engine{
		Config: cfg,
		Repo:   repo,
	}
}

// CreateCollector instantiates a configured Colly collector with user-agent, timeouts, and rate limits
func (e *Engine) CreateCollector(domainFilter ...string) *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent(e.Config.ScraperUserAgent),
		colly.Async(true),
	)

	// Set request timeout
	c.SetRequestTimeout(time.Duration(e.Config.ScraperTimeoutSec) * time.Second)

	// Set Rate Limiting & Concurrency
	limitRule := &colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: e.Config.ScraperMaxThreads,
		Delay:       time.Duration(e.Config.ScraperDelayMs) * time.Millisecond,
		RandomDelay: 500 * time.Millisecond,
	}

	if len(domainFilter) > 0 {
		limitRule.DomainGlob = domainFilter[0]
		c.AllowedDomains = domainFilter
	}

	if err := c.Limit(limitRule); err != nil {
		log.Printf("[WARN] Failed to apply rate limit rule: %v\n", err)
	}

	// Setup standard hooks for logging & debugging
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9,id;q=0.8")
		r.Headers.Set("Sec-Ch-Ua", `"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`)
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		log.Printf("[SCRAPER] Visiting: %s\n", r.URL.String())
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("[ERROR] Request URL: %s failed with response code: %d, error: %v\n",
			r.Request.URL, r.StatusCode, err)
	})

	c.OnResponse(func(r *colly.Response) {
		if r.StatusCode != http.StatusOK {
			log.Printf("[WARN] Response code: %d from %s\n", r.StatusCode, r.Request.URL)
		}
	})

	return c
}
