package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"gitlab.com/my-game873206/auth-service/internal/model"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByOAuth(provider, providerID string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT u.id, u.email, u.name, u.picture, u.created_at, u.updated_at
		 FROM users u
		 JOIN oauth_connections o ON u.id = o.user_id
		 WHERE o.provider = $1 AND o.provider_id = $2`, provider, providerID,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Picture, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepository.FindByOAuth: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRow(
		`SELECT id, email, name, picture, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Picture, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepository.FindByID: %w", err)
	}
	return user, nil
}

func (r *UserRepository) Upsert(info *model.OAuthUserInfo) (*model.User, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("UserRepository.Upsert: failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	user := &model.User{}
	err = tx.QueryRow(
		`INSERT INTO users (email, name, picture)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO UPDATE
		   SET name = EXCLUDED.name,
		       picture = EXCLUDED.picture,
		       updated_at = NOW()
		 RETURNING id, email, name, picture, created_at, updated_at`,
		info.Email, info.Name, info.Picture,
	).Scan(&user.ID, &user.Email, &user.Name, &user.Picture, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("UserRepository.Upsert: failed to upsert user: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO oauth_connections (user_id, provider, provider_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (provider, provider_id) DO NOTHING`,
		user.ID, info.Provider, info.ProviderID,
	)
	if err != nil {
		return nil, fmt.Errorf("UserRepository.Upsert: failed to upsert oauth connection: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("UserRepository.Upsert: failed to commit tx: %w", err)
	}

	return user, nil
}

func (r *UserRepository) SaveRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("UserRepository.SaveRefreshToken: %w", err)
	}
	return nil
}

func (r *UserRepository) FindRefreshToken(tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	err := r.db.QueryRow(
		`SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("UserRepository.FindRefreshToken: %w", err)
	}
	if time.Now().After(expiresAt) {
		return uuid.Nil, fmt.Errorf("UserRepository.FindRefreshToken: refresh token expired")
	}
	return userID, nil
}

func (r *UserRepository) DeleteRefreshToken(tokenHash string) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return 0, fmt.Errorf("UserRepository.DeleteRefreshToken: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("UserRepository.DeleteRefreshToken get rows: %w", err)
	}
	return rowsAffected, nil
}
