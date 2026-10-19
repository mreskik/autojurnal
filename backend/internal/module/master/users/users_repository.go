package users

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

func (r *Repository) List(ctx context.Context, q ListUserQuery) ([]User, int, error) {
	var list []User
	query := r.db.NewSelect().
		Model(&list).
		OrderExpr("id ASC").
		Limit(int64(q.Limit)).
		Offset(int64((q.Page - 1) * q.Limit))

	if q.Search != "" {
		like := "%" + q.Search + "%"
		query.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("username ILIKE ?", like).
				WhereOr("name ILIKE ?", like).
				WhereOr("email ILIKE ?", like)
		})
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int(total), nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := new(User)
	err := r.db.NewSelect().Model(user).Where("id = ?", id).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) FindByUsername(ctx context.Context, username string) (*User, error) {
	user := new(User)
	err := r.db.NewSelect().Model(user).Where("lower(username) = lower(?)", username).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *Repository) UsernameExists(ctx context.Context, username string, excludeID uuid.UUID) (bool, error) {
	return r.db.NewSelect().
		Model((*User)(nil)).
		Where("lower(username) = lower(?)", username).
		Where("id <> ?", excludeID).
		Exists(ctx)
}

func (r *Repository) EmailExists(ctx context.Context, email string, excludeID uuid.UUID) (bool, error) {
	return r.db.NewSelect().
		Model((*User)(nil)).
		Where("lower(email) = lower(?)", email).
		Where("id <> ?", excludeID).
		Exists(ctx)
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	_, err := r.db.NewInsert().Model(user).Returning("*").Exec(ctx)
	return err
}

func (r *Repository) Update(ctx context.Context, user *User, columns ...string) error {
	_, err := r.db.NewUpdate().
		Model(user).
		Column(columns...).
		WherePK().
		Returning("*").
		Exec(ctx)
	return err
}

// Delete melakukan soft delete (mengisi deleted_at).
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.NewDelete().Model((*User)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUserNotFound
	}
	return nil
}
