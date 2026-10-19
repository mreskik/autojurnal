# Frontend Brief — Autojurnal

Brief teknologi dan arah desain untuk aplikasi web Autojurnal (trading journal + monitoring agent). Brief ini akan dikerjakan oleh **AI coding agent (Antigravity)**. Bagian 0 berisi aturan kerja agent dan **wajib dibaca dulu**.

**Arah desain dalam satu kalimat:** terminal trading yang tenang. Gelap, padat data, rapi, angka jadi pusat perhatian. Modern dan minimalis, tidak ramai, tidak "template AI".

### Lokasi file

Buka workspace di folder **`autojurnal/`** (induk `backend/` dan `frontend/`) supaya semua dokumen terbaca.

```
autojurnal/
  backend/
    FRONTEND_BRIEF.md            ← dokumen ini
    DOKUMENTASI API/             ← referensi API
  frontend/                      ← kode frontend dibuat di sini
    design-references/           ← gambar referensi desain (ref-1 … ref-4)
```

---

## 0. Aturan Kerja untuk AI Agent

### 0.1 Urutan baca sebelum menulis kode

1. Dokumen ini, seluruhnya.
2. Gambar di `frontend/design-references/` (keempatnya), cocokkan dengan tabel di bagian 2.
3. `backend/DOKUMENTASI API/README.md`: base URL, autentikasi, format response.
4. `backend/DOKUMENTASI API/frontend-dashboard-guide.md`: halaman, widget, tipe TypeScript, daftar tugas.
5. Dokumen endpoint sesuai halaman yang sedang dikerjakan (`auth.md`, `journals.md`, `trades.md`, `runners.md`, `dashboard.md`) dan `error-codes.md`.

### 0.2 Batasan keras

- **Jangan mengarang endpoint, field, atau nilai enum.** Semua harus ada di `DOKUMENTASI API/`. Kalau data yang dibutuhkan tidak ada, buat catatan `TODO(api): ...` di kode dan laporkan, jangan membuat mock permanen.
- **Jangan memanggil `/api/runner/*`** dan jangan membaca `runner-guide.md`. Itu untuk agent trading, bukan aplikasi.
- **Jangan menambah library** di luar tabel bagian 1 tanpa alasan tertulis di laporan.
- **Jangan menulis hex warna atau ukuran font baru** di komponen. Semua lewat token bagian 3.
- **Jangan menghitung uang dengan float.** Format pakai helper `lib/format.ts`, hitung pakai decimal.js.
- **Patuhi bagian 5 (Anti AI Slop)** di setiap halaman.
- Base URL API dari env `VITE_API_URL` (contoh `http://localhost:1000/api`). Token session disimpan setelah `POST /auth/login`.
- Jalankan dev server Vite di port default **5173**: origin itu sudah diizinkan CORS oleh backend. Kalau port / domain lain, origin harus ditambahkan di `CORS_ALLOWED_ORIGINS` backend (lihat `README.md`). Jangan mengakali CORS dengan proxy aneh atau mematikan keamanan browser; `server.proxy` bawaan Vite boleh dipakai.

### 0.3 Cara kerja

Kerjakan berurutan per tugas di `frontend-dashboard-guide.md` §10 (T1 → T10), didahului **T0**:

| ID | Tugas |
|----|-------|
| T0 | Setup Vite + React + TS + Tailwind + shadcn, pasang token bagian 3, font Geist, layout `AppShell` (topbar, sidebar, status bar), halaman login (`auth.md`), API client + `ApiError` + helper format |
| T1 - T10 | Sesuai `frontend-dashboard-guide.md` §10 |
| T11 | Halaman manajemen di luar dashboard: daftar & form jurnal, trades (list, buka, ubah, tutup), deposit/withdraw, runner di jurnal (dropdown options, rules/strategy, download AGENT.md), manajemen runner, users |

Setiap selesai satu tugas:

1. Jalankan `npm run build` dan `npm run lint` tanpa error.
2. Cek tampilan dengan backend lokal (data asli), di lebar 1440px dan 390px.
3. Jalankan checklist bagian 5 dan §10 "Kriteria selesai" di `frontend-dashboard-guide.md`.
4. Tulis laporan singkat: apa yang dibuat, file utama, hal yang belum / `TODO(api)`, keputusan desain yang diambil sendiri.

