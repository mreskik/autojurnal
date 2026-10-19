package runners

import (
	"context"
	"errors"
	"log"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/token"

	"github.com/google/uuid"
)

var (
	ErrRunnerNotFound = errors.New("runner not found")
	ErrInvalidToken   = errors.New("missing or invalid runner token")
	ErrTokenExpired   = errors.New("runner token expired")
	ErrRunnerInactive = errors.New("runner inactive")
)

// last_used_at hanya di-update kalau sudah lewat interval ini
const touchInterval = time.Minute

type Service struct {
	repo     *Repository
	userRepo *users.Repository
}

func NewService(repo *Repository, userRepo *users.Repository) *Service {
	return &Service{repo: repo, userRepo: userRepo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, q ListRunnerQuery) ([]RunnerResponse, int, error) {
	list, total, err := s.repo.ListByUser(ctx, userID, q)
	if err != nil {
		return nil, 0, err
	}
	result := make([]RunnerResponse, 0, len(list))
	for i := range list {
		result = append(result, ToRunnerResponse(&list[i]))
	}
	return result, total, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (RunnerResponse, error) {
	runner, err := s.repo.FindByUser(ctx, userID, id)
	if err != nil {
		return RunnerResponse{}, err
	}
	return ToRunnerResponse(runner), nil
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRunnerRequest) (RunnerResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return RunnerResponse{}, err
	}

	token, err := generateToken()
	if err != nil {
		return RunnerResponse{}, err
	}
	runner := &Runner{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Token:       token,
		IsActive:    true,
		ExpiresAt:   req.ExpiresAt,
	}
	if err := s.repo.Create(ctx, runner); err != nil {
		return RunnerResponse{}, err
	}
	return ToRunnerResponse(runner), nil
}

func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateRunnerRequest) (RunnerResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return RunnerResponse{}, err
	}

	runner, err := s.repo.FindByUser(ctx, userID, id)
	if err != nil {
		return RunnerResponse{}, err
	}

	columns := []string{"updated_at"}
	if req.Name != nil {
		runner.Name = *req.Name
		columns = append(columns, "name")
	}
	if req.Description != nil {
		runner.Description = normalizeDescription(req.Description) // "" -> NULL
		columns = append(columns, "description")
	}
	if req.IsActive != nil {
		runner.IsActive = *req.IsActive
		columns = append(columns, "is_active")
	}
	if req.ExpiresAt.Set {
		runner.ExpiresAt = req.ExpiresAt.Value
		columns = append(columns, "expires_at")
	}
	runner.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, runner, columns...); err != nil {
		return RunnerResponse{}, err
	}
	return ToRunnerResponse(runner), nil
}

// RegenerateToken membuat token baru; token lama langsung tidak berlaku.
func (s *Service) RegenerateToken(ctx context.Context, userID, id uuid.UUID) (RunnerResponse, error) {
	runner, err := s.repo.FindByUser(ctx, userID, id)
	if err != nil {
		return RunnerResponse{}, err
	}
	token, err := generateToken()
	if err != nil {
		return RunnerResponse{}, err
	}
	runner.Token = token
	runner.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, runner, "token", "updated_at"); err != nil {
		return RunnerResponse{}, err
	}
	return ToRunnerResponse(runner), nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

// Authenticate memvalidasi token runner dari header.
func (s *Service) Authenticate(ctx context.Context, token string) (*Runner, error) {
	if token == "" {
		return nil, ErrInvalidToken
	}
	runner, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !runner.IsActive {
		return nil, ErrRunnerInactive
	}
	if runner.ExpiresAt != nil && !runner.ExpiresAt.After(time.Now()) {
		return nil, ErrTokenExpired
	}

	// pemilik yang sudah dihapus membuat runner-nya ikut tidak berlaku
	if _, err := s.userRepo.FindByID(ctx, runner.UserID); err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}

	if runner.LastUsedAt == nil || time.Since(*runner.LastUsedAt) > touchInterval {
		if err := s.repo.Touch(ctx, runner.ID); err != nil {
			log.Printf("runners: touch: %v", err)
		}
	}
	return runner, nil
}

func generateToken() (string, error) {
	return token.Generate(token.Length)
}
