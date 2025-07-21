package console

import (
	"crypto/rand"
	"crypto/tls"
	_ "embed"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var hmacKey = []byte("shared-secret-key")

func sign(data []byte) []byte {
	//mac := hmac.New(sha256.New, hmacKey)
	//mac.Write(data)
	//return append(mac.Sum(nil), data...)
	return data
}

func verify(packet []byte) ([]byte, bool) {
	return packet, true
	//if len(packet) < 32 {
	//	return nil, false
	//}
	//sig := packet[:32]
	//data := packet[32:]
	//
	//mac := hmac.New(sha256.New, hmacKey)
	//mac.Write(data)
	//expected := mac.Sum(nil)
	//if hmac.Equal(sig, expected) {
	//	return data, true
	//}
	//return nil, false
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

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

var jwtSecret = []byte("your-very-secret-key")

func generateJWT(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateJWT(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid claims")
	}
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("user_id missing")
	}
	return int64(userID), nil
}
