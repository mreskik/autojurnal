package journals

import (
	"context"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/types"

	"github.com/shopspring/decimal"
)

// Totals: angka agregat jurnal untuk balance & target jurnal.
type Totals struct {
	Deposit  decimal.Decimal
	Withdraw decimal.Decimal
	PnL      decimal.Decimal // Σ pnl trade closed
}

// Balance = initial + deposit - withdraw + pnl.
func (t Totals) Balance(initial decimal.Decimal) decimal.Decimal {
	return initial.Add(t.Deposit).Sub(t.Withdraw).Add(t.PnL)
}

// GoalResponse: progres target akhir jurnal.
type GoalResponse struct {
	GoalType        string          `json:"goal_type"`
	GoalValue       decimal.Decimal `json:"goal_value"`
	CurrentValue    decimal.Decimal `json:"current_value"`
	ProgressPercent decimal.Decimal `json:"progress_percent"`
	Remaining       decimal.Decimal `json:"remaining"`
	Status          string          `json:"status"` // in_progress / achieved
	AchievedAt      *time.Time      `json:"achieved_at"`
}

// goalCurrentValue: nilai saat ini sesuai tipe target.
//
//	balance : total balance
//	profit  : Σ pnl trade closed
//	percent : Σ pnl / (initial_balance + Σ deposit) × 100
func goalCurrentValue(j *Journal, t Totals) decimal.Decimal {
	switch *j.GoalType {
	case GoalBalance:
		return t.Balance(j.InitialBalance)
	case GoalPercent:
		capital := j.InitialBalance.Add(t.Deposit)
		return t.PnL.Div(capital).Mul(hundred).Round(4)
	default:
		return t.PnL
	}
}

func buildGoal(j *Journal, t Totals) *GoalResponse {
	if j.GoalType == nil || j.GoalValue == nil {
		return nil
	}
	current := goalCurrentValue(j, t)
	remaining := j.GoalValue.Sub(current)
	if remaining.IsNegative() {
		remaining = decimal.Zero
	}
	status := "in_progress"
	if j.GoalAchievedAt != nil {
		status = "achieved"
	}
	return &GoalResponse{
		GoalType:        *j.GoalType,
		GoalValue:       *j.GoalValue,
		CurrentValue:    current,
		ProgressPercent: current.Div(*j.GoalValue).Mul(hundred).Round(2),
		Remaining:       remaining,
		Status:          status,
		AchievedAt:      j.GoalAchievedAt,
	}
}

func validateGoal(errs users.ValidationErrors, goalType *string, goalValue *decimal.Decimal) {
	if goalType == nil && goalValue == nil {
		return
	}
	if goalType == nil {
		errs["goal_type"] = "is required when goal_value is set"
		return
	}
	if *goalType != GoalBalance && *goalType != GoalProfit && *goalType != GoalPercent {
		errs["goal_type"] = "must be balance, profit or percent"
	}
	if goalValue == nil {
		errs["goal_value"] = "is required when goal_type is set"
	} else if !goalValue.IsPositive() {
		errs["goal_value"] = "must be greater than 0"
	}
}

// applyGoalUpdate menerapkan perubahan target jurnal dari request update.
// goal_type null = hapus target. Kalau target berubah, goal_achieved_at di-reset.
// Mengembalikan true kalau ada perubahan.
func applyGoalUpdate(j *Journal, goalType types.Nullable[string], goalValue *decimal.Decimal) (bool, error) {
	if !goalType.Set && goalValue == nil {
		return false, nil
	}
	if goalType.Set && goalType.Value == nil {
		changed := j.GoalType != nil
		j.GoalType, j.GoalValue, j.GoalAchievedAt = nil, nil, nil
		return changed, nil
	}

	newType := j.GoalType
	if goalType.Set {
		newType = goalType.Value
	}
	newValue := j.GoalValue
	if goalValue != nil {
		newValue = goalValue
	}
	errs := users.ValidationErrors{}
	validateGoal(errs, newType, newValue)
	if len(errs) > 0 {
		return false, errs
	}

	changed := j.GoalType == nil || *j.GoalType != *newType || !j.GoalValue.Equal(*newValue)
	j.GoalType, j.GoalValue = newType, newValue
	if changed {
		j.GoalAchievedAt = nil
	}
	return changed, nil
}

// syncGoal mencatat goal_achieved_at saat target jurnal pertama kali tercapai.
// Sekali tercapai tidak di-reset walaupun nilai turun lagi.
func syncGoal(ctx context.Context, repo *Repository, j *Journal) error {
	if j.GoalType == nil || j.GoalAchievedAt != nil {
		return nil
	}
	t, err := repo.Totals(ctx, j.ID)
	if err != nil {
		return err
	}
	if goalCurrentValue(j, t).LessThan(*j.GoalValue) {
		return nil
	}
	now := time.Now()
	j.GoalAchievedAt = &now
	return repo.Update(ctx, j, "goal_achieved_at")
}

// afterBalanceChange dipanggil di dalam transaction setelah perubahan yang mempengaruhi balance:
// tolak kalau balance minus, lalu cek target jurnal.
func afterBalanceChange(ctx context.Context, repo *Repository, j *Journal) error {
	if err := ensureNonNegativeBalance(ctx, repo, j); err != nil {
		return err
	}
	return syncGoal(ctx, repo, j)
}