Kalau ada hal yang tidak diatur di dokumen mana pun, pilih opsi **paling sederhana dan paling mirip referensi 1**, lalu catat di laporan.

---

Dokumen terkait (di folder `backend/DOKUMENTASI API/`):

| Dokumen | Isi |
|---------|-----|
| `README.md` | Base URL, autentikasi, format response, rate limit |
| `frontend-dashboard-guide.md` | 5 halaman dashboard: wireframe, widget → field API, tipe TypeScript, daftar tugas |
| `dashboard.md`, `journals.md`, `trades.md`, `runners.md`, `auth.md` | Referensi endpoint |
| `error-codes.md` | Semua kode error |

> Frontend **tidak** memakai `runner-guide.md` dan endpoint `/api/runner/*`. Itu khusus agent.

---

## 1. Teknologi

### Wajib

| Kebutuhan | Pakai | Catatan |
|-----------|-------|---------|
| Framework | **React** + **TypeScript** + **Vite** | SPA. TypeScript strict |
| UI components | **shadcn/ui** | Basis semua komponen (Button, Table, Tabs, Dialog, Sheet, Select, Popover, Tooltip, Badge, Skeleton, Sonner, Chart). Komponen di-copy ke repo, boleh dimodifikasi |
| Styling | **Tailwind CSS** | Semua warna lewat CSS variables / token (bagian 3). Tidak ada hex langsung di komponen |
| Ikon | **lucide-react** | Bawaan shadcn. Stroke 1.5, ukuran 14-16px |

### Disarankan (supaya tim tidak memilih sendiri-sendiri)

| Kebutuhan | Pakai | Alasan |
|-----------|-------|--------|
| Data fetching & cache | **TanStack Query** | Polling, invalidate, loading/error state konsisten |
| Routing | **React Router** | |
| Tabel | **TanStack Table** + `Table` shadcn | Sort, kolom, virtualisasi kalau trade banyak |
| Chart | **shadcn Chart** (Recharts) | Equity, bar, donut. Sudah menyatu dengan token tema |
| Chart equity ala terminal (opsional) | **lightweight-charts** (TradingView) | Kalau butuh zoom/crosshair halus untuk equity curve |
| Form & validasi | **react-hook-form** + **zod** | Pesan error API (`data.<field>`) dipetakan ke field form |
| Angka decimal | **decimal.js** atau **big.js** | API mengirim uang sebagai string. Jangan `parseFloat` untuk hitungan |
| Tanggal & timezone | **date-fns** + **date-fns-tz** | Semua tanggal ditampilkan di timezone **jurnal**, bukan browser |
| Toast | **Sonner** (bawaan shadcn) | |
| Animasi aksen | **React Bits** (terbatas, lihat 6) | |

### Struktur folder (saran)

```
src/
  app/                 router, providers (QueryClient, Theme, Toaster)
  components/
    ui/                komponen shadcn (hasil generate, boleh diubah)
    data/              PnlText, Money, Percent, StatusBadge, HealthDot, KpiCard, ProgressBar, Sparkline
    charts/            EquityChart, PnlBarChart, BreakdownBar, CalendarHeatmap, DonutChart
    layout/            AppShell, Sidebar, Topbar, PageHeader, Section
  features/
    auth/  journals/  trades/  runners/  dashboard/
      api.ts           fungsi fetch + hooks TanStack Query
      types.ts         tipe response (lihat frontend-dashboard-guide.md §9)
      components/      komponen khusus fitur
      pages/
  lib/
    api.ts             fetch wrapper + ApiError
    format.ts          fmtMoney, fmtPnl, fmtPct, fmtDuration, fmtDateInTz
    query-keys.ts
  styles/globals.css   token tema
```

---

## 2. Referensi Visual

Gambar ada di `frontend/design-references/`. Referensi dipakai untuk **gaya**, bukan untuk ditiru 1:1 (konten, logo, dan menu milik produk lain jangan disalin).

