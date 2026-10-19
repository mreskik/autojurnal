package journals

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Status periode target.
const (
	PeriodAchieved   = "achieved"    // pnl periode >= target
	PeriodMissed     = "missed"      // periode sudah lewat, ada trade closed, pnl < target
	PeriodNoTrade    = "no_trade"    // periode sudah lewat, tidak ada trade closed
	PeriodInProgress = "in_progress" // periode berjalan, belum tercapai
)

// ---------- periode target ----------

type periodStat struct {
	start, end     time.Time // tanggal sipil, end inklusif
	startAt, endAt time.Time // instant di timezone jurnal, endAt eksklusif
	targetType     string
	targetValue    decimal.Decimal
	targetAmount   decimal.Decimal
	opening        decimal.Decimal
	pnl            decimal.Decimal
	flow           decimal.Decimal
	closing        decimal.Decimal
	trades         int
	wins, losses   int
	status         string
	current        bool
}

// buildPeriods: semua periode dari start_date sampai periode berjalan (urut lama → baru).
// Status dihitung dari hasil akhir periode.
func buildPeriods(d *journalData) []periodStat {
	today := d.today()
	p := d.startDate()
	if today.Before(p) {
		return nil
	}

	var (
		list   []periodStat
		ti, xi int
		bal    = d.j.InitialBalance
	)
	for !p.After(today) {
		next := nextPeriodStart(p, d.j.PeriodType)
		ps := periodStat{
			start:   p,
			end:     next.AddDate(0, 0, -1),
			startAt: d.instant(p),
			endAt:   d.instant(next),
			opening: bal,
			current: !today.Before(p) && today.Before(next),
		}
		for ti < len(d.closed) && d.closed[ti].ClosedAt.Before(ps.endAt) {
			pnl := *d.closed[ti].PnL
			ps.pnl = ps.pnl.Add(pnl)
			ps.trades++
			if pnl.IsPositive() {
				ps.wins++
			} else if pnl.IsNegative() {
				ps.losses++
			}
			ti++
		}
		for xi < len(d.txs) && d.txs[xi].OccurredAt.Before(ps.endAt) {
			if d.txs[xi].Type == TxDeposit {
				ps.flow = ps.flow.Add(d.txs[xi].Amount)
			} else {
				ps.flow = ps.flow.Sub(d.txs[xi].Amount)
			}
			xi++
		}
		ps.closing = ps.opening.Add(ps.pnl).Add(ps.flow)
		bal = ps.closing

		if t := activeTargetAt(d.targets, p.Format(dateLayout)); t != nil {
			ps.targetType, ps.targetValue = t.TargetType, t.TargetValue
			ps.targetAmount = t.TargetValue
			if t.TargetType == TargetPercent {
				ps.targetAmount = ps.opening.Mul(t.TargetValue).Div(hundred).Round(pnlPrecision)
			}
		}

		switch {
		case ps.targetAmount.IsPositive() && ps.pnl.GreaterThanOrEqual(ps.targetAmount):
			ps.status = PeriodAchieved
		case ps.current:
			ps.status = PeriodInProgress
		case ps.trades == 0:
			ps.status = PeriodNoTrade
		default:
			ps.status = PeriodMissed
		}
		list = append(list, ps)
		p = next
	}
	return list
}

func activeTargetAt(targets []JournalTarget, date string) *JournalTarget {
	for i := len(targets) - 1; i >= 0; i-- {
		if targets[i].EffectiveFrom <= date {
			return &targets[i]
		}
	}
	return nil
}

// summarizePeriods: hitungan status, achievement rate, dan streak.
// Streak: achieved menambah, missed mereset, no_trade & in_progress tidak mengubah.
func summarizePeriods(list []periodStat) PeriodSummary {
	var s PeriodSummary
	for _, p := range list {
		s.Total++
		switch p.status {
		case PeriodAchieved:
			s.Achieved++
			s.CurrentStreak++
			if s.CurrentStreak > s.BestStreak {
				s.BestStreak = s.CurrentStreak
			}
		case PeriodMissed:
			s.Missed++
			s.CurrentStreak = 0
		case PeriodNoTrade:
			s.NoTrade++
		case PeriodInProgress:
			s.InProgress++
		}
	}
	s.AchievementRate = percent(s.Achieved, s.Achieved+s.Missed)
	return s
}

