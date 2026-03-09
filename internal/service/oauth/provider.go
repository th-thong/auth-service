package oauth

import (
	"context"

	"gitlab.com/my-game873206/auth-service/internal/model"
)

type Provider interface {
	ExchangeCode(ctx context.Context, code string) (*model.OAuthUserInfo, error)
}
