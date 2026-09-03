package omdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cinembrot/config"
	"cinembrot/model"
)

const BaseURL = "https://www.omdbapi.com/"

// FreekeysOMDbPool contains OMDb API key pool from rickylawson/freekeys
var FreekeysOMDbPool = []string{
	"4b447405",
	"eb0c0475",
	"7776cbde",
	"ff28f90b",
	"6c3a2d45",
	"b07b58c8",
	"ad04b643",
	"a95b5205",
	"777d9323",
	"2c2c3314",
	"b5cff164",
	"89a9f57d",
	"73a9858a",
	"efbd8357",
}

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

// GetAPIKey returns user configured key or rotates from freekeys pool
func (c *Client) GetAPIKey() string {
	if c.cfg.OMDBAPIKey != "" {
		return c.cfg.OMDBAPIKey
	}
	return FreekeysOMDbPool[0]
}

type RatingItem struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
}

type MovieResponse struct {
	Title      string       `json:"Title"`
	Year       string       `json:"Year"`
	Rated      string       `json:"Rated"`
	Released   string       `json:"Released"`
	Runtime    string       `json:"Runtime"`
	Genre      string       `json:"Genre"`
	Director   string       `json:"Director"`
	Writer     string       `json:"Writer"`
	Actors     string       `json:"Actors"`
	Plot       string       `json:"Plot"`
	Language   string       `json:"Language"`
	Country    string       `json:"Country"`
	Awards     string       `json:"Awards"`
	Poster     string       `json:"Poster"`
	Ratings    []RatingItem `json:"Ratings"`
	Metascore  string       `json:"Metascore"`
	IMDbRating string       `json:"imdbRating"`
	IMDbVotes  string       `json:"imdbVotes"`
	IMDbID     string       `json:"imdbID"`
	Type       string       `json:"Type"`
	Response   string       `json:"Response"`
	Error      string       `json:"Error"`
}

// FetchByTitle fetches ratings & metadata by title and year
func (c *Client) FetchByTitle(title string, year int) (*MovieResponse, error) {
	apiKey := c.GetAPIKey()

	queryURL := fmt.Sprintf("%s?apikey=%s&t=%s&plot=full", BaseURL, apiKey, url.QueryEscape(title))
	if year > 0 {
		queryURL += fmt.Sprintf("&y=%d", year)
	}

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result MovieResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Response == "False" {
		return nil, fmt.Errorf("OMDb API error: %s", result.Error)
	}

	return &result, nil
}

// EnrichMovie adds IMDb / Rotten Tomatoes ratings to an existing movie model
func (c *Client) EnrichMovie(movie *model.Movie) error {
	omdbData, err := c.FetchByTitle(movie.Title, movie.Year)
	if err != nil {
		return err
	}

	if rating, err := strconv.ParseFloat(omdbData.IMDbRating, 64); err == nil {
		movie.IMDbRating = rating
		if movie.Rating == 0 {
			movie.Rating = rating
		}
	}

	votesClean := strings.ReplaceAll(omdbData.IMDbVotes, ",", "")
	if votes, err := strconv.Atoi(votesClean); err == nil {
		movie.IMDbVotes = votes
	}

	if movie.AgeRating == "" && omdbData.Rated != "N/A" {
		movie.AgeRating = omdbData.Rated
	}

	if movie.Country == "" && omdbData.Country != "N/A" {
		movie.Country = omdbData.Country
	}

	return nil
}