func toPeriodItem(p periodStat) PeriodItem {
	remaining := p.targetAmount.Sub(p.pnl)
	if remaining.IsNegative() {
		remaining = decimal.Zero
	}
	return PeriodItem{
		PeriodStart:     p.start.Format(dateLayout),
		PeriodEnd:       p.end.Format(dateLayout),
		IsCurrent:       p.current,
		Status:          p.status,
		TargetType:      p.targetType,
		TargetValue:     p.targetValue,
		TargetAmount:    p.targetAmount,
		PnL:             p.pnl,
		ProgressPercent: ratioPercent(p.pnl, p.targetAmount),
		Remaining:       remaining,
		OpeningBalance:  p.opening,
		ClosingBalance:  p.closing,
		NetFlow:         p.flow,
		Trades:          p.trades,
		Wins:            p.wins,
		Losses:          p.losses,
	}
}

// ---------- statistik trade ----------

func computeTradeStats(trades []Trade) TradeStats {
	var (
		st      TradeStats
		roiSum  decimal.Decimal
		holdSum int64
	)
	for i := range trades {
		t := &trades[i]
		pnl := *t.PnL
		st.TotalTrades++
		st.NetPnL = st.NetPnL.Add(pnl)
		st.TotalFee = st.TotalFee.Add(t.Fee)
		switch {
		case pnl.IsPositive():
			st.Wins++
			st.GrossProfit = st.GrossProfit.Add(pnl)
			if pnl.GreaterThan(st.LargestWin) {
				st.LargestWin = pnl
			}
		case pnl.IsNegative():
			st.Losses++
			st.GrossLoss = st.GrossLoss.Add(pnl)
			if pnl.LessThan(st.LargestLoss) {
				st.LargestLoss = pnl
			}
		default:
			st.Breakeven++
		}
		roiSum = roiSum.Add(pnl.Div(t.Margin).Mul(hundred))
		holdSum += int64(t.ClosedAt.Sub(t.OpenedAt).Seconds())
	}

	st.WinRate = percent(st.Wins, st.Wins+st.Losses)
	if st.TotalTrades > 0 {
		n := decimal.NewFromInt(int64(st.TotalTrades))
		st.AvgPnL = st.NetPnL.Div(n).Round(pnlPrecision)
		st.AvgROI = roiSum.Div(n).Round(2)
		st.AvgHoldingSeconds = holdSum / int64(st.TotalTrades)
	}
	if st.Wins > 0 {
		st.AvgWin = st.GrossProfit.Div(decimal.NewFromInt(int64(st.Wins))).Round(pnlPrecision)
	}
	if st.Losses > 0 {
		st.AvgLoss = st.GrossLoss.Div(decimal.NewFromInt(int64(st.Losses))).Round(pnlPrecision)
	}
	if !st.GrossLoss.IsZero() {
		pf := st.GrossProfit.Div(st.GrossLoss.Abs()).Round(2)
		st.ProfitFactor = &pf
	}
	if st.Wins+st.Losses > 0 && st.Losses > 0 && st.Wins > 0 {
		// payoff ratio = rata-rata menang / |rata-rata kalah|
		pr := st.AvgWin.Div(st.AvgLoss.Abs()).Round(2)
		st.PayoffRatio = &pr
	}
	return st
}

// computeStreaks: beruntun menang / kalah berdasarkan urutan closed_at. Breakeven dilewati.
func computeStreaks(trades []Trade) Streaks {
	var s Streaks
	s.CurrentType = "none"
	var cur int
	var curType string
	for i := range trades {
		pnl := *trades[i].PnL
		var typ string
		switch {
		case pnl.IsPositive():
			typ = "win"
		case pnl.IsNegative():
			typ = "loss"
		default:
			continue
		}
		if typ == curType {
			cur++
		} else {
			curType, cur = typ, 1
		}
		if typ == "win" && cur > s.MaxWins {
			s.MaxWins = cur
		}
		if typ == "loss" && cur > s.MaxLosses {
			s.MaxLosses = cur
		}
	}
	if curType != "" {
		s.CurrentType, s.CurrentCount = curType, cur
	}
	return s
}

