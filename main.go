package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cinembrot/config"
	"cinembrot/database"
	"cinembrot/model"
	"cinembrot/pipeline"
	"cinembrot/scheduler"
	"cinembrot/scraper"
	"cinembrot/server"
	"cinembrot/validator"

	"gorm.io/gorm"
)

func main() {
	serveWeb := flag.Bool("serve", false, "Start CINEMBROT Web Server on http://localhost:8080 (with background auto-scraper)")
	autoScrapeOnly := flag.Bool("auto-scrape", false, "Run full automated polite scraping cycle across all years")
	cronDaemon := flag.Bool("daemon", false, "Run standalone background scraping scheduler without web server")
	scrapeURL := flag.String("url", "", "Scrape custom website URL using Colly engine")
	fetchArchive := flag.Bool("archive", false, "Ingest legal public domain feature films from Internet Archive API")
	archiveLimit := flag.Int("archive-limit", 10, "Number of films to fetch from Internet Archive")
	archivePage := flag.Int("archive-page", 1, "Page number for Internet Archive search")
	fetchOpenMovies := flag.Bool("openmovies", false, "Ingest Creative Commons / Blender Open Movies")
	tmdbQuery := flag.String("tmdb", "", "Search & ingest rich metadata from TMDb REST API")
	tmdbYear := flag.Int("year", 0, "Optional release year filter for single TMDb search")
	byYear := flag.Int("by-year", 0, "Automatically scrape and discover top movies by release year (e.g. 2024, 2023, 1999)")
	pages := flag.Int("pages", 1, "Number of pages to scrape for year discovery (1 page = 20 movies)")
	source := flag.String("source", "tmdb", "Source for year discovery: 'tmdb', 'archive', 'yts', or 'all'")
	syncAll := flag.Bool("sync", false, "Run full pipeline (Open Movies + Internet Archive)")
	convertImages := flag.Bool("convert-images", false, "Download and convert existing movie images in database to local WebP")
	checkLinks := flag.Bool("check-links", false, "Scan and validate all download file links in database for broken/dead URLs")

	flag.Parse()

	fmt.Println("================================================================")
	fmt.Println("             CINEMBROT - GOLANG MOVIE ENGINE & WEB               ")
	fmt.Println("================================================================")

	// 1. Load Configurations
	cfg := config.LoadConfig()
	fmt.Printf("[CONFIG] Target MariaDB : %s@tcp(%s:%s)/%s\n", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// 2. Initialize MariaDB Connection & Auto-Migrate Schemas
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatalf("[FATAL] MariaDB Connection Failed: %v\n", err)
	}

	// 3. Initialize Repositories, Pipelines, and Auto-Scraper
	repo := scraper.NewRepository(db)
	engine := scraper.NewEngine(cfg, repo)
	pipe := pipeline.NewPipeline(cfg, repo)
	sampleScraper := scraper.NewSampleMovieScraper(engine)
	autoScraper := scheduler.NewAutoScraper(cfg, db, pipe, repo)

	// 4. Handle Dedicated Auto-Scraping Flags
	if *autoScrapeOnly {
		fmt.Println("\n[ACTION] Running one full polite auto-scraping cycle across years...")
		autoScraper.RunFullCycle()
		printDatabaseStats(db)
		return
	}

	if *cronDaemon {
		fmt.Println("\n[ACTION] Running standalone background scheduler daemon (Ctrl+C to stop)...")
		autoScraper.Start()
		// Wait for terminate signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\n[INFO] Scheduler shutting down...")
		return
	}

	// 5. Handle CLI Action Flags
	if *scrapeURL != "" {
		fmt.Printf("\n[ACTION] Starting HTML Scraper for: %s\n", *scrapeURL)
		if err := sampleScraper.ScrapeURL(*scrapeURL); err != nil {
			log.Printf("[WARN] Scraper notice: %v\n", err)
		}
		printDatabaseStats(db)
		return
	}

	if *byYear > 0 {
		fmt.Printf("\n[ACTION] Automated Year-Based Scraping for Year: %d (Pages: %d, Source: %s)...\n",
			*byYear, *pages, *source)
		count, err := pipe.IngestByYear(*byYear, *pages, *source)
		if err != nil {
			log.Printf("[ERROR] Year scraping error: %v\n", err)
		} else {
			fmt.Printf("\n[SUCCESS] Successfully scraped and saved %d movies for year %d into MariaDB!\n", count, *byYear)
		}
		printDatabaseStats(db)
		return
	}

	if *tmdbQuery != "" {
		fmt.Printf("\n[ACTION] Querying TMDb REST API for: '%s' (Year: %d)...\n", *tmdbQuery, *tmdbYear)
		movie, err := pipe.SearchAndIngestTMDb(*tmdbQuery, *tmdbYear)
		if err != nil {
			log.Printf("[ERROR] TMDb Ingestion failed: %v\n", err)
		} else {
			fmt.Printf("[SUCCESS] Ingested '%s' (%d) | Rating: %.1f | Genres: %d | Actors: %d\n",
				movie.Title, movie.Year, movie.Rating, len(movie.Genres), len(movie.Actors))
		}
		printDatabaseStats(db)
		return
	}

	if *fetchOpenMovies || *syncAll {
		fmt.Println("\n[ACTION] Ingesting Creative Commons / Blender Open Movies...")
		count, err := pipe.IngestOpenMovies()
		if err == nil {
			fmt.Printf("[SUCCESS] Ingested %d Creative Commons films!\n", count)
		}
	}

	if *fetchArchive || *syncAll {
		fmt.Printf("\n[ACTION] Ingesting %d Feature Films from Internet Archive (Page %d)...\n", *archiveLimit, *archivePage)
		count, err := pipe.IngestArchiveFeatureFilms(*archiveLimit, *archivePage)
		if err == nil {
			fmt.Printf("[SUCCESS] Ingested %d feature films from Internet Archive!\n", count)
		}
	}

	if *convertImages {
		fmt.Println("\n[ACTION] Mengonversi gambar film di database ke format WebP lokal (Original + Thumbnail)...")
		count := pipeline.ConvertExistingImagesToWebP(db, cfg.ScraperUserAgent, 500)
		fmt.Printf("\n[SUCCESS] Berhasil mengonversi %d film ke WebP lokal!\n", count)
		printDatabaseStats(db)
		return
	}

	if *checkLinks {
		fmt.Println("\n[ACTION] Memulai validasi dan pengecekan kesehatan link download file di database...")
		total, valid, dead := validator.ValidateAllDatabaseLinks(db, cfg.ScraperUserAgent, 0)
		fmt.Printf("\n[RESULT] Selesai! Total Diperiksa: %d | Aktif (Valid): %d | Rusak/Mati (Dead): %d\n", total, valid, dead)
		printDatabaseStats(db)
		return
	}

	// 6. Start Web Server with background Auto-Scraper enabled
	if *serveWeb || (!*fetchArchive && !*fetchOpenMovies && !*syncAll && *tmdbQuery == "" && *scrapeURL == "" && *byYear == 0 && !*convertImages && !*checkLinks) {
		// Launch background polite auto-scraper
		autoScraper.Start()

		// Background routine: Convert any existing movie images to local WebP & validate links
		go func() {
			time.Sleep(3 * time.Second)
			pipeline.ConvertExistingImagesToWebP(db, cfg.ScraperUserAgent, 200)
			validator.ValidateAllDatabaseLinks(db, cfg.ScraperUserAgent, 100)
		}()

		webServer := server.NewServer(cfg, db)
		if err := webServer.Start(); err != nil {
			log.Fatalf("[FATAL] Web server error: %v\n", err)
		}
		return
	}

	printDatabaseStats(db)

	fmt.Println("\n[INFO] Perintah CLI yang dapat digunakan:")
	fmt.Println("  go run main.go -serve                # 🚀 JALANKAN WEB SERVER & AUTO-SCRAPER BACKGROUND")
	fmt.Println("  go run main.go -check-links          # 🔍 Pengecekan & validasi link download file di database")
	fmt.Println("  go run main.go -convert-images       # 🖼️ Download & konversi semua poster/backdrop di DB ke WebP lokal")
	fmt.Println("  go run main.go -auto-scrape          # 🤖 Jalankan 1 siklus scraping ramah server semua tahun")
	fmt.Println("  go run main.go -daemon               # ⏳ Jalankan scraper otomatis di background (cron)")
	fmt.Println("  go run main.go -by-year 2024         # Scrape film-film rilis tahun 2024")
	fmt.Println("  go run main.go -archive              # Ambil film legal dari Internet Archive API")
	fmt.Println("  go run main.go -tmdb \"Inception\"     # Ambil metadata HD dari TMDb API")
	_ = os.Stdout.Sync()
}

