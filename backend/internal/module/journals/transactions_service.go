package journals

import (
	"context"
	"errors"
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ErrTransactionNotFound = errors.New("transaction not found")
	ErrInsufficientBalance = errors.New("insufficient balance")
)

// InsufficientBalanceError: perubahan transaksi membuat balance jadi minus.
type InsufficientBalanceError struct {
	BalanceAfter decimal.Decimal
}

func (e *InsufficientBalanceError) Error() string { return ErrInsufficientBalance.Error() }
func (e *InsufficientBalanceError) Unwrap() error { return ErrInsufficientBalance }

func (s *Service) ListTransactions(ctx context.Context, userID, journalID uuid.UUID, q ListTransactionQuery) ([]TransactionResponse, int, error) {
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return nil, 0, err
	}
	from, to, err := dateRange(j.Timezone, q.From, q.To)
	if err != nil {
		return nil, 0, err
	}
	list, total, err := s.repo.ListTransactions(ctx, j.ID, q, from, to)
	if err != nil {
		return nil, 0, err
	}
	result := make([]TransactionResponse, 0, len(list))
	for i := range list {
		result = append(result, ToTransactionResponse(&list[i]))
	}
	return result, total, nil
}

func (s *Service) CreateTransaction(ctx context.Context, userID, journalID uuid.UUID, req CreateTransactionRequest) (TransactionResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return TransactionResponse{}, err
	}

	var res TransactionResponse
	err := s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := repo.FindByUserForUpdate(ctx, userID, journalID)
		if err != nil {
			return err
		}
		t := &JournalTransaction{
			JournalID:  j.ID,
			Type:       req.Type,
			Amount:     req.Amount,
			Note:       req.Note,
			OccurredAt: time.Now(),
		}
		if req.OccurredAt != nil {
			t.OccurredAt = *req.OccurredAt
		}
		if err := validateOccurredAt(j, t.OccurredAt); err != nil {
			return err
		}
		if err := repo.CreateTransaction(ctx, t); err != nil {
			return err
		}
		if err := afterBalanceChange(ctx, repo, j); err != nil {
			return err
		}
		res = ToTransactionResponse(t)
		return nil
	})
	return res, err
}

func (s *Service) UpdateTransaction(ctx context.Context, userID, journalID, id uuid.UUID, req UpdateTransactionRequest) (TransactionResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return TransactionResponse{}, err
	}

	var res TransactionResponse
	err := s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := repo.FindByUserForUpdate(ctx, userID, journalID)
		if err != nil {
			return err
		}
		t, err := repo.FindTransaction(ctx, j.ID, id)
		if err != nil {
			return err
		}

		columns := []string{"updated_at"}
		if req.Type != nil {
			t.Type = *req.Type
			columns = append(columns, "type")
		}
		if req.Amount != nil {
			t.Amount = *req.Amount
			columns = append(columns, "amount")
		}
		if req.Note != nil {
			t.Note = normalizeText(req.Note) // "" -> NULL
			columns = append(columns, "note")
		}
		if req.OccurredAt != nil {
			if err := validateOccurredAt(j, *req.OccurredAt); err != nil {
				return err
			}
			t.OccurredAt = *req.OccurredAt
			columns = append(columns, "occurred_at")
		}
		t.UpdatedAt = time.Now()

		if err := repo.UpdateTransaction(ctx, t, columns...); err != nil {
			return err
		}
		if err := afterBalanceChange(ctx, repo, j); err != nil {
			return err
		}
		res = ToTransactionResponse(t)
		return nil
	})
	return res, err
}

func (s *Service) DeleteTransaction(ctx context.Context, userID, journalID, id uuid.UUID) error {
	return s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := repo.FindByUserForUpdate(ctx, userID, journalID)
		if err != nil {
			return err
		}
		if err := repo.DeleteTransaction(ctx, j.ID, id); err != nil {
			return err
		}
		// misal hapus deposit yang sudah dipakai untuk withdraw
		return afterBalanceChange(ctx, repo, j)
	})
}

// ensureNonNegativeBalance dipanggil setelah perubahan, di dalam transaction:
// kalau balance jadi minus, error dikembalikan dan semua perubahan di-rollback.
func ensureNonNegativeBalance(ctx context.Context, repo *Repository, j *Journal) error {
	balance, err := repo.Balance(ctx, j)
	if err != nil {
		return err
	}
	if balance.IsNegative() {
		return &InsufficientBalanceError{BalanceAfter: balance}
	}
	return nil
}

// dateRange mengubah from/to (YYYY-MM-DD, inklusif) di timezone jurnal jadi rentang waktu [from, to+1hari).
func dateRange(tz, from, to string) (*time.Time, *time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, nil, err
	}
	var start, end *time.Time
	if from != "" {
		t, err := time.ParseInLocation(dateLayout, from, loc)
		if err != nil {
			return nil, nil, users.ValidationErrors{"from": "must be a date in YYYY-MM-DD format"}
		}
		start = &t
	}
	if to != "" {
		t, err := time.ParseInLocation(dateLayout, to, loc)
		if err != nil {
			return nil, nil, users.ValidationErrors{"to": "must be a date in YYYY-MM-DD format"}
		}
		t = t.AddDate(0, 0, 1)
		end = &t
	}
	return start, end, nil
}
