package journals

import (
	"context"
	"errors"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/module/runners"
	"autojurnal/internal/token"

	"github.com/google/uuid"
)

var (
	ErrJournalNotFound     = errors.New("journal not found")
	ErrJournalLocked       = errors.New("journal settings are locked")
	ErrInvalidJournalToken = errors.New("missing or invalid journal token")
	ErrRunnerNotJoined     = errors.New("runner has not joined this journal")
)

// JournalLockedError: field yang ditolak karena jurnal sudah punya aktivitas.
type JournalLockedError struct {
	Fields users.ValidationErrors
}

func (e *JournalLockedError) Error() string { return ErrJournalLocked.Error() }
func (e *JournalLockedError) Unwrap() error { return ErrJournalLocked }

type Service struct {
	repo       *Repository
	runnerRepo *runners.Repository
}

func NewService(repo *Repository, runnerRepo *runners.Repository) *Service {
	return &Service{repo: repo, runnerRepo: runnerRepo}
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, q ListJournalQuery) ([]JournalResponse, int, error) {
	list, total, err := s.repo.ListByUser(ctx, userID, q)
	if err != nil {
		return nil, 0, err
	}
	result, err := s.toResponses(ctx, list)
	return result, total, err
}

// toResponses menyusun response banyak jurnal dengan query target & totals sekaligus.
func (s *Service) toResponses(ctx context.Context, list []Journal) ([]JournalResponse, error) {
	ids := make([]uuid.UUID, 0, len(list))
	for _, j := range list {
		ids = append(ids, j.ID)
	}
	targets, err := s.repo.ListTargets(ctx, ids...)
	if err != nil {
		return nil, err
	}
	byJournal := make(map[uuid.UUID][]JournalTarget, len(list))
	for _, t := range targets {
		byJournal[t.JournalID] = append(byJournal[t.JournalID], t)
	}
	totals, err := s.repo.TotalsMany(ctx, ids...)
	if err != nil {
		return nil, err
	}

	result := make([]JournalResponse, 0, len(list))
	for i := range list {
		result = append(result, ToJournalResponse(&list[i], byJournal[list[i].ID], totals[list[i].ID]))
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (JournalResponse, error) {
	return s.GetScoped(ctx, UserScope(userID), id)
}

// GetScoped: detail jurnal sesuai scope (user atau runner).
func (s *Service) GetScoped(ctx context.Context, sc Scope, id uuid.UUID) (JournalResponse, error) {
	j, err := sc.journal(ctx, s.repo, id, false)
	if err != nil {
		return JournalResponse{}, err
	}
	return s.toResponse(ctx, s.repo, j)
}

// Create membuat jurnal sekaligus target pertamanya (berlaku sejak start_date).
func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateJournalRequest) (JournalResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return JournalResponse{}, err
	}

	tok, err := token.Generate(token.Length)
	if err != nil {
		return JournalResponse{}, err
	}
	j := &Journal{
		UserID:         userID,
		Token:          tok,
		Name:           req.Name,
		Description:    req.Description,
		InitialBalance: req.InitialBalance,
		PeriodType:     req.PeriodType,
		Timezone:       req.Timezone,
		StartDate:      req.StartDate,
		Status:         StatusActive,
		GoalType:       req.GoalType,
		GoalValue:      req.GoalValue,
		Rules:          req.Rules,
		Strategy:       req.Strategy,
	}
	var res JournalResponse
	err = s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		if err := repo.Create(ctx, j); err != nil {
			return err
		}
		target := &JournalTarget{
			JournalID:     j.ID,
			TargetType:    req.TargetType,
			TargetValue:   req.TargetValue,
			EffectiveFrom: j.StartDate,
		}
		if err := repo.CreateTarget(ctx, target); err != nil {
			return err
		}
		// misal goal balance <= initial_balance: langsung tercapai
		if err := syncGoal(ctx, repo, j); err != nil {
			return err
		}
		var err error
		res, err = s.toResponse(ctx, repo, j)
		return err
	})
	return res, err
}