func printDatabaseStats(db *gorm.DB) {
	var movieCount, freeCount, legalCount, genreCount, directorCount, actorCount, downloadCount, streamCount, logCount int64
	db.Model(&model.Movie{}).Count(&movieCount)
	db.Model(&model.Movie{}).Where("is_free = ?", true).Count(&freeCount)
	db.Model(&model.Movie{}).Where("is_legal = ?", true).Count(&legalCount)
	db.Model(&model.Genre{}).Count(&genreCount)
	db.Model(&model.Director{}).Count(&directorCount)
	db.Model(&model.Actor{}).Count(&actorCount)
	db.Model(&model.DownloadLink{}).Count(&downloadCount)
	db.Model(&model.StreamLink{}).Count(&streamCount)
	db.Model(&model.ScrapeLog{}).Count(&logCount)

	fmt.Println("\n======================== DATABASE STATS ========================")
	fmt.Printf("Total Film (Movies)       : %d\n", movieCount)
	fmt.Printf("  ├── Legal / Sah         : %d film\n", legalCount)
	fmt.Printf("  ├── Gratis (Open/Public): %d film (Free to watch/download)\n", freeCount)
	fmt.Printf("  └── Berlisensi Komersil : %d film (Copyrighted / TMDb)\n", movieCount-freeCount)
	fmt.Printf("Total Genres              : %d\n", genreCount)
	fmt.Printf("Total Directors           : %d\n", directorCount)
	fmt.Printf("Total Actors              : %d\n", actorCount)
	fmt.Printf("Total Download Links      : %d\n", downloadCount)
	fmt.Printf("Total Stream Players      : %d\n", streamCount)
	fmt.Printf("Total Scrape Logs         : %d\n", logCount)
	fmt.Println("================================================================")
}
