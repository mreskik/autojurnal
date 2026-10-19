package users

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrUsernameTaken = errors.New("username already used")
	ErrEmailTaken    = errors.New("email already used")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, q ListUserQuery) ([]UserResponse, int, error) {
	list, total, err := s.repo.List(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	result := make([]UserResponse, 0, len(list))
	for i := range list {
		result = append(result, ToUserResponse(&list[i]))
	}
	return result, total, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (UserResponse, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}
	return ToUserResponse(user), nil
}

func (s *Service) Create(ctx context.Context, req CreateUserRequest) (UserResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return UserResponse{}, err
	}
	if err := s.ensureUnique(ctx, &req.Username, req.Email, uuid.Nil); err != nil {
		return UserResponse{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserResponse{}, err
	}

	user := &User{
		Username:     req.Username,
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return UserResponse{}, mapUniqueViolation(err)
	}
	return ToUserResponse(user), nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (UserResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return UserResponse{}, err
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}

	var email *string
	if req.Email != nil && *req.Email != "" {
		email = req.Email
	}
	if err := s.ensureUnique(ctx, req.Username, email, id); err != nil {
		return UserResponse{}, err
	}

	columns := []string{"updated_at"}
	if req.Username != nil {
		user.Username = *req.Username
		columns = append(columns, "username")
	}
	if req.Name != nil {
		user.Name = *req.Name
		columns = append(columns, "name")
	}
	if req.Email != nil {
		user.Email = email // "" -> NULL
		columns = append(columns, "email")
	}
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return UserResponse{}, err
		}
		user.PasswordHash = string(hash)
		columns = append(columns, "password_hash")
	}
	user.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, user, columns...); err != nil {
		return UserResponse{}, mapUniqueViolation(err)
	}
	return ToUserResponse(user), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ensureUnique(ctx context.Context, username, email *string, excludeID uuid.UUID) error {
	if username != nil {
		exists, err := s.repo.UsernameExists(ctx, *username, excludeID)
		if err != nil {
			return err
		}
		if exists {
			return ErrUsernameTaken
		}
	}
	if email != nil {
		exists, err := s.repo.EmailExists(ctx, *email, excludeID)
		if err != nil {
			return err
		}
		if exists {
			return ErrEmailTaken
		}
	}
	return nil
}

// mapUniqueViolation menangani race condition saat dua request insert bersamaan.
func mapUniqueViolation(err error) error {
	var pgErr pgdriver.Error
	if errors.As(err, &pgErr) && pgErr.IntegrityViolation() {
		switch pgErr.Field('n') {
		case "users_username_unique":
			return ErrUsernameTaken
		case "users_email_unique":
			return ErrEmailTaken
		}
	}
	return err
}
