package journals

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Repository struct {
	db bun.IDB
}

func NewRepository(db bun.IDB) *Repository {
	return &Repository{db: db}
}

// RunInTx menjalankan fn dengan repository yang memakai transaction yang sama.
func (r *Repository) RunInTx(ctx context.Context, fn func(ctx context.Context, repo *Repository) error) error {
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return fn(ctx, NewRepository(tx))
	})
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, q ListJournalQuery) ([]Journal, int, error) {
	var list []Journal
	query := r.db.NewSelect().
		Model(&list).
		Where("user_id = ?", userID).
		OrderExpr("id DESC").
		Limit(int64(q.Limit)).
		Offset(int64((q.Page - 1) * q.Limit))

	if q.Status != "" {
		query.Where("status = ?", q.Status)
	}
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

// FindByUser mencari jurnal milik user tertentu; milik user lain dianggap tidak ada.
func (r *Repository) FindByUser(ctx context.Context, userID, id uuid.UUID) (*Journal, error) {
	j := new(Journal)
	err := r.db.NewSelect().
		Model(j).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJournalNotFound
	}
	if err != nil {
		return nil, err
	}
	return j, nil
}

// FindByUserForUpdate sama dengan FindByUser tapi mengunci baris jurnal (SELECT ... FOR UPDATE)
// sampai transaction selesai, supaya perubahan balance tidak balapan.
func (r *Repository) FindByUserForUpdate(ctx context.Context, userID, id uuid.UUID) (*Journal, error) {
	j := new(Journal)
	err := r.db.NewSelect().
		Model(j).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		For("UPDATE").
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJournalNotFound
	}
	if err != nil {
		return nil, err
	}
	return j, nil
}

// runnerJoined: syarat jurnal milik ownerID dan runner terdaftar di jurnal tersebut.
func runnerJoined(q *bun.SelectQuery, ownerID, runnerID uuid.UUID) *bun.SelectQuery {
	return q.
		Where("?TableAlias.user_id = ?", ownerID).
		Where("EXISTS (SELECT 1 FROM journal_runners AS jr WHERE jr.journal_id = ?TableAlias.id AND jr.runner_id = ?)", runnerID)
}

// FindForRunner: jurnal yang boleh diakses runner (milik pemilik runner & runner sudah join).
func (r *Repository) FindForRunner(ctx context.Context, ownerID, runnerID, id uuid.UUID, forUpdate bool) (*Journal, error) {
	j := new(Journal)
	q := runnerJoined(r.db.NewSelect().Model(j).Where("?TableAlias.id = ?", id), ownerID, runnerID)
	if forUpdate {
		q.For("UPDATE")
	}
	err := q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJournalNotFound
	}
	if err != nil {
		return nil, err
	}
	return j, nil
}

// FindByToken mencari jurnal dari journal token (header X-Journal-Token).
func (r *Repository) FindByToken(ctx context.Context, token string) (*Journal, error) {
	j := new(Journal)
	err := r.db.NewSelect().Model(j).Where("token = ?", token).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJournalNotFound
	}
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (r *Repository) Create(ctx context.Context, j *Journal) error {
	_, err := r.db.NewInsert().Model(j).Returning("*").Exec(ctx)
	return err
}

func (r *Repository) Update(ctx context.Context, j *Journal, columns ...string) error {
	_, err := r.db.NewUpdate().
		Model(j).
		Column(columns...).
		WherePK().
		Returning("*").
		Exec(ctx)
	return err
}

// Delete melakukan soft delete.
func (r *Repository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	res, err := r.db.NewDelete().
		Model((*Journal)(nil)).
		Where("id = ?", id).
		Where("user_id = ?", userID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrJournalNotFound
	}
	return nil
}

// ListTargets mengambil riwayat target beberapa jurnal sekaligus,
// urut effective_from lalu id (baris terakhir = terbaru).
func (r *Repository) ListTargets(ctx context.Context, journalIDs ...uuid.UUID) ([]JournalTarget, error) {
	var list []JournalTarget
	if len(journalIDs) == 0 {
		return list, nil
	}
	err := r.db.NewSelect().
		Model(&list).
		Where("journal_id IN (?)", bun.In(journalIDs)).
		OrderExpr("journal_id, effective_from, id").
		Scan(ctx)
	return list, err
}

func (r *Repository) CreateTarget(ctx context.Context, t *JournalTarget) error {
	_, err := r.db.NewInsert().Model(t).Returning("*").Exec(ctx)
	return err
}

func (r *Repository) DeleteTargets(ctx context.Context, journalID uuid.UUID) error {
	_, err := r.db.NewDelete().
		Model((*JournalTarget)(nil)).
		Where("journal_id = ?", journalID).
		Exec(ctx)
	return err
}
