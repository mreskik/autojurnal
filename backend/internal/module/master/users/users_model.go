package users

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	// UUID v7, di-generate oleh PostgreSQL (DEFAULT uuidv7())
	ID           uuid.UUID  `bun:"id,pk,type:uuid,nullzero,default:uuidv7()" json:"id"`
	Username     string     `bun:"username,notnull" json:"username"`
	Name         string     `bun:"name,notnull" json:"name"`
	Email        *string    `bun:"email" json:"email"`
	PasswordHash string     `bun:"password_hash,notnull" json:"-"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt    time.Time  `bun:"updated_at,notnull,default:current_timestamp" json:"updated_at"`
	DeletedAt    *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"-"`
}
