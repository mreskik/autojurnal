package journals

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---------- summary (Journal Performance) ----------

type DashboardRange struct {
	From *string `json:"from"`
	To   *string `json:"to"`
}

type BalanceSummary struct {
	Initial     decimal.Decimal `json:"initial"`
	Current     decimal.Decimal `json:"current"`
	Deposits    decimal.Decimal `json:"deposits"`
	Withdrawals decimal.Decimal `json:"withdrawals"`
	NetFlow     decimal.Decimal `json:"net_flow"`
}

type PerformanceSummary struct {
	NetPnL decimal.Decimal `json:"net_pnl"`
	// net_pnl / (initial_balance + total deposit) × 100
	ROIPercent decimal.Decimal `json:"roi_percent"`
	TotalFee   decimal.Decimal `json:"total_fee"`
}

type TradeStats struct {
	TotalTrades       int              `json:"total_trades"`
	Wins              int              `json:"wins"`
	Losses            int              `json:"losses"`
	Breakeven         int              `json:"breakeven"`
	WinRate           decimal.Decimal  `json:"win_rate"`
	NetPnL            decimal.Decimal  `json:"net_pnl"`
	GrossProfit       decimal.Decimal  `json:"gross_profit"`
	GrossLoss         decimal.Decimal  `json:"gross_loss"`
	TotalFee          decimal.Decimal  `json:"total_fee"`
	ProfitFactor      *decimal.Decimal `json:"profit_factor"`
	PayoffRatio       *decimal.Decimal `json:"payoff_ratio"`
	AvgPnL            decimal.Decimal  `json:"avg_pnl"`
	AvgWin            decimal.Decimal  `json:"avg_win"`
	AvgLoss           decimal.Decimal  `json:"avg_loss"`
	LargestWin        decimal.Decimal  `json:"largest_win"`
	LargestLoss       decimal.Decimal  `json:"largest_loss"`
	AvgROI            decimal.Decimal  `json:"avg_roi"`
	AvgHoldingSeconds int64            `json:"avg_holding_seconds"`
}

type Streaks struct {
	CurrentType  string `json:"current_type"` // win / loss / none
	CurrentCount int    `json:"current_count"`
	MaxWins      int    `json:"max_wins"`
	MaxLosses    int    `json:"max_losses"`
}

type Drawdown struct {
	Max            decimal.Decimal `json:"max"`
	MaxPercent     decimal.Decimal `json:"max_percent"`
	Current        decimal.Decimal `json:"current"`
	CurrentPercent decimal.Decimal `json:"current_percent"`
}

type OpenPositions struct {
	Count             int             `json:"count"`
	Long              int             `json:"long"`
	Short             int             `json:"short"`
	TotalMargin       decimal.Decimal `json:"total_margin"`
	TotalPositionSize decimal.Decimal `json:"total_position_size"`
}

type TradeBrief struct {
	ID       uuid.UUID       `json:"id"`
	RunnerID *uuid.UUID      `json:"runner_id"`
	Pair     string          `json:"pair"`
	Position string          `json:"position"`
	PnL      decimal.Decimal `json:"pnl"`
	ROI      decimal.Decimal `json:"roi"`
	OpenedAt time.Time       `json:"opened_at"`
	ClosedAt time.Time       `json:"closed_at"`
}

type PeriodBrief struct {
	PeriodStart string          `json:"period_start"`
	PeriodEnd   string          `json:"period_end"`
	PnL         decimal.Decimal `json:"pnl"`
	Status      string          `json:"status"`
}

type TargetSummary struct {
	PeriodType    string        `json:"period_type"`
	CurrentPeriod *PeriodItem   `json:"current_period"`
	Stats         PeriodSummary `json:"stats"`
	BestPeriod    *PeriodBrief  `json:"best_period"`
	WorstPeriod   *PeriodBrief  `json:"worst_period"`
}

type DashboardSummary struct {
	Range         DashboardRange     `json:"range"`
	Balance       BalanceSummary     `json:"balance"`
	Performance   PerformanceSummary `json:"performance"`
	Trades        TradeStats         `json:"trades"`
	Streaks       Streaks            `json:"streaks"`
	BestTrade     *TradeBrief        `json:"best_trade"`
	WorstTrade    *TradeBrief        `json:"worst_trade"`
	Drawdown      Drawdown           `json:"drawdown"`
	OpenPositions OpenPositions      `json:"open_positions"`
	Target        TargetSummary      `json:"target"`
	Goal          *GoalResponse      `json:"goal"`
}

