package journals

import "time"

// Aturan periode:
//   daily   : setiap hari (termasuk Sabtu/Minggu)
//   weekly  : Senin - Minggu
//   monthly : tanggal 1 - akhir bulan
// Periode pertama dimulai dari start_date (bisa tidak penuh, misal mulai hari Rabu).
// Semua tanggal dihitung di timezone jurnal dan berbentuk "YYYY-MM-DD".

const dateLayout = "2006-01-02"

// today mengembalikan tanggal hari ini di timezone jurnal.
func today(tz string) (time.Time, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, err
	}
	y, m, d := time.Now().In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), nil
}

// periodStart: awal periode yang berisi tanggal d.
func periodStart(d time.Time, periodType string) time.Time {
	switch periodType {
	case PeriodWeekly:
		offset := (int(d.Weekday()) + 6) % 7 // Senin = 0
		return d.AddDate(0, 0, -offset)
	case PeriodMonthly:
		return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return d
	}
}

// nextPeriodStart: awal periode setelah periode yang berisi tanggal d.
func nextPeriodStart(d time.Time, periodType string) time.Time {
	start := periodStart(d, periodType)
	switch periodType {
	case PeriodWeekly:
		return start.AddDate(0, 0, 7)
	case PeriodMonthly:
		return start.AddDate(0, 1, 0)
	default:
		return start.AddDate(0, 0, 1)
	}
}

// newTargetEffectiveFrom: kapan target baru mulai berlaku.
// Jurnal belum mulai -> sejak start_date. Sudah berjalan -> periode berikutnya.
func newTargetEffectiveFrom(j *Journal) (string, error) {
	now, err := today(j.Timezone)
	if err != nil {
		return "", err
	}
	if now.Format(dateLayout) < j.StartDate {
		return j.StartDate, nil
	}
	return nextPeriodStart(now, j.PeriodType).Format(dateLayout), nil
}

// period: satu periode target jurnal.
type period struct {
	Start   string    // YYYY-MM-DD, inklusif
	End     string    // YYYY-MM-DD, inklusif
	StartAt time.Time // awal periode di timezone jurnal
	EndAt   time.Time // awal periode berikutnya (eksklusif)
}

// currentPeriod: periode berjalan jurnal (atau periode pertama kalau jurnal belum mulai).
func currentPeriod(j *Journal) (period, error) {
	start, err := currentRefDate(j)
	if err != nil {
		return period{}, err
	}
	startDate, err := time.Parse(dateLayout, start)
	if err != nil {
		return period{}, err
	}
	next := nextPeriodStart(startDate, j.PeriodType)

	loc, err := time.LoadLocation(j.Timezone)
	if err != nil {
		return period{}, err
	}
	return period{
		Start:   start,
		End:     next.AddDate(0, 0, -1).Format(dateLayout),
		StartAt: time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, loc),
		EndAt:   time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, loc),
	}, nil
}

// currentRefDate: tanggal acuan "target aktif" = awal periode berjalan,
// atau start_date kalau jurnal belum mulai.
func currentRefDate(j *Journal) (string, error) {
	now, err := today(j.Timezone)
	if err != nil {
		return "", err
	}
	ref := periodStart(now, j.PeriodType).Format(dateLayout)
	if ref < j.StartDate {
		ref = j.StartDate
	}
	return ref, nil
}
