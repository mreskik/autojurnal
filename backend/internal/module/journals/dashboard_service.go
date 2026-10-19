package journals

import (
	"context"
	"sort"
	"time"

	"autojurnal/internal/module/master/users"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	maxEquityPoints    = 1000
	recentTradesLimit  = 10
	runnerOnlineWindow = 5 * time.Minute
	runnerIdleWindow   = time.Hour
)

// ---------- Journal Performance ----------

// DashboardSummary: KPI lengkap satu jurnal.
// from/to (YYYY-MM-DD, closed_at, timezone jurnal) memfilter bagian performance, trades, streaks, best/worst trade.
// Balance, drawdown, open positions, target, dan goal selalu kondisi keseluruhan / saat ini.
func (s *Service) DashboardSummary(ctx context.Context, userID, journalID uuid.UUID, from, to string) (DashboardSummary, error) {
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return DashboardSummary{}, err
	}
	fromT, toT, err := dateRange(j.Timezone, from, to)
	if err != nil {
		return DashboardSummary{}, err
	}
	d, err := s.loadJournalData(ctx, j)
	if err != nil {
		return DashboardSummary{}, err
	}

	totals := d.totals()
	trades := d.closedInRange(fromT, toT)
	stats := computeTradeStats(trades)
	best, worst := bestWorstTrade(trades)
	capital := j.InitialBalance.Add(totals.Deposit)

	res := DashboardSummary{
		Range: rangeOf(from, to),
		Balance: BalanceSummary{
			Initial:     j.InitialBalance,
			Current:     totals.Balance(j.InitialBalance),
			Deposits:    totals.Deposit,
			Withdrawals: totals.Withdraw,
			NetFlow:     totals.Deposit.Sub(totals.Withdraw),
		},
		Performance: PerformanceSummary{
			NetPnL:     stats.NetPnL,
			ROIPercent: ratioPercent(stats.NetPnL, capital),
			TotalFee:   stats.TotalFee,
		},
		Trades:        stats,
		Streaks:       computeStreaks(trades),
		BestTrade:     best,
		WorstTrade:    worst,
		Drawdown:      computeDrawdown(d),
		OpenPositions: openPositions(d.open),
		Goal:          buildGoal(j, totals),
	}

	periods := buildPeriods(d)
	res.Target = TargetSummary{PeriodType: j.PeriodType, Stats: summarizePeriods(periods)}
	if n := len(periods); n > 0 && periods[n-1].current {
		cur := toPeriodItem(periods[n-1])
		res.Target.CurrentPeriod = &cur
	}
	res.Target.BestPeriod, res.Target.WorstPeriod = bestWorstPeriod(periods)
	return res, nil
}

func openPositions(open []Trade) OpenPositions {
	var p OpenPositions
	for _, t := range open {
		p.Count++
		if t.Position == PositionLong {
			p.Long++
		} else {
			p.Short++
		}
		p.TotalMargin = p.TotalMargin.Add(t.Margin)
		p.TotalPositionSize = p.TotalPositionSize.Add(t.PositionSize())
	}
	return p
}

// bestWorstPeriod: hanya periode yang punya trade closed.
func bestWorstPeriod(periods []periodStat) (best, worst *PeriodBrief) {
	for _, p := range periods {
		if p.trades == 0 {
			continue
		}
		brief := PeriodBrief{
			PeriodStart: p.start.Format(dateLayout),
			PeriodEnd:   p.end.Format(dateLayout),
			PnL:         p.pnl,
			Status:      p.status,
		}
		if best == nil || p.pnl.GreaterThan(best.PnL) {
			b := brief
			best = &b
		}
		if worst == nil || p.pnl.LessThan(worst.PnL) {
			w := brief
			worst = &w
		}
	}
	return best, worst
}

func rangeOf(from, to string) DashboardRange {
	var r DashboardRange
	if from != "" {
		r.From = &from
	}
	if to != "" {
		r.To = &to
	}
	return r
}

// ---------- Target Tracker ----------