// ---------- periods (Target Tracker) ----------

type PeriodItem struct {
	PeriodStart     string          `json:"period_start"`
	PeriodEnd       string          `json:"period_end"`
	IsCurrent       bool            `json:"is_current"`
	Status          string          `json:"status"`
	TargetType      string          `json:"target_type"`
	TargetValue     decimal.Decimal `json:"target_value"`
	TargetAmount    decimal.Decimal `json:"target_amount"`
	PnL             decimal.Decimal `json:"pnl"`
	ProgressPercent decimal.Decimal `json:"progress_percent"`
	Remaining       decimal.Decimal `json:"remaining"`
	OpeningBalance  decimal.Decimal `json:"opening_balance"`
	ClosingBalance  decimal.Decimal `json:"closing_balance"`
	NetFlow         decimal.Decimal `json:"net_flow"`
	Trades          int             `json:"trades"`
	Wins            int             `json:"wins"`
	Losses          int             `json:"losses"`
}

type PeriodSummary struct {
	Total      int `json:"total"`
	Achieved   int `json:"achieved"`
	Missed     int `json:"missed"`
	NoTrade    int `json:"no_trade"`
	InProgress int `json:"in_progress"`
	// achieved / (achieved + missed) × 100
	AchievementRate decimal.Decimal `json:"achievement_rate"`
	CurrentStreak   int             `json:"current_streak"`
	BestStreak      int             `json:"best_streak"`
}

type PeriodsResponse struct {
	PeriodType string        `json:"period_type"`
	Summary    PeriodSummary `json:"summary"`
	Items      []PeriodItem  `json:"items"`
}

// ---------- equity ----------

type EquityPoint struct {
	Date            string          `json:"date"`
	PnL             decimal.Decimal `json:"pnl"`
	NetFlow         decimal.Decimal `json:"net_flow"`
	Balance         decimal.Decimal `json:"balance"`
	CumulativePnL   decimal.Decimal `json:"cumulative_pnl"`
	Drawdown        decimal.Decimal `json:"drawdown"`
	DrawdownPercent decimal.Decimal `json:"drawdown_percent"`
	Trades          int             `json:"trades"`
	Wins            int             `json:"wins"`
	Losses          int             `json:"losses"`
}

type EquityResponse struct {
	Interval string        `json:"interval"`
	From     string        `json:"from"`
	To       string        `json:"to"`
	Points   []EquityPoint `json:"points"`
}

// ---------- calendar ----------

type CalendarDay struct {
	Date   string          `json:"date"`
	PnL    decimal.Decimal `json:"pnl"`
	Trades int             `json:"trades"`
	Wins   int             `json:"wins"`
	Losses int             `json:"losses"`
	// hanya untuk jurnal daily
	TargetAmount *decimal.Decimal `json:"target_amount"`
	TargetStatus *string          `json:"target_status"`
}

type CalendarTotals struct {
	PnL            decimal.Decimal `json:"pnl"`
	Trades         int             `json:"trades"`
	Wins           int             `json:"wins"`
	Losses         int             `json:"losses"`
	ProfitableDays int             `json:"profitable_days"`
	LosingDays     int             `json:"losing_days"`
	BestDay        *string         `json:"best_day"`
	WorstDay       *string         `json:"worst_day"`
}

type CalendarResponse struct {
	Month  string         `json:"month"`
	Totals CalendarTotals `json:"totals"`
	Days   []CalendarDay  `json:"days"`
}

// ---------- breakdown ----------

type BreakdownItem struct {
	Key          string           `json:"key"`
	Label        string           `json:"label"`
	Trades       int              `json:"trades"`
	Wins         int              `json:"wins"`
	Losses       int              `json:"losses"`
	Breakeven    int              `json:"breakeven"`
	WinRate      decimal.Decimal  `json:"win_rate"`
	NetPnL       decimal.Decimal  `json:"net_pnl"`
	AvgPnL       decimal.Decimal  `json:"avg_pnl"`
	GrossProfit  decimal.Decimal  `json:"gross_profit"`
	GrossLoss    decimal.Decimal  `json:"gross_loss"`
	ProfitFactor *decimal.Decimal `json:"profit_factor"`
	TotalFee     decimal.Decimal  `json:"total_fee"`
}

type BreakdownResponse struct {
	By    string          `json:"by"`
	Range DashboardRange  `json:"range"`
	Items []BreakdownItem `json:"items"`
}

