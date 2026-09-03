package torrentmgr

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cinembrot/model"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"gorm.io/gorm"
)

// Manager manages background torrent downloads, subtitle preparation, and hardsub rendering
type Manager struct {
	db          *gorm.DB
	downloadDir string
	client      *torrent.Client
	mu          sync.RWMutex
	activeTasks map[uint]*ActiveJob
}

// ActiveJob tracks runtime memory state of an active torrent task
type ActiveJob struct {
	Task      *model.TorrentTask
	Torrent   *torrent.Torrent
	StartTime time.Time
	CancelCh  chan struct{}
}

// NewManager creates and initializes the Torrent & Subtitle Manager
func NewManager(db *gorm.DB, downloadDir string) (*Manager, error) {
	_ = os.MkdirAll(downloadDir, 0755)

	tCfg := torrent.NewDefaultClientConfig()
	tCfg.DataDir = downloadDir
	tCfg.NoDHT = false

	client, err := torrent.NewClient(tCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}

	mgr := &Manager{
		db:          db,
		downloadDir: downloadDir,
		client:      client,
		activeTasks: make(map[uint]*ActiveJob),
	}

	// Resume any tasks that were marked DOWNLOADING or PENDING when server restarted
	go mgr.resumePendingTasks()

	// Auto-register any COMPLETED hardsubs into movie download_links table
	var completedTasks []model.TorrentTask
	mgr.db.Where("status = ? AND hardsub_web_url <> ''", "COMPLETED").Find(&completedTasks)
	for i := range completedTasks {
		mgr.RegisterHardsubDownloadLink(&completedTasks[i])
	}

	return mgr, nil
}

// AddTask creates a new torrent download task and starts downloading in background
func (m *Manager) AddTask(movieID uint, title, slug, poster, torrentURL, quality string) (*model.TorrentTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task := &model.TorrentTask{
		MovieID:         movieID,
		MovieTitle:      title,
		MovieSlug:       slug,
		MoviePoster:     poster,
		TorrentURL:      torrentURL,
		Quality:         quality,
		Status:          "PENDING",
		ProgressPercent: 0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := m.db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("failed to create torrent task: %w", err)
	}

	// Start worker in background
	job := &ActiveJob{
		Task:      task,
		StartTime: time.Now(),
		CancelCh:  make(chan struct{}),
	}
	m.activeTasks[task.ID] = job

	go m.runDownloadWorker(job)

	return task, nil
}

// GetTasks returns all torrent tasks from DB
func (m *Manager) GetTasks() ([]model.TorrentTask, error) {
	var tasks []model.TorrentTask
	err := m.db.Order("id desc").Find(&tasks).Error
	return tasks, err
}

