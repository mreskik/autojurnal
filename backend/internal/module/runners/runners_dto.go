package runners

import (
	"strings"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/types"

	"github.com/google/uuid"
)

type CreateRunnerRequest struct {
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// UpdateRunnerRequest: field yang tidak dikirim tidak diubah.
// description "" = dikosongkan, expires_at null = berlaku selamanya.
type UpdateRunnerRequest struct {
	Name        *string                   `json:"name"`
	Description *string                   `json:"description"`
	IsActive    *bool                     `json:"is_active"`
	ExpiresAt   types.Nullable[time.Time] `json:"expires_at"`
}

type ListRunnerQuery struct {
	Page   int
	Limit  int
	Search string
}

type RunnerResponse struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	Token       string     `json:"token"`
	IsActive    bool       `json:"is_active"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func ToRunnerResponse(r *Runner) RunnerResponse {
	return RunnerResponse{
		ID:          r.ID,
		UserID:      r.UserID,
		Name:        r.Name,
		Description: r.Description,
		Token:       r.Token,
		IsActive:    r.IsActive,
		ExpiresAt:   r.ExpiresAt,
		LastUsedAt:  r.LastUsedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// RunnerSelfResponse: data yang dilihat runner sendiri (tanpa token).
type RunnerSelfResponse struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func ToRunnerSelfResponse(r *Runner) RunnerSelfResponse {
	return RunnerSelfResponse{ID: r.ID, UserID: r.UserID, Name: r.Name, ExpiresAt: r.ExpiresAt}
}

func (r *CreateRunnerRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = normalizeDescription(r.Description)
}

func (r *CreateRunnerRequest) Validate() error {
	errs := users.ValidationErrors{}
	validateName(errs, r.Name)
	validateDescription(errs, r.Description)
	validateExpiresAt(errs, r.ExpiresAt)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *UpdateRunnerRequest) Normalize() {
	if r.Name != nil {
		v := strings.TrimSpace(*r.Name)
		r.Name = &v
	}
	if r.Description != nil {
		v := strings.TrimSpace(*r.Description)
		r.Description = &v
	}
}

func (r *UpdateRunnerRequest) Validate() error {
	errs := users.ValidationErrors{}
	if r.Name != nil {
		validateName(errs, *r.Name)
	}
	validateDescription(errs, r.Description)
	if r.ExpiresAt.Set {
		validateExpiresAt(errs, r.ExpiresAt.Value)
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func normalizeDescription(v *string) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	return &s
}

func validateName(errs users.ValidationErrors, v string) {
	if v == "" || len(v) > 100 {
		errs["name"] = "is required, max 100 characters"
	}
}

func validateDescription(errs users.ValidationErrors, v *string) {
	if v != nil && len(*v) > 500 {
		errs["description"] = "max 500 characters"
	}
}

func validateExpiresAt(errs users.ValidationErrors, v *time.Time) {
	if v != nil && !v.After(time.Now()) {
		errs["expires_at"] = "must be in the future"
	}
}