| Ref | File | Sumber | Ambil | Jangan ambil |
|-----|------|--------|-------|--------------|
| 1 | `ref-1-hyperdash-wallet.png` | Hyperdash, halaman wallet | Baris 4 kartu KPI (judul kecil, angka besar, bar bersegmen, keterangan 1 baris). Area chart dengan **garis nol**: hijau di atas, merah di bawah. Tab uppercase mono dengan garis bawah. Tabel posisi yang rapat. Panel kiri berisi ringkasan akun berupa daftar label-nilai | Badge "copytrading score" merah besar |
| 2 | `ref-2-hyperdash-explore.png` | Hyperdash, explore | Sidebar dengan judul + deskripsi kecil per menu. Kartu grid berisi sparkline + bar long/short di bawahnya. Badge status kecil (Bullish ↗) | |
| 3 | `ref-3-bybit-trading.png` | Bybit, trading | Kepadatan informasi. Baris "ticker" horizontal (label kecil abu di atas, nilai di bawah). Tabel order book dengan bar latar tipis sebagai indikator besaran | Banner promosi, oranye berlebihan, popup |
| 4 | `ref-4-finwise-dashboard.png` | FinWise | Kartu target dengan progress bar + "x / y" + persen + status ("On track"). Kejelasan hierarki KPI | Radius sangat besar, warna lime di mana-mana, bar chart tebal bergradasi, emoji di judul |

Inti yang dicari: **kepadatan dan presisi Hyperdash/Bybit**, dengan **kejelasan kartu target FinWise**.

---

## 3. Design System

### 3.1 Prinsip

1. **Angka adalah konten utama.** Desain mendukung keterbacaan angka, bukan dekorasi.
2. **Warna punya arti.** Hijau = profit/naik, merah = rugi/turun, biru = berjalan, kuning = peringatan. Di luar itu, UI netral (abu).
3. **Satu warna aksen** untuk aksi utama dan elemen aktif. Jangan dipakai untuk dekorasi.
4. **Garis, bukan bayangan.** Pemisah memakai border 1px, bukan shadow.
5. **Padat tapi bernapas.** Baris tabel rapat, tapi jarak antar section konsisten.
6. **Dark first.** Mode gelap adalah mode utama. Mode terang menyusul memakai token yang sama.

### 3.2 Warna (token)

Definisikan sebagai CSS variables dan petakan ke tema shadcn (`--background`, `--card`, `--border`, dst).

| Token | Hex | Pemakaian |
|-------|-----|-----------|
| `--bg` | `#0A0B0D` | Latar aplikasi |
| `--surface` | `#111316` | Kartu, panel, sidebar |
| `--surface-2` | `#171A1E` | Hover baris, input, area dalam kartu |
| `--surface-3` | `#1E2227` | Tombol sekunder, track progress bar |
| `--border` | `#1F2329` | Border kartu & pemisah |
| `--border-strong` | `#2B3038` | Border input, divider tebal |
| `--text` | `#E7E9EC` | Teks utama, angka |
| `--text-muted` | `#8B929B` | Label, keterangan |
| `--text-subtle` | `#5C636C` | Placeholder, sumbu chart, teks non-aktif |
| `--accent` | `#FF7A2F` | Tombol utama, tab aktif, focus ring, link. **Hemat** |
| `--accent-fg` | `#0A0B0D` | Teks di atas aksen |
| `--profit` | `#2BD39A` | PnL positif, achieved, long |
| `--profit-bg` | `rgba(43,211,154,.12)` | Latar badge/area profit |
| `--loss` | `#F4506A` | PnL negatif, missed, short |
| `--loss-bg` | `rgba(244,80,106,.12)` | Latar badge/area rugi |
| `--info` | `#4C8DFF` | in_progress, berjalan |
| `--warning` | `#F2B23A` | idle, mendekati batas |
| `--neutral` | `#6B7280` | 0, breakeven, no_trade |

Aturan:

- Teks di atas `--surface` minimal `--text-muted`. `--text-subtle` hanya untuk elemen sekunder (kontras rendah).
- Area chart: isi warna `profit`/`loss` dengan opacity 10-20%, garis 1.5px warna penuh.
- **Tidak ada gradien** di UI, kecuali isi area chart yang memudar ke bawah.
- Jangan pakai warna aksen untuk angka. Angka hanya `--text`, `--profit`, atau `--loss`.

Contoh `globals.css` (Tailwind v4 + shadcn):

