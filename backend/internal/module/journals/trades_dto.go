package journals

import (
	"regexp"
	"strings"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	PositionLong  = "long"
	PositionShort = "short"

	TradeOpen   = "open"
	TradeClosed = "closed"
)

var (
	closeReasons = map[string]bool{
		"take_profit":   true,
		"stop_loss":     true,
		"manual":        true,
		"liquidation":   true,
		"trailing_stop": true,
	}
	pairPattern  = regexp.MustCompile(`^[A-Z0-9][A-Z0-9/._-]{1,29}$`)
	maxLeverage  = decimal.NewFromInt(500)
	hundred      = decimal.NewFromInt(100)
	pnlPrecision = int32(8)
)

// Trade: satu posisi trading di dalam jurnal. Semua nominal dalam USD.
type Trade struct {
	bun.BaseModel `bun:"table:trades"`

	ID          uuid.UUID        `bun:"id,pk,type:uuid,nullzero,default:uuidv7()"`
	JournalID   uuid.UUID        `bun:"journal_id,type:uuid,notnull"`
	RunnerID    *uuid.UUID       `bun:"runner_id,type:uuid"`
	Exchange    *string          `bun:"exchange"`
	Pair        string           `bun:"pair,notnull"`
	Position    string           `bun:"position,notnull"`
	Margin      decimal.Decimal  `bun:"margin,type:numeric,notnull"`
	Leverage    decimal.Decimal  `bun:"leverage,type:numeric,notnull"`
	EntryPrice  decimal.Decimal  `bun:"entry_price,type:numeric,notnull"`
	ExitPrice   *decimal.Decimal `bun:"exit_price,type:numeric"`
	StopLoss    *decimal.Decimal `bun:"stop_loss,type:numeric"`
	TakeProfit  *decimal.Decimal `bun:"take_profit,type:numeric"`
	Status      string           `bun:"status,notnull"`
	OpenReason  *string          `bun:"open_reason"`
	CloseReason *string          `bun:"close_reason"`
	CloseNote   *string          `bun:"close_note"`
	PnL         *decimal.Decimal `bun:"pnl,type:numeric"`
	Fee         decimal.Decimal  `bun:"fee,type:numeric,notnull"`
	OpenedAt    time.Time        `bun:"opened_at,notnull"`
	ClosedAt    *time.Time       `bun:"closed_at"`
	CreatedAt   time.Time        `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time        `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt   *time.Time       `bun:"deleted_at,soft_delete,nullzero"`
}

// PositionSize = margin × leverage.
func (t *Trade) PositionSize() decimal.Decimal {
	return t.Margin.Mul(t.Leverage)
}

// CalculatePnL: PnL bersih dari harga, dipakai kalau pnl tidak dikirim saat close.
//
//	long : (exit - entry) / entry × margin × leverage - fee
//	short: (entry - exit) / entry × margin × leverage - fee
func (t *Trade) CalculatePnL(exit, fee decimal.Decimal) decimal.Decimal {
	move := exit.Sub(t.EntryPrice)
	if t.Position == PositionShort {
		move = move.Neg()
	}
	return move.Div(t.EntryPrice).Mul(t.PositionSize()).Sub(fee).Round(pnlPrecision)
}

// CloseFields: data saat menutup trade (dipakai di endpoint close & create langsung closed).
type CloseFields struct {
	ExitPrice   *decimal.Decimal `json:"exit_price"`
	CloseReason string           `json:"close_reason"`
	CloseNote   *string          `json:"close_note"`
	PnL         *decimal.Decimal `json:"pnl"`       // kosong = dihitung server
	Fee         *decimal.Decimal `json:"fee"`       // kosong = 0
	ClosedAt    *time.Time       `json:"closed_at"` // kosong = sekarang
}

