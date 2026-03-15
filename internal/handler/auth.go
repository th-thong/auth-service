package handler

import (
	"net/http"

	"firebase.google.com/go/v4/auth"
	"gitlab.com/my-game873206/auth-service/internal/config"
	"gitlab.com/my-game873206/auth-service/internal/model"
	"gitlab.com/my-game873206/auth-service/internal/service"
	"gitlab.com/my-game873206/auth-service/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"strings"
)

type AuthHandler struct {
	oauthService *service.OAuthService
	cfg          *config.Config
	authClient   *auth.Client
}

func NewAuthHandler(oauthService *service.OAuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{oauthService: oauthService, cfg: cfg}
}

func (h *AuthHandler) SetFirebaseClient(authClient *auth.Client) {
	h.authClient = authClient
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	logger := utils.GetLogger(c)

	var req struct {
		Code        string `json:"code" binding:"required"`
		RedirectURI string `json:"redirect_uri"`
		ClientID    string `json:"client_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("GoogleLogin bind error", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	user, tokens, err := h.oauthService.ExchangeCode(c.Request.Context(), "google", req.Code)
	if err != nil {
		logger.Error("ExchangeCode error", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to exchange code for access token"})
		return
	}

	logger.Info("OAuth exchange successfully", zap.String("user_id", user.ID.String()))

	h.setTokenCookies(c, tokens)
	c.JSON(http.StatusOK, gin.H{
		"access":  tokens.AccessToken,
		"refresh": tokens.RefreshToken,
		"user":    formatUser(user),
	})
}

func (h *AuthHandler) FirebaseLogin(c *gin.Context) {
	logger := utils.GetLogger(c)

	if h.authClient == nil {
		logger.Error("Firebase auth client not initialized")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Firebase service not available"})
		return
	}

	var req struct {
		IDToken string `json:"id_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("FirebaseLogin bind error", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token is required"})
		return
	}

	decodedToken, err := h.authClient.VerifyIDToken(c.Request.Context(), req.IDToken)
	if err != nil {
		logger.Error("Firebase token verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
		return
	}

	firebaseUID := decodedToken.UID
	email, emailOk := decodedToken.Claims["email"].(string)
	name, _ := decodedToken.Claims["name"].(string)
	picture, _ := decodedToken.Claims["picture"].(string)

	if !emailOk || email == "" {
		logger.Warn("Firebase token does not contain an email")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required for login"})
		return
	}

	isAllowed := false
	emailLower := strings.ToLower(email)
	for _, allowedEmail := range h.cfg.WhiteList {
		if emailLower == strings.ToLower(allowedEmail) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		logger.Warn("Unauthorised login", zap.String("email", email))
		c.JSON(http.StatusForbidden, gin.H{
			"error": "This is a private system; you do not have access to it.",
		})
		return
	}

	userInfo := &model.OAuthUserInfo{
		Provider:   "firebase",
		ProviderID: firebaseUID,
		Email:      email,
		Name:       name,
		Picture:    picture,
	}

	user, err := h.oauthService.UpsertUser(c.Request.Context(), userInfo)
	if err != nil {
		logger.Error("Failed to upsert user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync user data"})
		return
	}

	claims := map[string]interface{}{
		"user_id": user.ID.String(),
	}

	err = h.authClient.SetCustomUserClaims(c.Request.Context(), firebaseUID, claims)
	if err != nil {
		logger.Error("Lỗi khi set custom claims", zap.Error(err))
	}

	logger.Info("Firebase login successful - User synced", zap.String("user_id", user.ID.String()))

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    formatUser(user),
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	logger := utils.GetLogger(c)
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		var req struct {
			Refresh string `json:"refresh"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Refresh == "" {
			logger.Error("RefreshToken bind error", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"detail": "Refresh token is required."})
			return
		}
		refreshToken = req.Refresh
	}

	user, tokens, err := h.oauthService.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		logger.Error("RefreshTokens error", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid or expired refresh token."})
		return
	}

	h.setTokenCookies(c, tokens)
	logger.Info("Refresh token sucessfully")
	c.JSON(http.StatusOK, gin.H{
		"access":  tokens.AccessToken,
		"refresh": tokens.RefreshToken,
		"user":    formatUser(user),
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	logger := utils.GetLogger(c)
	refreshToken, _ := c.Cookie("refresh_token")

	rows, err := h.oauthService.Logout(refreshToken)
	if err != nil || rows != 1 {
		if err != nil {
			logger.Error("Logout error", zap.Error(err))
		}
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Logout failed."})
		return
	}

	logger.Info("Logout sucessfullly")
	h.clearTokenCookies(c)
	c.JSON(http.StatusOK, gin.H{"detail": "Successfully logged out."})
}

func (h *AuthHandler) GetUser(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	email := c.MustGet("user_email").(string)

	c.JSON(http.StatusOK, gin.H{
		"pk":    userID.String(),
		"email": email,
	})
}

func (h *AuthHandler) setTokenCookies(c *gin.Context, tokens *model.TokenPair) {
	accessMaxAge := h.cfg.AccessTokenMaxAge * 60
	refreshMaxAge := h.cfg.RefreshTokenMaxAge * 24 * 60 * 60

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokens.AccessToken, accessMaxAge,
		"/", h.cfg.CookieDomain, true, true)
	c.SetCookie("refresh_token", tokens.RefreshToken, refreshMaxAge,
		"/", h.cfg.CookieDomain, true, true)
}

func (h *AuthHandler) clearTokenCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", h.cfg.CookieDomain, true, true)
	c.SetCookie("refresh_token", "", -1, "/", h.cfg.CookieDomain, true, true)
}

func formatUser(user *model.User) gin.H {
	return gin.H{
		"pk":      user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"picture": user.Picture,
	}
}
