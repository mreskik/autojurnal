package journals

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// journalData: semua data satu jurnal yang dibutuhkan dashboard, dimuat sekali lalu dihitung di memori.
type journalData struct {
	j       *Journal
	loc     *time.Location
	now     time.Time
	closed  []Trade              // trade closed, urut closed_at
	open    []Trade              // trade open
	txs     []JournalTransaction // deposit / withdraw, urut occurred_at
	targets []JournalTarget      // riwayat target, urut effective_from, id
	events  []balanceEvent       // trade closed + transaksi, urut waktu
}

// balanceEvent: kejadian yang mengubah balance.
type balanceEvent struct {
	at    time.Time
	pnl   decimal.Decimal // dari trade closed
	flow  decimal.Decimal // deposit (+) / withdraw (-)
	trade *Trade          // nil untuk transaksi
}

func (s *Service) loadJournalData(ctx context.Context, j *Journal) (*journalData, error) {
	loc, err := time.LoadLocation(j.Timezone)
	if err != nil {
		return nil, err
	}
	d := &journalData{j: j, loc: loc, now: time.Now()}

	if d.closed, err = s.repo.ListClosedTrades(ctx, j.ID); err != nil {
		return nil, err
	}
	if d.open, err = s.repo.ListOpenTrades(ctx, j.ID); err != nil {
		return nil, err
	}
	if d.txs, err = s.repo.ListAllTransactions(ctx, j.ID); err != nil {
		return nil, err
	}
	if d.targets, err = s.repo.ListTargets(ctx, j.ID); err != nil {
		return nil, err
	}

	d.events = make([]balanceEvent, 0, len(d.closed)+len(d.txs))
	for i := range d.closed {
		t := &d.closed[i]
		d.events = append(d.events, balanceEvent{at: *t.ClosedAt, pnl: *t.PnL, trade: t})
	}
	for _, tx := range d.txs {
		flow := tx.Amount
		if tx.Type == TxWithdraw {
			flow = flow.Neg()
		}
		d.events = append(d.events, balanceEvent{at: tx.OccurredAt, flow: flow})
	}
	sort.SliceStable(d.events, func(a, b int) bool { return d.events[a].at.Before(d.events[b].at) })
	return d, nil
}

// totals: agregat deposit / withdraw / pnl dari data yang sudah dimuat.
func (d *journalData) totals() Totals {
	var t Totals
	for _, tx := range d.txs {
		if tx.Type == TxDeposit {
			t.Deposit = t.Deposit.Add(tx.Amount)
		} else {
			t.Withdraw = t.Withdraw.Add(tx.Amount)
		}
	}
	for _, tr := range d.closed {
		t.PnL = t.PnL.Add(*tr.PnL)
	}
	return t
}

// today: tanggal hari ini di timezone jurnal (UTC midnight sebagai tanggal sipil).
func (d *journalData) today() time.Time {
	return civilDate(d.now, d.loc)
}

// instant: awal tanggal sipil di timezone jurnal.
func (d *journalData) instant(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, d.loc)
}

func (d *journalData) startDate() time.Time {
	t, _ := time.Parse(dateLayout, d.j.StartDate)
	return t
}

