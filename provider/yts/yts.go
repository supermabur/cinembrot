package yts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/scraper"
)

var DefaultMirrors = []string{
	"https://yts.lt/api/v2",
	"https://yts.bz/api/v2",
	"https://yts.ag/api/v2",
	"https://yts.am/api/v2",
	"https://yts.mx/api/v2",
}

// Client handles communication with YTS / YIFY Movies REST API
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
	baseURL    string
}

// NewClient initializes a new YTS API client
func NewClient(cfg *config.Config) *Client {
	timeout := 15
	if cfg != nil && cfg.ScraperTimeoutSec > 0 {
		timeout = cfg.ScraperTimeoutSec
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	return &Client{
		cfg:     cfg,
		baseURL: DefaultMirrors[0],
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   time.Duration(timeout) * time.Second,
		},
	}
}

// SetBaseURL allows overriding the default YTS endpoint with a mirror or proxy
func (c *Client) SetBaseURL(u string) {
	if strings.TrimSpace(u) != "" {
		c.baseURL = strings.TrimRight(strings.TrimSpace(u), "/")
	}
}

type ListMoviesResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Data          struct {
		MovieCount int        `json:"movie_count"`
		Limit      int        `json:"limit"`
		PageNumber int        `json:"page_number"`
		Movies     []YTSMovie `json:"movies"`
	} `json:"data"`
}

type YTSMovie struct {
	ID                      int64        `json:"id"`
	URL                     string       `json:"url"`
	IMDbCode                string       `json:"imdb_code"`
	Title                   string       `json:"title"`
	TitleEnglish            string       `json:"title_english"`
	TitleLong               string       `json:"title_long"`
	Slug                    string       `json:"slug"`
	Year                    int          `json:"year"`
	Rating                  float64      `json:"rating"`
	Runtime                 int          `json:"runtime"`
	Genres                  []string     `json:"genres"`
	Summary                 string       `json:"summary"`
	DescriptionFull         string       `json:"description_full"`
	Synopsis                string       `json:"synopsis"`
	YTTrailerCode           string       `json:"yt_trailer_code"`
	Language                string       `json:"language"`
	MPARating               string       `json:"mpa_rating"`
	BackgroundImage         string       `json:"background_image"`
	BackgroundImageOriginal string       `json:"background_image_original"`
	SmallCoverImage         string       `json:"small_cover_image"`
	MediumCoverImage        string       `json:"medium_cover_image"`
	LargeCoverImage         string       `json:"large_cover_image"`
	Torrents                []YTSTorrent `json:"torrents"`
}

type YTSTorrent struct {
	URL        string `json:"url"`
	Hash       string `json:"hash"`
	Quality    string `json:"quality"`
	Type       string `json:"type"`
	Seeds      int    `json:"seeds"`
	Peers      int    `json:"peers"`
	Size       string `json:"size"`
	SizeBytes  int64  `json:"size_bytes"`
	VideoCodec string `json:"video_codec"`
}

// FetchMoviesByYear fetches top movies released in a specific year from YTS API
func (c *Client) FetchMoviesByYear(year int, limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("list_movies.json?query_term=%d&limit=%d&page=%d&sort_by=download_count&order_by=desc",
		year, limit, page)

	return c.fetchAndParseMovies(endpoint, year)
}

// FetchLatestMovies fetches recently added movies from YTS API
func (c *Client) FetchLatestMovies(limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("list_movies.json?limit=%d&page=%d&sort_by=date_added&order_by=desc",
		limit, page)

	return c.fetchAndParseMovies(endpoint, 0)
}

// SearchMovies searches for movies matching title or term from YTS API
func (c *Client) SearchMovies(query string, limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	endpoint := fmt.Sprintf("list_movies.json?query_term=%s&limit=%d&page=%d&sort_by=seeds&order_by=desc",
		url.QueryEscape(query), limit, page)

	return c.fetchAndParseMovies(endpoint, 0)
}

// Helper to execute GET request and convert YTS JSON to model.Movie
func (c *Client) fetchAndParseMovies(pathQuery string, targetYear int) ([]model.Movie, error) {
	// Build list of candidate endpoints (configured baseURL first, then other mirrors)
	candidates := []string{c.baseURL}
	for _, m := range DefaultMirrors {
		if m != c.baseURL {
			candidates = append(candidates, m)
		}
	}

	var lastErr error
	for _, base := range candidates {
		fullURL := fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), strings.TrimLeft(pathQuery, "/"))

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			lastErr = err
			continue
		}

		userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
		if c.cfg != nil && c.cfg.ScraperUserAgent != "" {
			userAgent = c.cfg.ScraperUserAgent
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("YTS API returned HTTP %d", resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		var ytsResp ListMoviesResponse
		if err := json.Unmarshal(body, &ytsResp); err != nil {
			lastErr = fmt.Errorf("failed to parse YTS JSON from %s: %w", base, err)
			continue
		}

		if ytsResp.Status != "ok" || len(ytsResp.Data.Movies) == 0 {
			return nil, nil
		}

		var results []model.Movie
		for _, ym := range ytsResp.Data.Movies {
			if targetYear > 0 && ym.Year != targetYear {
				continue
			}
			if strings.TrimSpace(ym.Title) == "" {
				continue
			}

			m := c.toModelMovie(&ym)
			if m != nil {
				results = append(results, *m)
			}
		}

		// Update c.baseURL to working mirror for subsequent calls
		c.baseURL = base
		return results, nil
	}

	if lastErr != nil {
		if strings.Contains(lastErr.Error(), "internetsehat") || strings.Contains(lastErr.Error(), "x509") || strings.Contains(lastErr.Error(), "No such host") {
			return nil, fmt.Errorf("akses ke domain YTS diblokir oleh ISP lokal / DNS. Gunakan VPN atau atur URL Mirror/Proxy YTS di menu 'Sumber Website' CMS")
		}
		return nil, fmt.Errorf("semua mirror YTS gagal diakses: %w", lastErr)
	}

	return nil, nil
}