func bestWorstTrade(trades []Trade) (best, worst *TradeBrief) {
	for i := range trades {
		t := &trades[i]
		if best == nil || t.PnL.GreaterThan(best.PnL) {
			b := toTradeBrief(t)
			best = &b
		}
		if worst == nil || t.PnL.LessThan(worst.PnL) {
			w := toTradeBrief(t)
			worst = &w
		}
	}
	return best, worst
}

func toTradeBrief(t *Trade) TradeBrief {
	roi := t.PnL.Div(t.Margin).Mul(hundred).Round(2)
	return TradeBrief{
		ID:       t.ID,
		RunnerID: t.RunnerID,
		Pair:     t.Pair,
		Position: t.Position,
		PnL:      *t.PnL,
		ROI:      roi,
		OpenedAt: t.OpenedAt,
		ClosedAt: *t.ClosedAt,
	}
}

// ---------- equity & drawdown ----------

// equityWalker: menelusuri event balance secara berurutan sambil melacak drawdown.
// Drawdown dihitung dari kurva PnL kumulatif (deposit / withdraw tidak dianggap profit / rugi).
type equityWalker struct {
	events     []balanceEvent
	idx        int
	balance    decimal.Decimal
	cumPnL     decimal.Decimal
	peakCum    decimal.Decimal
	balAtPeak  decimal.Decimal
	maxDD      decimal.Decimal
	maxDDPct   decimal.Decimal
	lastTrades int
	lastWins   int
	lastLosses int
}

func newEquityWalker(d *journalData) *equityWalker {
	return &equityWalker{events: d.events, balance: d.j.InitialBalance, balAtPeak: d.j.InitialBalance}
}

// advance memproses event sebelum `until`, mengembalikan pnl & flow yang diproses.
func (w *equityWalker) advance(until time.Time) (pnl, flow decimal.Decimal) {
	w.lastTrades, w.lastWins, w.lastLosses = 0, 0, 0
	for w.idx < len(w.events) && w.events[w.idx].at.Before(until) {
		e := w.events[w.idx]
		w.idx++
		w.balance = w.balance.Add(e.pnl).Add(e.flow)
		pnl, flow = pnl.Add(e.pnl), flow.Add(e.flow)
		if e.trade == nil {
			continue
		}
		w.lastTrades++
		if e.pnl.IsPositive() {
			w.lastWins++
		} else if e.pnl.IsNegative() {
			w.lastLosses++
		}
		w.cumPnL = w.cumPnL.Add(e.pnl)
		if w.cumPnL.GreaterThan(w.peakCum) {
			w.peakCum, w.balAtPeak = w.cumPnL, w.balance
		}
		dd, ddPct := w.drawdown()
		if dd.GreaterThan(w.maxDD) {
			w.maxDD = dd
		}
		if ddPct.GreaterThan(w.maxDDPct) {
			w.maxDDPct = ddPct
		}
	}
	return pnl, flow
}

func (w *equityWalker) drawdown() (decimal.Decimal, decimal.Decimal) {
	dd := w.peakCum.Sub(w.cumPnL)
	if !w.balAtPeak.IsPositive() {
		return dd, decimal.Zero
	}
	return dd, dd.Div(w.balAtPeak).Mul(hundred).Round(2)
}

func computeDrawdown(d *journalData) Drawdown {
	w := newEquityWalker(d)
	w.advance(d.now.Add(time.Hour))
	cur, curPct := w.drawdown()
	return Drawdown{Max: w.maxDD, MaxPercent: w.maxDDPct, Current: cur, CurrentPercent: curPct}
}

// equityPoints: kurva equity per bucket (day / week / month) untuk rentang tanggal [from, to].
func equityPoints(d *journalData, interval string, from, to time.Time) []EquityPoint {
	periodType := map[string]string{"day": PeriodDaily, "week": PeriodWeekly, "month": PeriodMonthly}[interval]
	w := newEquityWalker(d)
	b := periodStart(from, periodType)
	w.advance(d.instant(b)) // event sebelum rentang masuk ke balance awal

	points := make([]EquityPoint, 0)
	for !b.After(to) {
		next := nextPeriodStart(b, periodType)
		pnl, flow := w.advance(d.instant(next))
		dd, ddPct := w.drawdown()
		points = append(points, EquityPoint{
			Date:            b.Format(dateLayout),
			PnL:             pnl,
			NetFlow:         flow,
			Balance:         w.balance,
			CumulativePnL:   w.cumPnL,
			Drawdown:        dd,
			DrawdownPercent: ddPct,
			Trades:          w.lastTrades,
			Wins:            w.lastWins,
			Losses:          w.lastLosses,
		})
		b = next
	}
	return points
}

