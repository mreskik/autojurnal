# Panduan Frontend: Dashboard Monitoring

Dokumen ini untuk tim frontend yang membangun halaman dashboard monitoring Autojurnal. Isinya: daftar halaman, layout, widget beserta sumber datanya, aturan tampilan angka & warna, tipe TypeScript, dan daftar tugas siap dibagi.

Referensi endpoint dan definisi setiap metrik ada di [dashboard.md](dashboard.md). Kalau ada perbedaan, **dashboard.md yang benar**.

---

## 1. Ringkasan

Dashboard terdiri dari **5 model (halaman)**. Setiap model menjawab pertanyaan yang berbeda dan bisa dikerjakan terpisah.

| # | Model | Route (saran) | Pertanyaan yang dijawab | Endpoint utama |
|---|-------|---------------|-------------------------|----------------|
| 1 | **Overview** | `/dashboard` | "Semua jurnalku hari ini bagaimana?" | `GET /api/dashboard/overview` |
| 2 | **Journal Performance** | `/journals/:id/performance` | "Jurnal ini profit? Seberapa sehat?" | `summary`, `equity` |
| 3 | **Target Tracker** | `/journals/:id/targets` | "Target hari / minggu / bulan ini tercapai?" | `summary`, `periods`, `calendar` |
| 4 | **Analytics** | `/journals/:id/analytics` | "Pair / jam / runner mana yang paling bagus?" | `breakdown`, `calendar` |
| 5 | **Runner Monitor** | `/runners/monitor` | "Agent-ku masih jalan? Siapa yang profit?" | `GET /api/dashboard/runners` |

Navigasi per jurnal (model 2-4) sebaiknya berupa **tab** di halaman detail jurnal:

```
Scalping BTC                                   [● active]  [daily · Asia/Jakarta]
┌─────────────┬────────────────┬─────────────┬─────────┬─────────┬──────────┐
│ Performance │ Target Tracker │ Analytics   │ Trades  │ Runners │ Settings │
└─────────────┴────────────────┴─────────────┴─────────┴─────────┴──────────┘
```

---

## 2. Aturan Umum (wajib dibaca)

### 2.1 Autentikasi

Semua endpoint dashboard memakai token session user:

```
Authorization: Bearer <token session>
```

Kalau response `401` dengan code `1201`, hapus token lalu arahkan ke halaman login.

### 2.1.1 Endpoint yang TIDAK dipakai frontend

Endpoint **`/api/runner/*`** (`/api/runner/me`, `/api/runner/journal`, `/api/runner/trades`, ...) khusus untuk **agent / bot**. Endpoint itu memakai token runner + `X-Journal-Token`, bukan token session, dan didokumentasikan di [runner-guide.md](runner-guide.md). **Frontend tidak boleh memanggilnya** dan tidak perlu membaca runner-guide.

Kebutuhan yang mirip sudah tersedia lewat endpoint aplikasi:

| Kebutuhan di aplikasi | Pakai endpoint (token session) | Jangan pakai |
|-----------------------|--------------------------------|--------------|
| Detail jurnal, balance, target | `GET /api/journals/:id`, `GET /api/journals/:id/dashboard/summary` | `GET /api/runner/journal` |
| List trade / trade open | `GET /api/journals/:id/trades?status=open` | `GET /api/runner/trades` |
| Buka / ubah / tutup trade manual | `POST`, `PUT`, `POST .../close` di `/api/journals/:id/trades` | `/api/runner/trades` |
| Lihat token runner | `GET /api/runners/:id` | - |
| Lihat token jurnal | `GET /api/journals/:id` (field `token`) | - |
| Atur rules / strategy | `PUT /api/journals/:id`, `PUT /api/journals/:id/runners/:runnerId` | - |
| Kasih instruksi lengkap ke agent | Tombol download `GET /api/journals/:id/runners/:runnerId/agent.md` | - |
| Status agent online / offline | `GET /api/dashboard/runners` | `GET /api/runner/me` |

### 2.2 Angka uang = string decimal

API mengirim angka uang & persen sebagai **string** (`"14.16666667"`, `"66.67"`) supaya presisi tidak hilang.

- **Jangan** `parseFloat` untuk perhitungan. Pakai library decimal (`decimal.js`, `big.js`) kalau perlu menghitung (misal selisih).
- Untuk **chart**, boleh dikonversi ke `Number(...)` karena hanya untuk visual.
- Untuk **tampilan**, format dari string dengan helper di bawah.

### 2.3 Format tampilan

| Jenis | Aturan | Contoh |
|-------|--------|--------|
| Uang (USD) | 2 desimal, pemisah ribuan, prefix `$` | `"1585"` → `$1,585.00` |
| PnL | Seperti uang + tanda `+` / `−` + warna | `"85"` → **`+$85.00`** (hijau), `"-15"` → **`−$15.00`** (merah), `"0"` → `$0.00` (abu) |
| Persen | 2 desimal + `%` | `"66.67"` → `66.67%` |
| ROI / progress | Persen + tanda kalau bisa negatif | `"-15"` → `−15.00%` |
| Rasio (profit factor, payoff) | 2 desimal, tanpa satuan. `null` → `∞` (profit factor tanpa kalah) atau `—` | `"5.25"` → `5.25` |
| Durasi (`avg_holding_seconds`) | Ringkas | `7800` → `2j 10m` |
| Harga (`entry_price`, dll) | Sesuai presisi string, maks 8 desimal | `"60000"` → `60,000` |
| Tanggal periode | Sesuai `period_type` | daily `Sen, 20 Okt`; weekly `14 – 20 Okt`; monthly `Okt 2026` |
| Waktu | Konversi ke timezone **jurnal** (`timezone`), bukan timezone browser | |

