package runners

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

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, q ListRunnerQuery) ([]Runner, int, error) {
	var list []Runner
	query := r.db.NewSelect().
		Model(&list).
		Where("user_id = ?", userID).
		OrderExpr("id ASC").
		Limit(int64(q.Limit)).
		Offset(int64((q.Page - 1) * q.Limit))

	if q.Search != "" {
		like := "%" + q.Search + "%"
		query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("name ILIKE ?", like).WhereOr("description ILIKE ?", like)
		})
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int(total), nil
}

// FindByUser mencari runner milik user tertentu; milik user lain dianggap tidak ada.
func (r *Repository) FindByUser(ctx context.Context, userID, id uuid.UUID) (*Runner, error) {
	runner := new(Runner)
	err := r.db.NewSelect().
		Model(runner).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRunnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return runner, nil
}

func (r *Repository) FindByToken(ctx context.Context, token string) (*Runner, error) {
	runner := new(Runner)
	err := r.db.NewSelect().Model(runner).Where("token = ?", token).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, err
	}
	return runner, nil
}

func (r *Repository) Create(ctx context.Context, runner *Runner) error {
	_, err := r.db.NewInsert().Model(runner).Returning("*").Exec(ctx)
	return err
}

func (r *Repository) Update(ctx context.Context, runner *Runner, columns ...string) error {
	_, err := r.db.NewUpdate().
		Model(runner).
		Column(columns...).
		WherePK().
		Returning("*").
		Exec(ctx)
	return err
}

func (r *Repository) Touch(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewUpdate().
		Model((*Runner)(nil)).
		Set("last_used_at = now()").
		Where("id = ?", id).
		Exec(ctx)
	return err
}

// Delete melakukan soft delete (mengisi deleted_at), token ikut tidak berlaku.
func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	res, err := r.db.NewDelete().
		Model((*Runner)(nil)).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRunnerNotFound
	}
	return nil
}
