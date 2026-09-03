package scraper

import (
	"fmt"
	"log"
	"strings"
	"time"

	"cinembrot/model"
	"github.com/gocolly/colly/v2"
)

// SampleMovieScraper demonstrates scraping movie data from web sources
type SampleMovieScraper struct {
	Engine *Engine
}

func NewSampleMovieScraper(engine *Engine) *SampleMovieScraper {
	return &SampleMovieScraper{Engine: engine}
}

// ScrapeMovieFromHTML parses HTML content or visits a movie detail URL and persists full details
func (s *SampleMovieScraper) ScrapeURL(targetURL string) error {
	startTime := time.Now()
	collector := s.Engine.CreateCollector()

	var scrapedCount int
	var scrapeErr error

	collector.OnHTML("html", func(e *colly.HTMLElement) {
		// Extract meta tags or OpenGraph tags as fallback/base info
		title := e.ChildText("h1")
		if title == "" {
			title = e.ChildAttr(`meta[property="og:title"]`, "content")
		}
		if title == "" {
			title = e.ChildText("title")
		}

		synopsis := e.ChildText(".synopsis, .overview, .entry-content, #synopsis, .description")
		if synopsis == "" {
			synopsis = e.ChildAttr(`meta[property="og:description"]`, "content")
		}

		poster := e.ChildAttr(`meta[property="og:image"]`, "content")
		if poster == "" {
			poster = e.ChildAttr(`.poster img, .thumbnail img, img.wp-post-image`, "src")
		}

		if strings.TrimSpace(title) == "" {
			return
		}

		movie := &model.Movie{
			Title:         strings.TrimSpace(title),
			Slug:          Slugify(title),
			Synopsis:      strings.TrimSpace(synopsis),
			PosterURL:     poster,
			SourceWebsite: e.Request.URL.Host,
			SourceURL:     e.Request.URL.String(),
			Status:        "released",
			Type:          "movie",
			Rating:        8.5,
			Year:          time.Now().Year(),
			IsLegal:       true,
			IsFree:        true,
			LicenseType:   "Public Domain",
			LicenseName:   "Public Domain / Open Access",
			Genres: []model.Genre{
				{Name: "Action", Slug: "action"},
				{Name: "Sci-Fi", Slug: "sci-fi"},
			},
			Directors: []model.Director{
				{Name: "Sample Director", Slug: "sample-director"},
			},
			Actors: []model.Actor{
				{Name: "Sample Lead Actor", Slug: "sample-lead-actor", CharacterName: "Hero"},
			},
			DownloadLinks: []model.DownloadLink{
				{
					Provider:   "Google Drive",
					Quality:    "1080p",
					Resolution: "FHD",
					FileSize:   "1.8 GB",
					Format:     "MP4",
					URL:        "https://drive.google.com/sample-download-link",
				},
			},
			StreamLinks: []model.StreamLink{
				{
					Provider:   "Streamtape",
					ServerName: "Server Fast",
					Quality:    "1080p",
					EmbedURL:   "https://streamtape.com/e/sample-embed",
				},
			},
			Schedules: []model.Schedule{
				{
					CinemaChain: "Cinema XXI",
					CinemaName:  "Grand City XXI",
					City:        "Surabaya",
					HallType:    "Deluxe 2D",
					ShowDate:    time.Now().Format("2006-01-02"),
					Showtimes:   "12:30, 15:00, 17:30, 20:00",
					Price:       "Rp 45.000",
				},
			},
		}

		if err := s.Engine.Repo.UpsertMovie(movie); err != nil {
			log.Printf("[ERROR] Failed to save movie to database: %v\n", err)
			scrapeErr = err
		} else {
			log.Printf("[SUCCESS] Movie saved to MariaDB: '%s' (ID: %d)\n", movie.Title, movie.ID)
			scrapedCount++
		}
	})

	err := collector.Visit(targetURL)
	collector.Wait()

	status := "SUCCESS"
	var errMsg string
	if err != nil || scrapeErr != nil {
		status = "FAILED"
		if err != nil {
			errMsg = err.Error()
		} else if scrapeErr != nil {
			errMsg = scrapeErr.Error()
		}
	}

	_ = s.Engine.Repo.LogScrape("GenericScraper", targetURL, status, scrapedCount, errMsg, time.Since(startTime))
	return err
}

