package journals

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrRunnerAlreadyJoined = errors.New("runner already joined this journal")
	ErrRunnerNotInJournal  = errors.New("runner not in this journal")
)

func (s *Service) ListRunners(ctx context.Context, userID, journalID uuid.UUID) ([]JoinedRunnerResponse, error) {
	if _, err := s.repo.FindByUser(ctx, userID, journalID); err != nil {
		return nil, err
	}
	return s.repo.ListRunners(ctx, journalID)
}

// ListRunnerOptions: pilihan runner untuk ditambahkan ke jurnal (belum join).
func (s *Service) ListRunnerOptions(ctx context.Context, userID, journalID uuid.UUID, search string) ([]RunnerOptionResponse, error) {
	if _, err := s.repo.FindByUser(ctx, userID, journalID); err != nil {
		return nil, err
	}
	return s.repo.ListRunnerOptions(ctx, userID, journalID, search)
}

// AddRunner: hanya runner milik pemilik jurnal yang bisa ditambahkan.
func (s *Service) AddRunner(ctx context.Context, userID, journalID uuid.UUID, req AddRunnerRequest) ([]JoinedRunnerResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.repo.FindByUser(ctx, userID, journalID); err != nil {
		return nil, err
	}
	// runner milik user lain / sudah dihapus -> runners.ErrRunnerNotFound
	if _, err := s.runnerRepo.FindByUser(ctx, userID, req.RunnerID); err != nil {
		return nil, err
	}
	jr := &JournalRunner{
		JournalID: journalID,
		RunnerID:  req.RunnerID,
		Rules:     normalizeText(req.Rules),
		Strategy:  normalizeText(req.Strategy),
	}
	if err := s.repo.AddRunner(ctx, jr); err != nil {
		return nil, err
	}
	return s.repo.ListRunners(ctx, journalID)
}

// UpdateRunner mengubah rules / strategy khusus runner di jurnal.
func (s *Service) UpdateRunner(ctx context.Context, userID, journalID, runnerID uuid.UUID, req UpdateJoinedRunnerRequest) (*JoinedRunnerResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if _, err := s.repo.FindByUser(ctx, userID, journalID); err != nil {
		return nil, err
	}
	jr := &JournalRunner{JournalID: journalID, RunnerID: runnerID}
	var columns []string
	if req.Rules != nil {
		jr.Rules = normalizeText(req.Rules) // "" -> NULL
		columns = append(columns, "rules")
	}
	if req.Strategy != nil {
		jr.Strategy = normalizeText(req.Strategy)
		columns = append(columns, "strategy")
	}
	if len(columns) == 0 {
		// tidak ada yang diubah: tetap pastikan runner memang ada di jurnal
		return s.repo.FindJoinedRunner(ctx, journalID, runnerID)
	}
	if err := s.repo.UpdateRunnerInstructions(ctx, jr, columns...); err != nil {
		return nil, err
	}
	return s.repo.FindJoinedRunner(ctx, journalID, runnerID)
}

func (s *Service) RemoveRunner(ctx context.Context, userID, journalID, runnerID uuid.UUID) error {
	if _, err := s.repo.FindByUser(ctx, userID, journalID); err != nil {
		return err
	}
	return s.repo.RemoveRunner(ctx, journalID, runnerID)
}
