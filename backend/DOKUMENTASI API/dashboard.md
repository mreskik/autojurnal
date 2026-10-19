# Dashboard & Monitoring

Endpoint data untuk dashboard monitoring di aplikasi. Semua endpoint **wajib login** (token session) dan hanya mengembalikan data milik user yang login. Jurnal milik user lain = `1400 journal not found`.

Panduan membangun tampilan dashboard (layout, chart, aturan tampilan) ada di [frontend-dashboard-guide.md](frontend-dashboard-guide.md). Format response umum di [README.md](README.md), error di [error-codes.md](error-codes.md).

| Method | Endpoint | Isi | Dipakai di model |
|--------|----------|-----|------------------|
| GET | `/api/dashboard/overview` | Ringkasan semua jurnal, aktivitas trade terbaru | Overview |
| GET | `/api/dashboard/runners` | Kesehatan & performa semua runner | Runner Monitor |
| GET | `/api/journals/:id/dashboard/summary` | KPI lengkap satu jurnal | Journal Performance, Target Tracker |
| GET | `/api/journals/:id/dashboard/equity` | Kurva balance, PnL kumulatif, drawdown | Journal Performance |
| GET | `/api/journals/:id/dashboard/periods` | Target vs pencapaian per periode | Target Tracker |
| GET | `/api/journals/:id/dashboard/calendar` | PnL harian satu bulan (heatmap) | Target Tracker, Analytics |
| GET | `/api/journals/:id/dashboard/breakdown` | Statistik dikelompokkan (pair, jam, runner, dll) | Analytics |

---

## Konsep & Definisi Metrik

Semua angka uang dalam **USD**, dikirim sebagai **string decimal** (`"85"`, `"14.16666667"`). Persentase juga string dengan 2 desimal (`"66.67"`).

### Trade yang dihitung

- Statistik performa hanya memakai trade **closed**. Trade open masuk ke `open_positions`.
- Trade yang dihapus tidak dihitung.
- Semua trade dihitung (manual pemilik + semua runner), kecuali breakdown yang memang memisahkannya.

### Definisi

| Metrik | Rumus / arti |
|--------|--------------|
| `net_pnl` | Σ `pnl` trade closed (pnl sudah bersih setelah fee) |
| `wins` / `losses` / `breakeven` | Jumlah trade dengan `pnl` > 0 / < 0 / = 0 |
| `win_rate` | `wins / (wins + losses) × 100`. Breakeven **tidak** dihitung |
| `gross_profit` | Σ pnl trade menang (positif) |
| `gross_loss` | Σ pnl trade kalah (**negatif**, contoh `"-20"`) |
| `profit_factor` | `gross_profit / |gross_loss|`. `null` kalau belum ada trade kalah. `> 1` = profit |
| `payoff_ratio` | `avg_win / |avg_loss|`. `null` kalau belum ada menang atau kalah |
| `avg_pnl` | Expectancy per trade = `net_pnl / total_trades` |
| `avg_win` / `avg_loss` | Rata-rata pnl trade menang / kalah (`avg_loss` negatif) |
| `largest_win` / `largest_loss` | pnl tertinggi / terendah |
| `avg_roi` | Rata-rata `pnl / margin × 100` per trade |
| `avg_holding_seconds` | Rata-rata `closed_at − opened_at` dalam detik |
| `roi_percent` (jurnal) | `net_pnl / (initial_balance + Σ deposit) × 100` |
| Streak trade | Beruntun menang / kalah berdasarkan urutan `closed_at`. Breakeven dilewati (tidak memutus) |
| Drawdown | Penurunan dari puncak **PnL kumulatif** (deposit / withdraw tidak dianggap profit / rugi). `max` = terbesar sepanjang waktu, `current` = jarak dari puncak saat ini. Persen = drawdown ÷ balance saat puncak × 100 |

### Periode target

| Status | Arti |
|--------|------|
| `achieved` | PnL periode ≥ target periode (dihitung dari **hasil akhir** periode) |
| `missed` | Periode sudah lewat, ada trade closed, PnL < target |
| `no_trade` | Periode sudah lewat, tidak ada trade closed |
| `in_progress` | Periode berjalan dan belum tercapai |

- Target periode `percent` = persen × **balance di awal periode** (compounding). Deposit / withdraw / PnL di dalam periode baru berpengaruh ke periode berikutnya.
- PnL masuk ke periode tanggal `closed_at` (timezone jurnal).
- `achievement_rate` = `achieved / (achieved + missed) × 100` (no_trade & in_progress tidak dihitung).
- Streak target: `achieved` +1, `missed` reset ke 0, `no_trade` dan `in_progress` **tidak** memutus streak.
- Periode pertama dimulai dari `start_date` dan bisa tidak penuh (misal weekly mulai hari Kamis).