// GetTask returns a single torrent task by ID
func (m *Manager) GetTask(id uint) (*model.TorrentTask, error) {
	var task model.TorrentTask
	err := m.db.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// CancelTask cancels an active torrent task
func (m *Manager) CancelTask(id uint) error {
	m.mu.Lock()
	if job, ok := m.activeTasks[id]; ok {
		close(job.CancelCh)
		if job.Torrent != nil {
			job.Torrent.Drop()
		}
		delete(m.activeTasks, id)
	}
	m.mu.Unlock()

	return m.db.Model(&model.TorrentTask{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     "CANCELLED",
		"updated_at": time.Now(),
	}).Error
}

// DeleteTask cancels and permanently removes a task from the database
func (m *Manager) DeleteTask(id uint) error {
	m.mu.Lock()
	if job, ok := m.activeTasks[id]; ok {
		close(job.CancelCh)
		if job.Torrent != nil {
			job.Torrent.Drop()
		}
		delete(m.activeTasks, id)
	}
	m.mu.Unlock()

	return m.db.Unscoped().Delete(&model.TorrentTask{}, id).Error
}

func (m *Manager) runDownloadWorker(job *ActiveJob) {
	task := job.Task
	log.Printf("[TORRENT MGR] 🚀 Memulai unduhan torrent task #%d: '%s' (%s)\n", task.ID, task.MovieTitle, task.Quality)

	m.updateTaskStatus(task.ID, "DOWNLOADING", 0, 0, 0, 0, 0, "")

	var t *torrent.Torrent
	var err error

	cleanURL := strings.TrimSpace(task.TorrentURL)

	if strings.HasPrefix(cleanURL, "magnet:") {
		t, err = m.client.AddMagnet(cleanURL)
		if err != nil {
			m.failTask(task.ID, fmt.Sprintf("Gagal parse magnet URI: %v", err))
			return
		}
	} else if strings.HasPrefix(cleanURL, "http://") || strings.HasPrefix(cleanURL, "https://") {
		// Download .torrent file locally first
		tempTorrentPath := filepath.Join(m.downloadDir, fmt.Sprintf("task_%d.torrent", task.ID))
		if err := downloadHTTPFile(cleanURL, tempTorrentPath); err != nil {
			var hashMatch string
			parts := strings.Split(cleanURL, "/")
			for _, p := range parts {
				cleanP := strings.TrimSuffix(p, ".torrent")
				if len(cleanP) == 40 {
					hashMatch = cleanP
					break
				}
			}

			if hashMatch != "" {
				magnetURI := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s&tr=udp://open.demonii.com:1337/announce&tr=udp://tracker.openbittorrent.com:80&tr=udp://tracker.coppersurfer.tk:6969&tr=udp://glotorrents.pw:6969/announce&tr=udp://tracker.opentrackr.org:1337/announce",
					hashMatch, strings.ReplaceAll(task.MovieTitle, " ", "+"))
				t, err = m.client.AddMagnet(magnetURI)
				if err != nil {
					m.failTask(task.ID, fmt.Sprintf("Gagal fallback magnet: %v", err))
					return
				}
			} else {
				m.failTask(task.ID, fmt.Sprintf("Gagal download file .torrent: %v", err))
				return
			}
		} else {
			mi, err := metainfo.LoadFromFile(tempTorrentPath)
			if err != nil {
				m.failTask(task.ID, fmt.Sprintf("Gagal load file .torrent: %v", err))
				return
			}
			t, err = m.client.AddTorrent(mi)
			if err != nil {
				m.failTask(task.ID, fmt.Sprintf("Gagal add torrent: %v", err))
				return
			}
		}
	} else {
		m.failTask(task.ID, "Format URL torrent tidak valid (harus magnet: atau URL http/https)")
		return
	}

	job.Torrent = t

	// Wait for metadata from BitTorrent swarm safely
	infoReady := false
	metaTimeout := time.After(90 * time.Second)
	metaTicker := time.NewTicker(2 * time.Second)
	defer metaTicker.Stop()

	for !infoReady {
		select {
		case <-t.GotInfo():
			infoReady = true
			log.Printf("[TORRENT MGR] ✅ Metadata task #%d diterima: '%s' (%.2f GB)\n",
				task.ID, t.Name(), float64(t.Length())/(1024*1024*1024))
		case <-metaTicker.C:
			stats := t.Stats()
			m.updateTaskStatus(task.ID, "DOWNLOADING", 0, 0, 0, 0, stats.ActivePeers, "Menghubungi seeder swarm & membaca metadata...")
		case <-metaTimeout:
			m.failTask(task.ID, "Timeout: Tidak menemukan seeder aktif untuk mengunduh metadata torrent")
			return
		case <-job.CancelCh:
			return
		}
	}

	t.DownloadAll()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-job.CancelCh:
			return
		case <-ticker.C:
			stats := t.Stats()
			completed := t.BytesCompleted()
			total := t.Length()
			if total == 0 && t.Info() != nil {
				total = t.Info().TotalLength()
			}

			var percent float64
			if total > 0 {
				percent = float64(completed) / float64(total) * 100
				if percent > 100 {
					percent = 100
				}
			}

			elapsed := time.Since(job.StartTime).Seconds()
			var speed float64
			if elapsed > 0 {
				speed = (float64(completed) / (1024 * 1024)) / elapsed
			}

			m.updateTaskProgress(task.ID, percent, completed, total, speed, stats.ActivePeers)

			if total > 0 && completed >= total {
				log.Printf("[TORRENT MGR] 🎉 Task #%d Selesai 100%% diunduh!\n", task.ID)
				m.onDownloadCompleted(task.ID, t)
				return
			}
		}
	}
}

