package runners

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Runner: klien mesin (agent AI, bot, script) yang akses API pakai token.
// Token disimpan apa adanya (belum di-hash) sesuai keputusan saat ini.
type Runner struct {
	bun.BaseModel `bun:"table:runners"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid,nullzero,default:uuidv7()"`
	UserID      uuid.UUID  `bun:"user_id,type:uuid,notnull"`
	Name        string     `bun:"name,notnull"`
	Description *string    `bun:"description"`
	Token       string     `bun:"token,notnull"`
	IsActive    bool       `bun:"is_active,notnull,default:true"`
	ExpiresAt   *time.Time `bun:"expires_at"`
	LastUsedAt  *time.Time `bun:"last_used_at"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time  `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt   *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}