### Filter tanggal (`from` / `to`)

Format `YYYY-MM-DD`, inklusif, di timezone jurnal. Untuk summary & breakdown, filter berlaku ke `closed_at` trade.

---

## GET /api/dashboard/overview

Ringkasan semua jurnal milik user untuk halaman utama.

**Query parameter**

| Nama | Default | Keterangan |
|------|---------|------------|
| `include_archived` | `false` | `true` = ikut tampilkan jurnal `archived` di `journals` (totals tetap hanya jurnal aktif) |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "totals": {
      "journals_active": 1,
      "journals_archived": 0,
      "total_balance": "1585",
      "total_initial": "1000",
      "net_pnl": "85",
      "today_pnl": "0",
      "open_trades": 1,
      "total_position_size": "500",
      "targets_achieved": 0,
      "targets_in_progress": 1,
      "runners_total": 3,
      "runners_active": 3
    },
    "journals": [
      {
        "id": "01a1551c-94f0-77dc-b77d-8b7b8adf8410",
        "name": "Scalping BTC",
        "status": "active",
        "period_type": "daily",
        "balance": "1585",
        "net_pnl": "85",
        "roi_percent": "5.67",
        "today_pnl": "0",
        "win_rate": "66.67",
        "open_trades": 1,
        "runners_joined": 2,
        "current_target": {
          "target_amount": "31.7",
          "pnl": "0",
          "progress_percent": "0",
          "status": "in_progress",
          "period_start": "2026-10-20",
          "period_end": "2026-10-20"
        },
        "goal": { "goal_type": "balance", "progress_percent": "79.25", "status": "in_progress" },
        "last_trade_at": "2026-10-20T00:01:43.092761+07:00"
      }
    ],
    "recent_trades": [
      {
        "id": "01a1551c-a536-7453-b4ed-ad02829b787c",
        "journal_id": "01a1551c-94f0-77dc-b77d-8b7b8adf8410",
        "journal_name": "Scalping BTC",
        "runner_id": "01a1551c-905e-7599-8646-eda492697cb2",
        "runner_name": "Bot A",
        "pair": "BTCUSDT",
        "position": "long",
        "status": "open",
        "pnl": null,
        "opened_at": "2026-10-20T00:01:43.092761+07:00",
        "closed_at": null,
        "updated_at": "2026-10-20T00:01:43.092407+07:00"
      }
    ]
  }
}
```

| Field | Keterangan |
|-------|------------|
| `totals.*` | Agregat jurnal **aktif** saja. `runners_*` dari semua runner milik user |
| `totals.today_pnl` | Σ pnl trade yang closed hari ini (hari dihitung di timezone tiap jurnal) |
| `totals.targets_achieved` / `targets_in_progress` | Jumlah jurnal aktif yang target periode berjalannya sudah / belum tercapai |
| `journals[]` | Diurutkan dari jurnal terbaru dibuat |
| `journals[].current_target` | `null` kalau jurnal belum mulai |
| `journals[].goal` | `null` kalau jurnal tidak punya target akhir |
| `journals[].last_trade_at` | Waktu trade terakhir dibuka / ditutup, `null` kalau belum ada trade |
| `recent_trades` | 10 trade terakhir yang berubah (dibuka / diubah / ditutup) di semua jurnal. `runner_name` `null` = input manual |

---

## GET /api/dashboard/runners

Kesehatan dan performa semua runner milik user (runner yang sudah dihapus tidak ikut).

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a1551c-905e-7599-8646-eda492697cb2",
      "name": "Bot A",
      "is_active": true,
      "expires_at": null,
      "last_used_at": "2026-10-20T00:01:39.99452+07:00",
      "idle_seconds": 69,
      "health": "online",
      "journals_joined": 1,
      "open_trades": 1,
      "closed_trades": 3,
      "wins": 2,
      "losses": 1,
      "win_rate": "66.67",
      "net_pnl": "20",
      "trades_last_24h": 1
    }
  ]
}
```

