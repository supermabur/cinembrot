-- ============================================================================
-- CINEMBROT DATABASE SCHEMA & SEED DATA
-- Generated At: 2026-09-01 17:02:26
-- Database Engine: MariaDB / MySQL (InnoDB, utf8mb4)
--
-- CATATAN:
-- 1. Seluruh perintah menggunakan 'CREATE TABLE IF NOT EXISTS' (Aman, Non-Destructive).
-- 2. Seluruh tabel hanya menyertakan struktur tabel, KECUALI:
--    - system_settings (Struktur + Data konfigurasi sistem & scraper)
--    - users (Struktur + Data akun administrator awal)
-- ============================================================================

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

-- ----------------------------------------------------------------------------
-- STRUKTUR TABEL DATABASE (SCHEMA ONLY)
-- ----------------------------------------------------------------------------
/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19-11.8.8-MariaDB, for Win64 (AMD64)
--
-- Host: localhost    Database: CINEMBROT
-- ------------------------------------------------------
-- Server version	11.8.8-MariaDB

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;

--
-- Table structure for table `actors`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `actors` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `character_name` varchar(255) DEFAULT NULL,
  `photo_url` varchar(1000) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `photo_thumb_url` varchar(1000) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_actors_name` (`name`),
  KEY `idx_actors_slug` (`slug`),
  KEY `idx_actors_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4519 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `comments`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `comments` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `movie_id` bigint(20) unsigned NOT NULL,
  `author_name` varchar(150) NOT NULL,
  `author_email` varchar(255) DEFAULT NULL,
  `content` text NOT NULL,
  `rating` double DEFAULT 10,
  `is_approved` tinyint(1) DEFAULT 1,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_comments_movie_id` (`movie_id`),
  KEY `idx_comments_is_approved` (`is_approved`),
  KEY `idx_comments_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_movies_comments` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `directors`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `directors` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `photo_url` varchar(1000) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_directors_name` (`name`),
  KEY `idx_directors_slug` (`slug`),
  KEY `idx_directors_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=541 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `download_links`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `download_links` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `movie_id` bigint(20) unsigned NOT NULL,
  `episode_id` bigint(20) unsigned DEFAULT NULL,
  `provider` varchar(100) DEFAULT NULL,
  `quality` varchar(50) DEFAULT NULL,
  `resolution` varchar(50) DEFAULT NULL,
  `file_size` varchar(50) DEFAULT NULL,
  `format` varchar(50) DEFAULT NULL,
  `url` varchar(1000) NOT NULL,
  `password` varchar(100) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `is_valid` tinyint(1) DEFAULT 1,
  `status` varchar(50) DEFAULT 'ACTIVE',
  `http_status` bigint(20) DEFAULT NULL,
  `response_time_ms` bigint(20) DEFAULT NULL,
  `last_checked_at` datetime(3) DEFAULT NULL,
  `url_hash` char(40) GENERATED ALWAYS AS (sha(`url`)) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_movie_download_url_hash` (`movie_id`,`url_hash`),
  KEY `idx_download_links_movie_id` (`movie_id`),
  KEY `idx_download_links_episode_id` (`episode_id`),
  KEY `idx_download_links_provider` (`provider`),
  KEY `idx_download_links_deleted_at` (`deleted_at`),
  KEY `idx_download_links_is_valid` (`is_valid`),
  KEY `idx_download_links_status` (`status`),
  CONSTRAINT `fk_episodes_download_links` FOREIGN KEY (`episode_id`) REFERENCES `episodes` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_movies_download_links` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=53389 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `episodes`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `episodes` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `movie_id` bigint(20) unsigned NOT NULL,
  `season_number` bigint(20) DEFAULT 1,
  `episode_number` bigint(20) NOT NULL,
  `title` varchar(255) DEFAULT NULL,
  `slug` varchar(255) DEFAULT NULL,
  `synopsis` text DEFAULT NULL,
  `duration` varchar(50) DEFAULT NULL,
  `release_date` datetime(3) DEFAULT NULL,
  `stream_url` varchar(1000) DEFAULT NULL,
  `source_url` varchar(500) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_episodes_movie_id` (`movie_id`),
  KEY `idx_episodes_season_number` (`season_number`),
  KEY `idx_episodes_episode_number` (`episode_number`),
  KEY `idx_episodes_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_movies_episodes` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `genres`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `genres` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(100) NOT NULL,
  `slug` varchar(100) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_genres_name` (`name`),
  UNIQUE KEY `idx_genres_slug` (`slug`),
  KEY `idx_genres_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=32 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `movie_actors`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `movie_actors` (
  `movie_id` bigint(20) unsigned NOT NULL,
  `actor_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`movie_id`,`actor_id`),
  KEY `fk_movie_actors_actor` (`actor_id`),
  CONSTRAINT `fk_movie_actors_actor` FOREIGN KEY (`actor_id`) REFERENCES `actors` (`id`),
  CONSTRAINT `fk_movie_actors_movie` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `movie_directors`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `movie_directors` (
  `movie_id` bigint(20) unsigned NOT NULL,
  `director_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`movie_id`,`director_id`),
  KEY `fk_movie_directors_director` (`director_id`),
  CONSTRAINT `fk_movie_directors_director` FOREIGN KEY (`director_id`) REFERENCES `directors` (`id`),
  CONSTRAINT `fk_movie_directors_movie` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `movie_genres`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `movie_genres` (
  `movie_id` bigint(20) unsigned NOT NULL,
  `genre_id` bigint(20) unsigned NOT NULL,
  PRIMARY KEY (`movie_id`,`genre_id`),
  KEY `fk_movie_genres_genre` (`genre_id`),
  CONSTRAINT `fk_movie_genres_genre` FOREIGN KEY (`genre_id`) REFERENCES `genres` (`id`),
  CONSTRAINT `fk_movie_genres_movie` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `movies`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `movies` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `slug` varchar(255) NOT NULL,
  `title` varchar(255) NOT NULL,
  `original_title` varchar(255) DEFAULT NULL,
  `alternative_titles` varchar(500) DEFAULT NULL,
  `type` varchar(50) DEFAULT 'movie',
  `status` varchar(50) DEFAULT 'released',
  `tagline` varchar(500) DEFAULT NULL,
  `synopsis` longtext DEFAULT NULL,
  `release_date` datetime(3) DEFAULT NULL,
  `year` bigint(20) DEFAULT NULL,
  `duration_minutes` bigint(20) DEFAULT NULL,
  `duration_formatted` varchar(50) DEFAULT NULL,
  `country` varchar(100) DEFAULT NULL,
  `language` varchar(100) DEFAULT NULL,
  `age_rating` varchar(50) DEFAULT NULL,
  `quality` varchar(50) DEFAULT NULL,
  `im_db_rating` double DEFAULT NULL,
  `im_db_votes` bigint(20) DEFAULT NULL,
  `tm_db_rating` double DEFAULT NULL,
  `rating` double DEFAULT NULL,
  `vote_count` bigint(20) DEFAULT NULL,
  `popularity` double DEFAULT NULL,
  `views` bigint(20) DEFAULT 0,
  `poster_url` varchar(1000) DEFAULT NULL,
  `backdrop_url` varchar(1000) DEFAULT NULL,
  `thumbnail_url` varchar(1000) DEFAULT NULL,
  `trailer_url` varchar(1000) DEFAULT NULL,
  `source_website` varchar(100) DEFAULT NULL,
  `source_url` varchar(500) NOT NULL,
  `raw_metadata` longtext DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `is_legal` tinyint(1) DEFAULT 1,
  `is_free` tinyint(1) DEFAULT 0,
  `license_type` varchar(100) DEFAULT NULL,
  `license_name` varchar(255) DEFAULT NULL,
  `license_url` varchar(500) DEFAULT NULL,
  `poster_thumb_url` varchar(1000) DEFAULT NULL,
  `backdrop_thumb_url` varchar(1000) DEFAULT NULL,
  `is_manual_edit` tinyint(1) DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_movies_slug` (`slug`),
  UNIQUE KEY `idx_movies_source_url` (`source_url`),
  KEY `idx_movies_title` (`title`),
  KEY `idx_movies_type` (`type`),
  KEY `idx_movies_status` (`status`),
  KEY `idx_movies_year` (`year`),
  KEY `idx_movies_country` (`country`),
  KEY `idx_movies_rating` (`rating`),
  KEY `idx_movies_source_website` (`source_website`),
  KEY `idx_movies_deleted_at` (`deleted_at`),
  KEY `idx_movies_is_legal` (`is_legal`),
  KEY `idx_movies_is_free` (`is_free`),
  KEY `idx_movies_license_type` (`license_type`),
  KEY `idx_movies_is_manual_edit` (`is_manual_edit`)
) ENGINE=InnoDB AUTO_INCREMENT=930 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `schedules`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `schedules` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `movie_id` bigint(20) unsigned NOT NULL,
  `cinema_chain` varchar(100) DEFAULT NULL,
  `cinema_name` varchar(255) NOT NULL,
  `city` varchar(100) DEFAULT NULL,
  `address` varchar(500) DEFAULT NULL,
  `hall_type` varchar(100) DEFAULT NULL,
  `show_date` varchar(20) DEFAULT NULL,
  `showtimes` text NOT NULL,
  `price` varchar(100) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_schedules_movie_id` (`movie_id`),
  KEY `idx_schedules_cinema_chain` (`cinema_chain`),
  KEY `idx_schedules_cinema_name` (`cinema_name`),
  KEY `idx_schedules_city` (`city`),
  KEY `idx_schedules_show_date` (`show_date`),
  KEY `idx_schedules_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_movies_schedules` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `scrape_logs`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `scrape_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `source_website` varchar(100) DEFAULT NULL,
  `target_url` varchar(500) NOT NULL,
  `status` varchar(50) DEFAULT NULL,
  `items_scraped` bigint(20) DEFAULT NULL,
  `error_message` text DEFAULT NULL,
  `execution_time_ms` bigint(20) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_scrape_logs_source_website` (`source_website`),
  KEY `idx_scrape_logs_status` (`status`),
  KEY `idx_scrape_logs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=760 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `scrape_sources`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `scrape_sources` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(150) NOT NULL,
  `code` varchar(50) NOT NULL,
  `base_url` varchar(500) NOT NULL,
  `api_key` varchar(255) DEFAULT NULL,
  `type` varchar(50) DEFAULT 'api',
  `category` varchar(100) DEFAULT 'General',
  `description` text DEFAULT NULL,
  `is_active` tinyint(1) DEFAULT 1,
  `rate_limit_per_sec` bigint(20) DEFAULT 5,
  `total_scraped` bigint(20) DEFAULT 0,
  `last_scraped_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_scrape_sources_code` (`code`),
  KEY `idx_scrape_sources_is_active` (`is_active`),
  KEY `idx_scrape_sources_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `stream_links`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `stream_links` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `movie_id` bigint(20) unsigned NOT NULL,
  `episode_id` bigint(20) unsigned DEFAULT NULL,
  `provider` varchar(100) DEFAULT NULL,
  `server_name` varchar(100) DEFAULT NULL,
  `quality` varchar(50) DEFAULT NULL,
  `embed_url` varchar(1000) NOT NULL,
  `direct_url` varchar(1000) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `is_valid` tinyint(1) DEFAULT 1,
  `status` varchar(50) DEFAULT 'ACTIVE',
  `last_checked_at` datetime(3) DEFAULT NULL,
  `url_hash` char(40) GENERATED ALWAYS AS (sha(`embed_url`)) STORED,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_movie_stream_url_hash` (`movie_id`,`url_hash`),
  KEY `idx_stream_links_movie_id` (`movie_id`),
  KEY `idx_stream_links_episode_id` (`episode_id`),
  KEY `idx_stream_links_provider` (`provider`),
  KEY `idx_stream_links_deleted_at` (`deleted_at`),
  KEY `idx_stream_links_is_valid` (`is_valid`),
  KEY `idx_stream_links_status` (`status`),
  CONSTRAINT `fk_episodes_stream_links` FOREIGN KEY (`episode_id`) REFERENCES `episodes` (`id`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_movies_stream_links` FOREIGN KEY (`movie_id`) REFERENCES `movies` (`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=436 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `system_settings`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `system_settings` (
  `key` varchar(100) NOT NULL,
  `value` text NOT NULL,
  `description` varchar(255) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `torrent_tasks`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `torrent_tasks` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `movie_id` bigint(20) unsigned NOT NULL,
  `movie_title` varchar(255) NOT NULL,
  `movie_slug` varchar(255) NOT NULL,
  `movie_poster` varchar(500) DEFAULT NULL,
  `torrent_url` text NOT NULL,
  `quality` varchar(100) DEFAULT NULL,
  `status` varchar(50) DEFAULT 'PENDING',
  `progress_percent` double DEFAULT 0,
  `downloaded_bytes` bigint(20) DEFAULT 0,
  `total_bytes` bigint(20) DEFAULT 0,
  `download_speed_mbs` double DEFAULT 0,
  `peers_count` bigint(20) DEFAULT 0,
  `video_file_path` text DEFAULT NULL,
  `video_web_url` text DEFAULT NULL,
  `available_subtitles` text DEFAULT NULL,
  `selected_subtitle_id` varchar(100) DEFAULT NULL,
  `hardsub_file_path` text DEFAULT NULL,
  `hardsub_web_url` text DEFAULT NULL,
  `error_message` text DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_torrent_tasks_movie_id` (`movie_id`),
  KEY `idx_torrent_tasks_status` (`status`),
  KEY `idx_torrent_tasks_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Table structure for table `users`
--

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE IF NOT EXISTS `users` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(100) NOT NULL,
  `password_hash` varchar(255) NOT NULL,
  `full_name` varchar(150) DEFAULT NULL,
  `role` varchar(50) DEFAULT 'admin',
  `is_active` tinyint(1) DEFAULT 1,
  `last_login_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_users_username` (`username`),
  KEY `idx_users_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping routines for database 'CINEMBROT'
--
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

-- Dump completed on 2026-09-01 17:02:26

-- ----------------------------------------------------------------------------
-- DATA SEED (HANYA UNTUK: system_settings & users)
-- ----------------------------------------------------------------------------
/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19-11.8.8-MariaDB, for Win64 (AMD64)
--
-- Host: localhost    Database: CINEMBROT
-- ------------------------------------------------------
-- Server version	11.8.8-MariaDB

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;

--
-- Dumping data for table `users`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
REPLACE INTO `users` VALUES
(1,'admin','$s256$a4e6589397d9f4ac2c47c706a0e0f5c9$3ae42e21e7d71e52b8b4af021d6bbf5629e545bc0415199cdb0552c62aacf6e2','Administrator','admin',1,'2026-09-01 16:24:36.160','2026-08-31 09:35:45.802','2026-09-01 16:24:36.160',NULL),
(2,'budi_editor','$s256$e3a1f83e86a8bab76adf94f901dad2c6$81fe5cffc7684d93a6c4c1c589afc44e4142e726c0289281b810b41fad95f8c1','Budi Pratama','editor',1,'2026-08-31 09:46:06.983','2026-08-31 09:36:04.143','2026-08-31 09:46:06.983',NULL);
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Dumping data for table `system_settings`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `system_settings` WRITE;
/*!40000 ALTER TABLE `system_settings` DISABLE KEYS */;
REPLACE INTO `system_settings` VALUES
('auto_scrape_delay_ms','500','Jeda waktu ramah server antar permintaan film (ms)','2026-08-31 14:18:48.238'),
('auto_scrape_enabled','true','Saklar ON/OFF Auto-Scraper Latar Belakang','2026-08-31 14:18:48.234'),
('auto_scrape_end_year','2015','Tahun akhir penelusuran katalog film','2026-08-31 14:18:48.237'),
('auto_scrape_interval_minutes','360','Interval waktu scraping otomatis (menit)','2026-08-31 14:18:48.235'),
('auto_scrape_pages_per_year','5','Jumlah halaman yang discraping per tahun (1 halaman = 20 film)','2026-08-31 14:18:48.237'),
('auto_scrape_start_year','2026','Tahun awal penelusuran katalog film','2026-08-31 14:18:48.236'),
('download_movie_path','public/download/movie','Direktori penyimpanan file unduhan film dan hardsub subtitle','2026-09-01 16:22:25.000');
/*!40000 ALTER TABLE `system_settings` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

-- Dump completed on 2026-09-01 17:02:26


-- ============================================================================
-- PETUNJUK PERUBAHAN STRUKTUR TABEL (ALTER TABLE / NON-DESTRUCTIVE MIGRATION)
-- ============================================================================
-- Sesuai aturan: Jika ada perubahan struktur, DILARANG me-DROP tabel yang sudah ada.
-- Gunakan skrip inkremental (ALTER TABLE) di bawah ini untuk menambahkan/memperbarui
-- struktur tabel tanpa menghilangkan data film yang sudah ada di database.
--
-- 1. Menambahkan kolom baru jika belum ada:
--    ALTER TABLE movies ADD COLUMN IF NOT EXISTS imdb_id VARCHAR(50) DEFAULT NULL AFTER id;
--
-- 2. Mengubah tipe atau panjang kolom yang sudah ada:
--    ALTER TABLE movies MODIFY COLUMN synopsis MEDIUMTEXT DEFAULT NULL;
--
-- 3. Menambahkan index / unique index baru:
--    ALTER TABLE download_links ADD INDEX IF NOT EXISTS idx_download_links_quality (quality);
--    ALTER TABLE download_links ADD UNIQUE INDEX IF NOT EXISTS uq_movie_download_url_hash (movie_id, url_hash);
--
-- 4. Menghapus kolom (hanya jika benar-benar tidak terpakai):
--    ALTER TABLE movies DROP COLUMN IF EXISTS unused_column;
-- ============================================================================

/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;
/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;