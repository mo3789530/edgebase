package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	NodeID uuid.UUID `json:"node_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type Manager struct {
	jwtSecret string
}

func NewManager(jwtSecret string) *Manager {
	return &Manager{jwtSecret: jwtSecret}
}

// GenerateToken creates a JWT token for a node
func (m *Manager) GenerateToken(nodeID uuid.UUID, expiresIn time.Duration) (string, error) {
	claims := Claims{
		NodeID: nodeID,
		Role:   "node",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   nodeID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.jwtSecret))
}

// VerifyToken validates and parses a JWT token
func (m *Manager) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// RefreshToken generates a new token if the old one is still valid
func (m *Manager) RefreshToken(tokenString string, expiresIn time.Duration) (string, error) {
	claims, err := m.VerifyToken(tokenString)
	if err != nil {
		return "", err
	}

	// Check if token is expired
	if time.Now().After(claims.ExpiresAt.Time) {
		return "", errors.New("token expired")
	}

	// Generate new token
	return m.GenerateToken(claims.NodeID, expiresIn)
}

// HashToken creates a SHA256 hash of a token for storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