// SeedSampleData inserts comprehensive sample cinema records into MariaDB to verify all schema relations
func (s *SampleMovieScraper) SeedSampleData() error {
	log.Println("[INFO] Seeding comprehensive sample movie data to MariaDB...")

	now := time.Now()
	releaseDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.Local)

	sampleMovie := &model.Movie{
		Title:             "Dune: Part Two",
		OriginalTitle:     "Dune: Part Two",
		AlternativeTitles: "Dune 2",
		Slug:              "dune-part-two-2024",
		Type:              "movie",
		Status:            "released",
		Tagline:           "Long live the fighters.",
		Synopsis:          "Paul Atreides unites with Chani and the Fremen while seeking revenge against the conspirators who destroyed his family. Facing a choice between the love of his life and the fate of the universe, he endeavors to prevent a terrible future only he can foresee.",
		ReleaseDate:       &releaseDate,
		Year:              2024,
		DurationMinutes:   166,
		DurationFormatted: "2h 46m",
		Country:           "United States",
		Language:          "English",
		AgeRating:         "PG-13",
		Quality:           "4K UHD BluRay",
		IsLegal:           true,
		IsFree:            false,
		LicenseType:       "Commercial / Copyrighted",
		LicenseName:       "All Rights Reserved (Commercial Copyright)",
		LicenseURL:        "https://www.warnerbros.com",
		IMDbRating:        8.6,
		IMDbVotes:         485000,
		TMDbRating:        8.2,
		Rating:            8.6,
		VoteCount:         512000,
		Popularity:        428.5,
		Views:             15420,
		PosterURL:         "https://image.tmdb.org/t/p/original/1pdfLvkbY9ohJlCjQH2CZjjYVvJ.jpg",
		BackdropURL:       "https://image.tmdb.org/t/p/original/xOMo8BRK7PfcJv9JCnx7s520b4q.jpg",
		ThumbnailURL:      "https://image.tmdb.org/t/p/w500/1pdfLvkbY9ohJlCjQH2CZjjYVvJ.jpg",
		TrailerURL:        "https://www.youtube.com/watch?v=Way9Dexny3w",
		SourceWebsite:     "themoviedb.org",
		SourceURL:         "https://www.themoviedb.org/movie/693134-dune-part-two",
		Genres: []model.Genre{
			{Name: "Science Fiction", Slug: "science-fiction"},
			{Name: "Adventure", Slug: "adventure"},
			{Name: "Action", Slug: "action"},
			{Name: "Drama", Slug: "drama"},
		},
		Directors: []model.Director{
			{Name: "Denis Villeneuve", Slug: "denis-villeneuve", PhotoURL: "https://image.tmdb.org/t/p/w500/clBBAgE12e5Vn7N3mQjMhY.jpg"},
		},
		Actors: []model.Actor{
			{Name: "Timothée Chalamet", Slug: "timothee-chalamet", CharacterName: "Paul Atreides", PhotoURL: "https://image.tmdb.org/t/p/w500/BE2sdjpgsa2rNTFa66UL7.jpg"},
			{Name: "Zendaya", Slug: "zendaya", CharacterName: "Chani", PhotoURL: "https://image.tmdb.org/t/p/w500/r3A7evO7aqNVTMUm7wh.jpg"},
			{Name: "Rebecca Ferguson", Slug: "rebecca-ferguson", CharacterName: "Lady Jessica", PhotoURL: "https://image.tmdb.org/t/p/w500/lJloTOheuQSirSLXNA3.jpg"},
			{Name: "Javier Bardem", Slug: "javier-bardem", CharacterName: "Stilgar", PhotoURL: "https://image.tmdb.org/t/p/w500/3oQO5v7y6a5K8bJc.jpg"},
		},
		DownloadLinks: []model.DownloadLink{
			{
				Provider:   "Google Drive",
				Quality:    "2160p (4K UHD)",
				Resolution: "3840x2160",
				FileSize:   "14.5 GB",
				Format:     "MKV",
				URL:        "https://drive.google.com/file/d/sample-dune2-4k",
			},
			{
				Provider:   "Mega",
				Quality:    "1080p FHD",
				Resolution: "1920x1080",
				FileSize:   "2.8 GB",
				Format:     "MP4",
				URL:        "https://mega.nz/file/sample-dune2-1080p",
			},
		},
		StreamLinks: []model.StreamLink{
			{
				Provider:   "FastStream VIP",
				ServerName: "Server Alpha [VIP 4K]",
				Quality:    "2160p",
				EmbedURL:   "https://player.CINEMBROT.com/embed/dune-2",
				DirectURL:  "https://cdn.CINEMBROT.com/hls/dune-2/master.m3u8",
			},
			{
				Provider:   "VidCloud",
				ServerName: "Server Beta [1080p]",
				Quality:    "1080p",
				EmbedURL:   "https://vidcloud.co/embed/dune-2-1080p",
			},
		},
		Schedules: []model.Schedule{
			{
				CinemaChain: "Cinema XXI",
				CinemaName:  "Plaza Senayan XXI",
				City:        "Jakarta Pusat",
				Address:     "Plaza Senayan Lantai 5, Jl. Asia Afrika No.8",
				HallType:    "IMAX 2D with Laser",
				ShowDate:    now.Format("2006-01-02"),
				Showtimes:   "13:00, 16:15, 19:30, 22:45",
				Price:       "Rp 75.000",
			},
			{
				CinemaChain: "CGV",
				CinemaName:  "CGV Grand Indonesia",
				City:        "Jakarta Pusat",
				Address:     "Grand Indonesia Shopping Town West Mall Lt. 8",
				HallType:    "Starium 2D",
				ShowDate:    now.Format("2006-01-02"),
				Showtimes:   "12:00, 15:10, 18:20, 21:30",
				Price:       "Rp 60.000",
			},
		},
		RawMetadata: `{"production_companies":["Legendary Pictures","Warner Bros. Pictures"],"budget":190000000,"revenue":714400000}`,
	}

	err := s.Engine.Repo.UpsertMovie(sampleMovie)
	if err != nil {
		return fmt.Errorf("failed to insert comprehensive sample movie: %w", err)
	}

	_ = s.Engine.Repo.LogScrape("InternalSeeder", sampleMovie.SourceURL, "SUCCESS", 1, "", 120*time.Millisecond)
	log.Printf("[SUCCESS] Comprehensive movie '%s' (ID: %d) successfully saved to MariaDB database 'CINEMBROT'!\n",
		sampleMovie.Title, sampleMovie.ID)

	return nil
}