// DashboardPeriods: riwayat target per periode, terbaru dulu.
func (s *Service) DashboardPeriods(ctx context.Context, userID, journalID uuid.UUID, status string, page, limit int) (PeriodsResponse, int, error) {
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return PeriodsResponse{}, 0, err
	}
	d, err := s.loadJournalData(ctx, j)
	if err != nil {
		return PeriodsResponse{}, 0, err
	}
	periods := buildPeriods(d)

	filtered := make([]PeriodItem, 0, len(periods))
	for i := len(periods) - 1; i >= 0; i-- {
		if status == "" || periods[i].status == status {
			filtered = append(filtered, toPeriodItem(periods[i]))
		}
	}
	total := len(filtered)
	startIdx := (page - 1) * limit
	if startIdx > total {
		startIdx = total
	}
	endIdx := startIdx + limit
	if endIdx > total {
		endIdx = total
	}
	return PeriodsResponse{
		PeriodType: j.PeriodType,
		Summary:    summarizePeriods(periods),
		Items:      filtered[startIdx:endIdx],
	}, total, nil
}

// ---------- equity ----------

func (s *Service) DashboardEquity(ctx context.Context, userID, journalID uuid.UUID, interval, from, to string) (EquityResponse, error) {
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return EquityResponse{}, err
	}
	d, err := s.loadJournalData(ctx, j)
	if err != nil {
		return EquityResponse{}, err
	}

	if interval == "" {
		interval = "day"
	}
	errs := users.ValidationErrors{}
	if interval != "day" && interval != "week" && interval != "month" {
		errs["interval"] = "must be day, week or month"
	}
	fromD := d.startDate()
	if from != "" {
		if t, err := time.Parse(dateLayout, from); err != nil {
			errs["from"] = "must be a date in YYYY-MM-DD format"
		} else {
			fromD = t
		}
	}
	toD := d.today()
	if to != "" {
		if t, err := time.Parse(dateLayout, to); err != nil {
			errs["to"] = "must be a date in YYYY-MM-DD format"
		} else {
			toD = t
		}
	}
	if len(errs) == 0 && toD.Before(fromD) {
		if from == "" && to == "" {
			// jurnal belum mulai: kurva kosong
			return EquityResponse{Interval: interval, From: j.StartDate, To: j.StartDate, Points: []EquityPoint{}}, nil
		}
		errs["to"] = "must not be before from"
	}
	if len(errs) == 0 {
		days := int(toD.Sub(fromD).Hours()/24) + 1
		perPoint := map[string]int{"day": 1, "week": 7, "month": 28}[interval]
		if days/perPoint > maxEquityPoints {
			errs["interval"] = "too many points for this range, use a larger interval or a shorter range (max 1000 points)"
		}
	}
	if len(errs) > 0 {
		return EquityResponse{}, errs
	}

	return EquityResponse{
		Interval: interval,
		From:     fromD.Format(dateLayout),
		To:       toD.Format(dateLayout),
		Points:   equityPoints(d, interval, fromD, toD),
	}, nil
}

// ---------- calendar ----------