// CreateTradeRequest: kalau exit_price dikirim, trade langsung dibuat dalam status closed
// (untuk mencatat trade yang sudah selesai).
type CreateTradeRequest struct {
	Exchange   *string          `json:"exchange"`
	Pair       string           `json:"pair"`
	Position   string           `json:"position"`
	Margin     decimal.Decimal  `json:"margin"`
	Leverage   decimal.Decimal  `json:"leverage"`
	EntryPrice decimal.Decimal  `json:"entry_price"`
	StopLoss   *decimal.Decimal `json:"stop_loss"`
	TakeProfit *decimal.Decimal `json:"take_profit"`
	OpenReason *string          `json:"open_reason"`
	OpenedAt   *time.Time       `json:"opened_at"` // kosong = sekarang
	CloseFields
}

// UpdateTradeRequest: trade open boleh ubah semua field open.
// Trade closed hanya boleh ubah close_note.
// stop_loss / take_profit null = dikosongkan; teks "" = dikosongkan.
type UpdateTradeRequest struct {
	Exchange   *string                         `json:"exchange"`
	Pair       *string                         `json:"pair"`
	Position   *string                         `json:"position"`
	Margin     *decimal.Decimal                `json:"margin"`
	Leverage   *decimal.Decimal                `json:"leverage"`
	EntryPrice *decimal.Decimal                `json:"entry_price"`
	StopLoss   types.Nullable[decimal.Decimal] `json:"stop_loss"`
	TakeProfit types.Nullable[decimal.Decimal] `json:"take_profit"`
	OpenReason *string                         `json:"open_reason"`
	OpenedAt   *time.Time                      `json:"opened_at"`
	CloseNote  *string                         `json:"close_note"`
}

// openFieldsSent: field yang hanya boleh diubah saat trade masih open.
func (r *UpdateTradeRequest) openFieldsSent() []string {
	var fields []string
	add := func(sent bool, name string) {
		if sent {
			fields = append(fields, name)
		}
	}
	add(r.Exchange != nil, "exchange")
	add(r.Pair != nil, "pair")
	add(r.Position != nil, "position")
	add(r.Margin != nil, "margin")
	add(r.Leverage != nil, "leverage")
	add(r.EntryPrice != nil, "entry_price")
	add(r.StopLoss.Set, "stop_loss")
	add(r.TakeProfit.Set, "take_profit")
	add(r.OpenReason != nil, "open_reason")
	add(r.OpenedAt != nil, "opened_at")
	return fields
}

type ListTradeQuery struct {
	Page     int
	Limit    int
	Status   string
	Position string
	Pair     string
	Source   string // manual / runner
	From     string // opened_at, YYYY-MM-DD di timezone jurnal
	To       string
}

