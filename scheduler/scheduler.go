package scheduler

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/pipeline"
	"cinembrot/scraper"
	"gorm.io/gorm"
)

// AutoScraper manages background automated scraping with polite rate limiting
type AutoScraper struct {
	cfg      *config.Config
	db       *gorm.DB
	pipe     *pipeline.Pipeline
	repo     *scraper.Repository
	stopChan chan struct{}
}

func NewAutoScraper(cfg *config.Config, db *gorm.DB, pipe *pipeline.Pipeline, repo *scraper.Repository) *AutoScraper {
	return &AutoScraper{
		cfg:      cfg,
		db:       db,
		pipe:     pipe,
		repo:     repo,
		stopChan: make(chan struct{}),
	}
}

// GetSchedulerConfig loads dynamic scheduler configuration from MariaDB SystemSetting table
func GetSchedulerConfig(db *gorm.DB, defaultCfg *config.Config) model.SchedulerConfig {
	cfg := model.SchedulerConfig{
		Enabled:         defaultCfg.AutoScrapeEnabled,
		IntervalMinutes: defaultCfg.AutoScrapeIntervalMinutes,
		StartYear:       defaultCfg.AutoScrapeStartYear,
		EndYear:         defaultCfg.AutoScrapeEndYear,
		PagesPerYear:    defaultCfg.AutoScrapePagesPerYear,
		DelayMs:         defaultCfg.AutoScrapeDelayMs,
	}

	if db != nil {
		var settings []model.SystemSetting
		if err := db.Find(&settings).Error; err == nil {
			for _, s := range settings {
				switch s.Key {
				case "auto_scrape_enabled":
					cfg.Enabled = s.Value == "true" || s.Value == "1" || s.Value == "on"
				case "auto_scrape_interval_minutes":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.IntervalMinutes = v
					}
				case "auto_scrape_start_year":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.StartYear = v
					}
				case "auto_scrape_end_year":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.EndYear = v
					}
				case "auto_scrape_pages_per_year":
					if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
						cfg.PagesPerYear = v
					}
				case "auto_scrape_delay_ms":
					if v, err := strconv.Atoi(s.Value); err == nil && v >= 0 {
						cfg.DelayMs = v
					}
				}
			}
		}
	}

	if cfg.IntervalMinutes < 5 {
		cfg.IntervalMinutes = 5
	}
	if cfg.PagesPerYear < 1 {
		cfg.PagesPerYear = 1
	}
	if cfg.StartYear == 0 {
		cfg.StartYear = time.Now().Year()
	}
	if cfg.EndYear == 0 {
		cfg.EndYear = 2015
	}

	return cfg
}

