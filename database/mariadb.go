package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cinembrot/auth"
	"cinembrot/config"
	"cinembrot/model"
	"cinembrot/scraper"

	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes MariaDB connection and auto-migrates all tables
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	// Step 1: Ensure database exists for local development
	if cfg.DBHost == "localhost" || cfg.DBHost == "127.0.0.1" {
		if err := ensureDatabaseExists(cfg); err != nil {
			log.Printf("[INFO] Notice checking local database creation: %v\n", err)
		}
	}

	// Step 2: Connect to the specific database
	gormDB, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MariaDB: %w", err)
	}

	// Step 3: Setup connection pooling
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get generic database object: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Step 4: Auto-migrate schemas
	log.Println("[INFO] Running auto-migration for database tables...")
	err = gormDB.AutoMigrate(
		&model.Movie{},
		&model.Genre{},
		&model.Director{},
		&model.Actor{},
		&model.Episode{},
		&model.DownloadLink{},
		&model.StreamLink{},
		&model.Schedule{},
		&model.ScrapeLog{},
		&model.Comment{},
		&model.User{},
		&model.ScrapeSource{},
		&model.SystemSetting{},
		&model.TorrentTask{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-migrate tables: %w", err)
	}

	// Seed default admin user if none exists
	seedAdminUser(gormDB, cfg)

	// Seed default website sources in database
	seedDefaultSources(gormDB)

	// Seed default system & scheduler settings
	seedDefaultSettings(gormDB)

	// Sanitize any existing movie synopses containing HTML tags
	SanitizeExistingSynopses(gormDB)

	log.Printf("[SUCCESS] Connected to MariaDB '%s' on %s:%s and migrations completed.\n",
		cfg.DBName, cfg.DBHost, cfg.DBPort)

	DB = gormDB
	return gormDB, nil
}

func seedAdminUser(db *gorm.DB, cfg *config.Config) {
	adminUser := cfg.AdminDefaultUser
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := cfg.AdminDefaultPass
	if adminPass == "" {
		adminPass = "CINEMBROT123"
	}

	var user model.User
	if err := db.Where("username = ?", adminUser).First(&user).Error; err != nil {
		user = model.User{
			Username:     adminUser,
			PasswordHash: auth.HashPassword(adminPass),
			FullName:     "Administrator",
			Role:         "admin",
			IsActive:     true,
		}
		_ = db.Create(&user)
		log.Printf("[CMS] 🔑 Akun Admin Default dibuat: Username '%s' | Password '%s'\n", adminUser, adminPass)
	} else if !auth.CheckPasswordHash(adminPass, user.PasswordHash) {
		// Update password hash if needed
		db.Model(&user).Update("password_hash", auth.HashPassword(adminPass))
		log.Printf("[CMS] 🔑 Password Admin Default diperbarui: '%s'\n", adminUser)
	}
}

