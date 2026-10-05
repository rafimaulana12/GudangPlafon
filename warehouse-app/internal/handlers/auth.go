package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
	"warehouse-app/internal/models"

	"golang.org/x/crypto/bcrypt"
)


func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	username := strings.TrimSpace(req.Username)
	password := strings.TrimSpace(req.Password)
	if username == "" || password == "" {
		jsonError(w, http.StatusBadRequest, "Username dan password wajib diisi")
		return
	}

	var userID int64
	var passwordHash string
	err := h.db.QueryRow(`SELECT id, password_hash FROM users WHERE username = ?`, username).Scan(&userID, &passwordHash)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		jsonError(w, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	token, err := generateToken()
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Gagal membuat sesi")
		return
	}

	// 7 hari durasi sesi
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = h.db.Exec(`
		INSERT INTO user_sessions (session_token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, token, userID, expiresAt)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Gagal menyimpan sesi")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "warehouse_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   7 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Log audit login
	_, _ = h.db.Exec(`
		INSERT INTO item_audit_logs (item_id, sku, name, action, details)
		VALUES (NULL, 'AUTH', ?, 'LOGIN', ?)
	`, username, fmt.Sprintf(`{"ip": "%s", "time": "%s"}`, getClientIP(r), time.Now().Format(time.RFC3339)))

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"username": username,
	})
}

// POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("warehouse_session"); err == nil && cookie.Value != "" {
		_, _ = h.db.Exec(`DELETE FROM user_sessions WHERE session_token = ?`, cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "warehouse_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

// GET /api/auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("warehouse_session")
	if err != nil || cookie.Value == "" {
		jsonResponse(w, http.StatusOK, models.AuthUserResponse{Authenticated: false})
		return
	}

	var username string
	err = h.db.QueryRow(`
		SELECT u.username
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_token = ? AND s.expires_at > NOW()
	`, cookie.Value).Scan(&username)
	if err != nil {
		jsonResponse(w, http.StatusOK, models.AuthUserResponse{Authenticated: false})
		return
	}

	jsonResponse(w, http.StatusOK, models.AuthUserResponse{
		Authenticated: true,
		Username:      username,
	})
}

// POST /api/auth/change-password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("warehouse_session")
	if err != nil || cookie.Value == "" {
		jsonError(w, http.StatusUnauthorized, "Sesi tidak ditemukan")
		return
	}

	var userID int64
	var currentHash string
	err = h.db.QueryRow(`
		SELECT u.id, u.password_hash
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_token = ? AND s.expires_at > NOW()
	`, cookie.Value).Scan(&userID, &currentHash)
	if err != nil {
		jsonError(w, http.StatusUnauthorized, "Sesi tidak valid")
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if strings.TrimSpace(req.NewPassword) == "" || len(req.NewPassword) < 6 {
		jsonError(w, http.StatusBadRequest, "Password baru minimal 6 karakter")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		jsonError(w, http.StatusBadRequest, "Password lama tidak sesuai")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Gagal mengenkripsi password baru")
		return
	}

	_, err = h.db.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, string(newHash), userID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "Gagal memperbarui password")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password berhasil diubah",
	})
}

// Middleware: RequireAuth
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("warehouse_session")
		if err != nil || cookie.Value == "" {
			jsonError(w, http.StatusUnauthorized, "Sesi tidak valid atau telah berakhir")
			return
		}

		var userID int64
		err = h.db.QueryRow(`
			SELECT user_id FROM user_sessions
			WHERE session_token = ? AND expires_at > NOW()
		`, cookie.Value).Scan(&userID)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "warehouse_session",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			jsonError(w, http.StatusUnauthorized, "Sesi telah berakhir. Silakan login kembali.")
			return
		}

		next(w, r)
	}
}
