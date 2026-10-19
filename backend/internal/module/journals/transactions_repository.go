package journals

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

func (r *Repository) ListTransactions(ctx context.Context, journalID uuid.UUID, q ListTransactionQuery, from, to *time.Time) ([]JournalTransaction, int, error) {
	var list []JournalTransaction
	query := r.db.NewSelect().
		Model(&list).
		Where("journal_id = ?", journalID).
		OrderExpr("occurred_at DESC, id DESC").
		Limit(int64(q.Limit)).
		Offset(int64((q.Page - 1) * q.Limit))

	if q.Type != "" {
		query.Where("type = ?", q.Type)
	}
	if from != nil {
		query.Where("occurred_at >= ?", *from)
	}
	if to != nil {
		query.Where("occurred_at < ?", *to)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}
	return list, int(total), nil
}

func (r *Repository) FindTransaction(ctx context.Context, journalID, id uuid.UUID) (*JournalTransaction, error) {
	t := new(JournalTransaction)
	err := r.db.NewSelect().
		Model(t).
		Where("id = ?", id).
		Where("journal_id = ?", journalID).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTransactionNotFound
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *Repository) CreateTransaction(ctx context.Context, t *JournalTransaction) error {
	_, err := r.db.NewInsert().Model(t).Returning("*").Exec(ctx)
	return err
}

func (r *Repository) UpdateTransaction(ctx context.Context, t *JournalTransaction, columns ...string) error {
	_, err := r.db.NewUpdate().
		Model(t).
		Column(columns...).
		WherePK().
		Returning("*").
		Exec(ctx)
	return err
}

// DeleteTransaction melakukan soft delete.
func (r *Repository) DeleteTransaction(ctx context.Context, journalID, id uuid.UUID) error {
	res, err := r.db.NewDelete().
		Model((*JournalTransaction)(nil)).
		Where("id = ?", id).
		Where("journal_id = ?", journalID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTransactionNotFound
	}
	return nil
}

// Balance: total balance saat ini = initial_balance + deposit - withdraw + Σ pnl trade closed.
func (r *Repository) Balance(ctx context.Context, j *Journal) (decimal.Decimal, error) {
	t, err := r.Totals(ctx, j.ID)
	if err != nil {
		return decimal.Zero, err
	}
	return t.Balance(j.InitialBalance), nil
}

// BalanceBefore: balance sebelum waktu tertentu (dasar target percent di awal periode).
func (r *Repository) BalanceBefore(ctx context.Context, j *Journal, before time.Time) (decimal.Decimal, error) {
	var net, pnl decimal.Decimal
	err := r.db.NewSelect().
		Model((*JournalTransaction)(nil)).
		ColumnExpr("COALESCE(SUM(CASE WHEN type = ? THEN amount ELSE -amount END), 0)", TxDeposit).
		Where("journal_id = ?", j.ID).
		Where("occurred_at < ?", before).
		Scan(ctx, &net)
	if err != nil {
		return decimal.Zero, err
	}
	err = r.db.NewSelect().
		Model((*Trade)(nil)).
		ColumnExpr("COALESCE(SUM(pnl), 0)").
		Where("journal_id = ?", j.ID).
		Where("status = ?", TradeClosed).
		Where("closed_at < ?", before).
		Scan(ctx, &pnl)
	if err != nil {
		return decimal.Zero, err
	}
	return j.InitialBalance.Add(net).Add(pnl), nil
}

// ClosedPnLBetween: Σ pnl trade yang closed_at-nya di [from, to).
func (r *Repository) ClosedPnLBetween(ctx context.Context, journalID uuid.UUID, from, to time.Time) (decimal.Decimal, error) {
	var pnl decimal.Decimal
	err := r.db.NewSelect().
		Model((*Trade)(nil)).
		ColumnExpr("COALESCE(SUM(pnl), 0)").
		Where("journal_id = ?", journalID).
		Where("status = ?", TradeClosed).
		Where("closed_at >= ?", from).
		Where("closed_at < ?", to).
		Scan(ctx, &pnl)
	return pnl, err
}

func (r *Repository) Totals(ctx context.Context, journalID uuid.UUID) (Totals, error) {
	m, err := r.TotalsMany(ctx, journalID)
	if err != nil {
		return Totals{}, err
	}
	return m[journalID], nil
}

// TotalsMany menghitung deposit, withdraw, dan pnl closed beberapa jurnal sekaligus (2 query).
func (r *Repository) TotalsMany(ctx context.Context, journalIDs ...uuid.UUID) (map[uuid.UUID]Totals, error) {
	result := make(map[uuid.UUID]Totals, len(journalIDs))
	if len(journalIDs) == 0 {
		return result, nil
	}

	var txRows []struct {
		JournalID uuid.UUID       `bun:"journal_id"`
		Deposit   decimal.Decimal `bun:"deposit"`
		Withdraw  decimal.Decimal `bun:"withdraw"`
	}
	err := r.db.NewSelect().
		Model((*JournalTransaction)(nil)).
		Column("journal_id").
		ColumnExpr("COALESCE(SUM(amount) FILTER (WHERE type = ?), 0) AS deposit", TxDeposit).
		ColumnExpr("COALESCE(SUM(amount) FILTER (WHERE type = ?), 0) AS withdraw", TxWithdraw).
		Where("journal_id IN (?)", bun.In(journalIDs)).
		Group("journal_id").
		Scan(ctx, &txRows)
	if err != nil {
		return nil, err
	}

	var pnlRows []struct {
		JournalID uuid.UUID       `bun:"journal_id"`
		PnL       decimal.Decimal `bun:"pnl"`
	}
	err = r.db.NewSelect().
		Model((*Trade)(nil)).
		Column("journal_id").
		ColumnExpr("COALESCE(SUM(pnl), 0) AS pnl").
		Where("journal_id IN (?)", bun.In(journalIDs)).
		Where("status = ?", TradeClosed).
		Group("journal_id").
		Scan(ctx, &pnlRows)
	if err != nil {
		return nil, err
	}

	for _, row := range txRows {
		t := result[row.JournalID]
		t.Deposit, t.Withdraw = row.Deposit, row.Withdraw
		result[row.JournalID] = t
	}
	for _, row := range pnlRows {
		t := result[row.JournalID]
		t.PnL = row.PnL
		result[row.JournalID] = t
	}
	return result, nil
}

// HasActivity: jurnal sudah punya transaksi atau trade, dipakai untuk mengunci setting jurnal.
func (r *Repository) HasActivity(ctx context.Context, journalID uuid.UUID) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*JournalTransaction)(nil)).
		Where("journal_id = ?", journalID).
		Exists(ctx)
	if err != nil || exists {
		return exists, err
	}
	return r.hasTrades(ctx, journalID)
}