| Field | Keterangan |
|-------|------------|
| `health` | Lihat tabel di bawah |
| `idle_seconds` | Detik sejak token runner terakhir dipakai. `null` kalau belum pernah dipakai |
| `last_used_at` | Diperbarui paling sering 1× per menit, jadi `idle_seconds` < 60 dianggap "baru saja" |
| `journals_joined` | Jumlah jurnal (belum dihapus) tempat runner terdaftar |
| `open_trades`, `closed_trades`, `wins`, `losses`, `win_rate`, `net_pnl` | Dari semua trade runner ini di semua jurnal |
| `trades_last_24h` | Trade yang dibuka atau ditutup runner ini dalam 24 jam terakhir |

| `health` | Kondisi |
|----------|---------|
| `online` | Aktif, token dipakai ≤ 5 menit lalu |
| `idle` | Aktif, terakhir dipakai 5 menit - 1 jam lalu |
| `offline` | Aktif, terakhir dipakai > 1 jam lalu |
| `never_used` | Aktif, token belum pernah dipakai |
| `expired` | Lewat `expires_at` |
| `inactive` | Dinonaktifkan pemilik |

Urutan: `online`, `idle`, `offline`, `never_used`, `expired`, `inactive`, lalu dari yang terakhir dipakai.

---

## GET /api/journals/:id/dashboard/summary

KPI lengkap satu jurnal.

**Query parameter**

| Nama | Keterangan |
|------|------------|
| `from` | Opsional, `YYYY-MM-DD`. Filter `closed_at` mulai tanggal ini |
| `to` | Opsional, `YYYY-MM-DD`. Filter `closed_at` sampai tanggal ini (inklusif) |

Filter **hanya** berlaku untuk: `performance`, `trades`, `streaks`, `best_trade`, `worst_trade`. Bagian lain (`balance`, `drawdown`, `open_positions`, `target`, `goal`) selalu kondisi keseluruhan / saat ini.

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "range": { "from": null, "to": null },
    "balance": {
      "initial": "1000",
      "current": "1585",
      "deposits": "500",
      "withdrawals": "0",
      "net_flow": "500"
    },
    "performance": { "net_pnl": "85", "roi_percent": "5.67", "total_fee": "0" },
    "trades": {
      "total_trades": 6,
      "wins": 4,
      "losses": 2,
      "breakeven": 0,
      "win_rate": "66.67",
      "net_pnl": "85",
      "gross_profit": "105",
      "gross_loss": "-20",
      "total_fee": "0",
      "profit_factor": "5.25",
      "payoff_ratio": "2.63",
      "avg_pnl": "14.16666667",
      "avg_win": "26.25",
      "avg_loss": "-10",
      "largest_win": "40",
      "largest_loss": "-15",
      "avg_roi": "10",
      "avg_holding_seconds": 7800
    },
    "streaks": { "current_type": "win", "current_count": 1, "max_wins": 2, "max_losses": 1 },
    "best_trade": {
      "id": "01a1551c-a318-73f9-8c06-385288d63b1c",
      "runner_id": "01a1551c-9222-75aa-b4b3-98c43be088b9",
      "pair": "BTCUSDT",
      "position": "short",
      "pnl": "40",
      "roi": "20",
      "opened_at": "2026-10-06T01:00:00+07:00",
      "closed_at": "2026-10-06T03:00:00+07:00"
    },
    "worst_trade": { "id": "...", "runner_id": "...", "pair": "ETHUSDT", "position": "short", "pnl": "-15", "roi": "-15", "opened_at": "...", "closed_at": "..." },
    "drawdown": { "max": "15", "max_percent": "1.42", "current": "0", "current_percent": "0" },
    "open_positions": { "count": 1, "long": 1, "short": 0, "total_margin": "50", "total_position_size": "500" },
    "target": {
      "period_type": "daily",
      "current_period": {
        "period_start": "2026-10-20",
        "period_end": "2026-10-20",
        "is_current": true,
        "status": "in_progress",
        "target_type": "percent",
        "target_value": "2",
        "target_amount": "31.7",
        "pnl": "0",
        "progress_percent": "0",
        "remaining": "31.7",
        "opening_balance": "1585",
        "closing_balance": "1585",
        "net_flow": "0",
        "trades": 0,
        "wins": 0,
        "losses": 0
      },
      "stats": {
        "total": 20,
        "achieved": 3,
        "missed": 2,
        "no_trade": 14,
        "in_progress": 1,
        "achievement_rate": "60",
        "current_streak": 1,
        "best_streak": 1
      },
      "best_period": { "period_start": "2026-10-06", "period_end": "2026-10-06", "pnl": "40", "status": "achieved" },
      "worst_period": { "period_start": "2026-10-05", "period_end": "2026-10-05", "pnl": "-15", "status": "missed" }
    },
    "goal": {
      "goal_type": "balance",
      "goal_value": "2000",
      "current_value": "1585",
      "progress_percent": "79.25",
      "remaining": "415",
      "status": "in_progress",
      "achieved_at": null
    }
  }
}
```

| Field | Keterangan |
|-------|------------|
| `streaks.current_type` | `win` / `loss` / `none` (belum ada trade menang/kalah) |
| `best_trade` / `worst_trade` | `null` kalau belum ada trade closed (di rentang filter) |
| `target.current_period` | Objek Period (lihat endpoint periods). `null` kalau jurnal belum mulai |
| `target.best_period` / `worst_period` | Periode dengan pnl tertinggi / terendah di antara periode yang punya trade. `null` kalau belum ada |
| `goal` | Objek Goal (lihat [journals.md](journals.md)), `null` kalau tidak diset |

Jurnal tanpa trade: semua angka `"0"`, `profit_factor` & `payoff_ratio` `null`, `best_trade` `null`, `streaks.current_type` `"none"`.

---

## GET /api/journals/:id/dashboard/equity

Kurva balance, PnL kumulatif, dan drawdown per bucket waktu.

**Query parameter**

| Nama | Default | Keterangan |
|------|---------|------------|
| `interval` | `day` | `day` / `week` (Senin-Minggu) / `month` |
| `from` | `start_date` jurnal | `YYYY-MM-DD` |
| `to` | Hari ini | `YYYY-MM-DD` |

Maksimal 1000 titik. Rentang yang terlalu panjang untuk interval kecil ditolak `1002` (pakai interval lebih besar).

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "interval": "day",
    "from": "2026-10-01",
    "to": "2026-10-07",
    "points": [
      {
        "date": "2026-10-05",
        "pnl": "-15",
        "net_flow": "500",
        "balance": "1545",
        "cumulative_pnl": "45",
        "drawdown": "15",
        "drawdown_percent": "1.42",
        "trades": 1,
        "wins": 0,
        "losses": 1
      }
    ]
  }
}
```