Gunakan font angka **tabular** (`font-variant-numeric: tabular-nums`) supaya kolom angka rata.

Contoh helper:

```ts
const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", minimumFractionDigits: 2, maximumFractionDigits: 2 });

export const fmtMoney = (v: string) => usd.format(Number(v));
export const fmtPnl = (v: string) => {
  const n = Number(v);
  if (n === 0) return usd.format(0);
  return (n > 0 ? "+" : "−") + usd.format(Math.abs(n));
};
export const fmtPct = (v: string | null, signed = false) => {
  if (v === null) return "—";
  const n = Number(v);
  const s = Math.abs(n).toFixed(2) + "%";
  return signed && n !== 0 ? (n > 0 ? "+" : "−") + s : (n < 0 ? "−" : "") + s;
};
export const pnlTone = (v: string) => (Number(v) > 0 ? "profit" : Number(v) < 0 ? "loss" : "neutral");
```

### 2.4 Warna & status (design tokens)

| Token | Pemakaian | Saran warna |
|-------|-----------|-------------|
| `profit` | PnL positif, win, achieved | hijau `#16A34A` |
| `loss` | PnL negatif, loss, missed | merah `#DC2626` |
| `neutral` | 0, breakeven, no_trade | abu `#6B7280` |
| `progress` | in_progress, target berjalan | biru `#2563EB` |
| `warning` | idle, mendekati deadline target | kuning `#D97706` |

Status periode target (`status`):

| Status | Badge | Warna | Ikon (saran) |
|--------|-------|-------|--------------|
| `achieved` | Tercapai | profit | ✓ |
| `missed` | Tidak tercapai | loss | ✕ |
| `no_trade` | Tidak trading | neutral | – |
| `in_progress` | Berjalan | progress | ◷ |

Status kesehatan runner (`health`):

| Health | Label | Warna dot |
|--------|-------|-----------|
| `online` | Online | hijau (berkedip halus) |
| `idle` | Idle | kuning |
| `offline` | Offline | abu |
| `never_used` | Belum dipakai | abu outline |
| `expired` | Token expired | merah |
| `inactive` | Nonaktif | abu gelap |

Jangan hanya mengandalkan warna: selalu sertakan **tanda +/− atau ikon/label** (aksesibilitas buta warna).

### 2.5 Pengambilan data & refresh

Saran: **TanStack Query** (React Query) / SWR.

| Data | Query key (saran) | Refresh otomatis |
|------|-------------------|------------------|
| Overview | `["dashboard","overview"]` | tiap 30 detik saat tab aktif |
| Runner monitor | `["dashboard","runners"]` | tiap 15 detik |
| Summary jurnal | `["journal",id,"summary",from,to]` | tiap 30 detik |
| Equity | `["journal",id,"equity",interval,from,to]` | tiap 60 detik |
| Periods | `["journal",id,"periods",status,page]` | tiap 60 detik |
| Calendar | `["journal",id,"calendar",month]` | tiap 60 detik |
| Breakdown | `["journal",id,"breakdown",by,from,to]` | manual / saat filter berubah |

- Hentikan polling saat tab browser tidak aktif (`refetchIntervalInBackground: false`).
- Setelah user membuat / menutup trade atau deposit dari aplikasi, **invalidate** semua query `["journal", id, ...]` dan `["dashboard","overview"]`.
- Batas rate limit API ±120 request/menit per user. Interval di atas jauh di bawah batas itu.

### 2.6 State tampilan (wajib untuk setiap widget)

| State | Tampilan |
|-------|----------|
| Loading pertama | Skeleton sesuai bentuk widget (bukan spinner tengah halaman) |
| Refresh di background | Data lama tetap tampil, indikator kecil "memperbarui…" |
| Kosong | Ilustrasi + kalimat jelas + aksi (lihat tiap model) |
| Error | Pesan singkat + tombol "Coba lagi". `1400` → halaman "Jurnal tidak ditemukan" |
| Jurnal belum mulai | `target.current_period = null`: tampilkan "Jurnal mulai {start_date}" |

### 2.7 Penanganan error

| Code | Tindakan UI |
|------|-------------|
| `1201` | Logout, ke halaman login |
| `1400` | Halaman "Jurnal tidak ditemukan" + tombol kembali ke daftar jurnal |
| `1002` | Filter tidak valid: tampilkan pesan dari `data.<field>` di dekat input filter |
| `1006` | Tunggu `Retry-After`, jangan tampilkan error merah (cukup toast "Terlalu banyak permintaan") |
| `1000` / jaringan | Pesan "Gagal memuat data" + "Coba lagi" |