```css
@import "tailwindcss";

:root {
  --bg: #0a0b0d;
  --surface: #111316;
  --surface-2: #171a1e;
  --surface-3: #1e2227;
  --border-color: #1f2329;
  --border-strong: #2b3038;
  --text: #e7e9ec;
  --text-muted: #8b929b;
  --text-subtle: #5c636c;
  --accent-color: #ff7a2f;
  --profit: #2bd39a;
  --loss: #f4506a;
  --info: #4c8dff;
  --warning: #f2b23a;

  /* pemetaan ke nama token shadcn */
  --background: var(--bg);
  --foreground: var(--text);
  --card: var(--surface);
  --card-foreground: var(--text);
  --popover: var(--surface);
  --popover-foreground: var(--text);
  --primary: var(--accent-color);
  --primary-foreground: #0a0b0d;
  --secondary: var(--surface-3);
  --secondary-foreground: var(--text);
  --muted: var(--surface-2);
  --muted-foreground: var(--text-muted);
  --accent: var(--surface-2);
  --accent-foreground: var(--text);
  --destructive: var(--loss);
  --border: var(--border-color);
  --input: var(--border-strong);
  --ring: var(--accent-color);
  --radius: 0.375rem;
}

@theme inline {
  --color-profit: var(--profit);
  --color-loss: var(--loss);
  --color-info: var(--info);
  --color-warning: var(--warning);
  --color-surface: var(--surface);
  --color-surface-2: var(--surface-2);
  --color-surface-3: var(--surface-3);
  --color-subtle: var(--text-subtle);
  --font-sans: "Geist", ui-sans-serif, system-ui, sans-serif;
  --font-mono: "Geist Mono", ui-monospace, monospace;
}

body { background: var(--bg); color: var(--text); font-feature-settings: "tnum" 1; }
```

### 3.3 Tipografi

| Font | Pakai untuk |
|------|-------------|
| **Geist** (sans) | Teks UI, angka besar KPI, isi tabel |
| **Geist Mono** | Label kecil uppercase, tab, header kolom, kode/token, sumbu chart |

Semua angka memakai **tabular numerals** (`tabular-nums`) supaya kolom rata.

| Peran | Ukuran / berat | Contoh |
|-------|----------------|--------|
| Angka hero (balance utama) | 28-32px / 500 | `$26,168,736.22` |
| Angka KPI | 20-24px / 500 | `+$85.00` |
| Judul halaman | 18px / 600 | `Scalping BTC` |
| Judul kartu | 13-14px / 500, `--text` | `Performance` |
| Label section | Mono 11px / 500, UPPERCASE, tracking 0.06em, `--text-muted` | `ACCOUNT VALUE` |
| Isi tabel | 13px / 400 | |
| Header tabel | Mono 11px uppercase, `--text-muted` | `ENTRY` |
| Keterangan | 12px / 400, `--text-muted` | `48.1% Win Rate · 27 Trades` |

Batasi ukuran font ke daftar di atas. Jangan menambah ukuran baru per halaman.

### 3.4 Spasi, radius, border

- Skala spasi Tailwind default. Padding kartu `p-4` (16px), gap grid `gap-3` (12px), jarak antar section `gap-6`.
- **Radius kecil**: kartu & input `6px` (`rounded-md`), badge `4px`, tombol `6px`. Tidak memakai `rounded-2xl`/`rounded-3xl`.
- Border kartu 1px `--border`. Tidak memakai `shadow-*` kecuali popover/dialog (`shadow-lg` halus).
- Tinggi baris tabel 36-40px. Tinggi tombol 32px (default), 28px (kecil).

### 3.5 Layout aplikasi

```
┌──────────────────────────────────────────────────────────────────────────┐
│ ▣ AUTOJURNAL   Dashboard  Journals  Runners        [⌘K Search]  🔔  (◯) │  topbar 48px
├────────────┬─────────────────────────────────────────────────────────────┤
│ JOURNALS   │ Journals / Scalping BTC                       [● active] ⟳  │  breadcrumb
│ ▸ Scalping │ ─────────────────────────────────────────────────────────── │
│   BTC      │ PERFORMANCE  TARGETS  ANALYTICS  TRADES  RUNNERS  SETTINGS  │  tab mono
│   +$85.00  │ ┌────────┬────────┬────────┬────────┐                       │
│ ▸ Swing ETH│ │ KPI    │ KPI    │ KPI    │ KPI    │                       │
│            │ └────────┴────────┴────────┴────────┘                       │
│ RUNNERS    │ ┌──────────────────────────────────────┐                    │
│ ● Bot A    │ │ chart                                 │                    │
│ ● Bot B    │ └──────────────────────────────────────┘                    │
├────────────┴─────────────────────────────────────────────────────────────┤
│ ● Connected · API OK                       Last update 12:04:31 (WIB)    │  status bar 28px
└──────────────────────────────────────────────────────────────────────────┘
```

