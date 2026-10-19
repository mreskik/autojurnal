package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository struct {
	db *bun.DB
}

func NewRepository(db *bun.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, s *Session) error {
	_, err := r.db.NewInsert().Model(s).Returning("*").Exec(ctx)
	return err
}

// FindActiveByTokenHash mencari session yang belum expired.
func (r *Repository) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*Session, error) {
	s := new(Session)
	err := r.db.NewSelect().
		Model(s).
		Where("token_hash = ?", tokenHash).
		Where("expires_at > now()").
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) ListActiveByUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	var list []Session
	err := r.db.NewSelect().
		Model(&list).
		Where("user_id = ?", userID).
		Where("expires_at > now()").
		OrderExpr("last_used_at DESC").
		Scan(ctx)
	return list, err
}

func (r *Repository) Touch(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*Session)(nil)).
		Set("last_used_at = now()").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *Repository) DeleteByID(ctx context.Context, userID, id uuid.UUID) error {
	res, err := r.db.NewDelete().
		Model((*Session)(nil)).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *Repository) DeleteByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*Session)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)
	return err
}

func (r *Repository) DeleteExpiredByUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*Session)(nil)).
		Where("user_id = ?", userID).
		Where("expires_at <= now()").
		Exec(ctx)
	return err
}
