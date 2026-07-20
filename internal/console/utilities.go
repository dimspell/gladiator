package console

import (
	"crypto/rand"
	"crypto/tls"
	_ "embed"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/dimspell/gladiator/internal/backend/proxy/relay/types"
	"github.com/golang-jwt/jwt/v5"
)

func sign(data []byte) []byte { //nolint:unused // may be used in future
	return types.SignHMAC(data, types.HMACKey())
}

func verifyRelayPacket(packet []byte) ([]byte, bool) {
	return types.VerifyHMAC(packet, types.HMACKey())
}

func generateSelfSigned() tls.Certificate { //nolint:unused // may be used in future
	cert, _ := tls.X509KeyPair(devCertPEM, devKeyPEM)
	return cert
}

//go:embed cert.pem
var devCertPEM []byte //nolint:unused // may be used in future

//go:embed key.pem
var devKeyPEM []byte //nolint:unused // may be used in future

func generateToken() (string, error) { //nolint:unused // may be used in future
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

var jwtSecret = []byte("your-very-secret-key") //nolint:unused // may be used in future

func generateJWT(userID int64) (string, error) { //nolint:unused // may be used in future
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func validateJWT(tokenString string) (int64, error) { //nolint:unused // may be used in future
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