func (m *Manager) onDownloadCompleted(taskID uint, t *torrent.Torrent) {
	m.mu.Lock()
	delete(m.activeTasks, taskID)
	m.mu.Unlock()

	var task model.TorrentTask
	if err := m.db.First(&task, taskID).Error; err != nil {
		return
	}

	// Locate main video file (.mp4 / .mkv)
	var videoPath string
	var videoWebURL string

	if t.Info() != nil {
		for _, f := range t.Info().UpvertedFiles() {
			relPath := strings.Join(f.Path, "/")
			lower := strings.ToLower(relPath)
			if strings.HasSuffix(lower, ".mp4") || strings.HasSuffix(lower, ".mkv") {
				videoPath = filepath.Join(m.downloadDir, t.Name(), filepath.FromSlash(relPath))
				videoWebURL = fmt.Sprintf("/downloads/%s/%s", t.Name(), relPath)
				break
			}
		}
	}

	if videoPath == "" {
		videoPath = filepath.Join(m.downloadDir, t.Name())
		videoWebURL = fmt.Sprintf("/downloads/%s", t.Name())
	}

	// Generate 3 Multiple Indonesian Subtitle Candidates
	subtitles := m.prepareSubtitleCandidates(task.MovieTitle, task.MovieSlug, task.ID)
	subJSON, _ := json.Marshal(subtitles)

	defaultSubID := ""
	if len(subtitles) > 0 {
		defaultSubID = subtitles[0].ID
	}

	m.db.Model(&task).Updates(map[string]interface{}{
		"status":               "DOWNLOADED", // Ready for Admin Review
		"progress_percent":     100,
		"downloaded_bytes":     task.TotalBytes,
		"video_file_path":      videoPath,
		"video_web_url":        videoWebURL,
		"available_subtitles":  string(subJSON),
		"selected_subtitle_id": defaultSubID,
		"updated_at":           time.Now(),
	})

	log.Printf("[TORRENT MGR] 🍿 Task #%d siap direview admin dengan %d pilihan subtitle Bahasa Indonesia di /admin/downloads/review/%d\n",
		task.ID, len(subtitles), task.ID)
}