---

## 3. Model 1: Overview (Home)

**Tujuan:** sekali lihat, user tahu kondisi semua jurnal hari ini dan apakah agent berjalan.

**Endpoint:** `GET /api/dashboard/overview` (+ `GET /api/dashboard/runners` untuk panel runner ringkas).

### Layout (desktop)

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Dashboard                                        [ Tampilkan arsip ☐ ]  ⟳ 30s│
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────┤
│ Total Balance│ Net PnL      │ PnL Hari Ini │ Posisi Open  │ Target Hari Ini  │
│ $1,585.00    │ +$85.00      │ $0.00        │ 1            │ 0 / 1 tercapai   │
│ modal $1,000 │ +5.67% ROI   │              │ $500 exposure│ ▓▓░░░░░░         │
├──────────────┴──────────────┴──────────────┴──────────────┴──────────────────┤
│ Jurnal                                                                        │
│ ┌──────────────────────────────┐ ┌──────────────────────────────┐            │
│ │ Scalping BTC        ● active │ │ Swing ETH          ● active  │            │
│ │ $1,585.00   +$85.00 (+5.67%) │ │ ...                          │            │
│ │ Target hari ini   0 / 31.70  │ │                              │            │
│ │ ░░░░░░░░░░░░░░░░░  0%  ◷     │ │                              │            │
│ │ Goal $2,000  ▓▓▓▓▓▓▓▓░ 79%   │ │                              │            │
│ │ WR 66.67% · 1 open · 2 runner│ │                              │            │
│ │ Trade terakhir 5 menit lalu  │ │                              │            │
│ └──────────────────────────────┘ └──────────────────────────────┘            │
├───────────────────────────────────────────────┬──────────────────────────────┤
│ Aktivitas Terbaru                             │ Runner                        │
│ ● BTCUSDT long  open        Bot A · Scalping  │ ● Bot A   online   +$20.00   │
│ ✓ BTCUSDT short +$40.00     Bot B · Scalping  │ ● Bot B   online   +$40.00   │
│ ✕ ETHUSDT short −$15.00     Bot A · Scalping  │ ○ Bot C   belum dipakai      │
│ ...                                           │ 3 runner · 3 aktif  → Monitor│
└───────────────────────────────────────────────┴──────────────────────────────┘
```

### Widget

| Widget | Komponen | Field | Catatan |
|--------|----------|-------|---------|
| Total Balance | KPI card | `totals.total_balance`, subtext `totals.total_initial` | |
| Net PnL | KPI card | `totals.net_pnl` + ROI = `net_pnl / total_initial` (hitung di FE) atau tampilkan per jurnal saja | Warna sesuai tanda |
| PnL Hari Ini | KPI card | `totals.today_pnl` | |
| Posisi Open | KPI card | `totals.open_trades`, subtext `totals.total_position_size` | |
| Target Hari Ini | KPI card + mini progress | `totals.targets_achieved` / (`targets_achieved` + `targets_in_progress`) | Label "periode berjalan" kalau ada jurnal weekly/monthly |
| Kartu jurnal | Card grid (klik → Performance) | `journals[]` | Lihat detail di bawah |
| Aktivitas terbaru | List | `recent_trades[]` | Ikon: open ●, closed profit ✓, closed loss ✕ |
| Runner ringkas | List kecil | `/api/dashboard/runners` (maks 5) | Link ke Runner Monitor |

Kartu jurnal:

| Elemen | Field |
|--------|-------|
| Nama + badge status | `name`, `status` |
| Balance & PnL | `balance`, `net_pnl`, `roi_percent` |
| Progress target periode | `current_target.pnl` / `current_target.target_amount`, bar = `progress_percent` (clamp 0-100 untuk bar, tampilkan angka asli), badge `current_target.status`. `null` → "Belum mulai" |
| Progress goal | `goal.progress_percent`, `goal.status`. Sembunyikan kalau `goal = null` |
| Info kecil | `win_rate`, `open_trades`, `runners_joined`, `last_trade_at` (relatif: "5 menit lalu") |

### Kosong

- `journals = []`: "Belum ada jurnal. Buat jurnal pertama untuk mulai mencatat trade." + tombol **Buat Jurnal**.
- `recent_trades = []`: "Belum ada aktivitas trade."

### Mobile

KPI jadi carousel horizontal (2 kartu terlihat), kartu jurnal 1 kolom, aktivitas & runner jadi tab.

---

## 4. Model 2: Journal Performance

**Tujuan:** evaluasi kesehatan jurnal seperti laporan trader profesional.

**Endpoint:** `GET /api/journals/:id/dashboard/summary?from&to` dan `GET /api/journals/:id/dashboard/equity?interval&from&to`.

### Layout

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Rentang: [ Semua ▾ ]  7H  30H  90H  YTD  Kustom…                             │
├──────────────┬──────────────┬──────────────┬──────────────┬──────────────────┤
│ Balance      │ Net PnL      │ Win Rate     │ Profit Factor│ Max Drawdown     │
│ $1,585.00    │ +$85.00      │ 66.67%       │ 5.25         │ −$15.00          │
│ +$500 deposit│ +5.67% ROI   │ 4W · 2L · 0BE│ payoff 2.63  │ 1.42% · now 0%   │
├──────────────┴──────────────┴──────────────┴──────────────┴──────────────────┤
│ Equity Curve                              [ Balance | PnL Kumulatif ] [D W M] │
│  1600 ┤                                   ╭────────────────────────          │
│  1400 ┤                                   │  ← deposit +$500 (marker)        │
│  1000 ┼───╮╭──╮──╯                                                            │
│       └──────────────────────────────────────────────────────────────        │
│ Drawdown (area merah di bawah 0)                                              │
├──────────────────────────────────────┬───────────────────────────────────────┤
│ Statistik Trade                      │ Posisi Open                           │
│ Total trade        6                 │ 1 posisi · 1 long · 0 short            │
│ Expectancy         +$14.17 / trade   │ Margin $50.00 · Size $500.00          │
│ Rata-rata menang   +$26.25           │ → lihat di tab Trades                  │
│ Rata-rata kalah    −$10.00           ├───────────────────────────────────────┤
│ Terbesar menang    +$40.00           │ Streak                                │
│ Terbesar kalah     −$15.00           │ Saat ini: 1 menang                    │
│ Rata-rata ROI      +10.00%           │ Terpanjang: 2 menang · 1 kalah        │
│ Rata-rata durasi   2j 10m            ├───────────────────────────────────────┤
│ Total fee          $0.00             │ Trade Terbaik / Terburuk              │
│                                      │ ✓ BTCUSDT short +$40.00 (+20%)        │
│                                      │ ✕ ETHUSDT short −$15.00 (−15%)        │
└──────────────────────────────────────┴───────────────────────────────────────┘
```

