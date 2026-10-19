package journals

import (
	"context"
	"errors"
	"strings"
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun/driver/pgdriver"
)

var (
	ErrTradeNotFound      = errors.New("trade not found")
	ErrTradeAlreadyClosed = errors.New("trade already closed")
	ErrTradeNotOwned      = errors.New("trade not owned by this runner")
	ErrJournalArchived    = errors.New("journal is archived")
	ErrOpenTradeExists    = errors.New("runner already has an open trade for this pair and position")
)

const runnerOpenUniqueIndex = "trades_runner_open_unique"

// OpenTradeExistsError: runner sudah punya trade open untuk pair + position yang sama.
// Existing berisi trade tersebut (nil kalau tertangkap lewat constraint database saat race).
type OpenTradeExistsError struct {
	Existing *Trade
}

func (e *OpenTradeExistsError) Error() string { return ErrOpenTradeExists.Error() }
func (e *OpenTradeExistsError) Unwrap() error { return ErrOpenTradeExists }

// ensureNoDuplicateOpen: guard dobel untuk trade runner yang statusnya open.
func ensureNoDuplicateOpen(ctx context.Context, repo *Repository, t *Trade) error {
	if t.RunnerID == nil || t.Status != TradeOpen {
		return nil
	}
	existing, err := repo.FindRunnerOpenTrade(ctx, t.JournalID, *t.RunnerID, t.Pair, t.Position)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != t.ID {
		return &OpenTradeExistsError{Existing: existing}
	}
	return nil
}

// mapOpenUniqueViolation: request bersamaan yang lolos pengecekan, ditahan unique index database.
func mapOpenUniqueViolation(err error) error {
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.IntegrityViolation() && pgErr.Field('n') == runnerOpenUniqueIndex {
		return &OpenTradeExistsError{}
	}
	return err
}

// TradeClosedError: field yang ditolak karena trade sudah closed.
type TradeClosedError struct {
	Fields users.ValidationErrors
}

func (e *TradeClosedError) Error() string { return ErrTradeAlreadyClosed.Error() }
func (e *TradeClosedError) Unwrap() error { return ErrTradeAlreadyClosed }

func (s *Service) ListTrades(ctx context.Context, sc Scope, journalID uuid.UUID, q ListTradeQuery) ([]TradeResponse, int, error) {
	j, err := sc.journal(ctx, s.repo, journalID, false)
	if err != nil {
		return nil, 0, err
	}
	from, to, err := dateRange(j.Timezone, q.From, q.To)
	if err != nil {
		return nil, 0, err
	}
	list, total, err := s.repo.ListTrades(ctx, j.ID, q, from, to)
	if err != nil {
		return nil, 0, err
	}
	result := make([]TradeResponse, 0, len(list))
	for i := range list {
		result = append(result, ToTradeResponse(&list[i]))
	}
	return result, total, nil
}

func (s *Service) GetTrade(ctx context.Context, sc Scope, journalID, id uuid.UUID) (TradeResponse, error) {
	j, err := sc.journal(ctx, s.repo, journalID, false)
	if err != nil {
		return TradeResponse{}, err
	}
	t, err := s.repo.FindTrade(ctx, j.ID, id)
	if err != nil {
		return TradeResponse{}, err
	}
	return ToTradeResponse(t), nil
}

