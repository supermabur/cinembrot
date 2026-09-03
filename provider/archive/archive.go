package archive

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/scraper"
)

// Client handles requests to Internet Archive Public API
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.ScraperTimeoutSec) * time.Second,
		},
	}
}

type SearchResponse struct {
	Response struct {
		NumFound int `json:"numFound"`
		Start    int `json:"start"`
		Docs     []struct {
			Identifier  string      `json:"identifier"`
			Title       string      `json:"title"`
			Description interface{} `json:"description"` // Can be string or []string
			Year        interface{} `json:"year"`        // Can be int or string
			Downloads   int64       `json:"downloads"`
			PublicDate  string      `json:"publicdate"`
		} `json:"docs"`
	} `json:"response"`
}

type MetadataResponse struct {
	Server   string `json:"server"`
	Dir      string `json:"dir"`
	Metadata struct {
		Identifier  string      `json:"identifier"`
		Title       string      `json:"title"`
		Description interface{} `json:"description"`
		Year        interface{} `json:"year"`
		Date        string      `json:"date"`
		Creator     interface{} `json:"creator"`
		Director    interface{} `json:"director"`
		Runtime     string      `json:"runtime"`
		LicenseURL  string      `json:"licenseurl"`
		Mediatype   string      `json:"mediatype"`
	} `json:"metadata"`
	Files []struct {
		Name   string `json:"name"`
		Format string `json:"format"`
		Size   string `json:"size"`
		Length string `json:"length"`
		Height string `json:"height"`
		Width  string `json:"width"`
	} `json:"files"`
}

// FetchFeatureFilms fetches public domain feature films from Internet Archive
func (c *Client) FetchFeatureFilms(limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	searchURL := fmt.Sprintf(
		"https://archive.org/advancedsearch.php?q=collection:(feature_films)+AND+mediatype:(movies)&fl[]=identifier,title,description,year,downloads,publicdate&sort[]=downloads+desc&rows=%d&page=%d&output=json",
		limit, page,
	)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.ScraperUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("archive.org search failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive.org returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResult SearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse archive.org response: %w", err)
	}

	var movies []model.Movie
	for _, doc := range searchResult.Response.Docs {
		movie, err := c.FetchMovieByIdentifier(doc.Identifier)
		if err != nil {
			log.Printf("[WARN] Failed to fetch metadata for %s: %v\n", doc.Identifier, err)
			continue
		}
		if movie != nil {
			movie.Views = doc.Downloads
			movies = append(movies, *movie)
		}
	}

	return movies, nil
}

// FetchFeatureFilmsByYear fetches public domain feature films for a specific year from Internet Archive
func (c *Client) FetchFeatureFilmsByYear(year int, limit int, page int) ([]model.Movie, error) {
	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}

	searchURL := fmt.Sprintf(
		"https://archive.org/advancedsearch.php?q=collection:(feature_films)+AND+mediatype:(movies)+AND+year:%d&fl[]=identifier,title,description,year,downloads,publicdate&sort[]=downloads+desc&rows=%d&page=%d&output=json",
		year, limit, page,
	)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.ScraperUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("archive.org search by year failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive.org returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResult SearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to parse archive.org response: %w", err)
	}

	var movies []model.Movie
	for _, doc := range searchResult.Response.Docs {
		movie, err := c.FetchMovieByIdentifier(doc.Identifier)
		if err != nil {
			log.Printf("[WARN] Failed to fetch metadata for %s: %v\n", doc.Identifier, err)
			continue
		}
		if movie != nil {
			movie.Views = doc.Downloads
			movies = append(movies, *movie)
		}
	}

	return movies, nil
}