func (s *Service) DashboardCalendar(ctx context.Context, userID, journalID uuid.UUID, month string) (CalendarResponse, error) {
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return CalendarResponse{}, err
	}
	d, err := s.loadJournalData(ctx, j)
	if err != nil {
		return CalendarResponse{}, err
	}

	first := periodStart(d.today(), PeriodMonthly)
	if month != "" {
		t, err := time.Parse("2006-01", month)
		if err != nil {
			return CalendarResponse{}, users.ValidationErrors{"month": "must be a month in YYYY-MM format"}
		}
		first = t
	}
	last := first.AddDate(0, 1, -1)

	// target harian hanya relevan untuk jurnal daily
	targets := map[string]periodStat{}
	if j.PeriodType == PeriodDaily {
		for _, p := range buildPeriods(d) {
			targets[p.start.Format(dateLayout)] = p
		}
	}

	res := CalendarResponse{Month: first.Format("2006-01"), Days: make([]CalendarDay, 0, 31)}
	var bestPnL, worstPnL decimal.Decimal
	for _, pt := range equityPoints(d, "day", first, last) {
		day := CalendarDay{Date: pt.Date, PnL: pt.PnL, Trades: pt.Trades, Wins: pt.Wins, Losses: pt.Losses}
		if p, ok := targets[pt.Date]; ok {
			amount, status := p.targetAmount, p.status
			day.TargetAmount, day.TargetStatus = &amount, &status
		}
		res.Days = append(res.Days, day)

		res.Totals.PnL = res.Totals.PnL.Add(pt.PnL)
		res.Totals.Trades += pt.Trades
		res.Totals.Wins += pt.Wins
		res.Totals.Losses += pt.Losses
		if pt.Trades == 0 {
			continue
		}
		if pt.PnL.IsPositive() {
			res.Totals.ProfitableDays++
		} else if pt.PnL.IsNegative() {
			res.Totals.LosingDays++
		}
		date := pt.Date
		if res.Totals.BestDay == nil || pt.PnL.GreaterThan(bestPnL) {
			res.Totals.BestDay, bestPnL = &date, pt.PnL
		}
		if res.Totals.WorstDay == nil || pt.PnL.LessThan(worstPnL) {
			res.Totals.WorstDay, worstPnL = &date, pt.PnL
		}
	}
	return res, nil
}

// ---------- breakdown ----------

func (s *Service) DashboardBreakdown(ctx context.Context, userID, journalID uuid.UUID, by, from, to string) (BreakdownResponse, error) {
	if !breakdownDimensions[by] {
		return BreakdownResponse{}, users.ValidationErrors{
			"by": "must be pair, position, close_reason, source, runner, exchange, weekday or hour",
		}
	}
	j, err := s.repo.FindByUser(ctx, userID, journalID)
	if err != nil {
		return BreakdownResponse{}, err
	}
	fromT, toT, err := dateRange(j.Timezone, from, to)
	if err != nil {
		return BreakdownResponse{}, err
	}
	d, err := s.loadJournalData(ctx, j)
	if err != nil {
		return BreakdownResponse{}, err
	}
	var names map[uuid.UUID]string
	if by == "runner" {
		if names, err = s.repo.RunnerNames(ctx, userID); err != nil {
			return BreakdownResponse{}, err
		}
	}
	return BreakdownResponse{
		By:    by,
		Range: rangeOf(from, to),
		Items: computeBreakdown(d, d.closedInRange(fromT, toT), by, names),
	}, nil
}

// ---------- Overview (semua jurnal) ----------

func (s *Service) DashboardOverview(ctx context.Context, userID uuid.UUID, includeArchived bool) (OverviewResponse, error) {
	list, err := s.repo.ListAllByUser(ctx, userID, includeArchived)
	if err != nil {
		return OverviewResponse{}, err
	}
	res := OverviewResponse{Journals: make([]OverviewJournal, 0, len(list))}

	for i := range list {
		j := &list[i]
		d, err := s.loadJournalData(ctx, j)
		if err != nil {
			return OverviewResponse{}, err
		}
		joined, err := s.repo.ListRunners(ctx, j.ID)
		if err != nil {
			return OverviewResponse{}, err
		}
		item := overviewJournal(d, len(joined))
		res.Journals = append(res.Journals, item)

		if j.Status == StatusArchived {
			res.Totals.JournalsArchived++
			continue
		}
		res.Totals.JournalsActive++
		res.Totals.TotalBalance = res.Totals.TotalBalance.Add(item.Balance)
		res.Totals.TotalInitial = res.Totals.TotalInitial.Add(j.InitialBalance)
		res.Totals.NetPnL = res.Totals.NetPnL.Add(item.NetPnL)
		res.Totals.TodayPnL = res.Totals.TodayPnL.Add(item.TodayPnL)
		res.Totals.OpenTrades += item.OpenTrades
		res.Totals.TotalPositionSize = res.Totals.TotalPositionSize.Add(openPositions(d.open).TotalPositionSize)
		if item.CurrentTarget != nil {
			if item.CurrentTarget.Status == PeriodAchieved {
				res.Totals.TargetsAchieved++
			} else {
				res.Totals.TargetsInProgress++
			}
		}
	}

	runnersList, err := s.RunnerMonitor(ctx, userID)
	if err != nil {
		return OverviewResponse{}, err
	}
	res.Totals.RunnersTotal = len(runnersList)
	for _, r := range runnersList {
		if r.IsActive {
			res.Totals.RunnersActive++
		}
	}

	if res.RecentTrades, err = s.repo.RecentTrades(ctx, userID, recentTradesLimit); err != nil {
		return OverviewResponse{}, err
	}
	return res, nil
}

