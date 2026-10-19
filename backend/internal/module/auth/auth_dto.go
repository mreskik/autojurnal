package auth

import (
	"strings"
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	r.Username = strings.TrimSpace(r.Username)

	errs := users.ValidationErrors{}
	if r.Username == "" {
		errs["username"] = "is required"
	}
	if r.Password == "" {
		errs["password"] = "is required"
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

type LoginResponse struct {
	Token     string             `json:"token"`
	ExpiresAt time.Time          `json:"expires_at"`
	User      users.UserResponse `json:"user"`
}

type SessionResponse struct {
	ID         uuid.UUID `json:"id"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Current    bool      `json:"current"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func ToSessionResponse(s *Session, currentID uuid.UUID) SessionResponse {
	return SessionResponse{
		ID:         s.ID,
		IPAddress:  s.IPAddress,
		UserAgent:  s.UserAgent,
		Current:    s.ID == currentID,
		CreatedAt:  s.CreatedAt,
		LastUsedAt: s.LastUsedAt,
		ExpiresAt:  s.ExpiresAt,
	}
}
