package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var captchaSecret = []byte("CINEMBROT-anti-spam-captcha-token-salt-2026")

type CaptchaChallenge struct {
	Question string
	Token    string
}

// GenerateCaptcha creates a randomized math challenge with signed HMAC token
func GenerateCaptcha() CaptchaChallenge {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	num1 := r.Intn(10) + 1 // 1 - 10
	num2 := r.Intn(10) + 1 // 1 - 10
	op := r.Intn(2)        // 0: +, 1: -

	var question string
	var answer int

	if op == 0 {
		answer = num1 + num2
		question = fmt.Sprintf("Berapa %d + %d = ?", num1, num2)
	} else {
		// Ensure positive outcome
		if num1 < num2 {
			num1, num2 = num2, num1
		}
		answer = num1 - num2
		question = fmt.Sprintf("Berapa %d - %d = ?", num1, num2)
	}

	timestamp := time.Now().Unix()
	payload := fmt.Sprintf("%d:%d", answer, timestamp)

	mac := hmac.New(sha256.New, captchaSecret)
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	token := fmt.Sprintf("%s:%s", payload, sig)

	return CaptchaChallenge{
		Question: question,
		Token:    token,
	}
}

// VerifyCaptcha validates user input against the signed HMAC token
func VerifyCaptcha(userAnswerStr string, token string) bool {
	cleanAnswer := strings.TrimSpace(userAnswerStr)
	if cleanAnswer == "" {
		return false
	}

	userAns, err := strconv.Atoi(cleanAnswer)
	if err != nil {
		return false
	}

	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return false
	}

	expectedAns, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}

	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}

	// Verify expiration (Token valid for 15 minutes)
	if time.Now().Unix()-ts > 900 || time.Now().Unix() < ts-60 {
		return false
	}

	// Verify HMAC signature integrity
	payload := fmt.Sprintf("%d:%d", expectedAns, ts)
	mac := hmac.New(sha256.New, captchaSecret)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return false
	}

	return userAns == expectedAns
}