// toModelMovie transforms YTSMovie to CINEMBROT's model.Movie
func (c *Client) toModelMovie(ym *YTSMovie) *model.Movie {
	slug := ym.Slug
	if slug == "" {
		slug = scraper.Slugify(fmt.Sprintf("%s-%d", ym.Title, ym.Year))
	}

	synopsis := ym.DescriptionFull
	if synopsis == "" {
		synopsis = ym.Synopsis
	}
	if synopsis == "" {
		synopsis = ym.Summary
	}
	synopsis = scraper.CleanHTMLToPlainText(synopsis)

	poster := ym.LargeCoverImage
	if poster == "" {
		poster = ym.MediumCoverImage
	}

	backdrop := ym.BackgroundImageOriginal
	if backdrop == "" {
		backdrop = ym.BackgroundImage
	}

	var trailerURL string
	if ym.YTTrailerCode != "" {
		trailerURL = fmt.Sprintf("https://www.youtube.com/watch?v=%s", ym.YTTrailerCode)
	}

	// Build Genres
	var genres []model.Genre
	for _, gName := range ym.Genres {
		cleanG := strings.TrimSpace(gName)
		if cleanG != "" {
			genres = append(genres, model.Genre{
				Name: cleanG,
				Slug: scraper.Slugify(cleanG),
			})
		}
	}

	movie := &model.Movie{
		Title:           ym.Title,
		OriginalTitle:   ym.TitleEnglish,
		Slug:            slug,
		Year:            ym.Year,
		Rating:          ym.Rating,
		DurationMinutes: ym.Runtime,
		Country:         "United States",
		Language:        ym.Language,
		Quality:         "HD / 1080p / 4K",
		PosterURL:       poster,
		BackdropURL:     backdrop,
		ThumbnailURL:    ym.SmallCoverImage,
		TrailerURL:      trailerURL,
		Synopsis:        synopsis,
		IsFree:          false,
		IsLegal:         false,
		LicenseType:     "Commercial (YTS Torrent)",
		SourceWebsite:   "yts.mx",
		SourceURL:       fmt.Sprintf("https://yts.mx/movies/%s", slug),
		Genres:          genres,
	}

	// Build Torrent Download Links
	now := time.Now()
	var downloadLinks []model.DownloadLink

	for _, t := range ym.Torrents {
		cleanQuality := strings.ToUpper(t.Quality)
		res := "1920x1080"
		if strings.Contains(t.Quality, "720") {
			res = "1280x720"
		} else if strings.Contains(t.Quality, "2160") || strings.Contains(t.Quality, "4k") {
			res = "3840x2160"
		}

		// Download URL format: Direct .torrent file download or Magnet Link
		downloadURL := t.URL
		if downloadURL == "" && t.Hash != "" {
			downloadURL = fmt.Sprintf("https://yts.mx/torrent/download/%s", t.Hash)
		}

		// Generate BitTorrent Magnet Link URI
		trackers := []string{
			"udp://open.demonii.com:1337/announce",
			"udp://tracker.openbittorrent.com:6969/announce",
			"udp://tracker.coppersurfer.tk:6969/announce",
			"udp://glotorrents.pw:6969/announce",
			"udp://tracker.opentrackr.org:1337/announce",
		}
		magnetURI := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", t.Hash, url.QueryEscape(ym.Title))
		for _, tr := range trackers {
			magnetURI += "&tr=" + url.QueryEscape(tr)
		}

		// 1. Direct .torrent File Link
		downloadLinks = append(downloadLinks, model.DownloadLink{
			Provider:       "YTS Torrent File",
			URL:            downloadURL,
			Quality:        fmt.Sprintf("%s (%s)", cleanQuality, strings.ToUpper(t.Type)),
			Resolution:     res,
			Format:         fmt.Sprintf(".torrent (%s)", t.VideoCodec),
			FileSize:       t.Size,
			IsValid:        true,
			Status:         "ACTIVE",
			LastCheckedAt:  &now,
		})

		// 2. Direct Magnet Link
		downloadLinks = append(downloadLinks, model.DownloadLink{
			Provider:       "YTS Magnet Link",
			URL:            magnetURI,
			Quality:        fmt.Sprintf("%s (Magnet)", cleanQuality),
			Resolution:     res,
			Format:         "Magnet URI",
			FileSize:       t.Size,
			IsValid:        true,
			Status:         "ACTIVE",
			LastCheckedAt:  &now,
		})
	}

	movie.DownloadLinks = downloadLinks
	return movie
}
