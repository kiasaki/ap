package ap

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

func StringToBase64(message string) string {
	return base64.StdEncoding.EncodeToString([]byte(message))
}

func Base64ToString(message string) string {
	if s, err := base64.StdEncoding.DecodeString(message); err == nil {
		return string(s)
	}
	return ""
}

func SumSHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func SignHMAC256(message, secret string) string {
	sig := hmac.New(sha256.New, []byte(secret))
	sig.Write([]byte(message))
	return hex.EncodeToString(sig.Sum(nil))
}

func CreateToken(value, secret string, mins int) string {
	t := strconv.FormatInt(time.Now().UTC().Add(time.Duration(mins)*time.Minute).Unix(), 10)
	message := StringToBase64(t + "." + value)
	signature := SignHMAC256(message, secret)
	return message + "." + signature
}

func ValidateToken(token, secret string) (string, bool) {
	if token == "" {
		return "no token", false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "not 2 parts", false
	}
	signature := parts[1]
	if signature != SignHMAC256(parts[0], secret) {
		return "signature mismatch", false
	}
	parts = strings.SplitN(Base64ToString(parts[0]), ".", 2)
	expiry, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "can't parse time", false
	}
	if time.Now().UTC().Unix() >= expiry {
		return "expired", false
	}
	return parts[1], true
}