| Field per titik | Keterangan |
|-----------------|------------|
| `date` | Awal bucket (untuk `week` = Senin, `month` = tanggal 1). Bucket pertama bisa dimulai sebelum `from` |
| `pnl` | Σ pnl trade closed di bucket ini |
| `net_flow` | Deposit − withdraw di bucket ini |
| `balance` | Balance di **akhir** bucket |
| `cumulative_pnl` | PnL kumulatif sejak jurnal mulai sampai akhir bucket |
| `drawdown`, `drawdown_percent` | Drawdown di akhir bucket (dari puncak PnL kumulatif sepanjang waktu) |
| `trades`, `wins`, `losses` | Trade closed di bucket ini |

Bucket tanpa aktivitas tetap muncul (pnl `"0"`), jadi chart garis tidak bolong. Jurnal yang belum mulai: `points` = `[]`.

**Response 422**

```json
{ "code": 1002, "message": "validation failed", "data": { "interval": "must be day, week or month" } }
```

---

## GET /api/journals/:id/dashboard/periods

Target vs pencapaian per periode (daily / weekly / monthly sesuai `period_type` jurnal), **terbaru dulu**, dengan pagination.

**Query parameter**

| Nama | Default | Keterangan |
|------|---------|------------|
| `status` | - | Filter `achieved` / `missed` / `no_trade` / `in_progress` |
| `page` | 1 | |
| `limit` | 30 | Maks 100 |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "period_type": "daily",
    "summary": {
      "total": 20,
      "achieved": 3,
      "missed": 2,
      "no_trade": 14,
      "in_progress": 1,
      "achievement_rate": "60",
      "current_streak": 1,
      "best_streak": 1
    },
    "items": [
      {
        "period_start": "2026-10-06",
        "period_end": "2026-10-06",
        "is_current": false,
        "status": "achieved",
        "target_type": "percent",
        "target_value": "2",
        "target_amount": "30.9",
        "pnl": "40",
        "progress_percent": "129.45",
        "remaining": "0",
        "opening_balance": "1545",
        "closing_balance": "1585",
        "net_flow": "0",
        "trades": 1,
        "wins": 1,
        "losses": 0
      }
    ]
  },
  "meta": { "page": 1, "limit": 30, "total": 3, "total_page": 1 }
}
```

| Field | Keterangan |
|-------|------------|
| `summary` | Selalu dari **semua** periode (tidak terpengaruh filter `status` / pagination) |
| `meta.total` | Jumlah periode setelah filter `status` |
| `target_type`, `target_value` | Aturan target yang berlaku di periode itu (dari riwayat target) |
| `target_amount` | Target USD periode itu |
| `progress_percent` | `pnl / target_amount × 100` (bisa > 100 atau negatif) |
| `remaining` | Sisa menuju target, minimal `"0"` |
| `opening_balance` / `closing_balance` | Balance awal / akhir periode |
| `net_flow` | Deposit − withdraw di periode itu |

---

## GET /api/journals/:id/dashboard/calendar

PnL per hari dalam satu bulan, untuk heatmap kalender.

**Query parameter**

| Nama | Default | Keterangan |
|------|---------|------------|
| `month` | Bulan berjalan | `YYYY-MM` |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "month": "2026-10",
    "totals": {
      "pnl": "85",
      "trades": 6,
      "wins": 4,
      "losses": 2,
      "profitable_days": 4,
      "losing_days": 1,
      "best_day": "2026-10-06",
      "worst_day": "2026-10-05"
    },
    "days": [
      { "date": "2026-10-01", "pnl": "20", "trades": 2, "wins": 1, "losses": 1, "target_amount": "20", "target_status": "achieved" },
      { "date": "2026-10-02", "pnl": "10", "trades": 1, "wins": 1, "losses": 0, "target_amount": "20.4", "target_status": "missed" },
      { "date": "2026-10-03", "pnl": "0", "trades": 0, "wins": 0, "losses": 0, "target_amount": "20.6", "target_status": "no_trade" }
    ]
  }
}
```