// Update mengubah header jurnal.
// Kalau period_type / timezone / start_date berubah, batas periode ikut berubah,
// jadi riwayat target diringkas jadi satu (target terbaru) yang berlaku sejak start_date.
// initial_balance, period_type, timezone, start_date dikunci kalau jurnal sudah punya aktivitas.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, req UpdateJournalRequest) (JournalResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return JournalResponse{}, err
	}

	var res JournalResponse
	err := s.repo.RunInTx(ctx, func(ctx context.Context, repo *Repository) error {
		j, err := repo.FindByUserForUpdate(ctx, userID, id)
		if err != nil {
			return err
		}
		if err := checkLockedFields(ctx, repo, j, req); err != nil {
			return err
		}

		columns := []string{"updated_at"}
		goalChanged, err := applyGoalUpdate(j, req.GoalType, req.GoalValue)
		if err != nil {
			return err
		}
		if goalChanged {
			columns = append(columns, "goal_type", "goal_value", "goal_achieved_at")
		}
		periodChanged := false
		if req.Name != nil {
			j.Name = *req.Name
			columns = append(columns, "name")
		}
		if req.Description != nil {
			j.Description = normalizeText(req.Description) // "" -> NULL
			columns = append(columns, "description")
		}
		if req.Status != nil {
			j.Status = *req.Status
			columns = append(columns, "status")
		}
		if req.Rules != nil {
			j.Rules = normalizeText(req.Rules) // "" -> NULL
			columns = append(columns, "rules")
		}
		if req.Strategy != nil {
			j.Strategy = normalizeText(req.Strategy)
			columns = append(columns, "strategy")
		}
		if req.InitialBalance != nil {
			j.InitialBalance = *req.InitialBalance
			columns = append(columns, "initial_balance")
		}
		if req.PeriodType != nil && *req.PeriodType != j.PeriodType {
			j.PeriodType = *req.PeriodType
			columns = append(columns, "period_type")
			periodChanged = true
		}
		if req.Timezone != nil && *req.Timezone != j.Timezone {
			j.Timezone = *req.Timezone
			columns = append(columns, "timezone")
			periodChanged = true
		}
		if req.StartDate != nil && *req.StartDate != j.StartDate {
			j.StartDate = *req.StartDate
			columns = append(columns, "start_date")
			periodChanged = true
		}
		j.UpdatedAt = time.Now()

		if err := repo.Update(ctx, j, columns...); err != nil {
			return err
		}
		if periodChanged {
			if err := resetTargets(ctx, repo, j); err != nil {
				return err
			}
		}
		if err := syncGoal(ctx, repo, j); err != nil {
			return err
		}
		res, err = s.toResponse(ctx, repo, j)
		return err
	})
	return res, err
}

// RegenerateToken membuat journal token baru; token lama langsung tidak berlaku.
func (s *Service) RegenerateToken(ctx context.Context, userID, id uuid.UUID) (JournalResponse, error) {
	j, err := s.repo.FindByUser(ctx, userID, id)
	if err != nil {
		return JournalResponse{}, err
	}
	tok, err := token.Generate(token.Length)
	if err != nil {
		return JournalResponse{}, err
	}
	j.Token = tok
	j.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, j, "token", "updated_at"); err != nil {
		return JournalResponse{}, err
	}
	return s.toResponse(ctx, s.repo, j)
}

// RunnerJournal: jurnal dari journal token untuk runner yang sedang akses.
// Token salah / jurnal milik user lain -> ErrInvalidJournalToken; runner belum join -> ErrRunnerNotJoined.
func (s *Service) RunnerJournal(ctx context.Context, sc Scope, journalToken string) (*Journal, error) {
	if journalToken == "" {
		return nil, ErrInvalidJournalToken
	}
	j, err := s.repo.FindByToken(ctx, journalToken)
	if errors.Is(err, ErrJournalNotFound) || (err == nil && j.UserID != sc.UserID) {
		return nil, ErrInvalidJournalToken
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.FindForRunner(ctx, sc.UserID, *sc.RunnerID, j.ID, false); err != nil {
		if errors.Is(err, ErrJournalNotFound) {
			return nil, ErrRunnerNotJoined
		}
		return nil, err
	}
	return j, nil
}

func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.repo.Delete(ctx, userID, id)
}