// FetchMovieByIdentifier extracts video download links, embeds, and info from Archive.org metadata
func (c *Client) FetchMovieByIdentifier(identifier string) (*model.Movie, error) {
	metaURL := fmt.Sprintf("https://archive.org/metadata/%s", url.PathEscape(identifier))

	req, err := http.NewRequest("GET", metaURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.ScraperUserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("archive.org metadata returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var meta MetadataResponse
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}

	title := cleanString(meta.Metadata.Title)
	if title == "" {
		title = identifier
	}

	synopsis := extractDescription(meta.Metadata.Description)
	year := extractYear(meta.Metadata.Year, meta.Metadata.Date)

	posterURL := fmt.Sprintf("https://archive.org/services/img/%s", identifier)
	embedURL := fmt.Sprintf("https://archive.org/embed/%s", identifier)
	sourceURL := fmt.Sprintf("https://archive.org/details/%s", identifier)

	var downloadLinks []model.DownloadLink
	var streamLinks []model.StreamLink

	// Add primary embed player
	streamLinks = append(streamLinks, model.StreamLink{
		Provider:   "Internet Archive Player",
		ServerName: "Official Archive Stream",
		Quality:    "Auto",
		EmbedURL:   embedURL,
		DirectURL:  sourceURL,
	})

	// Process files for downloadable and streamable videos
	for _, f := range meta.Files {
		formatLower := strings.ToLower(f.Format)
		nameLower := strings.ToLower(f.Name)

		// Filter for primary video formats
		if strings.Contains(formatLower, "mp4") || strings.Contains(formatLower, "mkv") ||
			strings.Contains(formatLower, "mpeg4") || strings.Contains(formatLower, "h.264") ||
			strings.HasSuffix(nameLower, ".mp4") || strings.HasSuffix(nameLower, ".mkv") {

			// Skip thumbnails or sub-audio files
			if strings.Contains(nameLower, "thumb") || strings.Contains(nameLower, "_ia_thumb") || strings.HasSuffix(nameLower, ".gif") {
				continue
			}

			quality, resolution := detectQuality(f.Name, f.Format, f.Height)
			sizeFormatted := formatBytes(f.Size)
			fileURL := fmt.Sprintf("https://archive.org/download/%s/%s", identifier, url.PathEscape(f.Name))

			format := "MP4"
			if strings.HasSuffix(nameLower, ".mkv") {
				format = "MKV"
			}

			downloadLinks = append(downloadLinks, model.DownloadLink{
				Provider:   "Internet Archive",
				Quality:    quality,
				Resolution: resolution,
				FileSize:   sizeFormatted,
				Format:     format,
				URL:        fileURL,
			})
		}
	}

	// Directors
	var directors []model.Director
	directorName := cleanString(fmt.Sprintf("%v", meta.Metadata.Director))
	if directorName == "" || directorName == "<nil>" {
		directorName = cleanString(fmt.Sprintf("%v", meta.Metadata.Creator))
	}
	if directorName != "" && directorName != "<nil>" {
		directors = append(directors, model.Director{
			Name: directorName,
			Slug: scraper.Slugify(directorName),
		})
	}

	licenseURL := cleanString(meta.Metadata.LicenseURL)
	if licenseURL == "" {
		licenseURL = "https://creativecommons.org/publicdomain/mark/1.0/"
	}

	movie := &model.Movie{
		Title:             title,
		OriginalTitle:     title,
		Slug:              fmt.Sprintf("%s-%s", scraper.Slugify(title), identifier),
		Type:              "movie",
		Status:            "released",
		Synopsis:          synopsis,
		Year:              year,
		PosterURL:         posterURL,
		BackdropURL:       posterURL,
		Quality:           "HD (Public Domain)",
		AgeRating:         "SU / All Ages",
		IsLegal:           true,
		IsFree:            true,
		LicenseType:       "Public Domain",
		LicenseName:       "Public Domain Mark 1.0 / Open Access",
		LicenseURL:        licenseURL,
		SourceWebsite:     "archive.org",
		SourceURL:         sourceURL,
		Directors:         directors,
		DownloadLinks:     downloadLinks,
		StreamLinks:       streamLinks,
		Genres: []model.Genre{
			{Name: "Classic", Slug: "classic"},
			{Name: "Public Domain", Slug: "public-domain"},
		},
		RawMetadata: string(body),
	}

	return movie, nil
}

func cleanString(s string) string {
	return strings.TrimSpace(s)
}

func extractDescription(desc interface{}) string {
	if desc == nil {
		return ""
	}
	var raw string
	switch v := desc.(type) {
	case string:
		raw = strings.TrimSpace(v)
	case []interface{}:
		var parts []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				parts = append(parts, str)
			}
		}
		raw = strings.Join(parts, "\n\n")
	default:
		raw = fmt.Sprintf("%v", desc)
	}

	return scraper.CleanHTMLToPlainText(raw)
}

func extractYear(yearVal, dateVal interface{}) int {
	if yearVal != nil {
		if yStr, ok := yearVal.(string); ok {
			if y, err := strconv.Atoi(yStr); err == nil && y > 1800 {
				return y
			}
		} else if yInt, ok := yearVal.(float64); ok {
			return int(yInt)
		}
	}
	if dateVal != nil {
		if dStr, ok := dateVal.(string); ok && len(dStr) >= 4 {
			if y, err := strconv.Atoi(dStr[:4]); err == nil && y > 1800 {
				return y
			}
		}
	}
	return 0
}

func detectQuality(filename, format, height string) (string, string) {
	lower := strings.ToLower(filename + " " + format)
	if strings.Contains(lower, "1080p") || height == "1080" {
		return "1080p Full HD", "1920x1080"
	}
	if strings.Contains(lower, "720p") || height == "720" {
		return "720p HD", "1280x720"
	}
	if strings.Contains(lower, "480p") || strings.Contains(lower, "dvd") || height == "480" {
		return "480p SD", "854x480"
	}
	if strings.Contains(lower, "512kb") || strings.Contains(lower, "360p") {
		return "360p Low", "640x360"
	}
	return "Standard HD", "HD"
}

func formatBytes(sizeStr string) string {
	if sizeStr == "" {
		return "N/A"
	}
	bytes, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return sizeStr
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