- `days` selalu berisi **semua tanggal** di bulan itu (28-31 item).
- `target_amount` & `target_status` hanya terisi untuk jurnal **daily** dan tanggal yang sudah masuk periode jurnal. Selain itu `null`.
- `profitable_days` / `losing_days` hanya menghitung hari yang ada trade closed.

**Response 422**

```json
{ "code": 1002, "message": "validation failed", "data": { "month": "must be a month in YYYY-MM format" } }
```

---

## GET /api/journals/:id/dashboard/breakdown

Statistik trade closed yang dikelompokkan.

**Query parameter**

| Nama | Wajib | Keterangan |
|------|-------|------------|
| `by` | Ya | Dimensi pengelompokan (tabel di bawah) |
| `from`, `to` | Tidak | Filter `closed_at`, `YYYY-MM-DD` |

| `by` | `key` | `label` | Urutan |
|------|-------|---------|--------|
| `pair` | `BTCUSDT` | sama | `net_pnl` tertinggi dulu |
| `position` | `long` / `short` | sama | `net_pnl` |
| `close_reason` | `take_profit`, `stop_loss`, `manual`, `liquidation`, `trailing_stop` | sama | `net_pnl` |
| `source` | `manual` / `runner` | sama | `net_pnl` |
| `runner` | id runner, atau `manual` | nama runner / `Manual (pemilik)` | `net_pnl` |
| `exchange` | `binance`, ..., `unknown` (kosong) | sama | `net_pnl` |
| `weekday` | `monday` ... `sunday` (hari **dibuka** trade, timezone jurnal) | sama | Senin → Minggu, **selalu 7 item** |
| `hour` | `00` ... `23` (jam **dibuka** trade, timezone jurnal) | `00:00` ... `23:00` | 00 → 23, **selalu 24 item** |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "by": "pair",
    "range": { "from": null, "to": null },
    "items": [
      {
        "key": "BTCUSDT",
        "label": "BTCUSDT",
        "trades": 3,
        "wins": 3,
        "losses": 0,
        "breakeven": 0,
        "win_rate": "100",
        "net_pnl": "75",
        "avg_pnl": "25",
        "gross_profit": "75",
        "gross_loss": "0",
        "profit_factor": null,
        "total_fee": "0"
      }
    ]
  }
}
```

`profit_factor` `null` = belum ada trade kalah di grup itu, `"0"` = belum ada trade menang. Grup kosong (weekday / hour tanpa trade) berisi semua angka `0`.

**Response 422**

```json
{ "code": 1002, "message": "validation failed", "data": { "by": "must be pair, position, close_reason, source, runner, exchange, weekday or hour" } }
```

---

## Error Umum

| HTTP | Code | Kapan |
|------|------|-------|
| 401 | `1201` | Belum login |
| 404 | `1400` | Jurnal tidak ada / milik user lain |
| 400 | `1003` | `id` bukan UUID valid |
| 422 | `1002` | Parameter tidak valid (`from`, `to`, `interval`, `month`, `status`, `by`) |