// ---------- breakdown ----------

var breakdownDimensions = map[string]bool{
	"pair": true, "position": true, "close_reason": true, "source": true,
	"runner": true, "exchange": true, "weekday": true, "hour": true,
}

var weekdays = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

func computeBreakdown(d *journalData, trades []Trade, by string, runnerNames map[uuid.UUID]string) []BreakdownItem {
	groups := map[string][]Trade{}
	labels := map[string]string{}
	for _, t := range trades {
		key, label := breakdownKey(d, &t, by, runnerNames)
		groups[key] = append(groups[key], t)
		labels[key] = label
	}

	// weekday & hour selalu lengkap (7 hari / 24 jam) supaya chart tidak bolong
	switch by {
	case "weekday":
		for _, w := range weekdays {
			if _, ok := groups[w]; !ok {
				groups[w], labels[w] = nil, w
			}
		}
	case "hour":
		for h := 0; h < 24; h++ {
			key := fmt.Sprintf("%02d", h)
			if _, ok := groups[key]; !ok {
				groups[key], labels[key] = nil, key+":00"
			}
		}
	}

	items := make([]BreakdownItem, 0, len(groups))
	for key, list := range groups {
		st := computeTradeStats(list)
		items = append(items, BreakdownItem{
			Key:          key,
			Label:        labels[key],
			Trades:       st.TotalTrades,
			Wins:         st.Wins,
			Losses:       st.Losses,
			Breakeven:    st.Breakeven,
			WinRate:      st.WinRate,
			NetPnL:       st.NetPnL,
			AvgPnL:       st.AvgPnL,
			GrossProfit:  st.GrossProfit,
			GrossLoss:    st.GrossLoss,
			ProfitFactor: st.ProfitFactor,
			TotalFee:     st.TotalFee,
		})
	}

	switch by {
	case "weekday":
		order := map[string]int{}
		for i, w := range weekdays {
			order[w] = i
		}
		sort.Slice(items, func(a, b int) bool { return order[items[a].Key] < order[items[b].Key] })
	case "hour":
		sort.Slice(items, func(a, b int) bool { return items[a].Key < items[b].Key })
	default:
		sort.Slice(items, func(a, b int) bool {
			if !items[a].NetPnL.Equal(items[b].NetPnL) {
				return items[a].NetPnL.GreaterThan(items[b].NetPnL)
			}
			return items[a].Key < items[b].Key
		})
	}
	return items
}

func breakdownKey(d *journalData, t *Trade, by string, runnerNames map[uuid.UUID]string) (string, string) {
	switch by {
	case "position":
		return t.Position, t.Position
	case "close_reason":
		return *t.CloseReason, *t.CloseReason
	case "source":
		if t.RunnerID == nil {
			return "manual", "manual"
		}
		return "runner", "runner"
	case "runner":
		if t.RunnerID == nil {
			return "manual", "Manual (pemilik)"
		}
		name := runnerNames[*t.RunnerID]
		if name == "" {
			name = "Runner"
		}
		return t.RunnerID.String(), name
	case "exchange":
		if t.Exchange == nil {
			return "unknown", "unknown"
		}
		return *t.Exchange, *t.Exchange
	case "weekday":
		w := strings.ToLower(t.OpenedAt.In(d.loc).Weekday().String())
		return w, w
	case "hour":
		h := t.OpenedAt.In(d.loc).Format("15")
		return h, h + ":00"
	default:
		return t.Pair, t.Pair
	}
}

// ---------- helper angka ----------

// percent: a / b × 100, 2 desimal (0 kalau b = 0).
func percent(a, b int) decimal.Decimal {
	if b == 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(int64(a)).Div(decimal.NewFromInt(int64(b))).Mul(hundred).Round(2)
}

// ratioPercent: a / b × 100, 2 desimal (0 kalau b <= 0).
func ratioPercent(a, b decimal.Decimal) decimal.Decimal {
	if !b.IsPositive() {
		return decimal.Zero
	}
	return a.Div(b).Mul(hundred).Round(2)
}
