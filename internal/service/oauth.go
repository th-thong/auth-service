package service

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/my-game873206/auth-service/internal/config"
	"gitlab.com/my-game873206/auth-service/internal/model"
	"gitlab.com/my-game873206/auth-service/internal/repository"
	"gitlab.com/my-game873206/auth-service/internal/service/oauth"
	"gitlab.com/my-game873206/auth-service/internal/service/oauth/google"

	"golang.org/x/sync/singleflight"
)

type OAuthService struct {
	cfg          *config.Config
	userRepo     *repository.UserRepository
	jwtService   *JWTService
	refreshGroup singleflight.Group
	providers    map[string]oauth.Provider
}

func NewOAuthService(cfg *config.Config, userRepo *repository.UserRepository, jwtService *JWTService) *OAuthService {
	providers := map[string]oauth.Provider{
		"google": google.NewGoogleProvider(cfg),
	}

	return &OAuthService{cfg: cfg, userRepo: userRepo, jwtService: jwtService, providers: providers}
}

func (s *OAuthService) ExchangeCode(ctx context.Context, providerName string, code string) (*model.User, *model.TokenPair, error) {
	provider, ok := s.providers[providerName]
	if !ok {
		return nil, nil, fmt.Errorf("OAuthService.ExchangeCode: unsupported provider %s", providerName)
	}

	userInfo, err := provider.ExchangeCode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("OAuthService.ExchangeCode: failed provider exchange: %w", err)
	}

	user, err := s.userRepo.Upsert(userInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("OAuthService.ExchangeCode: failed to upsert user: %w", err)
	}

	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("OAuthService.ExchangeCode: failed to generate token pair: %w", err)
	}
	return user, tokens, nil
}

func (s *OAuthService) RefreshTokens(ctx context.Context, refreshToken string) (*model.User, *model.TokenPair, error) {
	type refreshResult struct {
		user   *model.User
		tokens *model.TokenPair
	}

	res, err, _ := s.refreshGroup.Do(refreshToken, func() (interface{}, error) {
		tokenHash := HashToken(refreshToken)

		userID, err := s.userRepo.FindRefreshToken(tokenHash)
		if err != nil {
			return nil, fmt.Errorf("OAuthService.RefreshTokens: invalid refresh token: %w", err)
		}

		if _, err := s.userRepo.DeleteRefreshToken(tokenHash); err != nil {
			return nil, fmt.Errorf("OAuthService.RefreshTokens: delete refresh token: %w", err)
		}

		user, err := s.userRepo.FindByID(userID)
		if err != nil {
			return nil, fmt.Errorf("OAuthService.RefreshTokens: find user by id: %w", err)
		}

		tokens, err := s.generateTokenPair(user)
		if err != nil {
			return nil, fmt.Errorf("OAuthService.RefreshTokens: generate tokens: %w", err)
		}

		return &refreshResult{user: user, tokens: tokens}, nil
	})

	if err != nil {
		return nil, nil, err
	}

	result := res.(*refreshResult)
	return result.user, result.tokens, nil
}

func (s *OAuthService) Logout(refreshToken string) (int64, error) {
	tokenHash := HashToken(refreshToken)
	rows, err := s.userRepo.DeleteRefreshToken(tokenHash)
	if err != nil {
		return 0, fmt.Errorf("OAuthService.Logout: %w", err)
	}
	return rows, nil
}

func (s *OAuthService) generateTokenPair(user *model.User) (*model.TokenPair, error) {
	accessToken, err := s.jwtService.GenerateAccessToken(
		user.ID, user.Email,
		time.Duration(s.cfg.AccessTokenMaxAge)*time.Minute,
	)
	if err != nil {
		return nil, fmt.Errorf("OAuthService.generateTokenPair: generate access token: %w", err)
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(
		user.ID,
		time.Duration(s.cfg.RefreshTokenMaxAge)*24*time.Hour,
	)
	if err != nil {
		return nil, fmt.Errorf("OAuthService.generateTokenPair: generate refresh token: %w", err)
	}

	refreshHash := HashToken(refreshToken)
	expiresAt := time.Now().Add(time.Duration(s.cfg.RefreshTokenMaxAge) * 24 * time.Hour)
	if err := s.userRepo.SaveRefreshToken(user.ID, refreshHash, expiresAt); err != nil {
		return nil, fmt.Errorf("OAuthService.generateTokenPair: save refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// UpsertUser creates or updates a user based on OAuth info
func (s *OAuthService) UpsertUser(ctx context.Context, userInfo *model.OAuthUserInfo) (*model.User, error) {
	return s.userRepo.Upsert(userInfo)
}

// GenerateTokenPair generates access and refresh tokens for a user
func (s *OAuthService) GenerateTokenPair(user *model.User) (*model.TokenPair, error) {
	return s.generateTokenPair(user)
}
