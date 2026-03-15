package main

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gitlab.com/my-game873206/auth-service/internal/config"
	"gitlab.com/my-game873206/auth-service/internal/handler"
	"gitlab.com/my-game873206/auth-service/internal/logger"
	"gitlab.com/my-game873206/auth-service/internal/middleware"
	"gitlab.com/my-game873206/auth-service/internal/repository"
	"gitlab.com/my-game873206/auth-service/internal/service"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

func main() {
	godotenv.Load(".env")
	cfg := config.Load()
	logger.InitLogger()

	// Database
	db, err := repository.NewDB(cfg.DatabaseURL)
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()
	if err := repository.RunMigrations(db); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.Error(err))
	}

	// Repo
	userRepo := repository.NewUserRepository(db)
	gameAccountRepo := repository.NewGameAccountRepository(db)

	jwtService, err := service.NewJWTService(cfg.JWTPrivateKeyB64, cfg.JWTPublicKeyB64)
	if err != nil {
		zap.L().Fatal("Failed to init JWT service", zap.Error(err))
	}

	// Auth
	jsonData := cfg.FbServiceAcc
	if jsonData == "" {
		zap.L().Fatal("FIREBASE_SERVICE_ACCOUNT is missing")
	}
	opt := option.WithCredentialsJSON([]byte(jsonData))
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		zap.L().Fatal("Failed to initialize Firebase app", zap.Error(err))
	}

	authClient, err := app.Auth(context.Background())
	if err != nil {
		zap.L().Fatal("Failed to get Firebase Auth client", zap.Error(err))
	}

	fbService := &middleware.FirebaseService{AuthClient: authClient}

	oauthService := service.NewOAuthService(cfg, userRepo, jwtService)
	gameAccountService := service.NewGameAccountService(gameAccountRepo)
	authHandler := handler.NewAuthHandler(oauthService, cfg)
	authHandler.SetFirebaseClient(authClient)
	gameAccountHandler := handler.NewGameAccountHandler(gameAccountService, cfg)

	r := gin.New()

	r.Use(gin.Recovery())

	account := r.Group("/account")
	{
		account.POST("/firebase-login", authHandler.FirebaseLogin)

		authMiddleware := account.Group("/")
		authMiddleware.Use(middleware.FirebaseAuth(fbService, userRepo))
		{
			authMiddleware.GET("/user/", authHandler.GetUser)
			authMiddleware.GET("/game-accounts/", gameAccountHandler.List)
			authMiddleware.POST("/game-accounts/", gameAccountHandler.Create)
			authMiddleware.DELETE("/game-accounts/:uid/", gameAccountHandler.Delete)
		}
	}

	zap.L().Info("Server starting", zap.String("port", cfg.Port))
	if err := r.Run("[::]:" + cfg.Port); err != nil {
		zap.L().Fatal("Failed to start server", zap.Error(err))
	}
}