// seedDefaultSources initializes website scraping sources in MariaDB
func seedDefaultSources(db *gorm.DB) {
	defaultSources := []model.ScrapeSource{
		{
			Name:            "The Movie Database (TMDb)",
			Code:            "tmdb",
			BaseURL:         "https://api.themoviedb.org/3",
			Type:            "api",
			Category:        "Metadata & Popular",
			Description:     "Penyedia katalog metadata film, rating, sinopsis, poster resolusi tinggi, sutradara, dan cast aktor.",
			IsActive:        true,
			RateLimitPerSec: 5,
		},
		{
			Name:            "Internet Archive (Feature Films)",
			Code:            "archive",
			BaseURL:         "https://archive.org/details/feature_films",
			Type:            "api",
			Category:        "Public Domain & Legal Downloads",
			Description:     "Arsip publik film bioskop klasik, public domain, dan open license dengan link download video langsung.",
			IsActive:        true,
			RateLimitPerSec: 3,
		},
		{
			Name:            "Blender Open Studio",
			Code:            "blender",
			BaseURL:         "https://studio.blender.org/films/",
			Type:            "html_scrape",
			Category:        "Creative Commons Open Movies",
			Description:     "Film animasi open source berkualitas 4K Creative Commons (Sintel, Tears of Steel, Big Buck Bunny, Spring, Charge).",
			IsActive:        true,
			RateLimitPerSec: 2,
		},
		{
			Name:            "Public Domain Movies Hub",
			Code:            "publicdomain",
			BaseURL:         "https://publicdomainmovies.info/",
			Type:            "html_scrape",
			Category:        "Public Domain Catalog",
			Description:     "Direktori kurasi film-film berlisensi domain publik bebas hak cipta komersial.",
			IsActive:        true,
			RateLimitPerSec: 2,
		},
		{
			Name:            "YTS Movies (YIFY Torrents)",
			Code:            "yts",
			BaseURL:         "https://yts.lt/api/v2",
			Type:            "api",
			Category:        "Torrent & Commercial Releases",
			Description:     "Penyedia REST API resmi film dengan link download file torrent dan magnet link resolusi 720p, 1080p, dan 4K.",
			IsActive:        true,
			RateLimitPerSec: 3,
		},
	}

	for _, s := range defaultSources {
		var existing model.ScrapeSource
		if err := db.Where("code = ?", s.Code).First(&existing).Error; err != nil {
			_ = db.Create(&s)
			log.Printf("[CMS DB] 🌐 Sumber website ditambahkan ke DB: '%s' (%s)\n", s.Name, s.Code)
		} else if s.Code == "yts" && existing.BaseURL == "https://yts.mx/api/v2" {
			db.Model(&existing).Update("base_url", "https://yts.lt/api/v2")
			log.Printf("[CMS DB] 🌐 Updated YTS BaseURL to active mirror: 'https://yts.lt/api/v2'\n")
		}
	}
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Key: "auto_scrape_enabled", Value: "true", Description: "Saklar ON/OFF Auto-Scraper Latar Belakang"},
		{Key: "auto_scrape_interval_minutes", Value: "30", Description: "Interval waktu scraping otomatis (menit)"},
		{Key: "auto_scrape_start_year", Value: "2026", Description: "Tahun awal penelusuran katalog film"},
		{Key: "auto_scrape_end_year", Value: "2015", Description: "Tahun akhir penelusuran katalog film"},
		{Key: "auto_scrape_pages_per_year", Value: "1", Description: "Jumlah halaman yang discraping per tahun (1 halaman = 20 film)"},
		{Key: "auto_scrape_delay_ms", Value: "500", Description: "Jeda waktu ramah server antar permintaan film (ms)"},
		{Key: "download_movie_path", Value: "public/download/movie", Description: "Direktori penyimpanan file unduhan film dan hardsub subtitle"},
	}

	for _, s := range defaults {
		var existing model.SystemSetting
		if err := db.Where("`key` = ?", s.Key).First(&existing).Error; err != nil {
			_ = db.Create(&s)
		}
	}
}

// GetDownloadMoviePath returns the absolute download path configured in system_settings or default
func GetDownloadMoviePath(db *gorm.DB) string {
	var s model.SystemSetting
	val := filepath.Join("public", "download", "movie")
	if err := db.Where("`key` = ?", "download_movie_path").First(&s).Error; err == nil && strings.TrimSpace(s.Value) != "" {
		val = strings.TrimSpace(s.Value)
	}

	if !filepath.IsAbs(val) {
		if abs, err := filepath.Abs(val); err == nil {
			val = abs
		}
	}
	_ = os.MkdirAll(val, 0755)
	return val
}

// SanitizeExistingSynopses cleans up any existing synopses in the database that still contain HTML tags
func SanitizeExistingSynopses(db *gorm.DB) {
	var movies []model.Movie
	db.Where("synopsis LIKE ? OR synopsis LIKE ? OR synopsis LIKE ?", "%<p>%", "%<a %", "%</div>%").Find(&movies)
	for _, m := range movies {
		clean := scraper.CleanHTMLToPlainText(m.Synopsis)
		if clean != m.Synopsis {
			db.Model(&model.Movie{}).Where("id = ?", m.ID).Update("synopsis", clean)
		}
	}
}

// ensureDatabaseExists connects without DB name to check or create target database
func ensureDatabaseExists(cfg *config.Config) error {
	rawDB, err := sql.Open("mysql", cfg.ServerDSN())
	if err != nil {
		return err
	}
	defer rawDB.Close()

	if err := rawDB.Ping(); err != nil {
		return fmt.Errorf("cannot ping MariaDB server: %w", err)
	}

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", cfg.DBName)
	_, err = rawDB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to execute create database query: %w", err)
	}

	return nil
}