### Widget

| Widget | Komponen | Field (`summary` kecuali disebut) |
|--------|----------|-----------------------------------|
| Balance | KPI | `balance.current`, subtext `balance.net_flow` ("+$500 deposit") |
| Net PnL | KPI | `performance.net_pnl`, `performance.roi_percent` |
| Win Rate | KPI + donut kecil | `trades.win_rate`, `trades.wins` / `losses` / `breakeven` |
| Profit Factor | KPI | `trades.profit_factor` (`null` → `∞` kalau ada win, `—` kalau tidak ada trade), subtext `trades.payoff_ratio` |
| Max Drawdown | KPI | `drawdown.max` (tampilkan sebagai negatif), `drawdown.max_percent`, `drawdown.current_percent` |
| Equity curve | Line / area chart | `equity.points[]`: x = `date`, y = `balance` atau `cumulative_pnl` (toggle). Marker di titik `net_flow ≠ 0` |
| Drawdown chart | Area chart (di bawah equity, sumbu x sama) | `equity.points[].drawdown_percent` (gambar sebagai nilai negatif) |
| Bar PnL per bucket (opsional) | Bar chart hijau/merah | `equity.points[].pnl` |
| Statistik trade | Tabel 2 kolom | `trades.*` |
| Posisi open | Card | `open_positions.*` |
| Streak | Card | `streaks.*` (`current_type` `none` → "Belum ada") |
| Trade terbaik / terburuk | Card | `best_trade`, `worst_trade` (klik → detail trade) |

### Interaksi

- **Filter rentang** (`from` / `to`) mempengaruhi: Net PnL, Win Rate, Profit Factor, statistik trade, streak, trade terbaik/terburuk, **dan** equity chart. **Tidak** mempengaruhi Balance, Drawdown (KPI), Posisi Open. Beri label kecil "sepanjang waktu" pada KPI yang tidak terfilter.
- **Interval equity** default otomatis: rentang ≤ 60 hari → `day`, ≤ 1 tahun → `week`, lebih → `month`. User boleh ganti.
- Rentang terlalu panjang untuk interval kecil → API `1002` pada field `interval`: otomatis turunkan ke interval lebih besar.

### Kosong

`trades.total_trades = 0`: semua KPI tampil `$0.00` / `—`, chart diganti "Belum ada trade yang ditutup. Data performa muncul setelah trade pertama ditutup."

---

## 5. Model 3: Target Tracker

**Tujuan:** memantau target per periode (daily / weekly / monthly) dan konsistensi.

**Endpoint:** `summary` (bagian `target`, `goal`), `GET /periods`, `GET /calendar` (untuk jurnal daily).

