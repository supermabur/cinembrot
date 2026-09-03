package server

import (
	"net/http"

	"cinembrot/auth"
	"cinembrot/model"
)

// GetLoggedInUser extracts the authenticated user from request cookie
func (s *Server) GetLoggedInUser(r *http.Request) *model.User {
	cookie, err := r.Cookie(auth.CookieSessionName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	username, valid := auth.ValidateSessionToken(cookie.Value)
	if !valid {
		return nil
	}

	var user model.User
	if err := s.db.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		// Fallback for default config admin if not yet in DB
		if username == s.cfg.AdminDefaultUser {
			return &model.User{
				Username: s.cfg.AdminDefaultUser,
				FullName: "Administrator",
				Role:     "admin",
				IsActive: true,
			}
		}
		return nil
	}

	return &user
}

// RequireAdmin middleware guards protected CMS routes
func (s *Server) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := s.GetLoggedInUser(r)
		if user == nil {
			http.Redirect(w, r, "/admin/login?redirect="+r.URL.Path, http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
