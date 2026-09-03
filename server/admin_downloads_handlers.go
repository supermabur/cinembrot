package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"cinembrot/model"
)

// HandleAdminDownloads renders the Torrent & Subtitle downloads dashboard
func (s *Server) HandleAdminDownloads(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)

	var tasks []model.TorrentTask
	if s.torrentMgr != nil {
		tasks, _ = s.torrentMgr.GetTasks()
	}

	data := AdminPageData{
		Title:      "Pengelola Unduhan Torrent & Subtitle - CMS CINEMBROT",
		ActiveMenu: "downloads",
		User:       user,
		Tasks:      tasks,
		SuccessMsg: r.URL.Query().Get("success"),
		ErrorMsg:   r.URL.Query().Get("error"),
	}

	s.RenderHTML(w, "admin_downloads.html", "admin_layout.html", data)
}

// HandleAdminDownloadsStatusAPI returns live JSON status of all torrent tasks for polling
func (s *Server) HandleAdminDownloadsStatusAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.torrentMgr == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"tasks": []model.TorrentTask{}})
		return
	}

	tasks, err := s.torrentMgr.GetTasks()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"tasks":   tasks,
	})
}

// HandleAdminDownloadsAddAPI adds a movie torrent to the background download queue
func (s *Server) HandleAdminDownloadsAddAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	_ = r.ParseMultipartForm(10 << 20)
	_ = r.ParseForm()
	movieIDStr := r.FormValue("movie_id")
	torrentURL := strings.TrimSpace(r.FormValue("torrent_url"))
	quality := strings.TrimSpace(r.FormValue("quality"))

	if torrentURL == "" {
		// Try JSON body
		var payload struct {
			MovieID    uint   `json:"movie_id"`
			TorrentURL string `json:"torrent_url"`
			Quality    string `json:"quality"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			if payload.MovieID > 0 {
				movieIDStr = strconv.Itoa(int(payload.MovieID))
			}
			torrentURL = strings.TrimSpace(payload.TorrentURL)
			quality = strings.TrimSpace(payload.Quality)
		}
	}

	movieID, _ := strconv.Atoi(movieIDStr)
	if torrentURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "URL torrent / magnet tidak boleh kosong"})
		return
	}

	var movie model.Movie
	if err := s.db.First(&movie, movieID).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Film tidak ditemukan di database"})
		return
	}

	if s.torrentMgr == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Torrent manager belum siap"})
		return
	}

	poster := movie.PosterThumbURL
	if poster == "" {
		poster = movie.PosterURL
	}

	task, err := s.torrentMgr.AddTask(movie.ID, movie.Title, movie.Slug, poster, torrentURL, quality)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Gagal membuat antrean download: %v", err)})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Film '%s' (%s) berhasil ditambahkan ke antrean unduhan torrent!", movie.Title, quality),
		"task_id": task.ID,
	})
}

// HandleAdminDownloadsCancelAPI cancels or deletes a download task
func (s *Server) HandleAdminDownloadsCancelAPI(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, _ := strconv.Atoi(idStr)

	if s.torrentMgr != nil && id > 0 {
		var task model.TorrentTask
		if err := s.db.First(&task, id).Error; err == nil {
			if task.Status == "DOWNLOADING" || task.Status == "PENDING" {
				_ = s.torrentMgr.CancelTask(uint(id))
			} else {
				_ = s.torrentMgr.DeleteTask(uint(id))
			}
		} else {
			_ = s.torrentMgr.DeleteTask(uint(id))
		}
	}

	if r.Header.Get("Accept") == "application/json" || strings.Contains(r.Header.Get("Content-Type"), "json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		return
	}

	http.Redirect(w, r, "/admin/downloads?success=Task+berhasil+dihapus", http.StatusSeeOther)
}

// HandleAdminDownloadsReview renders the video player preview with multi-subtitle switcher
func (s *Server) HandleAdminDownloadsReview(w http.ResponseWriter, r *http.Request) {
	user := s.GetLoggedInUser(r)
	idStr := r.PathValue("id")
	id, _ := strconv.Atoi(idStr)

	if s.torrentMgr == nil || id == 0 {
		http.Redirect(w, r, "/admin/downloads?error=Task+tidak+valid", http.StatusSeeOther)
		return
	}

	task, err := s.torrentMgr.GetTask(uint(id))
	if err != nil {
		http.Redirect(w, r, "/admin/downloads?error=Task+tidak+ditemukan", http.StatusSeeOther)
		return
	}

	var subtitles []model.SubtitleOption
	if task.AvailableSubtitles != "" {
		_ = json.Unmarshal([]byte(task.AvailableSubtitles), &subtitles)
	}

	data := AdminPageData{
		Title:      fmt.Sprintf("Review Subtitle: %s - CMS CINEMBROT", task.MovieTitle),
		ActiveMenu: "downloads",
		User:       user,
		Task:       task,
		Subtitles:  subtitles,
		SuccessMsg: r.URL.Query().Get("success"),
		ErrorMsg:   r.URL.Query().Get("error"),
	}

	s.RenderHTML(w, "admin_download_review.html", "admin_layout.html", data)
}

// HandleAdminDownloadsSelectSubtitleAPI updates the selected subtitle candidate for hardsubbing
func (s *Server) HandleAdminDownloadsSelectSubtitleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("id")
	taskID, _ := strconv.Atoi(idStr)
	subtitleID := r.FormValue("subtitle_id")

	if s.torrentMgr == nil || taskID == 0 || subtitleID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Parameter tidak valid"})
		return
	}

	if err := s.torrentMgr.SelectSubtitle(uint(taskID), subtitleID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"subtitle_id": subtitleID,
	})
}

// HandleAdminDownloadsUploadSubtitleAPI saves an uploaded custom .srt file for the task
func (s *Server) HandleAdminDownloadsUploadSubtitleAPI(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	taskID, _ := strconv.Atoi(idStr)

	if s.torrentMgr == nil || taskID == 0 {
		http.Redirect(w, r, "/admin/downloads?error=Task+tidak+valid", http.StatusSeeOther)
		return
	}

	_ = r.ParseMultipartForm(10 << 20) // 10MB max
	file, header, err := r.FormFile("subtitle_file")
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/downloads/review/%d?error=Gagal+membaca+file+subtitle", taskID), http.StatusSeeOther)
		return
	}
	defer file.Close()

	srtBytes, err := io.ReadAll(file)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/downloads/review/%d?error=Gagal+membaca+konten+file", taskID), http.StatusSeeOther)
		return
	}

	_, err = s.torrentMgr.SaveCustomSubtitle(uint(taskID), header.Filename, srtBytes)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/admin/downloads/review/%d?error=Gagal+menyimpan+subtitle:+%v", taskID, err), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/admin/downloads/review/%d?success=Subtitle+kustom+berhasil+diunggah+dan+dikonversi!", taskID), http.StatusSeeOther)
}

// HandleAdminDownloadsHardsubAPI triggers the FFmpeg hardsub render process
func (s *Server) HandleAdminDownloadsHardsubAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := r.PathValue("id")
	taskID, _ := strconv.Atoi(idStr)

	if s.torrentMgr == nil || taskID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Task tidak valid"})
		return
	}

	if err := s.torrentMgr.StartHardsub(uint(taskID)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Rendering hardsub dimulai di latar belakang! Subtitle akan dicetak permanen ke video.",
	})
}
