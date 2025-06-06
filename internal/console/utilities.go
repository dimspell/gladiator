package console

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	_ "embed"
)

var hmacKey = []byte("shared-secret-key")

func sign(data []byte) []byte {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(data)
	return append(mac.Sum(nil), data...)
}

func verify(packet []byte) ([]byte, bool) {
	if len(packet) < 32 {
		return nil, false
	}
	sig := packet[:32]
	data := packet[32:]

	mac := hmac.New(sha256.New, hmacKey)
	mac.Write(data)
	expected := mac.Sum(nil)
	if hmac.Equal(sig, expected) {
		return data, true
	}
	return nil, false
}

func generateSelfSigned() tls.Certificate {
	// For development only. Replace with proper TLS cert in production.
	cert, _ := tls.X509KeyPair(devCertPEM, devKeyPEM)
	return cert
}

// Dev-only TLS cert
//
// You can generate real self-signed certs using:
//
// 	openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

//go:embed cert.pem
var devCertPEM []byte

//go:embed key.pem
var devKeyPEM []byte