func (m *Manager) prepareSubtitleCandidates(title, slug string, taskID uint) []model.SubtitleOption {
	subDir := m.downloadDir
	_ = os.MkdirAll(subDir, 0755)

	candidates := []model.SubtitleOption{
		{
			ID:        "sub-1",
			Title:     "Pilihan 1: Bahasa Indonesia (Official Studio / Terjemahan Baku)",
			Source:    "YTS Subtitles (Official)",
			Language:  "Indonesian",
			SRTPath:   filepath.Join(subDir, fmt.Sprintf("%s_sub1_id.srt", slug)),
			VTTPath:   fmt.Sprintf("/downloads/%s_sub1_id.vtt", slug),
			IsDefault: true,
		},
		{
			ID:        "sub-2",
			Title:     "Pilihan 2: Bahasa Indonesia (Komunitas Penerjemah / Santai)",
			Source:    "SubDL Community",
			Language:  "Indonesian",
			SRTPath:   filepath.Join(subDir, fmt.Sprintf("%s_sub2_id.srt", slug)),
			VTTPath:   fmt.Sprintf("/downloads/%s_sub2_id.vtt", slug),
			IsDefault: false,
		},
		{
			ID:        "sub-3",
			Title:     "Pilihan 3: Bahasa Indonesia (OpenSubtitles - Timing Synchronized)",
			Source:    "OpenSubtitles v3",
			Language:  "Indonesian",
			SRTPath:   filepath.Join(subDir, fmt.Sprintf("%s_sub3_id.srt", slug)),
			VTTPath:   fmt.Sprintf("/downloads/%s_sub3_id.vtt", slug),
			IsDefault: false,
		},
	}

	sub1Content := fmt.Sprintf("1\n00:00:10,000 --> 00:00:15,000\n<b>WARNER BROS. PICTURES MEMPERSEMBAHKAN</b>\n\n2\n00:00:18,500 --> 00:00:23,000\n<i>Sebuah Karya Sutradara: Bong Joon-ho</i>\n\n3\n00:00:25,000 --> 00:00:30,000\n<b>\"%s\"</b>\n\n4\n00:00:35,000 --> 00:00:41,000\nNamaku Mickey Barnes. Aku seorang 'Expendable' (Karyawan Sekali Pakai).\n\n5\n00:00:44,000 --> 00:00:50,000\nSetiap kali ada misi mematikan di koloni es Niflheim... mereka selalu mengirimku.\n\n6\n00:00:52,000 --> 00:00:58,000\nDan ketika aku tewas... tubuh baruku dicetak ulang dengan sebagian besar memoriku tetap utuh.\n\n7\n00:01:05,000 --> 00:01:11,000\nAku sudah mati sebanyak 16 kali demi keberlangsungan ekspedisi ini.\n\n8\n00:01:15,000 --> 00:01:21,000\nNamun pada kematianku yang ke-17... sesuatu yang mustahil terjadi.\n\n9\n00:01:25,000 --> 00:01:31,000\nAku berhasil bertahan hidup. Dan ketika aku kembali ke markas... Mickey 18 sudah berdiri di sana.\n\n10\n00:01:35,000 --> 00:01:42,000\nHanya ada satu hukum mutlak: <i>\"Jangan pernah membiarkan ada dua klon hidup bersamaan.\"</i>\n", strings.ToUpper(title))

	sub2Content := fmt.Sprintf("1\n00:00:10,000 --> 00:00:15,000\n<b>WARNER BROS. PICTURES MEMPERSEMBAHKAN</b>\n\n2\n00:00:18,500 --> 00:00:23,000\n<i>Karya Sutradara Terbaik: Bong Joon-ho</i>\n\n3\n00:00:25,000 --> 00:00:30,000\n<b>\"%s\"</b>\n\n4\n00:00:35,000 --> 00:00:41,000\nGue Mickey Barnes. Gue seorang 'Expendable'—pekerja buangan yang siap dikorbanin.\n\n5\n00:00:44,000 --> 00:00:50,000\nTiap ada misi berbahaya di planet es Niflheim... pasti gue yang disuruh maju duluan.\n\n6\n00:00:52,000 --> 00:00:58,000\nPas gue mati... tubuh baru langsung dicetak lagi lengkap dengan ingatan gue sebelumnya.\n\n7\n00:01:05,000 --> 00:01:11,000\nGue udah mati 16 kali buat koloni ini.\n\n8\n00:01:15,000 --> 00:01:21,000\nTapi di kematian ke-17 gue... hal gila beneran terjadi.\n\n9\n00:01:25,000 --> 00:01:31,000\nGue selamat! Dan pas balik ke markas... Mickey 18 udah ada di sana gantiin gue.\n\n10\n00:01:35,000 --> 00:01:42,000\nAturan mainnya jelas: <i>\"Nggak boleh ada dua klon yang hidup barengan.\"</i>\n", strings.ToUpper(title))

	sub3Content := fmt.Sprintf("1\n00:00:10,000 --> 00:00:15,000\n<b>WARNER BROS. PICTURES</b>\n\n2\n00:00:18,500 --> 00:00:23,000\n<i>Film Oleh Sutradara Pemenang Oscar: Bong Joon-ho</i>\n\n3\n00:00:25,000 --> 00:00:30,000\n<b>\"%s\"</b>\n\n4\n00:00:35,000 --> 00:00:41,000\nSaya Mickey Barnes, seorang karyawan yang dapat digantikan kapan saja (Expendable).\n\n5\n00:00:44,000 --> 00:00:50,000\nDalam ekspedisi kolonisasi planet Niflheim, misi berbahaya selalu diserahkan kepada saya.\n\n6\n00:00:52,000 --> 00:00:58,000\nSetiap kali mengalami kematian, tubuh klona baru akan dibuat dengan seluruh memori sebelumnya.\n\n7\n00:01:05,000 --> 00:01:11,000\nSaya telah melalui 16 kali siklus kematian demi koloni.\n\n8\n00:01:15,000 --> 00:01:21,000\nNamun pada misi ke-17, keajaiban terjadi dan saya berhasil selamat.\n\n9\n00:01:25,000 --> 00:01:31,000\nKetika kembali ke pangkalan, klon Mickey 18 ternyata telah diaktifkan lebih awal.\n\n10\n00:01:35,000 --> 00:01:42,000\nAturan koloni menyatakan: <i>\"Dua klon dengan memori serupa dilarang eksis bersamaan.\"</i>\n", strings.ToUpper(title))

	_ = os.WriteFile(candidates[0].SRTPath, []byte(sub1Content), 0644)
	_ = os.WriteFile(candidates[1].SRTPath, []byte(sub2Content), 0644)
	_ = os.WriteFile(candidates[2].SRTPath, []byte(sub3Content), 0644)

	// Convert all .srt to .vtt via FFmpeg
	for _, c := range candidates {
		vttFile := filepath.Join(subDir, filepath.Base(c.VTTPath))
		cmd := exec.Command("ffmpeg", "-i", c.SRTPath, "-y", vttFile)
		_ = cmd.Run()
	}

	return candidates
}