// Start launches the background scheduler goroutine
func (s *AutoScraper) Start() {
	log.Printf("\n[SCHEDULER] 🤖 Background Auto-Scraper Scheduler aktif (Menggunakan pengaturan MariaDB & CMS)...\n")

	go func() {
		// Run initial check after 10 seconds of startup
		time.Sleep(10 * time.Second)
		s.RunFullCycle()

		for {
			schedCfg := GetSchedulerConfig(s.db, s.cfg)
			interval := time.Duration(schedCfg.IntervalMinutes) * time.Minute
			if interval < 5*time.Minute {
				interval = 5 * time.Minute
			}

			select {
			case <-time.After(interval):
				s.RunFullCycle()
			case <-s.stopChan:
				log.Println("[INFO] Auto-Scraper stopped.")
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *AutoScraper) Stop() {
	close(s.stopChan)
}

// RunFullCycle executes a gentle, multi-source movie scraping sweep
func (s *AutoScraper) RunFullCycle() {
	schedCfg := GetSchedulerConfig(s.db, s.cfg)
	if !schedCfg.Enabled {
		log.Println("[AUTO-SCRAPER] ⏸️ Jadwal scraper otomatis dinonaktifkan (OFF) di panel CMS. Siklus otomatis dilewati.")
		return
	}

	startTime := time.Now()
	log.Println("\n==============================================================")
	log.Printf(" [AUTO-SCRAPER] 🚀 Memulai siklus scraping otomatis (Tahun: %d - %d, Interval: %d menit, %d hal/tahun)\n",
		schedCfg.StartYear, schedCfg.EndYear, schedCfg.IntervalMinutes, schedCfg.PagesPerYear)
	log.Println("==============================================================")

	totalIngested := 0

	// 0. Ambil daftar sumber yang AKTIF dari tabel scrape_sources di MariaDB
	var activeSources []model.ScrapeSource
	s.db.Where("is_active = ?", true).Find(&activeSources)

	isSourceActive := func(code string) bool {
		for _, as := range activeSources {
			if as.Code == code {
				return true
			}
		}
		return false
	}

	// 1. Sync Creative Commons Open Movies (jika aktif di DB)
	if isSourceActive("blender") {
		log.Println("[AUTO-SCRAPER] 🎬 Memeriksa film Open Source / Creative Commons (Blender)...")
		openCount, err := s.pipe.IngestOpenMovies()
		if err == nil {
			totalIngested += openCount
		}
		s.politeSleep()
	}

	// 2. Discover movies across years (from StartYear down to EndYear)
	startYear := schedCfg.StartYear
	endYear := schedCfg.EndYear
	pagesPerYear := schedCfg.PagesPerYear

	if startYear < endYear {
		startYear, endYear = endYear, startYear
	}

	for year := startYear; year >= endYear; year-- {
		// Periksa kembali status ON/OFF di setiap tahun agar responsive jika user mematikan jadwal di tengah jalan
		currCfg := GetSchedulerConfig(s.db, s.cfg)
		if !currCfg.Enabled {
			log.Println("[AUTO-SCRAPER] ⏸️ Auto-scraper dimatikan dari CMS saat siklus berjalan. Menghentikan penelusuran tahun.")
			break
		}

		log.Printf("[AUTO-SCRAPER] 📅 Menelusuri film rilis tahun %d (Maks %d halaman)...\n", year, pagesPerYear)

		for page := 1; page <= pagesPerYear; page++ {
			// A. Ingest TMDb Popular movies (HANYA jika aktif di DB)
			if isSourceActive("tmdb") {
				count, err := s.pipe.IngestByYear(year, 1, "tmdb")
				if err == nil {
					totalIngested += count
				}
				s.politeSleep()
			}

			// B. Ingest YTS Torrents (HANYA jika aktif di DB)
			if isSourceActive("yts") {
				ytsCount, err := s.pipe.IngestByYear(year, 1, "yts")
				if err == nil {
					totalIngested += ytsCount
				}
				s.politeSleep()
			}
		}

		// C. Archive.org feature films (HANYA jika aktif di DB)
		if isSourceActive("archive") && (year%5 == 0 || year == startYear) {
			archCount, _ := s.pipe.IngestByYear(year, 1, "archive")
			totalIngested += archCount
			s.politeSleep()
		}
	}

	// 3. Log cycle summary
	duration := time.Since(startTime)
	log.Printf("\n[AUTO-SCRAPER] ✅ Siklus scraping selesai dalam %v. Total %d film tersimpan/diperbarui.\n",
		duration.Round(time.Second), totalIngested)

	_ = s.repo.LogScrape("AutoScraper-FullCycle", fmt.Sprintf("years=%d-%d", startYear, endYear),
		"SUCCESS", totalIngested, "", duration)
}

// politeSleep adds a polite, randomized delay so we never overwhelm target servers
func (s *AutoScraper) politeSleep() {
	baseMs := s.cfg.AutoScrapeDelayMs
	if baseMs <= 0 {
		baseMs = 2000
	}
	// Add 0-1000ms jitter to simulate natural human requests
	jitter := rand.Intn(1000)
	sleepDuration := time.Duration(baseMs+jitter) * time.Millisecond
	time.Sleep(sleepDuration)
}
