package types

import (
	"crypto/hmac"
	"crypto/sha256"
	"testing"
)

var testKey = []byte("test-hmac-key-0123456789")

func TestSignHMAC_NilKey(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	result := SignHMAC(data, nil)
	if string(result) != string(data) {
		t.Fatalf("SignHMAC with nil key should return data unchanged, got %x", result)
	}
}

func TestSignHMAC_EmptyKey(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	result := SignHMAC(data, []byte{})
	if string(result) != string(data) {
		t.Fatalf("SignHMAC with empty key should return data unchanged")
	}
}

func TestSignHMAC_WithKey(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	signed := SignHMAC(data, testKey)

	if len(signed) != HMACLen+len(data) {
		t.Fatalf("signed length = %d, want %d", len(signed), HMACLen+len(data))
	}

	// Verify that the HMAC is correct
	mac := hmac.New(sha256.New, testKey)
	mac.Write(data)
	expectedSig := mac.Sum(nil)
	sig, payload := signed[:HMACLen], signed[HMACLen:]

	if !hmac.Equal(sig, expectedSig) {
		t.Fatalf("signature mismatch: got %x, want %x", sig, expectedSig)
	}
	if string(payload) != string(data) {
		t.Fatalf("payload mismatch: got %s, want %s", payload, data)
	}
}

func TestVerifyHMAC_NilKey(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	result, ok := VerifyHMAC(data, nil)
	if !ok {
		t.Fatal("VerifyHMAC with nil key should return ok=true")
	}
	if string(result) != string(data) {
		t.Fatal("VerifyHMAC with nil key should return data unchanged")
	}
}

func TestVerifyHMAC_EmptyKey(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	result, ok := VerifyHMAC(data, []byte{})
	if !ok {
		t.Fatal("VerifyHMAC with empty key should return ok=true")
	}
	if string(result) != string(data) {
		t.Fatal("VerifyHMAC with empty key should return data unchanged")
	}
}

func TestVerifyHMAC_Valid(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	signed := SignHMAC(data, testKey)

	result, ok := VerifyHMAC(signed, testKey)
	if !ok {
		t.Fatal("VerifyHMAC should verify a valid signature")
	}
	if string(result) != string(data) {
		t.Fatalf("VerifyHMAC payload mismatch: got %s, want %s", result, data)
	}
}

func TestVerifyHMAC_Tampered(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	signed := SignHMAC(data, testKey)

	// Flip a bit in the payload
	signed[HMACLen+5] ^= 0x01

	_, ok := VerifyHMAC(signed, testKey)
	if ok {
		t.Fatal("VerifyHMAC should reject tampered payload")
	}
}

func TestVerifyHMAC_TamperedSig(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	signed := SignHMAC(data, testKey)

	// Flip a bit in the signature
	signed[0] ^= 0x01

	_, ok := VerifyHMAC(signed, testKey)
	if ok {
		t.Fatal("VerifyHMAC should reject tampered signature")
	}
}

func TestVerifyHMAC_TooShort(t *testing.T) {
	_, ok := VerifyHMAC([]byte{1, 2, 3}, testKey)
	if ok {
		t.Fatal("VerifyHMAC should reject data shorter than HMACLen")
	}
}

func TestVerifyHMAC_WrongKey(t *testing.T) {
	data := []byte(`{"type":"join","room":"R","from":"1"}`)
	signed := SignHMAC(data, testKey)

	wrongKey := []byte("wrong-key-0123456789abcdef")
	_, ok := VerifyHMAC(signed, wrongKey)
	if ok {
		t.Fatal("VerifyHMAC should reject signature with wrong key")
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	payloads := [][]byte{
		[]byte(`{"type":"join","room":"R","from":"1"}`),
		[]byte(`{"type":"udp","room":"R","from":"1","to":"2","payload":"aGVsbG8="}`),
		[]byte(`{"type":"ping"}`),
		{},
	}

	for _, data := range payloads {
		signed := SignHMAC(data, testKey)
		result, ok := VerifyHMAC(signed, testKey)
		if !ok {
			t.Fatalf("VerifyHMAC failed for payload %q", data)
		}
		if string(result) != string(data) {
			t.Fatalf("payload mismatch after round-trip: got %q, want %q", result, data)
		}
	}
}

func TestHMACKey_EnvVar(t *testing.T) {
	ResetHMACKeyForTest()
	defer ResetHMACKeyForTest()

	t.Setenv("REFLECT_RELAY_HMAC_KEY", "env-key-value")

	key := HMACKey()
	if string(key) != "env-key-value" {
		t.Fatalf("HMACKey() = %q, want %q", key, "env-key-value")
	}
}

func TestHMACKey_NoEnv(t *testing.T) {
	ResetHMACKeyForTest()
	defer ResetHMACKeyForTest()

	t.Setenv("REFLECT_RELAY_HMAC_KEY", "")
	key := HMACKey()
	if key != nil {
		t.Fatalf("HMACKey() should be nil when env var is not set, got %v", key)
	}
}
