package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"gitlab.com/my-game873206/auth-service/internal/config"
	"gitlab.com/my-game873206/auth-service/internal/model"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleProvider struct {
	cfg *config.Config
}

func NewGoogleProvider(cfg *config.Config) *GoogleProvider {
	return &GoogleProvider{cfg: cfg}
}

func (p *GoogleProvider) getConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.cfg.GoogleClientID,
		ClientSecret: p.cfg.GoogleClientSecret,
		RedirectURL:  p.cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email",
		},
		Endpoint: google.Endpoint,
	}
}

func (p *GoogleProvider) ExchangeCode(ctx context.Context, code string) (*model.OAuthUserInfo, error) {
	oauthToken, err := p.getConfig().Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("GoogleProvider.ExchangeCode: failed to exchange code: %w", err)
	}

	userInfo, err := p.fetchUserInfo(ctx, oauthToken.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("GoogleProvider.ExchangeCode: failed to fetch user info: %w", err)
	}

	return userInfo, nil
}

func (p *GoogleProvider) fetchUserInfo(ctx context.Context, accessToken string) (*model.OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("GoogleProvider.fetchUserInfo: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GoogleProvider.fetchUserInfo: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GoogleProvider.fetchUserInfo: read body: %w", err)
	}

	var rawInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &rawInfo); err != nil {
		return nil, fmt.Errorf("GoogleProvider.fetchUserInfo: unmarshal body: %w", err)
	}

	return &model.OAuthUserInfo{
		Provider:   "google",
		ProviderID: rawInfo.ID,
		Email:      rawInfo.Email,
		Name:       rawInfo.Name,
		Picture:    rawInfo.Picture,
	}, nil
}