// CreateTrade membuka trade baru oleh user (runner_id = null).
// Kalau exit_price dikirim, trade langsung disimpan sebagai closed.
func (s *Service) CreateTrade(ctx context.Context, sc Scope, journalID uuid.UUID, req CreateTradeRequest) (TradeResponse, error) {
	req.Normalize()
	if sc.RunnerID != nil {
		if err := req.CloseFields.rejectForRunner(); err != nil {
			return TradeResponse{}, err
		}
	}
	if err := req.Validate(); err != nil {
		return TradeResponse{}, err
	}

	var res TradeResponse
	err := s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := sc.journal(ctx, repo, journalID, true)
		if err != nil {
			return err
		}
		if j.Status == StatusArchived {
			return ErrJournalArchived
		}

		t := &Trade{
			JournalID:  j.ID,
			RunnerID:   sc.RunnerID,
			Exchange:   req.Exchange,
			Pair:       req.Pair,
			Position:   req.Position,
			Margin:     req.Margin,
			Leverage:   req.Leverage,
			EntryPrice: req.EntryPrice,
			StopLoss:   req.StopLoss,
			TakeProfit: req.TakeProfit,
			Status:     TradeOpen,
			OpenReason: req.OpenReason,
			Fee:        decimal.Zero,
			OpenedAt:   time.Now(),
		}
		if req.OpenedAt != nil {
			t.OpenedAt = *req.OpenedAt
		}
		if req.ExitPrice != nil {
			applyClose(t, req.CloseFields)
		}
		if err := validateTradeTimes(j, t.OpenedAt, t.ClosedAt); err != nil {
			return err
		}
		if err := ensureNoDuplicateOpen(ctx, repo, t); err != nil {
			return err
		}
		if err := repo.CreateTrade(ctx, t); err != nil {
			return mapOpenUniqueViolation(err)
		}
		if t.Status == TradeClosed {
			if err := afterBalanceChange(ctx, repo, j); err != nil {
				return err
			}
		}
		res = ToTradeResponse(t)
		return nil
	})
	return res, err
}

// UpdateTrade: trade open boleh diubah semua field open-nya;
// trade closed hanya close_note (status tidak bisa kembali ke open).
func (s *Service) UpdateTrade(ctx context.Context, sc Scope, journalID, id uuid.UUID, req UpdateTradeRequest) (TradeResponse, error) {
	req.Normalize()

	var res TradeResponse
	err := s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := sc.journal(ctx, repo, journalID, true)
		if err != nil {
			return err
		}
		t, err := sc.editableTrade(ctx, repo, j.ID, id)
		if err != nil {
			return err
		}

		if t.Status == TradeClosed {
			if fields := req.openFieldsSent(); len(fields) > 0 {
				errs := users.ValidationErrors{}
				for _, f := range fields {
					errs[f] = "cannot be changed after trade is closed, only close_note can be changed"
				}
				return &TradeClosedError{Fields: errs}
			}
		} else if req.CloseNote != nil {
			return users.ValidationErrors{"close_note": "can only be set when closing the trade"}
		}

		columns := applyTradeUpdate(t, req)
		if err := validateTrade(t); err != nil {
			return err
		}
		if req.OpenedAt != nil {
			if err := validateTradeTimes(j, t.OpenedAt, nil); err != nil {
				return err
			}
		}
		// pair / position diganti bisa bentrok dengan trade open lain milik runner yang sama
		if req.Pair != nil || req.Position != nil {
			if err := ensureNoDuplicateOpen(ctx, repo, t); err != nil {
				return err
			}
		}
		t.UpdatedAt = time.Now()
		if err := repo.UpdateTrade(ctx, t, append(columns, "updated_at")...); err != nil {
			return mapOpenUniqueViolation(err)
		}
		res = ToTradeResponse(t)
		return nil
	})
	return res, err
}

// CloseTrade menutup trade open. Trade yang sudah closed tidak bisa ditutup / dibuka lagi.
func (s *Service) CloseTrade(ctx context.Context, sc Scope, journalID, id uuid.UUID, req CloseFields) (TradeResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return TradeResponse{}, err
	}

	var res TradeResponse
	err := s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := sc.journal(ctx, repo, journalID, true)
		if err != nil {
			return err
		}
		t, err := sc.editableTrade(ctx, repo, j.ID, id)
		if err != nil {
			return err
		}
		if t.Status == TradeClosed {
			return ErrTradeAlreadyClosed
		}

		applyClose(t, req)
		if err := validateTradeTimes(j, t.OpenedAt, t.ClosedAt); err != nil {
			return err
		}
		t.UpdatedAt = time.Now()
		if err := repo.UpdateTrade(ctx, t,
			"exit_price", "close_reason", "close_note", "pnl", "fee", "closed_at", "status", "updated_at"); err != nil {
			return err
		}
		if err := afterBalanceChange(ctx, repo, j); err != nil {
			return err
		}
		res = ToTradeResponse(t)
		return nil
	})
	return res, err
}