// ---------- overview (semua jurnal) ----------

type OverviewTotals struct {
	JournalsActive    int             `json:"journals_active"`
	JournalsArchived  int             `json:"journals_archived"`
	TotalBalance      decimal.Decimal `json:"total_balance"`
	TotalInitial      decimal.Decimal `json:"total_initial"`
	NetPnL            decimal.Decimal `json:"net_pnl"`
	TodayPnL          decimal.Decimal `json:"today_pnl"`
	OpenTrades        int             `json:"open_trades"`
	TotalPositionSize decimal.Decimal `json:"total_position_size"`
	TargetsAchieved   int             `json:"targets_achieved"`
	TargetsInProgress int             `json:"targets_in_progress"`
	RunnersTotal      int             `json:"runners_total"`
	RunnersActive     int             `json:"runners_active"`
}

type OverviewTarget struct {
	TargetAmount    decimal.Decimal `json:"target_amount"`
	PnL             decimal.Decimal `json:"pnl"`
	ProgressPercent decimal.Decimal `json:"progress_percent"`
	Status          string          `json:"status"`
	PeriodStart     string          `json:"period_start"`
	PeriodEnd       string          `json:"period_end"`
}

type OverviewGoal struct {
	GoalType        string          `json:"goal_type"`
	ProgressPercent decimal.Decimal `json:"progress_percent"`
	Status          string          `json:"status"`
}

type OverviewJournal struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	PeriodType    string          `json:"period_type"`
	Balance       decimal.Decimal `json:"balance"`
	NetPnL        decimal.Decimal `json:"net_pnl"`
	ROIPercent    decimal.Decimal `json:"roi_percent"`
	TodayPnL      decimal.Decimal `json:"today_pnl"`
	WinRate       decimal.Decimal `json:"win_rate"`
	OpenTrades    int             `json:"open_trades"`
	RunnersJoined int             `json:"runners_joined"`
	CurrentTarget *OverviewTarget `json:"current_target"`
	Goal          *OverviewGoal   `json:"goal"`
	LastTradeAt   *time.Time      `json:"last_trade_at"`
}

type RecentTradeItem struct {
	ID          uuid.UUID        `bun:"id" json:"id"`
	JournalID   uuid.UUID        `bun:"journal_id" json:"journal_id"`
	JournalName string           `bun:"journal_name" json:"journal_name"`
	RunnerID    *uuid.UUID       `bun:"runner_id" json:"runner_id"`
	RunnerName  *string          `bun:"runner_name" json:"runner_name"`
	Pair        string           `bun:"pair" json:"pair"`
	Position    string           `bun:"position" json:"position"`
	Status      string           `bun:"status" json:"status"`
	PnL         *decimal.Decimal `bun:"pnl" json:"pnl"`
	OpenedAt    time.Time        `bun:"opened_at" json:"opened_at"`
	ClosedAt    *time.Time       `bun:"closed_at" json:"closed_at"`
	UpdatedAt   time.Time        `bun:"updated_at" json:"updated_at"`
}

type OverviewResponse struct {
	Totals       OverviewTotals    `json:"totals"`
	Journals     []OverviewJournal `json:"journals"`
	RecentTrades []RecentTradeItem `json:"recent_trades"`
}

// ---------- runner monitor ----------

type RunnerMonitorItem struct {
	ID             uuid.UUID       `bun:"id" json:"id"`
	Name           string          `bun:"name" json:"name"`
	IsActive       bool            `bun:"is_active" json:"is_active"`
	ExpiresAt      *time.Time      `bun:"expires_at" json:"expires_at"`
	LastUsedAt     *time.Time      `bun:"last_used_at" json:"last_used_at"`
	IdleSeconds    *int64          `bun:"-" json:"idle_seconds"`
	Health         string          `bun:"-" json:"health"`
	JournalsJoined int             `bun:"journals_joined" json:"journals_joined"`
	OpenTrades     int             `bun:"open_trades" json:"open_trades"`
	ClosedTrades   int             `bun:"closed_trades" json:"closed_trades"`
	Wins           int             `bun:"wins" json:"wins"`
	Losses         int             `bun:"losses" json:"losses"`
	WinRate        decimal.Decimal `bun:"-" json:"win_rate"`
	NetPnL         decimal.Decimal `bun:"net_pnl" json:"net_pnl"`
	TradesLast24h  int             `bun:"trades_last_24h" json:"trades_last_24h"`
}