- Lebar konten maksimum tidak dibatasi (terminal memakai seluruh layar). Sidebar 240px, bisa diciutkan ke 56px.
- **Status bar** bawah (seperti Hyperdash/Bybit): status koneksi API, waktu update terakhir, timezone. Memberi kesan terminal yang "hidup".
- Breakpoint: `≥1280` layout penuh, `768-1279` sidebar ciut, `<768` sidebar jadi `Sheet`, KPI jadi scroll horizontal.

---

## 4. Komponen Kunci

Spesifikasi visual. Kebutuhan datanya ada di `frontend-dashboard-guide.md`.

### KPI Card (ref 1)

```
┌───────────────────────────────┐
│ Performance             All ▾ │  judul 13px + kontrol kecil kanan
│                               │
│ +$25,731,761.27  PNL          │  angka 22px, warna sesuai tanda, satuan kecil muted
│ ▮▮▯▮▮▮▯▮▮▮▮▯▮▮▯▮▮▮▮▮▯▮▮▮▯▮▮  │  bar bersegmen / progress 6px
│ 48.1% Win Rate · 27 Trades    │  keterangan 12px, bagian penting diberi warna
└───────────────────────────────┘
```

- Satu angka utama per kartu. Maksimal satu baris keterangan.
- Bar bersegmen untuk win/loss (tiap segmen = 1 trade terakhir, hijau/merah). Progress bar polos untuk target / margin.

### Tabel data (ref 1, 3)

- Header mono uppercase 11px, sticky, bisa sort (ikon ⇅ kecil).
- Angka **rata kanan**, teks rata kiri. Kolom pair diberi ikon aset + badge leverage kecil (`40×`).
- Hover baris: `--surface-2`. Tidak memakai zebra stripe.
- PnL ditulis bertanda + persen dalam kurung: `+$19,551,900.00 (+1254.26%)`.
- Bar latar tipis opsional untuk kolom besaran (seperti order book Bybit), opacity 10%.

### Chart

- **Equity / PnL**: area chart dengan garis nol. Di atas nol `--profit`, di bawah `--loss` (ref 1). Grid horizontal tipis `--border`, tanpa grid vertikal. Sumbu mono 11px `--text-subtle`, angka diringkas (`$4.8M`).
- Watermark logo samar di tengah (opsional, opacity 4%) seperti Hyperdash.
- Kontrol rentang di kanan atas: segmented mono `24H 7D 30D ALL`, aktif = teks `--text` + garis bawah aksen.
- Tooltip: kartu kecil `--surface` border, berisi tanggal + nilai bertanda.
- **Sparkline** di kartu jurnal (ref 2): 1.5px, tanpa sumbu, warna sesuai tanda total.
- Bar chart: batang tipis, sudut 2px, hijau/merah per nilai. Tanpa gradien.

### Badge & status

- Badge: teks 11px, latar `*-bg` (12% opacity), teks warna penuh, radius 4px, opsional ikon panah kecil (`Bullish ↗`, `Tercapai ✓`).
- Dot status runner: 6px. Online berdenyut halus (opacity 1 → 0.4, 2s).

### Tab

Mono uppercase 12px, `--text-muted`. Aktif: `--text` + garis bawah 2px `--accent`. Tidak memakai tab berbentuk pill.

### Progress / target (ref 4, versi terminal)

```
TARGET HARI INI · SEN 20 OKT                ◷ BERJALAN
$18.40 / $31.70                                   58%
▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░░
2% × balance awal $1,585.00 · sisa $13.30
```

Track `--surface-3`, isi `--info` (berjalan) / `--profit` (tercapai) / `--loss` (gagal). Tinggi 6px, radius penuh.

---

## 5. Aturan Anti "AI Slop"

Daftar ini wajib dicek saat review UI.

**Jangan:**

