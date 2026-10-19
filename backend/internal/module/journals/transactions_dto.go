package journals

import (
	"strings"
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	TxDeposit  = "deposit"
	TxWithdraw = "withdraw"
)

// JournalTransaction: deposit / withdraw modal di tengah jalan.
type JournalTransaction struct {
	bun.BaseModel `bun:"table:journal_transactions"`

	ID         uuid.UUID       `bun:"id,pk,type:uuid,nullzero,default:uuidv7()"`
	JournalID  uuid.UUID       `bun:"journal_id,type:uuid,notnull"`
	Type       string          `bun:"type,notnull"`
	Amount     decimal.Decimal `bun:"amount,type:numeric,notnull"`
	Note       *string         `bun:"note"`
	OccurredAt time.Time       `bun:"occurred_at,notnull"`
	CreatedAt  time.Time       `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt  time.Time       `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt  *time.Time      `bun:"deleted_at,soft_delete,nullzero"`
}

type CreateTransactionRequest struct {
	Type       string          `json:"type"`
	Amount     decimal.Decimal `json:"amount"`
	Note       *string         `json:"note"`
	OccurredAt *time.Time      `json:"occurred_at"`
}

// UpdateTransactionRequest: field yang tidak dikirim tidak diubah. note "" = dikosongkan.
type UpdateTransactionRequest struct {
	Type       *string          `json:"type"`
	Amount     *decimal.Decimal `json:"amount"`
	Note       *string          `json:"note"`
	OccurredAt *time.Time       `json:"occurred_at"`
}

type ListTransactionQuery struct {
	Page  int
	Limit int
	Type  string
	From  string // YYYY-MM-DD, di timezone jurnal
	To    string // YYYY-MM-DD, inklusif
}

type TransactionResponse struct {
	ID         uuid.UUID       `json:"id"`
	JournalID  uuid.UUID       `json:"journal_id"`
	Type       string          `json:"type"`
	Amount     decimal.Decimal `json:"amount"`
	Note       *string         `json:"note"`
	OccurredAt time.Time       `json:"occurred_at"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

func ToTransactionResponse(t *JournalTransaction) TransactionResponse {
	return TransactionResponse{
		ID:         t.ID,
		JournalID:  t.JournalID,
		Type:       t.Type,
		Amount:     t.Amount,
		Note:       t.Note,
		OccurredAt: t.OccurredAt,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func (r *CreateTransactionRequest) Normalize() {
	r.Type = strings.TrimSpace(r.Type)
	r.Note = normalizeText(r.Note)
}

func (r *CreateTransactionRequest) Validate() error {
	errs := users.ValidationErrors{}
	validateTxType(errs, r.Type)
	validateAmount(errs, r.Amount)
	validateNote(errs, r.Note)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *UpdateTransactionRequest) Normalize() {
	if r.Type != nil {
		v := strings.TrimSpace(*r.Type)
		r.Type = &v
	}
	if r.Note != nil {
		v := strings.TrimSpace(*r.Note)
		r.Note = &v
	}
}

func (r *UpdateTransactionRequest) Validate() error {
	errs := users.ValidationErrors{}
	if r.Type != nil {
		validateTxType(errs, *r.Type)
	}
	if r.Amount != nil {
		validateAmount(errs, *r.Amount)
	}
	validateNote(errs, r.Note)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateTxType(errs users.ValidationErrors, v string) {
	if v != TxDeposit && v != TxWithdraw {
		errs["type"] = "must be deposit or withdraw"
	}
}

func validateAmount(errs users.ValidationErrors, v decimal.Decimal) {
	if !v.IsPositive() {
		errs["amount"] = "must be greater than 0"
	}
}

func validateNote(errs users.ValidationErrors, v *string) {
	if v != nil && len(*v) > 500 {
		errs["note"] = "max 500 characters"
	}
}

// validateOccurredAt: tidak boleh sebelum start_date jurnal dan tidak boleh di masa depan.
func validateOccurredAt(j *Journal, v time.Time) error {
	loc, err := time.LoadLocation(j.Timezone)
	if err != nil {
		return err
	}
	start, err := time.ParseInLocation(dateLayout, j.StartDate, loc)
	if err != nil {
		return err
	}
	if v.Before(start) {
		return users.ValidationErrors{"occurred_at": "must not be before journal start_date (" + j.StartDate + ")"}
	}
	if v.After(time.Now().Add(time.Minute)) {
		return users.ValidationErrors{"occurred_at": "must not be in the future"}
	}
	return nil
}
