package types

import (
	"crypto/hmac"
	"crypto/sha256"
	"os"
	"sync"
)

// HMACLen is the size of the HMAC-SHA256 signature in bytes (sha256.Size).
const HMACLen = 32

var (
	hmacKey     []byte
	hmacKeyOnce sync.Once
)

// loadHMACKey reads the HMAC key from the environment once.
func loadHMACKey() {
	if k := os.Getenv("REFLECT_RELAY_HMAC_KEY"); k != "" {
		hmacKey = []byte(k)
	}
}

// HMACKey returns the configured HMAC key, or nil if REFLECT_RELAY_HMAC_KEY
// is not set. The result is cached after the first call.
func HMACKey() []byte {
	hmacKeyOnce.Do(loadHMACKey)
	return hmacKey
}

// ResetHMACKeyForTest clears the cached key so the next call to HMACKey
// re-reads the environment variable. Only use in tests.
func ResetHMACKeyForTest() {
	hmacKeyOnce = sync.Once{}
	hmacKey = nil
}

// SignHMAC prepends HMAC-SHA256(key, data) to data.
// If key is empty, returns data unchanged (dev mode).
func SignHMAC(data, key []byte) []byte {
	if len(key) == 0 {
		return data
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	sig := mac.Sum(nil)
	out := make([]byte, HMACLen+len(data))
	copy(out, sig)
	copy(out[HMACLen:], data)
	return out
}

// VerifyHMAC extracts and checks the HMAC-SHA256 signature from signed.
// Returns the original data and true on success.
// If key is empty, accepts signed as-is (dev mode).
func VerifyHMAC(signed, key []byte) ([]byte, bool) {
	if len(key) == 0 {
		return signed, true
	}
	if len(signed) < HMACLen {
		return nil, false
	}
	sig, payload := signed[:HMACLen], signed[HMACLen:]
	mac := hmac.New(sha256.New, key)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if hmac.Equal(sig, expected) {
		return payload, true
	}
	return nil, false
}
