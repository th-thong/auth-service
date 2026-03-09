package middleware

import (
	"net/http"
	"strings"

	"gitlab.com/my-game873206/auth-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func JWTAuth(jwtService *service.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try cookie first, then Authorization header
		tokenStr, err := c.Cookie("access_token")
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Authentication credentials were not provided."})
			return
		}

		claims, err := jwtService.ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Token is invalid or expired."})
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Invalid token claims."})
			return
		}

		newLogger := zap.L().With(zap.String("user_id", claims.UserID))
		c.Set("zapLogger", newLogger)

		c.Set("user_id", userID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}
