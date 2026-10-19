package users

import (
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.]{3,50}$`)

type CreateUserRequest struct {
	Username string  `json:"username"`
	Name     string  `json:"name"`
	Email    *string `json:"email"`
	Password string  `json:"password"`
}

// UpdateUserRequest: field yang tidak dikirim (null) tidak diubah.
// Email dikirim "" untuk mengosongkan email.
type UpdateUserRequest struct {
	Username *string `json:"username"`
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
}

type ListUserQuery struct {
	Page   int
	Limit  int
	Search string
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// ValidationErrors: key = nama field, value = pesan error.
type ValidationErrors map[string]string

func (v ValidationErrors) Error() string { return "validation failed" }

func (r *CreateUserRequest) Normalize() {
	r.Username = strings.TrimSpace(r.Username)
	r.Name = strings.TrimSpace(r.Name)
	r.Email = normalizeEmail(r.Email)
}

func (r *CreateUserRequest) Validate() error {
	errs := ValidationErrors{}
	validateUsername(errs, r.Username)
	validateName(errs, r.Name)
	validateEmail(errs, r.Email)
	validatePassword(errs, r.Password)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *UpdateUserRequest) Normalize() {
	if r.Username != nil {
		v := strings.TrimSpace(*r.Username)
		r.Username = &v
	}
	if r.Name != nil {
		v := strings.TrimSpace(*r.Name)
		r.Name = &v
	}
	if r.Email != nil {
		v := strings.TrimSpace(*r.Email)
		r.Email = &v
	}
}

func (r *UpdateUserRequest) Validate() error {
	errs := ValidationErrors{}
	if r.Username != nil {
		validateUsername(errs, *r.Username)
	}
	if r.Name != nil {
		validateName(errs, *r.Name)
	}
	if r.Email != nil && *r.Email != "" {
		validateEmail(errs, r.Email)
	}
	if r.Password != nil {
		validatePassword(errs, *r.Password)
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func normalizeEmail(email *string) *string {
	if email == nil {
		return nil
	}
	v := strings.TrimSpace(*email)
	if v == "" {
		return nil
	}
	return &v
}

func validateUsername(errs ValidationErrors, v string) {
	if !usernamePattern.MatchString(v) {
		errs["username"] = "must be 3-50 characters: letters, numbers, underscore or dot"
	}
}

func validateName(errs ValidationErrors, v string) {
	if v == "" || len(v) > 100 {
		errs["name"] = "is required, max 100 characters"
	}
}

func validateEmail(errs ValidationErrors, v *string) {
	if v == nil {
		return
	}
	if len(*v) > 255 {
		errs["email"] = "max 255 characters"
		return
	}
	if addr, err := mail.ParseAddress(*v); err != nil || addr.Address != *v {
		errs["email"] = "invalid email format"
	}
}

func validatePassword(errs ValidationErrors, v string) {
	// bcrypt hanya memakai 72 byte pertama
	if len(v) < 8 || len(v) > 72 {
		errs["password"] = "must be 8-72 characters"
	}
}