- Gradien ungu/biru, glow neon, atau latar "aurora" / blob.
- Glassmorphism (blur + transparan) di kartu biasa.
- Emoji di UI (judul, tombol, status). Pakai ikon lucide.
- Kartu dengan radius besar (`rounded-2xl`+) dan shadow tebal.
- Hero section, ilustrasi 3D, atau kalimat sambutan panjang ("Welcome back! 👋 Ready to crush it?"). Cukup nama halaman.
- Ikon di setiap judul kartu hanya untuk "mengisi".
- Semua elemen di tengah halaman. Data rata kiri/kanan sesuai jenisnya.
- Banyak warna aksen sekaligus. Satu aksen saja.
- Animasi masuk pada setiap kartu saat halaman dibuka.
- Angka tanpa format (`1585`), desimal acak, atau `NaN` / `undefined` tampil ke user.
- Placeholder palsu ("Lorem ipsum", "John Doe", grafik dummy) di build yang di-review.
- Badge "AI" / "New" / "Pro" dekoratif.

**Lakukan:**

- Gunakan data asli dari API (backend lokal) saat develop. Kalau perlu data contoh, pakai skenario uji di `dashboard.md`.
- Konsistensi lebih penting dari variasi: komponen yang sama tampil sama di semua halaman.
- Setiap angka punya konteks (satuan, periode, pembanding): `+$85.00 · +5.67% ROI`.
- Empty state berupa kalimat jelas + satu aksi, bukan ilustrasi besar.
- Rapikan alignment ke grid: label sejajar, angka sejajar desimal.
- Uji tampilan dengan angka ekstrem (`$26,168,736.22`, `−$2,288,514.79`, `0`) supaya tidak terpotong.

---

## 6. React Bits (opsional, terbatas)

Boleh dipakai untuk memberi kesan "segar", dengan batasan:

| Boleh | Contoh pemakaian |
|-------|------------------|
| `CountUp` | Angka hero (balance / net PnL) saat pertama tampil. Durasi ≤ 600ms, **sekali saja**, tidak saat polling |
| Animated tab indicator / underline | Perpindahan tab |
| Efek halus di halaman login / empty state | Latar noise / grid sangat tipis (opacity ≤ 5%) |

| Tidak boleh |
|-------------|
| Animasi di tabel, chart, atau angka yang di-refresh polling |
| Teks berkilau / glitch / typewriter untuk data |
| Partikel, aurora, blob, cursor effect |
| Lebih dari 2 jenis efek React Bits di seluruh aplikasi |

Aturan gerak umum: durasi 120-200ms, `ease-out`, tanpa bounce. Hormati `prefers-reduced-motion`.

---

## 7. Halaman

Prioritas pengerjaan dan spesifikasi detail ada di `frontend-dashboard-guide.md` (§3-7 dan daftar tugas §10). Ringkasan halaman:

| Halaman | Referensi gaya |
|---------|----------------|
| Login | Minimal: logo, form di tengah, latar polos / grid tipis |
| Dashboard (Overview) | Ref 1 (baris KPI) + ref 2 (grid kartu jurnal dengan sparkline) |
| Journal: Performance | Ref 1 (KPI + equity chart garis nol + tabel) |
| Journal: Target Tracker | Ref 4 (kartu target) dalam gaya terminal + kalender heatmap |
| Journal: Analytics | Grid chart breakdown, tabel runner |
| Journal: Trades | Ref 1 / ref 3 (tabel padat, filter di atas, sheet detail di kanan) |
| Journal: Runners & Settings | Form shadcn standar, textarea rules/strategy, tombol download AGENT.md |
| Runner Monitor | Tabel dengan dot status live |

---

## 8. Kualitas & Serah Terima

- **Format angka & tanggal** hanya lewat helper di `lib/format.ts`. Tidak ada `toFixed` tersebar di komponen.
- **State lengkap** di setiap halaman: loading (skeleton bentuk asli), kosong, error, data.
- **Aksesibilitas**: kontras WCAG AA, fokus keyboard terlihat (`ring` aksen), status tidak hanya dibedakan warna, chart punya ringkasan teks / tooltip.
- **Responsif**: dicek di 360px, 768px, 1280px, 1920px.
- **Performa**: tabel trade > 200 baris memakai virtualisasi. Polling berhenti saat tab tidak aktif.
- **Review UI** memakai checklist bagian 5. PR yang melanggar daftar "Jangan" dikembalikan.

Kalau ada keputusan desain yang tidak tercakup di brief ini, utamakan: **lebih sederhana, lebih padat, lebih konsisten dengan referensi 1**.
