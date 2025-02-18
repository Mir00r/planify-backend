// middleware/auth/jwt.go

package auth

import (
	"fmt"
	"net/http"
	"strings"
	_ "time"

	"github.com/Mir00r/api-gateway/src/api-gateway/src/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware handles JWT authentication
type JWTAuthMiddleware struct {
	config *config.AuthConfig
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// NewJWTAuthMiddleware creates a new JWT authentication middleware
func NewJWTAuthMiddleware(config *config.AuthConfig) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		config: config,
	}
}

// Authenticate is the middleware function to authenticate requests
func (m *JWTAuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := m.extractToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		claims, err := m.validateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Add claims to context for handlers to use
		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// extractToken extracts the JWT token from the request header
func (m *JWTAuthMiddleware) extractToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("no authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// validateToken validates the JWT token
func (m *JWTAuthMiddleware) validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
