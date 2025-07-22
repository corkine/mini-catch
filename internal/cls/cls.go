package cls

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CLSService struct {
	PublicKey    *rsa.PublicKey
	MatchPurpose string
	RemoteServer string
}

func NewCLSService(publicKeyPEM, matchPurpose, remoteServer string) *CLSService {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		panic("failed to parse PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	return &CLSService{
		PublicKey:    pub.(*rsa.PublicKey),
		MatchPurpose: matchPurpose,
		RemoteServer: remoteServer,
	}
}

type JWTClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

// / Auth Using JWT Token, return User
func (s *CLSService) JwtAuth(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.PublicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		if claims.Purpose != s.MatchPurpose {
			return nil, fmt.Errorf("purpose mismatch")
		}
		if claims.ExpiresAt.Before(time.Now()) {
			return nil, fmt.Errorf("token expired")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// / Auth Using Remote Server by quickKey, return JWT Token
func (s *CLSService) TokenAuth(key string) (*JWTClaims, error) {
	url := fmt.Sprintf("%s/s/%s", s.RemoteServer, key)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote server returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tokenClaims struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(body, &tokenClaims)
	if err != nil || tokenClaims.Token == "" {
		return nil, err
	}
	return s.JwtAuth(tokenClaims.Token)
}
