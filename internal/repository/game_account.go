package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"gitlab.com/my-game873206/auth-service/internal/model"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type GameAccountRepository struct {
	db *sql.DB
}

var ErrDuplicate = errors.New("duplicate entry")

func NewGameAccountRepository(db *sql.DB) *GameAccountRepository {
	return &GameAccountRepository{db: db}
}

func (r *GameAccountRepository) ListByUserID(userID uuid.UUID) ([]model.GameAccount, error) {
	rows, err := r.db.Query(
		`SELECT user_id, uid, created_at
		 FROM game_accounts WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("GameAccountRepository.ListByUserID: %w", err)
	}
	defer rows.Close()

	var accounts []model.GameAccount
	for rows.Next() {
		var a model.GameAccount
		if err := rows.Scan(&a.UserID, &a.UID, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("GameAccountRepository.ListByUserID scan: %w", err)
		}
		accounts = append(accounts, a)
	}
	if rows.Err() != nil {
		return accounts, fmt.Errorf("GameAccountRepository.ListByUserID rows err: %w", rows.Err())
	}
	return accounts, nil
}

func (r *GameAccountRepository) Create(userID uuid.UUID, uid string) (*model.GameAccount, error) {
	a := &model.GameAccount{}
	err := r.db.QueryRow(
		`INSERT INTO game_accounts (user_id, uid)
         VALUES ($1, $2)
         RETURNING user_id, uid, created_at`,
		userID, uid,
	).Scan(&a.UserID, &a.UID, &a.CreatedAt)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("GameAccountRepository.Create: %w", err)
	}
	return a, nil
}

func (r *GameAccountRepository) Delete(userID uuid.UUID, uid string) error {
	result, err := r.db.Exec(
		`DELETE FROM game_accounts WHERE user_id = $1 AND uid = $2`,
		userID, uid,
	)
	if err != nil {
		return fmt.Errorf("GameAccountRepository.Delete: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}