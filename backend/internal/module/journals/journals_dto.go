package journals

import (
	"strings"
	"time"

	"autojurnal/internal/module/master/users"
	"autojurnal/internal/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var maxPercent = decimal.NewFromInt(100)

type CreateJournalRequest struct {
	Name           string           `json:"name"`
	Description    *string          `json:"description"`
	InitialBalance decimal.Decimal  `json:"initial_balance"`
	PeriodType     string           `json:"period_type"`
	Timezone       string           `json:"timezone"`
	StartDate      string           `json:"start_date"`
	TargetType     string           `json:"target_type"`
	TargetValue    decimal.Decimal  `json:"target_value"`
	GoalType       *string          `json:"goal_type"`
	GoalValue      *decimal.Decimal `json:"goal_value"`
	Rules          *string          `json:"rules"`
	Strategy       *string          `json:"strategy"`
}

// UpdateJournalRequest: field yang tidak dikirim tidak diubah.
// goal_type null = hapus target jurnal. rules / strategy "" = dikosongkan.
type UpdateJournalRequest struct {
	Name           *string                `json:"name"`
	Description    *string                `json:"description"`
	Status         *string                `json:"status"`
	InitialBalance *decimal.Decimal       `json:"initial_balance"`
	PeriodType     *string                `json:"period_type"`
	Timezone       *string                `json:"timezone"`
	StartDate      *string                `json:"start_date"`
	GoalType       types.Nullable[string] `json:"goal_type"`
	GoalValue      *decimal.Decimal       `json:"goal_value"`
	Rules          *string                `json:"rules"`
	Strategy       *string                `json:"strategy"`
}

type CreateTargetRequest struct {
	TargetType  string          `json:"target_type"`
	TargetValue decimal.Decimal `json:"target_value"`
	Note        *string         `json:"note"`
}

type ListJournalQuery struct {
	Page   int
	Limit  int
	Search string
	Status string
}

type TargetResponse struct {
	ID            uuid.UUID       `json:"id"`
	TargetType    string          `json:"target_type"`
	TargetValue   decimal.Decimal `json:"target_value"`
	EffectiveFrom string          `json:"effective_from"`
	Note          *string         `json:"note"`
	CreatedAt     time.Time       `json:"created_at"`
}

type JournalResponse struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	Name           string          `json:"name"`
	Description    *string         `json:"description"`
	InitialBalance decimal.Decimal `json:"initial_balance"`
	PeriodType     string          `json:"period_type"`
	Timezone       string          `json:"timezone"`
	StartDate      string          `json:"start_date"`
	Status         string          `json:"status"`
	// token jurnal untuk runner (header X-Journal-Token)
	Token string `json:"token"`
	// instruksi umum untuk semua runner di jurnal ini
	Rules    *string `json:"rules"`
	Strategy *string `json:"strategy"`
	// target yang berlaku di periode berjalan
	CurrentTarget *TargetResponse `json:"current_target"`
	// target terbaru (bisa baru berlaku periode berikutnya)
	LatestTarget *TargetResponse `json:"latest_target"`
	// total balance saat ini = initial + deposit - withdraw + pnl closed
	Balance decimal.Decimal `json:"balance"`
	// target akhir jurnal, null kalau tidak diset
	Goal      *GoalResponse `json:"goal"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func ToTargetResponse(t *JournalTarget) *TargetResponse {
	if t == nil {
		return nil
	}
	return &TargetResponse{
		ID:            t.ID,
		TargetType:    t.TargetType,
		TargetValue:   t.TargetValue,
		EffectiveFrom: t.EffectiveFrom,
		Note:          t.Note,
		CreatedAt:     t.CreatedAt,
	}
}

// ToJournalResponse: targets = riwayat target jurnal ini, urut effective_from lalu id.
func ToJournalResponse(j *Journal, targets []JournalTarget, totals Totals) JournalResponse {
	res := JournalResponse{
		ID:             j.ID,
		UserID:         j.UserID,
		Name:           j.Name,
		Description:    j.Description,
		InitialBalance: j.InitialBalance,
		PeriodType:     j.PeriodType,
		Timezone:       j.Timezone,
		StartDate:      j.StartDate,
		Status:         j.Status,
		Token:          j.Token,
		Rules:          j.Rules,
		Strategy:       j.Strategy,
		Balance:        totals.Balance(j.InitialBalance),
		Goal:           buildGoal(j, totals),
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
	}
	if len(targets) == 0 {
		return res
	}
	res.LatestTarget = ToTargetResponse(&targets[len(targets)-1])

	if ref, err := currentRefDate(j); err == nil {
		for i := len(targets) - 1; i >= 0; i-- {
			if targets[i].EffectiveFrom <= ref {
				res.CurrentTarget = ToTargetResponse(&targets[i])
				break
			}
		}
	}
	return res
}

func (r *CreateJournalRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = normalizeText(r.Description)
	r.Rules = normalizeText(r.Rules)
	r.Strategy = normalizeText(r.Strategy)
	r.PeriodType = strings.TrimSpace(r.PeriodType)
	r.Timezone = strings.TrimSpace(r.Timezone)
	if r.Timezone == "" {
		r.Timezone = "Asia/Jakarta"
	}
	r.StartDate = strings.TrimSpace(r.StartDate)
	if r.StartDate == "" {
		if now, err := today(r.Timezone); err == nil {
			r.StartDate = now.Format(dateLayout)
		}
	}
	r.TargetType = strings.TrimSpace(r.TargetType)
	if r.GoalType != nil {
		v := strings.ToLower(strings.TrimSpace(*r.GoalType))
		r.GoalType = &v
	}
}

func (r *CreateJournalRequest) Validate() error {
	errs := users.ValidationErrors{}
	validateName(errs, r.Name)
	validateDescription(errs, r.Description)
	validateInitialBalance(errs, r.InitialBalance)
	validatePeriodType(errs, r.PeriodType)
	validateTimezone(errs, r.Timezone)
	validateDate(errs, "start_date", r.StartDate)
	validateTarget(errs, r.TargetType, r.TargetValue)
	validateGoal(errs, r.GoalType, r.GoalValue)
	validateInstruction(errs, "rules", r.Rules)
	validateInstruction(errs, "strategy", r.Strategy)
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *UpdateJournalRequest) Normalize() {
	if r.GoalType.Value != nil {
		*r.GoalType.Value = strings.ToLower(strings.TrimSpace(*r.GoalType.Value))
	}
	for _, p := range []*string{r.Name, r.Description, r.Status, r.PeriodType, r.Timezone, r.StartDate, r.Rules, r.Strategy} {
		if p != nil {
			*p = strings.TrimSpace(*p)
		}
	}
}

func (r *UpdateJournalRequest) Validate() error {
	errs := users.ValidationErrors{}
	if r.Name != nil {
		validateName(errs, *r.Name)
	}
	validateDescription(errs, r.Description)
	validateInstruction(errs, "rules", r.Rules)
	validateInstruction(errs, "strategy", r.Strategy)
	if r.Status != nil && *r.Status != StatusActive && *r.Status != StatusArchived {
		errs["status"] = "must be active or archived"
	}
	if r.InitialBalance != nil {
		validateInitialBalance(errs, *r.InitialBalance)
	}
	if r.PeriodType != nil {
		validatePeriodType(errs, *r.PeriodType)
	}
	if r.Timezone != nil {
		validateTimezone(errs, *r.Timezone)
	}
	if r.StartDate != nil {
		validateDate(errs, "start_date", *r.StartDate)
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func (r *CreateTargetRequest) Normalize() {
	r.TargetType = strings.TrimSpace(r.TargetType)
	r.Note = normalizeText(r.Note)
}

func (r *CreateTargetRequest) Validate() error {
	errs := users.ValidationErrors{}
	validateTarget(errs, r.TargetType, r.TargetValue)
	if r.Note != nil && len(*r.Note) > 500 {
		errs["note"] = "max 500 characters"
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func normalizeText(v *string) *string {
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
	if v != nil && len(*v) > 1000 {
		errs["description"] = "max 1000 characters"
	}
}

func validateInitialBalance(errs users.ValidationErrors, v decimal.Decimal) {
	if !v.IsPositive() {
		errs["initial_balance"] = "must be greater than 0"
	}
}

func validatePeriodType(errs users.ValidationErrors, v string) {
	if v != PeriodDaily && v != PeriodWeekly && v != PeriodMonthly {
		errs["period_type"] = "must be daily, weekly or monthly"
	}
}

func validateTimezone(errs users.ValidationErrors, v string) {
	if _, err := time.LoadLocation(v); err != nil || v == "" || v == "Local" {
		errs["timezone"] = "invalid timezone, example: Asia/Jakarta"
	}
}

func validateDate(errs users.ValidationErrors, field, v string) {
	if _, err := time.Parse(dateLayout, v); err != nil {
		errs[field] = "must be a date in YYYY-MM-DD format"
	}
}

func validateTarget(errs users.ValidationErrors, targetType string, value decimal.Decimal) {
	switch targetType {
	case TargetAmount:
		if !value.IsPositive() {
			errs["target_value"] = "must be greater than 0"
		}
	case TargetPercent:
		if !value.IsPositive() || value.GreaterThan(maxPercent) {
			errs["target_value"] = "must be greater than 0 and at most 100"
		}
	default:
		errs["target_type"] = "must be amount or percent"
	}
}

// maxInstructionLength: batas panjang teks rules / strategy.
const maxInstructionLength = 5000

func validateInstruction(errs users.ValidationErrors, field string, v *string) {
	if v != nil && len([]rune(*v)) > maxInstructionLength {
		errs[field] = "max 5000 characters"
	}
}
