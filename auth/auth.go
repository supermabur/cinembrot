package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var sessionSecret = []byte("CINEMBROT-admin-cms-session-secret-key-2026")

const (
	CookieSessionName = "CINEMBROT_admin_session"
	SessionDuration   = 24 * time.Hour
)

// HashPassword hashes a plain text password with a random salt
func HashPassword(password string) string {
	saltBytes := make([]byte, 16)
	_, _ = rand.Read(saltBytes)
	salt := hex.EncodeToString(saltBytes)

	h := sha256.New()
	h.Write([]byte(salt + password))
	hash := hex.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("$s256$%s$%s", salt, hash)
}

// CheckPasswordHash verifies a plain text password against the hashed format
func CheckPasswordHash(password, storedHash string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 4 || parts[1] != "s256" {
		// Fallback for simple equality if plain text
		return password == storedHash
	}

	salt := parts[2]
	expectedHash := parts[3]

	h := sha256.New()
	h.Write([]byte(salt + password))
	hash := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(hash), []byte(expectedHash))
}

// GenerateSessionToken creates a signed HMAC token for cookies
func GenerateSessionToken(username string) string {
	ts := time.Now().Unix()
	payload := fmt.Sprintf("%s:%d", username, ts)

	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s:%s", payload, sig)
}

// ValidateSessionToken checks the cookie signature and expiration
func ValidateSessionToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}

	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return "", false
	}

	username := parts[0]
	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", false
	}

	// 24 Hour expiry check
	if time.Now().Unix()-ts > int64(SessionDuration.Seconds()) || time.Now().Unix() < ts-60 {
		return "", false
	}

	// Verify HMAC signature
	payload := fmt.Sprintf("%s:%d", username, ts)
	mac := hmac.New(sha256.New, sessionSecret)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return "", false
	}

	return username, true
}
