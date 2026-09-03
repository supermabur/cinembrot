package validator

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cinembrot/model"
	"gorm.io/gorm"
)

var checkClient = &http.Client{
	Timeout: 12 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		return nil
	},
}

// ValidateURL performs lightweight HEAD/GET request to verify if download link is online and valid
func ValidateURL(urlStr string, userAgent string) (isValid bool, httpStatus int, responseTimeMs int64, err error) {
	cleanURL := strings.TrimSpace(urlStr)
	if cleanURL == "" {
		return false, 0, 0, fmt.Errorf("empty URL")
	}

	// BitTorrent Magnet URIs: validate structure without sending HTTP requests
	if strings.HasPrefix(cleanURL, "magnet:?") {
		if strings.Contains(cleanURL, "xt=urn:btih:") {
			return true, 200, 1, nil
		}
		return false, 0, 0, fmt.Errorf("invalid magnet link format")
	}

	// Local CINEMBROT downloads (/downloads/...)
	if strings.HasPrefix(cleanURL, "/downloads/") || strings.HasPrefix(cleanURL, "/") {
		rel := strings.TrimPrefix(cleanURL, "/downloads/")
		rel = strings.TrimPrefix(rel, "/")
		// Primary check: public/download/movie
		p1 := filepath.Join("public", "download", "movie", rel)
		if _, err := os.Stat(p1); err == nil {
			return true, 200, 1, nil
		}
		// Fallback check: C:\CINEMBROT_downloads
		p2 := filepath.Join(`C:\CINEMBROT_downloads`, rel)
		if _, err := os.Stat(p2); err == nil {
			return true, 200, 1, nil
		}
		return false, 404, 1, fmt.Errorf("file local tidak ditemukan di disk: %s", p1)
	}

	start := time.Now()

	// 1. Try HEAD request first (fastest, no body downloaded)
	req, err := http.NewRequest("HEAD", cleanURL, nil)
	if err == nil {
		if userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		resp, headErr := checkClient.Do(req)
		if headErr == nil {
			defer resp.Body.Close()
			duration := time.Since(start).Milliseconds()

			// 200 OK, 206 Partial Content, 302 Found are healthy
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				return true, resp.StatusCode, duration, nil
			}
			// If server rejects HEAD (e.g. 405 Method Not Allowed), fallback to range GET below
			if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusForbidden {
				return false, resp.StatusCode, duration, fmt.Errorf("HTTP status %d", resp.StatusCode)
			}
		}
	}

	// 2. Fallback: Lightweight GET with Range header (bytes=0-0 to avoid downloading file)
	reqGet, err := http.NewRequest("GET", cleanURL, nil)
	if err != nil {
		return false, 0, time.Since(start).Milliseconds(), err
	}
	if userAgent != "" {
		reqGet.Header.Set("User-Agent", userAgent)
	}
	reqGet.Header.Set("Range", "bytes=0-1024")

	respGet, err := checkClient.Do(reqGet)
	if err != nil {
		return false, 0, time.Since(start).Milliseconds(), err
	}
	defer respGet.Body.Close()

	duration := time.Since(start).Milliseconds()
	if (respGet.StatusCode >= 200 && respGet.StatusCode < 400) || respGet.StatusCode == http.StatusPartialContent {
		return true, respGet.StatusCode, duration, nil
	}

	return false, respGet.StatusCode, duration, fmt.Errorf("HTTP status %d", respGet.StatusCode)
}

// ValidateMovieDownloadLinks checks and updates status for all download links of a movie
func ValidateMovieDownloadLinks(links []model.DownloadLink, userAgent string) []model.DownloadLink {
	now := time.Now()
	for i := range links {
		link := &links[i]
		valid, code, dur, err := ValidateURL(link.URL, userAgent)
		link.IsValid = valid
		link.HTTPStatus = code
		link.ResponseTimeMs = dur
		link.LastCheckedAt = &now

		if valid {
			link.Status = "ACTIVE"
		} else {
			link.Status = "DEAD"
			log.Printf("[LINK CHECK] ⚠️ Broken link found: [%s] %s (HTTP %d: %v)\n", link.Provider, link.URL, code, err)
		}
	}
	return links
}

// ValidateAllDatabaseLinks checks download links in MariaDB concurrently and updates health status
func ValidateAllDatabaseLinks(db *gorm.DB, userAgent string, limit int) (checked int, valid int, broken int) {
	var links []model.DownloadLink
	query := db.Model(&model.DownloadLink{})
	if limit > 0 {
		query = query.Limit(limit)
	}
	// Prioritize unchecked or oldest checked links
	query.Order("last_checked_at asc, id desc").Find(&links)

	if len(links) == 0 {
		log.Println("[LINK CHECK] Tidak ada link download yang perlu diperiksa.")
		return 0, 0, 0
	}

	log.Printf("[LINK CHECK] 🔍 Memulai pengecekan %d link download file di database...\n", len(links))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8) // Concurrency limit of 8 parallel checks

	var mu sync.Mutex
	validCount := 0
	brokenCount := 0

	for i := range links {
		link := links[i]
		wg.Add(1)

		go func(l model.DownloadLink) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			isValid, code, dur, err := ValidateURL(l.URL, userAgent)
			now := time.Now()
			status := "ACTIVE"
			if !isValid {
				status = "DEAD"
			}

			// Update in database
			db.Model(&model.DownloadLink{}).Where("id = ?", l.ID).Updates(map[string]interface{}{
				"is_valid":         isValid,
				"status":           status,
				"http_status":      code,
				"response_time_ms": dur,
				"last_checked_at":  &now,
			})

			mu.Lock()
			if isValid {
				validCount++
			} else {
				brokenCount++
				log.Printf("  -> [DEAD ❌] Link ID %d (%s): %s | HTTP %d: %v\n", l.ID, l.Provider, l.URL, code, err)
			}
			mu.Unlock()
		}(link)
	}

	wg.Wait()
	log.Printf("[LINK CHECK] ✅ Pengecekan selesai! Total Diperiksa: %d | Aktif (Valid): %d | Rusak (Dead): %d\n",
		len(links), validCount, brokenCount)

	return len(links), validCount, brokenCount
}
