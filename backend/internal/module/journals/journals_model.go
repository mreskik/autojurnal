package journals

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"

	StatusActive   = "active"
	StatusArchived = "archived"

	TargetAmount  = "amount"
	TargetPercent = "percent"

	GoalBalance = "balance" // total balance >= goal_value
	GoalProfit  = "profit"  // total pnl closed >= goal_value
	GoalPercent = "percent" // total pnl / (initial + deposit) x 100 >= goal_value
)

// Journal: header jurnal trading. Semua nominal dalam USD.
// Kolom DATE disimpan sebagai string "YYYY-MM-DD" supaya tidak terpengaruh timezone.
type Journal struct {
	bun.BaseModel `bun:"table:journals"`

	ID             uuid.UUID       `bun:"id,pk,type:uuid,nullzero,default:uuidv7()"`
	UserID         uuid.UUID       `bun:"user_id,type:uuid,notnull"`
	Name           string          `bun:"name,notnull"`
	Description    *string         `bun:"description"`
	InitialBalance decimal.Decimal `bun:"initial_balance,type:numeric,notnull"`
	PeriodType     string          `bun:"period_type,notnull"`
	Timezone       string          `bun:"timezone,notnull"`
	StartDate      string          `bun:"start_date,type:date,notnull"`
	Status         string          `bun:"status,notnull"`
	// token jurnal untuk runner (header X-Journal-Token), plain text
	Token string `bun:"token,notnull"`
	// instruksi umum untuk semua runner di jurnal ini (teks bebas)
	Rules    *string `bun:"rules"`
	Strategy *string `bun:"strategy"`
	// target akhir jurnal (opsional); goal_achieved_at = pertama kali tercapai, tidak hilang walau turun lagi
	GoalType       *string          `bun:"goal_type"`
	GoalValue      *decimal.Decimal `bun:"goal_value,type:numeric"`
	GoalAchievedAt *time.Time       `bun:"goal_achieved_at"`
	CreatedAt      time.Time        `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time        `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt      *time.Time       `bun:"deleted_at,soft_delete,nullzero"`
}

// JournalTarget: satu baris riwayat target (append-only).
type JournalTarget struct {
	bun.BaseModel `bun:"table:journal_targets"`

	ID            uuid.UUID       `bun:"id,pk,type:uuid,nullzero,default:uuidv7()"`
	JournalID     uuid.UUID       `bun:"journal_id,type:uuid,notnull"`
	TargetType    string          `bun:"target_type,notnull"`
	TargetValue   decimal.Decimal `bun:"target_value,type:numeric,notnull"`
	EffectiveFrom string          `bun:"effective_from,type:date,notnull"`
	Note          *string         `bun:"note"`
	CreatedAt     time.Time       `bun:"created_at,notnull,default:current_timestamp"`
}
