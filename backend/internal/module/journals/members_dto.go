package journals

import (
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// JournalRunner: runner yang join ke jurnal (boleh kirim trade ke jurnal ini),
// beserta instruksi khusus untuk runner ini di jurnal ini.
type JournalRunner struct {
	bun.BaseModel `bun:"table:journal_runners"`

	JournalID uuid.UUID `bun:"journal_id,pk,type:uuid"`
	RunnerID  uuid.UUID `bun:"runner_id,pk,type:uuid"`
	Rules     *string   `bun:"rules"`
	Strategy  *string   `bun:"strategy"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type AddRunnerRequest struct {
	RunnerID uuid.UUID `json:"runner_id"`
	Rules    *string   `json:"rules"`
	Strategy *string   `json:"strategy"`
}

// UpdateJoinedRunnerRequest: ubah instruksi runner di jurnal. "" = dikosongkan.
type UpdateJoinedRunnerRequest struct {
	Rules    *string `json:"rules"`
	Strategy *string `json:"strategy"`
}

func (r *AddRunnerRequest) Validate() error {
	errs := users.ValidationErrors{}
	if r.RunnerID == uuid.Nil {
		errs["runner_id"] = "is required"
	}
	validateInstruction(errs, "rules", r.Rules)
	validateInstruction(errs, "strategy", r.Strategy)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *UpdateJoinedRunnerRequest) Validate() error {
	errs := users.ValidationErrors{}
	validateInstruction(errs, "rules", r.Rules)
	validateInstruction(errs, "strategy", r.Strategy)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// RunnerOptionResponse: pilihan runner untuk dropdown "tambah runner ke jurnal".
type RunnerOptionResponse struct {
	ID       uuid.UUID `bun:"id" json:"id"`
	Name     string    `bun:"name" json:"name"`
	IsActive bool      `bun:"is_active" json:"is_active"`
}

// JoinedRunnerResponse: data runner di dalam jurnal (tanpa token).
type JoinedRunnerResponse struct {
	RunnerID   uuid.UUID  `bun:"runner_id" json:"runner_id"`
	Name       string     `bun:"name" json:"name"`
	IsActive   bool       `bun:"is_active" json:"is_active"`
	ExpiresAt  *time.Time `bun:"expires_at" json:"expires_at"`
	LastUsedAt *time.Time `bun:"last_used_at" json:"last_used_at"`
	Rules      *string    `bun:"rules" json:"rules"`
	Strategy   *string    `bun:"strategy" json:"strategy"`
	JoinedAt   time.Time  `bun:"joined_at" json:"joined_at"`
}
