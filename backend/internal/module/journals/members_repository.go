package journals

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func (r *Repository) joinedRunnerQuery(journalID uuid.UUID) *bun.SelectQuery {
	return r.db.NewSelect().
		TableExpr("journal_runners AS jr").
		Join("JOIN runners AS r ON r.id = jr.runner_id AND r.deleted_at IS NULL").
		ColumnExpr("r.id AS runner_id, r.name, r.is_active, r.expires_at, r.last_used_at, jr.rules, jr.strategy, jr.created_at AS joined_at").
		Where("jr.journal_id = ?", journalID)
}

// ListRunners: runner yang join, runner yang sudah dihapus tidak ikut.
func (r *Repository) ListRunners(ctx context.Context, journalID uuid.UUID) ([]JoinedRunnerResponse, error) {
	list := make([]JoinedRunnerResponse, 0)
	err := r.joinedRunnerQuery(journalID).
		OrderExpr("jr.created_at ASC").
		Scan(ctx, &list)
	return list, err
}

func (r *Repository) FindJoinedRunner(ctx context.Context, journalID, runnerID uuid.UUID) (*JoinedRunnerResponse, error) {
	var res JoinedRunnerResponse
	err := r.joinedRunnerQuery(journalID).
		Where("jr.runner_id = ?", runnerID).
		Scan(ctx, &res)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRunnerNotInJournal
	}
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// ListRunnerOptions: runner milik owner yang belum dihapus dan belum join ke jurnal ini.
func (r *Repository) ListRunnerOptions(ctx context.Context, ownerID, journalID uuid.UUID, search string) ([]RunnerOptionResponse, error) {
	list := make([]RunnerOptionResponse, 0)
	q := r.db.NewSelect().
		TableExpr("runners AS r").
		ColumnExpr("r.id, r.name, r.is_active").
		Where("r.user_id = ?", ownerID).
		Where("r.deleted_at IS NULL").
		Where("NOT EXISTS (SELECT 1 FROM journal_runners AS jr WHERE jr.journal_id = ? AND jr.runner_id = r.id)", journalID).
		OrderExpr("lower(r.name) ASC, r.id ASC")
	if search != "" {
		q.Where("r.name ILIKE ?", "%"+search+"%")
	}
	err := q.Scan(ctx, &list)
	return list, err
}

// AddRunner mengembalikan ErrRunnerAlreadyJoined kalau runner sudah ada di jurnal.
func (r *Repository) AddRunner(ctx context.Context, jr *JournalRunner) error {
	res, err := r.db.NewInsert().
		Model(jr).
		On("CONFLICT DO NOTHING").
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRunnerAlreadyJoined
	}
	return nil
}

// UpdateRunnerInstructions mengubah rules / strategy runner di jurnal (hanya kolom yang dikirim).
func (r *Repository) UpdateRunnerInstructions(ctx context.Context, jr *JournalRunner, columns ...string) error {
	if len(columns) == 0 {
		return nil
	}
	res, err := r.db.NewUpdate().
		Model(jr).
		Column(columns...).
		WherePK().
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRunnerNotInJournal
	}
	return nil
}

func (r *Repository) RemoveRunner(ctx context.Context, journalID, runnerID uuid.UUID) error {
	res, err := r.db.NewDelete().
		Model((*JournalRunner)(nil)).
		Where("journal_id = ?", journalID).
		Where("runner_id = ?", runnerID).
		Exec(ctx)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrRunnerNotInJournal
	}
	return nil
}