func (s *Service) DeleteTrade(ctx context.Context, sc Scope, journalID, id uuid.UUID) error {
	return s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := sc.journal(ctx, repo, journalID, true)
		if err != nil {
			return err
		}
		if err := repo.DeleteTrade(ctx, j.ID, id); err != nil {
			return err
		}
		// misal hapus trade profit yang sudah ditarik lewat withdraw
		return afterBalanceChange(ctx, repo, j)
	})
}

// applyClose mengisi data close ke trade; pnl dihitung kalau tidak dikirim.
func applyClose(t *Trade, req CloseFields) {
	fee := decimal.Zero
	if req.Fee != nil {
		fee = *req.Fee
	}
	pnl := t.CalculatePnL(*req.ExitPrice, fee)
	if req.PnL != nil {
		pnl = *req.PnL
	}
	closedAt := time.Now()
	if req.ClosedAt != nil {
		closedAt = *req.ClosedAt
	}
	reason := req.CloseReason

	t.ExitPrice = req.ExitPrice
	t.CloseReason = &reason
	t.CloseNote = req.CloseNote
	t.PnL = &pnl
	t.Fee = fee
	t.ClosedAt = &closedAt
	t.Status = TradeClosed
}

// applyTradeUpdate menerapkan field yang dikirim dan mengembalikan kolom yang berubah.
func applyTradeUpdate(t *Trade, req UpdateTradeRequest) []string {
	var columns []string
	if req.Exchange != nil {
		t.Exchange = normalizeExchange(req.Exchange)
		columns = append(columns, "exchange")
	}
	if req.Pair != nil {
		t.Pair = *req.Pair
		columns = append(columns, "pair")
	}
	if req.Position != nil {
		t.Position = *req.Position
		columns = append(columns, "position")
	}
	if req.Margin != nil {
		t.Margin = *req.Margin
		columns = append(columns, "margin")
	}
	if req.Leverage != nil {
		t.Leverage = *req.Leverage
		columns = append(columns, "leverage")
	}
	if req.EntryPrice != nil {
		t.EntryPrice = *req.EntryPrice
		columns = append(columns, "entry_price")
	}
	if req.StopLoss.Set {
		t.StopLoss = req.StopLoss.Value
		columns = append(columns, "stop_loss")
	}
	if req.TakeProfit.Set {
		t.TakeProfit = req.TakeProfit.Value
		columns = append(columns, "take_profit")
	}
	if req.OpenReason != nil {
		t.OpenReason = normalizeText(req.OpenReason)
		columns = append(columns, "open_reason")
	}
	if req.OpenedAt != nil {
		t.OpenedAt = *req.OpenedAt
		columns = append(columns, "opened_at")
	}
	if req.CloseNote != nil {
		t.CloseNote = normalizeText(req.CloseNote)
		columns = append(columns, "close_note")
	}
	return columns
}

func normalizeTradeFilter(q *ListTradeQuery) users.ValidationErrors {
	q.Status = strings.ToLower(q.Status)
	q.Position = strings.ToLower(q.Position)
	q.Pair = strings.ToUpper(q.Pair)
	q.Source = strings.ToLower(q.Source)

	errs := users.ValidationErrors{}
	if q.Status != "" && q.Status != TradeOpen && q.Status != TradeClosed {
		errs["status"] = "must be open or closed"
	}
	if q.Position != "" && q.Position != PositionLong && q.Position != PositionShort {
		errs["position"] = "must be long or short"
	}
	if q.Source != "" && q.Source != "manual" && q.Source != "runner" {
		errs["source"] = "must be manual or runner"
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}