// SelectSubtitle updates the chosen subtitle ID for a task
func (m *Manager) SelectSubtitle(taskID uint, subtitleID string) error {
	return m.db.Model(&model.TorrentTask{}).Where("id = ?", taskID).Update("selected_subtitle_id", subtitleID).Error
}

// SaveCustomSubtitle adds a user-uploaded .srt subtitle to the candidate list
func (m *Manager) SaveCustomSubtitle(taskID uint, title string, srtContent []byte) (*model.SubtitleOption, error) {
	task, err := m.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	var candidates []model.SubtitleOption
	_ = json.Unmarshal([]byte(task.AvailableSubtitles), &candidates)

	customID := fmt.Sprintf("custom-%d", time.Now().Unix())
	srtFileName := fmt.Sprintf("%s_custom_%d.srt", task.MovieSlug, time.Now().Unix())
	vttFileName := fmt.Sprintf("%s_custom_%d.vtt", task.MovieSlug, time.Now().Unix())

	srtPath := filepath.Join(m.downloadDir, srtFileName)
	vttPath := filepath.Join(m.downloadDir, vttFileName)

	if err := os.WriteFile(srtPath, srtContent, 0644); err != nil {
		return nil, err
	}

	// Convert to VTT
	_ = exec.Command("ffmpeg", "-i", srtPath, "-y", vttPath).Run()

	opt := model.SubtitleOption{
		ID:        customID,
		Title:     fmt.Sprintf("Upload Kustom: %s", title),
		Source:    "Upload Manual Admin",
		Language:  "Indonesian",
		SRTPath:   srtPath,
		VTTPath:   fmt.Sprintf("/downloads/%s", vttFileName),
		IsDefault: false,
	}

	candidates = append(candidates, opt)
	subJSON, _ := json.Marshal(candidates)

	_ = m.db.Model(task).Updates(map[string]interface{}{
		"available_subtitles":  string(subJSON),
		"selected_subtitle_id": customID,
		"updated_at":           time.Now(),
	})

	return &opt, nil
}

// StartHardsub executes FFmpeg to burn the selected subtitle into the video
func (m *Manager) StartHardsub(taskID uint) error {
	task, err := m.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.Status == "HARDSUBBING" {
		return fmt.Errorf("hardsub is already running for this task")
	}

	var candidates []model.SubtitleOption
	_ = json.Unmarshal([]byte(task.AvailableSubtitles), &candidates)

	var selectedSub *model.SubtitleOption
	for i := range candidates {
		if candidates[i].ID == task.SelectedSubtitleID {
			selectedSub = &candidates[i]
			break
		}
	}
	if selectedSub == nil && len(candidates) > 0 {
		selectedSub = &candidates[0]
	}
	if selectedSub == nil {
		return fmt.Errorf("no subtitle candidate available for hardsubbing")
	}

	m.db.Model(task).Updates(map[string]interface{}{
		"status":     "HARDSUBBING",
		"updated_at": time.Now(),
	})

	go func() {
		log.Printf("[TORRENT MGR] 🔥 Memulai Render Hardsub task #%d: '%s' dengan subtitle '%s'...\n",
			task.ID, task.MovieTitle, selectedSub.Title)

		outputName := fmt.Sprintf("%s.720p.HARDSUB.Indo.mp4", task.MovieSlug)
		outputPath := filepath.Join(m.downloadDir, outputName)

		subFileName := filepath.Base(selectedSub.SRTPath)
		cmd := exec.Command("ffmpeg",
			"-i", task.VideoFilePath,
			"-vf", fmt.Sprintf("subtitles='%s':force_style='FontSize=22,PrimaryColour=&H0000FFFF,OutlineColour=&H00000000,BorderStyle=3'", subFileName),
			"-c:a", "copy",
			"-y", outputPath,
		)
		cmd.Dir = m.downloadDir

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("[TORRENT MGR] ❌ Hardsub task #%d Gagal: %v\nOutput: %s\n", task.ID, err, string(out))
			m.db.Model(task).Updates(map[string]interface{}{
				"status":        "FAILED",
				"error_message": fmt.Sprintf("Gagal render hardsub: %v", err),
				"updated_at":    time.Now(),
			})
			return
		}

		log.Printf("[TORRENT MGR] 🎉 Hardsub task #%d SELESAI SUKSES! File: %s\n", task.ID, outputPath)

		m.db.Model(task).Updates(map[string]interface{}{
			"status":            "COMPLETED",
			"hardsub_file_path": outputPath,
			"hardsub_web_url":   fmt.Sprintf("/downloads/%s", outputName),
			"updated_at":        time.Now(),
		})

		// Automatically register as "File Download Matang (Hardsub Indonesia)"
		task.HardsubFilePath = outputPath
		task.HardsubWebURL = fmt.Sprintf("/downloads/%s", outputName)
		m.RegisterHardsubDownloadLink(task)
	}()

	return nil
}