### Layout

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ ┌───────────────────────────────┐ ┌────────────┬────────────┬──────────────┐ │
│ │ Target Hari Ini · Sen 20 Okt  │ │ Achievement│ Streak     │ Goal         │ │
│ │                               │ │ 60%        │ 1          │ 79.25%       │ │
│ │        ◔  0%                  │ │ 3 / 5      │ terbaik 1  │ $1,585/$2,000│ │
│ │   $0.00 / $31.70              │ └────────────┴────────────┴──────────────┘ │
│ │   sisa $31.70  ◷ berjalan     │ ┌──────────────────────────────────────┐  │
│ │   2% × balance awal $1,585    │ │ 3 tercapai · 2 gagal · 14 tidak trade │  │
│ └───────────────────────────────┘ └──────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────────────────────────┤
│ Kalender Oktober 2026                                   ‹ Sep    Nov ›      │
│  Sen   Sel   Rab   Kam   Jum   Sab   Min                                     │
│                    1     2     3     4                                       │
│                   +20✓  +10✕   –    +30✓                                     │
│   5     6     7  ...                                                         │
│  −15✕  +40✓   –                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│ Riwayat Periode                              [ Semua ▾ achieved missed … ]  │
│ Periode        Target    PnL       Progress           Status      Trade      │
│ 20 Okt         $31.70    $0.00     ░░░░░░░░  0%       ◷ Berjalan  0          │
│ 06 Okt         $30.90    +$40.00   ▓▓▓▓▓▓▓▓ 129%      ✓ Tercapai  1          │
│ 05 Okt         $21.20    −$15.00   ░░░░░░░░ −71%      ✕ Gagal     1  +$500 ↑ │
│ ...                                                    ‹ 1 2 3 ›             │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Widget

| Widget | Komponen | Field |
|--------|----------|-------|
| Target periode berjalan | Progress ring besar | `summary.target.current_period`: `pnl` / `target_amount`, `progress_percent`, `remaining`, `status`, `period_start`–`period_end`. Subtext aturan: `target_type` `percent` → "`target_value`% × balance awal `opening_balance`", `amount` → "target tetap" |
| Achievement rate | KPI | `summary.target.stats.achievement_rate`, `achieved` / (`achieved` + `missed`) |
| Streak | KPI | `stats.current_streak`, `stats.best_streak` |
| Goal | KPI + bar | `summary.goal` (sembunyikan kalau `null`). `status: achieved` → badge "Goal tercapai" + `achieved_at` |
| Distribusi status | Stacked bar horizontal | `stats.achieved`, `missed`, `no_trade`, `in_progress` |
| Kalender (daily) | Heatmap bulan | `calendar.days[]`: warna = tanda `pnl` (intensitas ∝ besar pnl), ikon = `target_status`. Tooltip: pnl, trades, wins/losses, target_amount |
| Kalender total bulan | Baris ringkas | `calendar.totals.*` |
| Riwayat periode | Tabel dengan pagination | `periods.items[]`, filter `status` |
| Bar target vs pnl (opsional) | Bar chart berpasangan | `periods.items[]`: `target_amount` (garis/ghost bar) vs `pnl` |

### Catatan

- **Progress bar**: clamp 0-100% untuk lebar bar, tapi tampilkan angka asli (`129%`, `−71%`).
- Untuk jurnal **weekly / monthly**, kalender tetap bisa dipakai (PnL harian), tapi `target_status` per hari `null`. Tampilkan target di level periode (tabel riwayat) saja.
- Baris periode dengan `net_flow ≠ 0`: beri ikon kecil "deposit/withdraw" karena mempengaruhi target periode berikutnya.
- `current_period = null` (jurnal belum mulai): ganti ring dengan "Jurnal mulai {start_date}".

---

## 6. Model 4: Analytics

**Tujuan:** menemukan apa yang paling menghasilkan (dan merugikan).

**Endpoint:** `GET /api/journals/:id/dashboard/breakdown?by=...&from&to`, `calendar`.

### Layout

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Rentang: [ 30 hari ▾ ]                                                        │
├────────────────────────────────────┬─────────────────────────────────────────┤
│ PnL per Pair            (bar)      │ Long vs Short          (2 kartu)         │
│ BTCUSDT  ▓▓▓▓▓▓▓▓▓▓▓▓  +$75.00     │ Long   3 trade  +$65.00  WR 100%         │
│ SOLUSDT  ▓▓▓▓▓        +$30.00     │ Short  3 trade  +$20.00  WR 33.33%       │
│ ETHUSDT  ▒▒▒          −$20.00     ├─────────────────────────────────────────┤
│                                    │ Alasan Close           (donut)          │
│                                    │ take_profit 2 · stop_loss 2 · …         │
├────────────────────────────────────┼─────────────────────────────────────────┤
│ PnL per Hari (Sen–Min)  (bar)      │ PnL per Jam (00–23)    (bar / heat)     │
├────────────────────────────────────┴─────────────────────────────────────────┤
│ Performa per Runner                                                           │
│ Runner            Trade  WR       Net PnL   Avg PnL  Profit Factor           │
│ Bot B             1      100%     +$40.00   +$40.00  ∞                        │
│ Manual (pemilik)  2      50%      +$25.00   +$12.50  7.00                     │
│ Bot A             3      66.67%   +$20.00   +$6.67   3.33                     │
├──────────────────────────────────────────────────────────────────────────────┤
│ Per Exchange (tabel kecil) · Manual vs Runner (2 kartu)                      │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Widget

| Widget | `by` | Komponen | Catatan |
|--------|------|----------|---------|
| PnL per pair | `pair` | Bar horizontal (hijau/merah) | Tampilkan top 10, sisanya "Lainnya" |
| Long vs Short | `position` | 2 kartu berdampingan | |
| Alasan close | `close_reason` | Donut (jumlah `trades`) + legend pnl | `liquidation` beri warna loss tegas |
| Per hari | `weekday` | Bar vertikal, selalu 7 item | Label `monday` → "Sen" |
| Per jam | `hour` | Bar vertikal / heat strip, selalu 24 item | Jam di timezone jurnal |
| Per runner | `runner` | Tabel, bisa sort | `key = "manual"` → label "Manual (pemilik)" |
| Per exchange | `exchange` | Tabel kecil | `unknown` → "Tidak diisi" |
| Manual vs runner | `source` | 2 kartu | |

