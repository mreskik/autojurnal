package journals

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RunnerJournalResponse: detail jurnal versi runner (tanpa id, user_id, dan riwayat target).
type RunnerJournalResponse struct {
	Name           string                `json:"name"`
	Description    *string               `json:"description"`
	Status         string                `json:"status"`
	InitialBalance decimal.Decimal       `json:"initial_balance"`
	Balance        decimal.Decimal       `json:"balance"`
	PeriodType     string                `json:"period_type"`
	Timezone       string                `json:"timezone"`
	StartDate      string                `json:"start_date"`
	CurrentTarget  *RunnerTargetResponse `json:"current_target"`
	// aturan & strategi yang harus diikuti runner (panduan, tidak dipaksakan server)
	Instructions RunnerInstructions `json:"instructions"`
	Goal         *GoalResponse      `json:"goal"`
	// semua trade open di jurnal (manual & semua runner), terbaru dulu
	OpenTrades []TradeResponse `json:"open_trades"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// InstructionSet: teks rules & strategy (null kalau tidak diisi).
type InstructionSet struct {
	Rules    *string `json:"rules"`
	Strategy *string `json:"strategy"`
}

// RunnerInstructions: journal = berlaku untuk semua runner di jurnal, runner = khusus runner ini.
type RunnerInstructions struct {
	Journal InstructionSet `json:"journal"`
	Runner  InstructionSet `json:"runner"`
}

// RunnerTargetResponse: target periode berjalan dalam USD beserta pencapaiannya.
type RunnerTargetResponse struct {
	Amount decimal.Decimal `json:"amount"`
	// Σ pnl trade (semua sumber) yang closed di dalam periode berjalan
	Achieved decimal.Decimal `json:"achieved"`
	// achieved / amount × 100, 2 desimal (bisa > 100 atau negatif)
	ProgressPercent decimal.Decimal `json:"progress_percent"`
	PeriodStart     string          `json:"period_start"`
	PeriodEnd       string          `json:"period_end"`
}

// RunnerJournalDetail: detail jurnal untuk runner; target periode berjalan dihitung jadi nominal USD.
func (s *Service) RunnerJournalDetail(ctx context.Context, sc Scope, id uuid.UUID) (RunnerJournalResponse, error) {
	j, err := sc.journal(ctx, s.repo, id, false)
	if err != nil {
		return RunnerJournalResponse{}, err
	}
	totals, err := s.repo.Totals(ctx, j.ID)
	if err != nil {
		return RunnerJournalResponse{}, err
	}
	target, err := s.currentTargetAmount(ctx, j)
	if err != nil {
		return RunnerJournalResponse{}, err
	}
	open, err := s.repo.ListOpenTrades(ctx, j.ID)
	if err != nil {
		return RunnerJournalResponse{}, err
	}
	member, err := s.repo.FindJoinedRunner(ctx, j.ID, *sc.RunnerID)
	if err != nil {
		return RunnerJournalResponse{}, err
	}
	openTrades := make([]TradeResponse, 0, len(open))
	for i := range open {
		openTrades = append(openTrades, ToTradeResponse(&open[i]))
	}
	return RunnerJournalResponse{
		OpenTrades:     openTrades,
		Name:           j.Name,
		Description:    j.Description,
		Status:         j.Status,
		InitialBalance: j.InitialBalance,
		Balance:        totals.Balance(j.InitialBalance),
		PeriodType:     j.PeriodType,
		Timezone:       j.Timezone,
		StartDate:      j.StartDate,
		CurrentTarget:  target,
		Instructions: RunnerInstructions{
			Journal: InstructionSet{Rules: j.Rules, Strategy: j.Strategy},
			Runner:  InstructionSet{Rules: member.Rules, Strategy: member.Strategy},
		},
		Goal:      buildGoal(j, totals),
		CreatedAt: j.CreatedAt,
		UpdatedAt: j.UpdatedAt,
	}, nil
}

// currentTargetAmount: target periode berjalan dalam USD.
//
//	amount  -> target_value
//	percent -> target_value% × balance di awal periode (deposit/withdraw & pnl closed sebelum periode mulai)
func (s *Service) currentTargetAmount(ctx context.Context, j *Journal) (*RunnerTargetResponse, error) {
	p, err := currentPeriod(j)
	if err != nil {
		return nil, err
	}
	targets, err := s.repo.ListTargets(ctx, j.ID)
	if err != nil {
		return nil, err
	}
	var active *JournalTarget
	for i := len(targets) - 1; i >= 0; i-- {
		if targets[i].EffectiveFrom <= p.Start {
			active = &targets[i]
			break
		}
	}
	if active == nil {
		return nil, nil
	}

	amount := active.TargetValue
	if active.TargetType == TargetPercent {
		opening, err := s.repo.BalanceBefore(ctx, j, p.StartAt)
		if err != nil {
			return nil, err
		}
		amount = opening.Mul(active.TargetValue).Div(hundred).Round(pnlPrecision)
	}

	achieved, err := s.repo.ClosedPnLBetween(ctx, j.ID, p.StartAt, p.EndAt)
	if err != nil {
		return nil, err
	}
	progress := decimal.Zero
	if amount.IsPositive() {
		progress = achieved.Div(amount).Mul(hundred).Round(2)
	}
	return &RunnerTargetResponse{
		Amount:          amount,
		Achieved:        achieved,
		ProgressPercent: progress,
		PeriodStart:     p.Start,
		PeriodEnd:       p.End,
	}, nil
}