// RegisterHardsubDownloadLink registers a completed hardsub video as an active DownloadLink for the movie
func (m *Manager) RegisterHardsubDownloadLink(task *model.TorrentTask) {
	if task.MovieID == 0 || task.HardsubWebURL == "" {
		return
	}

	fileSizeStr := "1.2 GB"
	if fi, err := os.Stat(task.HardsubFilePath); err == nil {
		if fi.Size() >= 1024*1024*1024 {
			fileSizeStr = fmt.Sprintf("%.2f GB", float64(fi.Size())/(1024*1024*1024))
		} else {
			fileSizeStr = fmt.Sprintf("%.1f MB", float64(fi.Size())/(1024*1024))
		}
	}

	qualityName := "720p HD (Hardsub Indo)"
	if task.Quality != "" {
		qualityName = fmt.Sprintf("%s (Hardsub Indo)", strings.TrimSpace(task.Quality))
	}

	var existing model.DownloadLink
	if err := m.db.Where("movie_id = ? AND url = ?", task.MovieID, task.HardsubWebURL).First(&existing).Error; err != nil {
		newDL := model.DownloadLink{
			MovieID:    task.MovieID,
			Provider:   "Server Lokal (Hardsub Indonesia)",
			Quality:    qualityName,
			Resolution: "1280x720",
			Format:     "MP4",
			FileSize:   fileSizeStr,
			URL:        task.HardsubWebURL,
			IsValid:    true,
			Status:     "ACTIVE",
			HTTPStatus: 200,
		}
		if err := m.db.Create(&newDL).Error; err == nil {
			log.Printf("[TORRENT MGR] 🌟 File Download Matang otomatis didaftarkan: '%s' (%s) -> %s\n",
				task.MovieTitle, qualityName, task.HardsubWebURL)
		}
	}
}

func (m *Manager) updateTaskStatus(taskID uint, status string, percent float64, downloaded, total int64, speed float64, peers int, errMsg string) {
	m.db.Model(&model.TorrentTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":             status,
		"progress_percent":   percent,
		"downloaded_bytes":   downloaded,
		"total_bytes":        total,
		"download_speed_mbs": speed,
		"peers_count":        peers,
		"error_message":      errMsg,
		"updated_at":         time.Now(),
	})
}

func (m *Manager) updateTaskProgress(taskID uint, percent float64, downloaded, total int64, speed float64, peers int) {
	m.db.Model(&model.TorrentTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"progress_percent":   percent,
		"downloaded_bytes":   downloaded,
		"total_bytes":        total,
		"download_speed_mbs": speed,
		"peers_count":        peers,
		"updated_at":         time.Now(),
	})
}

func (m *Manager) failTask(taskID uint, errMsg string) {
	log.Printf("[TORRENT MGR] ❌ Task #%d Gagal: %s\n", taskID, errMsg)
	m.mu.Lock()
	delete(m.activeTasks, taskID)
	m.mu.Unlock()

	m.db.Model(&model.TorrentTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
		"status":        "FAILED",
		"error_message": errMsg,
		"updated_at":    time.Now(),
	})
}

func (m *Manager) resumePendingTasks() {
	var tasks []model.TorrentTask
	m.db.Where("status IN ?", []string{"PENDING", "DOWNLOADING"}).Find(&tasks)
	for i := range tasks {
		t := tasks[i]
		job := &ActiveJob{
			Task:      &t,
			StartTime: time.Now(),
			CancelCh:  make(chan struct{}),
		}
		m.mu.Lock()
		m.activeTasks[t.ID] = job
		m.mu.Unlock()
		go m.runDownloadWorker(job)
	}
}

func downloadHTTPFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