func overviewJournal(d *journalData, runnersJoined int) OverviewJournal {
	j := d.j
	totals := d.totals()
	stats := computeTradeStats(d.closed)
	today := d.today()
	todayFrom, todayTo := d.instant(today), d.instant(today.AddDate(0, 0, 1))

	item := OverviewJournal{
		ID:            j.ID,
		Name:          j.Name,
		Status:        j.Status,
		PeriodType:    j.PeriodType,
		Balance:       totals.Balance(j.InitialBalance),
		NetPnL:        totals.PnL,
		ROIPercent:    ratioPercent(totals.PnL, j.InitialBalance.Add(totals.Deposit)),
		TodayPnL:      computeTradeStats(d.closedInRange(&todayFrom, &todayTo)).NetPnL,
		WinRate:       stats.WinRate,
		OpenTrades:    len(d.open),
		RunnersJoined: runnersJoined,
	}

	if periods := buildPeriods(d); len(periods) > 0 && periods[len(periods)-1].current {
		p := periods[len(periods)-1]
		item.CurrentTarget = &OverviewTarget{
			TargetAmount:    p.targetAmount,
			PnL:             p.pnl,
			ProgressPercent: ratioPercent(p.pnl, p.targetAmount),
			Status:          p.status,
			PeriodStart:     p.start.Format(dateLayout),
			PeriodEnd:       p.end.Format(dateLayout),
		}
	}
	if g := buildGoal(j, totals); g != nil {
		item.Goal = &OverviewGoal{GoalType: g.GoalType, ProgressPercent: g.ProgressPercent, Status: g.Status}
	}

	var last *time.Time
	for i := range d.closed {
		if last == nil || d.closed[i].ClosedAt.After(*last) {
			last = d.closed[i].ClosedAt
		}
	}
	for i := range d.open {
		if last == nil || d.open[i].OpenedAt.After(*last) {
			t := d.open[i].OpenedAt
			last = &t
		}
	}
	item.LastTradeAt = last
	return item
}

// ---------- Runner Monitor ----------

// RunnerMonitor: aktivitas & performa semua runner milik user.
//
//	health: inactive (dinonaktifkan) / expired / online (aktif <= 5 menit) / idle (<= 1 jam) / offline / never_used
func (s *Service) RunnerMonitor(ctx context.Context, userID uuid.UUID) ([]RunnerMonitorItem, error) {
	list, err := s.repo.RunnerMonitor(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range list {
		r := &list[i]
		r.WinRate = percent(r.Wins, r.Wins+r.Losses)
		if r.LastUsedAt != nil {
			idle := int64(now.Sub(*r.LastUsedAt).Seconds())
			r.IdleSeconds = &idle
		}
		switch {
		case !r.IsActive:
			r.Health = "inactive"
		case r.ExpiresAt != nil && !r.ExpiresAt.After(now):
			r.Health = "expired"
		case r.LastUsedAt == nil:
			r.Health = "never_used"
		case now.Sub(*r.LastUsedAt) <= runnerOnlineWindow:
			r.Health = "online"
		case now.Sub(*r.LastUsedAt) <= runnerIdleWindow:
			r.Health = "idle"
		default:
			r.Health = "offline"
		}
	}
	sort.SliceStable(list, func(a, b int) bool { return healthOrder[list[a].Health] < healthOrder[list[b].Health] })
	return list, nil
}

var healthOrder = map[string]int{"online": 0, "idle": 1, "offline": 2, "never_used": 3, "expired": 4, "inactive": 5}