type TradeResponse struct {
	ID           uuid.UUID        `json:"id"`
	JournalID    uuid.UUID        `json:"journal_id"`
	RunnerID     *uuid.UUID       `json:"runner_id"`
	Exchange     *string          `json:"exchange"`
	Pair         string           `json:"pair"`
	Position     string           `json:"position"`
	Margin       decimal.Decimal  `json:"margin"`
	Leverage     decimal.Decimal  `json:"leverage"`
	PositionSize decimal.Decimal  `json:"position_size"`
	EntryPrice   decimal.Decimal  `json:"entry_price"`
	ExitPrice    *decimal.Decimal `json:"exit_price"`
	StopLoss     *decimal.Decimal `json:"stop_loss"`
	TakeProfit   *decimal.Decimal `json:"take_profit"`
	Status       string           `json:"status"`
	OpenReason   *string          `json:"open_reason"`
	CloseReason  *string          `json:"close_reason"`
	CloseNote    *string          `json:"close_note"`
	PnL          *decimal.Decimal `json:"pnl"`
	Fee          decimal.Decimal  `json:"fee"`
	// ROI = pnl / margin × 100 (persen), hanya untuk trade closed
	ROI       *decimal.Decimal `json:"roi"`
	OpenedAt  time.Time        `json:"opened_at"`
	ClosedAt  *time.Time       `json:"closed_at"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func ToTradeResponse(t *Trade) TradeResponse {
	res := TradeResponse{
		ID:           t.ID,
		JournalID:    t.JournalID,
		RunnerID:     t.RunnerID,
		Exchange:     t.Exchange,
		Pair:         t.Pair,
		Position:     t.Position,
		Margin:       t.Margin,
		Leverage:     t.Leverage,
		PositionSize: t.PositionSize(),
		EntryPrice:   t.EntryPrice,
		ExitPrice:    t.ExitPrice,
		StopLoss:     t.StopLoss,
		TakeProfit:   t.TakeProfit,
		Status:       t.Status,
		OpenReason:   t.OpenReason,
		CloseReason:  t.CloseReason,
		CloseNote:    t.CloseNote,
		PnL:          t.PnL,
		Fee:          t.Fee,
		OpenedAt:     t.OpenedAt,
		ClosedAt:     t.ClosedAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.PnL != nil {
		roi := t.PnL.Div(t.Margin).Mul(hundred).Round(2)
		res.ROI = &roi
	}
	return res
}

func (r *CreateTradeRequest) Normalize() {
	r.Exchange = normalizeExchange(r.Exchange)
	r.Pair = strings.ToUpper(strings.TrimSpace(r.Pair))
	r.Position = strings.ToLower(strings.TrimSpace(r.Position))
	r.OpenReason = normalizeText(r.OpenReason)
	r.CloseFields.Normalize()
}

// Validate: memvalidasi field open, dan field close kalau exit_price dikirim.
func (r *CreateTradeRequest) Validate() error {
	errs := users.ValidationErrors{}
	validateExchange(errs, r.Exchange)
	validatePair(errs, r.Pair)
	validatePosition(errs, r.Position)
	validatePositive(errs, "margin", r.Margin)
	validateLeverage(errs, r.Leverage)
	validatePositive(errs, "entry_price", r.EntryPrice)
	validateOptionalPositive(errs, "stop_loss", r.StopLoss)
	validateOptionalPositive(errs, "take_profit", r.TakeProfit)
	validateText(errs, "open_reason", r.OpenReason)
	if len(errs) == 0 {
		validateTakeProfitDirection(errs, r.Position, r.EntryPrice, r.TakeProfit)
	}
	if r.ExitPrice != nil {
		r.CloseFields.validateInto(errs)
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *CloseFields) Normalize() {
	r.CloseReason = strings.ToLower(strings.TrimSpace(r.CloseReason))
	r.CloseNote = normalizeText(r.CloseNote)
}

func (r *CloseFields) Validate() error {
	errs := users.ValidationErrors{}
	r.validateInto(errs)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// rejectForRunner: runner tidak boleh membuat trade yang langsung closed
// (harus buka dulu lalu close), supaya guard trade dobel selalu berlaku.
func (r *CloseFields) rejectForRunner() error {
	const msg = "not allowed when opening a trade as runner, open the trade first then close it"
	errs := users.ValidationErrors{}
	if r.ExitPrice != nil {
		errs["exit_price"] = msg
	}
	if r.CloseReason != "" {
		errs["close_reason"] = msg
	}
	if r.CloseNote != nil {
		errs["close_note"] = msg
	}
	if r.PnL != nil {
		errs["pnl"] = msg
	}
	if r.Fee != nil {
		errs["fee"] = msg
	}
	if r.ClosedAt != nil {
		errs["closed_at"] = msg
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *CloseFields) validateInto(errs users.ValidationErrors) {
	if r.ExitPrice == nil {
		errs["exit_price"] = "is required"
	} else {
		validatePositive(errs, "exit_price", *r.ExitPrice)
	}
	if !closeReasons[r.CloseReason] {
		errs["close_reason"] = "must be take_profit, stop_loss, manual, liquidation or trailing_stop"
	}
	validateText(errs, "close_note", r.CloseNote)
	if r.Fee != nil && r.Fee.IsNegative() {
		errs["fee"] = "must not be negative"
	}
}

func (r *UpdateTradeRequest) Normalize() {
	if r.Exchange != nil {
		v := strings.ToLower(strings.TrimSpace(*r.Exchange))
		r.Exchange = &v
	}
	if r.Pair != nil {
		v := strings.ToUpper(strings.TrimSpace(*r.Pair))
		r.Pair = &v
	}
	if r.Position != nil {
		v := strings.ToLower(strings.TrimSpace(*r.Position))
		r.Position = &v
	}
	for _, p := range []*string{r.OpenReason, r.CloseNote} {
		if p != nil {
			*p = strings.TrimSpace(*p)
		}
	}
}

// validateTrade: validasi state trade setelah perubahan diterapkan (dipakai saat update).
func validateTrade(t *Trade) error {
	errs := users.ValidationErrors{}
	validateExchange(errs, t.Exchange)
	validatePair(errs, t.Pair)
	validatePosition(errs, t.Position)
	validatePositive(errs, "margin", t.Margin)
	validateLeverage(errs, t.Leverage)
	validatePositive(errs, "entry_price", t.EntryPrice)
	validateOptionalPositive(errs, "stop_loss", t.StopLoss)
	validateOptionalPositive(errs, "take_profit", t.TakeProfit)
	validateText(errs, "open_reason", t.OpenReason)
	validateText(errs, "close_note", t.CloseNote)
	if len(errs) == 0 {
		validateTakeProfitDirection(errs, t.Position, t.EntryPrice, t.TakeProfit)
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func normalizeExchange(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*v))
	if s == "" {
		return nil
	}
	return &s
}

func validateExchange(errs users.ValidationErrors, v *string) {
	if v != nil && len(*v) > 30 {
		errs["exchange"] = "max 30 characters"
	}
}

func validatePair(errs users.ValidationErrors, v string) {
	if !pairPattern.MatchString(v) {
		errs["pair"] = "is required, 2-30 characters: letters, numbers, / . _ -  (example: BTCUSDT)"
	}
}

func validatePosition(errs users.ValidationErrors, v string) {
	if v != PositionLong && v != PositionShort {
		errs["position"] = "must be long or short"
	}
}

func validatePositive(errs users.ValidationErrors, field string, v decimal.Decimal) {
	if !v.IsPositive() {
		errs[field] = "must be greater than 0"
	}
}

func validateOptionalPositive(errs users.ValidationErrors, field string, v *decimal.Decimal) {
	if v != nil {
		validatePositive(errs, field, *v)
	}
}

func validateLeverage(errs users.ValidationErrors, v decimal.Decimal) {
	if v.LessThan(decimal.NewFromInt(1)) || v.GreaterThan(maxLeverage) {
		errs["leverage"] = "must be between 1 and 500"
	}
}

func validateText(errs users.ValidationErrors, field string, v *string) {
	if v != nil && len(*v) > 1000 {
		errs[field] = "max 1000 characters"
	}
}

// validateTakeProfitDirection: TP long harus di atas entry, TP short di bawah entry.
// Stop loss sengaja tidak dicek arahnya karena SL bisa digeser ke area profit (breakeven / trailing).
func validateTakeProfitDirection(errs users.ValidationErrors, position string, entry decimal.Decimal, tp *decimal.Decimal) {
	if tp == nil {
		return
	}
	if position == PositionLong && !tp.GreaterThan(entry) {
		errs["take_profit"] = "must be above entry_price for long position"
	}
	if position == PositionShort && !tp.LessThan(entry) {
		errs["take_profit"] = "must be below entry_price for short position"
	}
}

// validateTradeTimes: opened_at tidak sebelum start_date jurnal & tidak di masa depan,
// closed_at tidak sebelum opened_at & tidak di masa depan.
func validateTradeTimes(j *Journal, openedAt time.Time, closedAt *time.Time) error {
	if err := validateOccurredAt(j, openedAt); err != nil {
		if verr, ok := err.(users.ValidationErrors); ok {
			return users.ValidationErrors{"opened_at": verr["occurred_at"]}
		}
		return err
	}
	if closedAt != nil {
		if closedAt.Before(openedAt) {
			return users.ValidationErrors{"closed_at": "must not be before opened_at"}
		}
		if closedAt.After(time.Now().Add(time.Minute)) {
			return users.ValidationErrors{"closed_at": "must not be in the future"}
		}
	}
	return nil
}