Setiap item breakdown punya: `trades`, `wins`, `losses`, `breakeven`, `win_rate`, `net_pnl`, `avg_pnl`, `gross_profit`, `gross_loss`, `profit_factor`, `total_fee`.

- Muat breakdown **secara paralel** (`Promise.all` / beberapa query) karena tiap `by` = 1 request.
- Grup dengan `trades = 0` (weekday / hour): tampilkan bar kosong, jangan disembunyikan.
- Weekday / hour dihitung dari **waktu buka** trade (`opened_at`), sebutkan di tooltip.

---

## 7. Model 5: Runner Monitor

**Tujuan:** memastikan agent berjalan dan membandingkan performanya.

**Endpoint:** `GET /api/dashboard/runners` (polling 15 detik).

### Layout

```
┌──────────────────────────────────────────────────────────────────────────────┐
│ Runner Monitor                     ● 2 online  ● 0 idle  ○ 1 belum dipakai   │
├──────────────────────────────────────────────────────────────────────────────┤
│ Runner     Status         Terakhir aktif   Jurnal  Open  Trade 24j  WR    PnL │
│ ● Bot B    Online         1 menit lalu     1       0     0          100%  +$40│
│ ● Bot A    Online         1 menit lalu     1       1     1          66.67% +$20│
│ ○ Bot C    Belum dipakai  —                0       0     0          —     $0  │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Widget

| Kolom | Field | Catatan |
|-------|-------|---------|
| Status | `health` | Dot + label (tabel di 2.4) |
| Terakhir aktif | `last_used_at` / `idle_seconds` | Relatif ("1 menit lalu"). Hitung ulang tiap detik di FE dari `last_used_at` supaya terasa live |
| Jurnal | `journals_joined` | |
| Open | `open_trades` | |
| Trade 24 jam | `trades_last_24h` | |
| Win rate | `win_rate` | `closed_trades = 0` → `—` |
| PnL | `net_pnl` | |
| Expired | `expires_at` | Kalau `health = expired`, tampilkan banner "Token runner expired, perpanjang di pengaturan runner" |

Ringkasan di header: hitung jumlah per `health` dari array.

Aksi per baris: buka detail runner (manajemen di [runners.md](runners.md)), aktifkan / nonaktifkan, regenerate token.

**Alert (opsional):** kalau runner yang sebelumnya `online` berubah jadi `offline` saat polling, tampilkan toast "Bot A berhenti mengirim data".

---

## 8. Komponen Reusable

| Komponen | Props utama | Dipakai di |
|----------|-------------|------------|
| `KpiCard` | `label`, `value`, `sub`, `tone` (profit/loss/neutral), `hint` (tooltip definisi) | Semua model |
| `PnlText` | `value: string`, `size` | Semua |
| `ProgressBar` | `percent: string`, `status` | Overview, Target |
| `ProgressRing` | `percent`, `status`, `center` | Target |
| `StatusBadge` | `status: PeriodStatus` | Target, Overview |
| `HealthDot` | `health: RunnerHealth` | Overview, Runner Monitor |
| `EquityChart` | `points`, `mode` (balance/cumulative), `showDrawdown` | Performance |
| `CalendarHeatmap` | `days`, `month`, `onMonthChange` | Target, Analytics |
| `BreakdownBar` | `items`, `metric` (`net_pnl` / `trades` / `win_rate`) | Analytics |
| `DateRangePicker` | preset 7H/30H/90H/YTD/Semua/Kustom, output `from` / `to` `YYYY-MM-DD` | Performance, Analytics |

Tooltip definisi (`hint`) pada KPI sangat dianjurkan, isi dari tabel definisi di [dashboard.md](dashboard.md#definisi), misal Profit Factor: "Total profit ÷ total rugi. Di atas 1 berarti menguntungkan."

Library chart yang cocok: **Recharts** atau **ECharts** (equity, bar, donut, heatmap). Untuk tampilan ala TradingView bisa **lightweight-charts** untuk equity curve.

---

## 9. Tipe TypeScript

```ts
// ===== umum =====
export type Decimal = string; // angka dari API selalu string
export interface ApiResponse<T> { code: number; message: string; data: T; meta?: PageMeta }
export interface PageMeta { page: number; limit: number; total: number; total_page: number }

export type PeriodType = "daily" | "weekly" | "monthly";
export type PeriodStatus = "achieved" | "missed" | "no_trade" | "in_progress";
export type RunnerHealth = "online" | "idle" | "offline" | "never_used" | "expired" | "inactive";

export interface Goal {
  goal_type: "balance" | "profit" | "percent";
  goal_value: Decimal;
  current_value: Decimal;
  progress_percent: Decimal;
  remaining: Decimal;
  status: "in_progress" | "achieved";
  achieved_at: string | null;
}