func (s *Service) ListTargets(ctx context.Context, userID, journalID uuid.UUID) ([]TargetResponse, error) {
	if _, err := s.repo.FindByUser(ctx, userID, journalID); err != nil {
		return nil, err
	}
	targets, err := s.repo.ListTargets(ctx, journalID)
	if err != nil {
		return nil, err
	}
	result := make([]TargetResponse, 0, len(targets))
	for i := range targets {
		result = append(result, *ToTargetResponse(&targets[i]))
	}
	return result, nil
}

// CreateTarget menambah target baru; berlaku mulai periode berikutnya
// (atau sejak start_date kalau jurnal belum mulai).
func (s *Service) CreateTarget(ctx context.Context, userID, journalID uuid.UUID, req CreateTargetRequest) (TargetResponse, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return TargetResponse{}, err
	}
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return TargetResponse{}, err
	}
	effectiveFrom, err := newTargetEffectiveFrom(j)
	if err != nil {
		return TargetResponse{}, err
	}
	target := &JournalTarget{
		JournalID:     j.ID,
		TargetType:    req.TargetType,
		TargetValue:   req.TargetValue,
		EffectiveFrom: effectiveFrom,
		Note:          req.Note,
	}
	if err := s.repo.CreateTarget(ctx, target); err != nil {
		return TargetResponse{}, err
	}
	return *ToTargetResponse(target), nil
}

func (s *Service) toResponse(ctx context.Context, repo *Repository, j *Journal) (JournalResponse, error) {
	targets, err := repo.ListTargets(ctx, j.ID)
	if err != nil {
		return JournalResponse{}, err
	}
	totals, err := repo.Totals(ctx, j.ID)
	if err != nil {
		return JournalResponse{}, err
	}
	return ToJournalResponse(j, targets, totals), nil
}

// checkLockedFields menolak perubahan setting dasar jurnal kalau sudah ada transaksi (nanti juga trade),
// karena mengubahnya akan mengubah riwayat balance & periode.
func checkLockedFields(ctx context.Context, repo *Repository, j *Journal, req UpdateJournalRequest) error {
	const msg = "cannot be changed after journal has transactions or trades"
	fields := users.ValidationErrors{}
	if req.InitialBalance != nil && !req.InitialBalance.Equal(j.InitialBalance) {
		fields["initial_balance"] = msg
	}
	if req.PeriodType != nil && *req.PeriodType != j.PeriodType {
		fields["period_type"] = msg
	}
	if req.Timezone != nil && *req.Timezone != j.Timezone {
		fields["timezone"] = msg
	}
	if req.StartDate != nil && *req.StartDate != j.StartDate {
		fields["start_date"] = msg
	}
	if len(fields) == 0 {
		return nil
	}
	active, err := repo.HasActivity(ctx, j.ID)
	if err != nil {
		return err
	}
	if active {
		return &JournalLockedError{Fields: fields}
	}
	return nil
}

// resetTargets meringkas riwayat target jadi target terbaru saja, berlaku sejak start_date.
func resetTargets(ctx context.Context, repo *Repository, j *Journal) error {
	targets, err := repo.ListTargets(ctx, j.ID)
	if err != nil || len(targets) == 0 {
		return err
	}
	latest := targets[len(targets)-1]
	if err := repo.DeleteTargets(ctx, j.ID); err != nil {
		return err
	}
	return repo.CreateTarget(ctx, &JournalTarget{
		JournalID:     j.ID,
		TargetType:    latest.TargetType,
		TargetValue:   latest.TargetValue,
		EffectiveFrom: j.StartDate,
		Note:          latest.Note,
	})
}
