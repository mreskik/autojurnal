package journals

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) ListTrades(ctx context.Context, journalID uuid.UUID, q ListTradeQuery, from, to *time.Time) ([]Trade, int, error) {
	var list []Trade
	query := r.db.NewSelect().
		Model(&list).
		Where("journal_id = ?", journalID).
		OrderExpr("opened_at DESC, id DESC").
		Limit(int64(q.Limit)).
		Offset(int64((q.Page - 1) * q.Limit))

	if q.Status != "" {
		query.Where("status = ?", q.Status)
	}
	if q.Position != "" {
		query.Where("position = ?", q.Position)
	}
	if q.Pair != "" {
		query.Where("pair = ?", q.Pair)
	}
	switch q.Source {
	case "manual":
		query.Where("runner_id IS NULL")
	case "runner":
		query.Where("runner_id IS NOT NULL")
	}
	if from != nil {
		query.Where("opened_at >= ?", *from)
	}
	if to != nil {
		query.Where("opened_at < ?", *to)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int(total), nil
}

// FindRunnerOpenTrade: trade open milik runner untuk pair + position tertentu (guard dobel).
func (r *Repository) FindRunnerOpenTrade(ctx context.Context, journalID, runnerID uuid.UUID, pair, position string) (*Trade, error) {
	t := new(Trade)
	err := r.db.NewSelect().
		Model(t).
		Where("journal_id = ?", journalID).
		Where("runner_id = ?", runnerID).
		Where("pair = ?", pair).
		Where("position = ?", position).
		Where("status = ?", TradeOpen).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return t, err
}

// ListOpenTrades: semua trade open di jurnal (tanpa pagination), terbaru dulu.
func (r *Repository) ListOpenTrades(ctx context.Context, journalID uuid.UUID) ([]Trade, error) {
	var list []Trade
	err := r.db.NewSelect().
		Model(&list).
		Where("journal_id = ?", journalID).
		Where("status = ?", TradeOpen).
		OrderExpr("opened_at DESC, id DESC").
		Scan(ctx)
	return list, err
}

func (r *Repository) FindTrade(ctx context.Context, journalID, id uuid.UUID) (*Trade, error) {
	t := new(Trade)
	err := r.db.NewSelect().
		Model(t).
		Where("id = ?", id).
		Where("journal_id = ?", journalID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTradeNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) CreateTrade(ctx context.Context, t *Trade) error {
	_, err := r.db.NewInsert().Model(t).Returning("*").Exec(ctx)
	return err
}

func (r *Repository) UpdateTrade(ctx context.Context, t *Trade, columns ...string) error {
	_, err := r.db.NewUpdate().
		Model(t).
		Column(columns...).
		WherePK().
		Returning("*").
		Exec(ctx)
	return err
}

// DeleteTrade melakukan soft delete.
func (r *Repository) DeleteTrade(ctx context.Context, journalID, id uuid.UUID) error {
	res, err := r.db.NewDelete().
		Model((*Trade)(nil)).
		Where("id = ?", id).
		Where("journal_id = ?", journalID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTradeNotFound
	}
	return nil
}

func (r *Repository) hasTrades(ctx context.Context, journalID uuid.UUID) (bool, error) {
	return r.db.NewSelect().
		Model((*Trade)(nil)).
		Where("journal_id = ?", journalID).
		Exists(ctx)
}