func civilDate(t time.Time, loc *time.Location) time.Time {
	y, m, day := t.In(loc).Date()
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

// closedInRange: trade closed dengan closed_at di [from, to); nil = tanpa batas.
func (d *journalData) closedInRange(from, to *time.Time) []Trade {
	if from == nil && to == nil {
		return d.closed
	}
	list := make([]Trade, 0, len(d.closed))
	for _, t := range d.closed {
		if from != nil && t.ClosedAt.Before(*from) {
			continue
		}
		if to != nil && !t.ClosedAt.Before(*to) {
			continue
		}
		list = append(list, t)
	}
	return list
}

// ---------- repository khusus dashboard ----------

func (r *Repository) ListClosedTrades(ctx context.Context, journalID uuid.UUID) ([]Trade, error) {
	var list []Trade
	err := r.db.NewSelect().
		Model(&list).
		Where("journal_id = ?", journalID).
		Where("status = ?", TradeClosed).
		OrderExpr("closed_at ASC, id ASC").
		Scan(ctx)
	return list, err
}

func (r *Repository) ListAllTransactions(ctx context.Context, journalID uuid.UUID) ([]JournalTransaction, error) {
	var list []JournalTransaction
	err := r.db.NewSelect().
		Model(&list).
		Where("journal_id = ?", journalID).
		OrderExpr("occurred_at ASC, id ASC").
		Scan(ctx)
	return list, err
}

// ListAllByUser: semua jurnal milik user (tanpa pagination) untuk overview.
func (r *Repository) ListAllByUser(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]Journal, error) {
	var list []Journal
	q := r.db.NewSelect().
		Model(&list).
		Where("user_id = ?", userID).
		OrderExpr("id DESC")
	if !includeArchived {
		q.Where("status = ?", StatusActive)
	}
	err := q.Scan(ctx)
	return list, err
}

// RunnerNames: nama semua runner milik user (termasuk yang sudah dihapus, untuk riwayat).
func (r *Repository) RunnerNames(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]string, error) {
	var rows []struct {
		ID   uuid.UUID `bun:"id"`
		Name string    `bun:"name"`
	}
	err := r.db.NewRaw("SELECT id, name FROM runners WHERE user_id = ?", userID).Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

func (r *Repository) RecentTrades(ctx context.Context, userID uuid.UUID, limit int) ([]RecentTradeItem, error) {
	list := make([]RecentTradeItem, 0)
	err := r.db.NewRaw(`
		SELECT t.id, t.journal_id, j.name AS journal_name, t.runner_id, r.name AS runner_name,
		       t.pair, t.position, t.status, t.pnl, t.opened_at, t.closed_at, t.updated_at
		FROM trades AS t
		JOIN journals AS j ON j.id = t.journal_id AND j.deleted_at IS NULL
		LEFT JOIN runners AS r ON r.id = t.runner_id
		WHERE j.user_id = ? AND t.deleted_at IS NULL
		ORDER BY t.updated_at DESC, t.id DESC
		LIMIT ?`, userID, limit).Scan(ctx, &list)
	return list, err
}

func (r *Repository) RunnerMonitor(ctx context.Context, userID uuid.UUID) ([]RunnerMonitorItem, error) {
	list := make([]RunnerMonitorItem, 0)
	err := r.db.NewRaw(`
		SELECT r.id, r.name, r.is_active, r.expires_at, r.last_used_at,
		       (SELECT count(*) FROM journal_runners AS jr
		          JOIN journals AS j ON j.id = jr.journal_id AND j.deleted_at IS NULL
		         WHERE jr.runner_id = r.id) AS journals_joined,
		       count(t.id) FILTER (WHERE t.status = 'open')                    AS open_trades,
		       count(t.id) FILTER (WHERE t.status = 'closed')                  AS closed_trades,
		       count(t.id) FILTER (WHERE t.status = 'closed' AND t.pnl > 0)    AS wins,
		       count(t.id) FILTER (WHERE t.status = 'closed' AND t.pnl < 0)    AS losses,
		       COALESCE(sum(t.pnl) FILTER (WHERE t.status = 'closed'), 0)      AS net_pnl,
		       count(t.id) FILTER (WHERE t.opened_at >= now() - interval '24 hours'
		                              OR t.closed_at >= now() - interval '24 hours') AS trades_last_24h
		FROM runners AS r
		LEFT JOIN trades AS t ON t.runner_id = r.id AND t.deleted_at IS NULL
		     AND t.journal_id IN (SELECT id FROM journals WHERE deleted_at IS NULL)
		WHERE r.user_id = ? AND r.deleted_at IS NULL
		GROUP BY r.id
		ORDER BY r.last_used_at DESC NULLS LAST, lower(r.name) ASC`, userID).Scan(ctx, &list)
	return list, err
}