// ===== overview =====
export interface Overview {
  totals: {
    journals_active: number; journals_archived: number;
    total_balance: Decimal; total_initial: Decimal; net_pnl: Decimal; today_pnl: Decimal;
    open_trades: number; total_position_size: Decimal;
    targets_achieved: number; targets_in_progress: number;
    runners_total: number; runners_active: number;
  };
  journals: OverviewJournal[];
  recent_trades: RecentTrade[];
}
export interface OverviewJournal {
  id: string; name: string; status: "active" | "archived"; period_type: PeriodType;
  balance: Decimal; net_pnl: Decimal; roi_percent: Decimal; today_pnl: Decimal; win_rate: Decimal;
  open_trades: number; runners_joined: number;
  current_target: {
    target_amount: Decimal; pnl: Decimal; progress_percent: Decimal;
    status: PeriodStatus; period_start: string; period_end: string;
  } | null;
  goal: { goal_type: Goal["goal_type"]; progress_percent: Decimal; status: Goal["status"] } | null;
  last_trade_at: string | null;
}
export interface RecentTrade {
  id: string; journal_id: string; journal_name: string;
  runner_id: string | null; runner_name: string | null;
  pair: string; position: "long" | "short"; status: "open" | "closed";
  pnl: Decimal | null; opened_at: string; closed_at: string | null; updated_at: string;
}

// ===== runner monitor =====
export interface RunnerMonitorItem {
  id: string; name: string; is_active: boolean; expires_at: string | null;
  last_used_at: string | null; idle_seconds: number | null; health: RunnerHealth;
  journals_joined: number; open_trades: number; closed_trades: number;
  wins: number; losses: number; win_rate: Decimal; net_pnl: Decimal; trades_last_24h: number;
}

// ===== summary =====
export interface TradeStats {
  total_trades: number; wins: number; losses: number; breakeven: number;
  win_rate: Decimal; net_pnl: Decimal; gross_profit: Decimal; gross_loss: Decimal; total_fee: Decimal;
  profit_factor: Decimal | null; payoff_ratio: Decimal | null;
  avg_pnl: Decimal; avg_win: Decimal; avg_loss: Decimal;
  largest_win: Decimal; largest_loss: Decimal; avg_roi: Decimal; avg_holding_seconds: number;
}
export interface TradeBrief {
  id: string; runner_id: string | null; pair: string; position: "long" | "short";
  pnl: Decimal; roi: Decimal; opened_at: string; closed_at: string;
}
export interface PeriodItem {
  period_start: string; period_end: string; is_current: boolean; status: PeriodStatus;
  target_type: "amount" | "percent"; target_value: Decimal; target_amount: Decimal;
  pnl: Decimal; progress_percent: Decimal; remaining: Decimal;
  opening_balance: Decimal; closing_balance: Decimal; net_flow: Decimal;
  trades: number; wins: number; losses: number;
}
export interface PeriodSummary {
  total: number; achieved: number; missed: number; no_trade: number; in_progress: number;
  achievement_rate: Decimal; current_streak: number; best_streak: number;
}
export interface PeriodBrief { period_start: string; period_end: string; pnl: Decimal; status: PeriodStatus }
export interface DashboardSummary {
  range: { from: string | null; to: string | null };
  balance: { initial: Decimal; current: Decimal; deposits: Decimal; withdrawals: Decimal; net_flow: Decimal };
  performance: { net_pnl: Decimal; roi_percent: Decimal; total_fee: Decimal };
  trades: TradeStats;
  streaks: { current_type: "win" | "loss" | "none"; current_count: number; max_wins: number; max_losses: number };
  best_trade: TradeBrief | null;
  worst_trade: TradeBrief | null;
  drawdown: { max: Decimal; max_percent: Decimal; current: Decimal; current_percent: Decimal };
  open_positions: { count: number; long: number; short: number; total_margin: Decimal; total_position_size: Decimal };
  target: {
    period_type: PeriodType;
    current_period: PeriodItem | null;
    stats: PeriodSummary;
    best_period: PeriodBrief | null;
    worst_period: PeriodBrief | null;
  };
  goal: Goal | null;
}

// ===== periods =====
export interface PeriodsResponse { period_type: PeriodType; summary: PeriodSummary; items: PeriodItem[] }

// ===== equity =====
export type EquityInterval = "day" | "week" | "month";
export interface EquityPoint {
  date: string; pnl: Decimal; net_flow: Decimal; balance: Decimal; cumulative_pnl: Decimal;
  drawdown: Decimal; drawdown_percent: Decimal; trades: number; wins: number; losses: number;
}
export interface EquityResponse { interval: EquityInterval; from: string; to: string; points: EquityPoint[] }

// ===== calendar =====
export interface CalendarDay {
  date: string; pnl: Decimal; trades: number; wins: number; losses: number;
  target_amount: Decimal | null; target_status: PeriodStatus | null;
}
export interface CalendarResponse {
  month: string;
  totals: {
    pnl: Decimal; trades: number; wins: number; losses: number;
    profitable_days: number; losing_days: number; best_day: string | null; worst_day: string | null;
  };
  days: CalendarDay[];
}

