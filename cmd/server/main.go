package main

import (
	"gitlab.com/my-game873206/auth-service/internal/config"
	"gitlab.com/my-game873206/auth-service/internal/handler"
	"gitlab.com/my-game873206/auth-service/internal/logger"
	"gitlab.com/my-game873206/auth-service/internal/middleware"
	"gitlab.com/my-game873206/auth-service/internal/repository"
	"gitlab.com/my-game873206/auth-service/internal/service"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")
	cfg := config.Load()
	logger.InitLogger()

	db, err := repository.NewDB(cfg.DatabaseURL)
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := repository.RunMigrations(db); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.Error(err))
	}

	userRepo := repository.NewUserRepository(db)
	gameAccountRepo := repository.NewGameAccountRepository(db)

	jwtService, err := service.NewJWTService(cfg.JWTPrivateKeyB64, cfg.JWTPublicKeyB64)
	if err != nil {
		zap.L().Fatal("Failed to init JWT service", zap.Error(err))
	}

	oauthService := service.NewOAuthService(cfg, userRepo, jwtService)
	gameAccountService := service.NewGameAccountService(gameAccountRepo)

	authHandler := handler.NewAuthHandler(oauthService, cfg)
	gameAccountHandler := handler.NewGameAccountHandler(gameAccountService)

	r := gin.New()

	r.Use(gin.Recovery())

	account := r.Group("/account")
	{
		account.POST("/login/google/", authHandler.GoogleCallback)
		account.POST("/refresh/", authHandler.RefreshToken)
		account.POST("/logout/", authHandler.Logout)

		auth := account.Group("/")
		auth.Use(middleware.JWTAuth(jwtService))
		{
			auth.GET("/user/", authHandler.GetUser)
			auth.GET("/game-accounts/", gameAccountHandler.List)
			auth.POST("/game-accounts/", gameAccountHandler.Create)
			auth.DELETE("/game-accounts/:uid/", gameAccountHandler.Delete)
			auth.PATCH("/game-accounts/:uid/", gameAccountHandler.UpdateOAuthCode)
		}
	}

	zap.L().Info("Server starting", zap.String("port", cfg.Port))
	if err := r.Run("[::]:" + cfg.Port); err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}
