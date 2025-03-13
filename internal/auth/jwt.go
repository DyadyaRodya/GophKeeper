package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Claims custom JWT claims
type Claims struct {
	jwt.RegisteredClaims
	UserUUID string
}

// JWTService service to extract user uuid from jwt token and generate new token and user uuid
type JWTService struct {
	secretKey []byte
	ttl       time.Duration
}

// NewJWTService Constructor for JWTService
func NewJWTService(secretKey []byte, ttl time.Duration) *JWTService {
	return &JWTService{
		secretKey: secretKey,
		ttl:       ttl,
	}
}

// ProcessToken verifies token claims, parses user UUID
func (s *JWTService) ProcessToken(token string) (userUUID string, authenticated bool) {
	if token == "" {
		return "", false
	}

	userUUID = s.getUserUUID(token, s.secretKey)
	if userUUID != "" {
		return userUUID, true
	}

	return "", false
}

// GenerateToken generates new jwt token with user uuid
func (s *JWTService) GenerateToken(userUUID string) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{},
		UserUUID:         userUUID,
	}

	if s.ttl > 0 { // allow to be infinite if ttl == 0
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(s.ttl))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", err
	}
	return tokenString, err
}

func (s *JWTService) getUserUUID(tokenString string, secretKey []byte) string {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}
	return claims.UserUUID
}