// ===== breakdown =====
export type BreakdownBy = "pair" | "position" | "close_reason" | "source" | "runner" | "exchange" | "weekday" | "hour";
export interface BreakdownItem {
  key: string; label: string; trades: number; wins: number; losses: number; breakeven: number;
  win_rate: Decimal; net_pnl: Decimal; avg_pnl: Decimal; gross_profit: Decimal; gross_loss: Decimal;
  profit_factor: Decimal | null; total_fee: Decimal;
}
export interface BreakdownResponse { by: BreakdownBy; range: { from: string | null; to: string | null }; items: BreakdownItem[] }
```

### Contoh API client

```ts
export class ApiError extends Error {
  constructor(public status: number, public code: number, message: string, public data: unknown) { super(message); }
}

export async function api<T>(path: string, init?: RequestInit): Promise<ApiResponse<T>> {
  const res = await fetch(`${import.meta.env.VITE_API_URL}${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${getSessionToken()}`, ...init?.headers },
  });
  const body = await res.json();
  if (body.code !== 0) throw new ApiError(res.status, body.code, body.message, body.data);
  return body;
}

// contoh hooks (TanStack Query)
export const useOverview = () =>
  useQuery({ queryKey: ["dashboard", "overview"], queryFn: () => api<Overview>("/dashboard/overview"), refetchInterval: 30_000 });

export const useSummary = (id: string, from?: string, to?: string) =>
  useQuery({
    queryKey: ["journal", id, "summary", from, to],
    queryFn: () => api<DashboardSummary>(`/journals/${id}/dashboard/summary?${new URLSearchParams(clean({ from, to }))}`),
    refetchInterval: 30_000,
  });
```

---

## 10. Daftar Tugas (siap dibagi)

Urutan disarankan dari atas ke bawah. Setiap tugas bisa dikerjakan terpisah setelah fondasi (T1) selesai.

| ID | Tugas | Isi | Bergantung |
|----|-------|-----|------------|
| T1 | Fondasi | API client + error handling, helper format angka (2.3), design tokens & status (2.4), komponen `KpiCard`, `PnlText`, `StatusBadge`, `HealthDot`, `ProgressBar`, skeleton | - |
| T2 | Overview | Model 1 lengkap + polling + empty state | T1 |
| T3 | Runner Monitor | Model 5 + idle time live + alert offline | T1 |
| T4 | Performance: KPI & statistik | KPI row, statistik trade, streak, best/worst trade, posisi open, `DateRangePicker` | T1 |
| T5 | Performance: chart | `EquityChart` (balance / cumulative toggle, marker deposit) + drawdown area + auto interval | T4 |
| T6 | Target Tracker: header | Progress ring periode berjalan, achievement, streak, goal, distribusi status | T1 |
| T7 | Target Tracker: riwayat | Tabel periods + filter status + pagination | T6 |
| T8 | Kalender heatmap | `CalendarHeatmap` + navigasi bulan, dipakai di Target & Analytics | T1 |
| T9 | Analytics | Semua widget breakdown (paralel) + filter rentang | T1 |
| T10 | Integrasi | Tab jurnal, invalidate query setelah aksi trade/deposit, responsif mobile, aksesibilitas | T2-T9 |

### Kriteria selesai (acceptance)

- [ ] Semua angka uang memakai helper format, tidak ada `NaN`, `undefined`, atau angka tanpa format.
- [ ] PnL selalu punya tanda `+` / `−` **dan** warna.
- [ ] Setiap widget punya state loading (skeleton), kosong, dan error.
- [ ] Waktu & tanggal ditampilkan di timezone jurnal.
- [ ] Polling berhenti saat tab tidak aktif.
- [ ] Filter rentang mengubah URL (`?from=&to=`) supaya bisa di-share / refresh.
- [ ] Data cocok dengan contoh di [dashboard.md](dashboard.md) untuk data uji yang sama.
- [ ] Tampilan rapi di lebar 360px (mobile) sampai 1440px.
- [ ] Kontras warna lulus WCAG AA, status tidak hanya dibedakan warna.

---

## 11. FAQ

**Kenapa win rate 66.67% padahal ada 6 trade dan 4 menang?**
Win rate = 4 / (4 menang + 2 kalah). Kalau ada breakeven, tidak dihitung.

**Kenapa balance naik tapi net PnL tidak?**
Ada deposit. Balance = modal + deposit − withdraw + PnL. Lihat `balance.net_flow`.

**Kenapa target hari ini tidak ikut naik padahal sudah profit / deposit?**
Target persen dihitung dari balance **di awal** periode. Profit / deposit hari ini baru mempengaruhi target besok.

**Kenapa ada hari "tidak trading" tapi streak tetap?**
`no_trade` tidak memutus streak, hanya `missed` yang mereset.

**Kenapa titik pertama equity weekly tanggalnya sebelum start jurnal?**
Bucket weekly selalu mulai hari Senin. Titik pertama = Senin minggu tempat jurnal mulai.

**Drawdown kok tidak turun saat withdraw?**
Drawdown dihitung dari PnL kumulatif, jadi withdraw / deposit tidak dianggap rugi / profit.
