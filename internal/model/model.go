package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Name      string    `db:"name" json:"name"`
	Picture   string    `db:"picture" json:"picture"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type GameAccount struct {
    UserID    uuid.UUID `db:"user_id" json:"user_id"`
    UID       string    `db:"uid" json:"uid"`
    CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type OAuthConnection struct {
    UserID     uuid.UUID `db:"user_id"`
    Provider   string    `db:"provider"`
    ProviderID string    `db:"provider_id"`
    CreatedAt  time.Time `db:"created_at"`
}

type OAuthUserInfo struct {
	Provider   string `json:"provider"`
	ProviderID string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Picture    string `json:"picture"`
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}
