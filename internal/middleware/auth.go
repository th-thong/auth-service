package middleware

import (
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"gitlab.com/my-game873206/auth-service/internal/repository"
	"go.uber.org/zap"
)

type FirebaseService struct {
	AuthClient *auth.Client
}

func FirebaseAuth(fbService *FirebaseService, userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		if fbService == nil || fbService.AuthClient == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"detail": "Firebase service not initialized."})
			return
		}

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

		decodedToken, err := fbService.AuthClient.VerifyIDToken(c.Request.Context(), tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "Token is invalid or expired."})
			return
		}

		user, err := userRepo.FindByOAuth("firebase", decodedToken.UID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"detail": "User not found in system."})
			return
		}

		newLogger := zap.L().With(zap.String("user_id", user.ID.String()))
		c.Set("zapLogger", newLogger)

		c.Set("user_id", user.ID)
		c.Set("user_email", user.Email)
		c.Set("firebase_uid", decodedToken.UID)

		c.Next()
	}
}
