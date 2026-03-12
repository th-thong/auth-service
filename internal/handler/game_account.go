package handler

import (
	"errors"
	"net/http"

	"gitlab.com/my-game873206/auth-service/internal/config"
	"gitlab.com/my-game873206/auth-service/internal/model"
	"gitlab.com/my-game873206/auth-service/internal/repository"
	"gitlab.com/my-game873206/auth-service/internal/service"
	"gitlab.com/my-game873206/auth-service/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type GameAccountHandler struct {
	service *service.GameAccountService
	cfg     *config.Config
}

func NewGameAccountHandler(service *service.GameAccountService, cfg *config.Config) *GameAccountHandler {
	return &GameAccountHandler{service: service, cfg: cfg}
}

func (h *GameAccountHandler) List(c *gin.Context) {
	logger := utils.GetLogger(c)
	userID := c.MustGet("user_id").(uuid.UUID)

	accounts, err := h.service.List(userID)
	if err != nil {
		logger.Error("Failed to list game accounts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to fetch game accounts."})
		return
	}

	if accounts == nil {
		accounts = []model.GameAccount{}
	}

	c.JSON(http.StatusOK, accounts)
}

func (h *GameAccountHandler) Create(c *gin.Context) {
	logger := utils.GetLogger(c)
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		UID       string  `json:"uid" binding:"required,max=20"`
		OAuthCode *string `json:"oauth_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for Create game account", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	account, err := h.service.Create(userID, req.UID, req.OAuthCode)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			logger.Warn("Game account creation conflict", zap.String("uid", req.UID), zap.Error(err))
			c.JSON(http.StatusConflict, gin.H{"detail": "Game account with this UID already exists."})
			return
		}
		logger.Error("Failed to create game account", zap.String("uid", req.UID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Internal server error."})
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *GameAccountHandler) Delete(c *gin.Context) {
	logger := utils.GetLogger(c)
	userID := c.MustGet("user_id").(uuid.UUID)
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "UID is required."})
		return
	}

	if err := h.service.Delete(userID, uid); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			logger.Warn("Game account not found for deletion", zap.String("uid", uid), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"detail": "Not found."})
			return
		}
		logger.Error("Failed to delete game account", zap.String("uid", uid), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to delete."})
		return
	}

	token := c.GetHeader("Authorization")
	if err := service.DeleteConvene(h.cfg.ConveneLogUrl, token, uid); err != nil {
		logger.Error("Failed to delete convene records", zap.String("uid", uid), zap.Error(err))
	}

	c.Status(http.StatusNoContent)
}

/*
func (h *GameAccountHandler) UpdateOAuthCode(c *gin.Context) {
	logger := utils.GetLogger(c)
	userID := c.MustGet("user_id").(uuid.UUID)
	uid := c.Param("uid")

	var req struct {
		OAuthCode *string `json:"oauth_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Failed to bind JSON for UpdateOAuthCode", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}

	account, err := h.service.UpdateOAuthCode(userID, uid, req.OAuthCode)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			logger.Warn("Game account not found for update", zap.String("uid", uid), zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"detail": "Not found."})
			return
		}
		logger.Error("Failed to update game account", zap.String("uid", uid), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to update."})
		return
	}

	c.JSON(http.StatusOK, account)
}
*/
