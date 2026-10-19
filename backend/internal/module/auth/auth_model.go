package auth

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Session login user. Token asli hanya dipegang client,
// di database yang disimpan hash SHA-256-nya.
type Session struct {
	bun.BaseModel `bun:"table:sessions"`

	ID         uuid.UUID `bun:"id,pk,type:uuid,nullzero,default:uuidv7()"`
	UserID     uuid.UUID `bun:"user_id,type:uuid,notnull"`
	TokenHash  string    `bun:"token_hash,notnull"`
	IPAddress  string    `bun:"ip_address"`
	UserAgent  string    `bun:"user_agent"`
	ExpiresAt  time.Time `bun:"expires_at,notnull"`
	LastUsedAt time.Time `bun:"last_used_at,notnull,default:current_timestamp"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:current_timestamp"`
}
