# Panduan Integrasi Runner (Agent)

Panduan langkah demi langkah untuk agent / bot / script yang mencatat trade ke jurnal Autojurnal lewat API: membaca detail jurnal & target, melihat trade aktif, membuka trade, mengubah, dan menutupnya.

> Dokumen ini untuk **developer agent / bot**, bukan untuk frontend aplikasi. Aplikasi tidak memanggil `/api/runner/*`; lihat [README.md](README.md#dokumen-per-pembaca).

Dokumen ini adalah referensi lengkap semua endpoint `/api/runner/*`. Format objek trade dan aturan validasinya sama dengan endpoint trade aplikasi di [trades.md](trades.md). Daftar error di [error-codes.md](error-codes.md). Cara pemilik membuat runner & token ada di [runners.md](runners.md).

## Ringkasan Alur

```
              ┌────────────────────────────┐
  (sekali)    │ 1. GET /runner/me          │  cek token runner
              └─────────────┬──────────────┘
                            │
  (tiap siklus)             ▼
              ┌────────────────────────────┐
              │ 2. GET /runner/journal     │  status, balance, target + progres,
              │                            │  dan semua trade yang masih open
              └─────────────┬──────────────┘
          ada sinyal entry  │  posisi berubah / kena TP/SL
              ┌─────────────┴──────────────┐
              ▼                            ▼
 ┌────────────────────────┐   ┌──────────────────────────────────┐
 │ 4. POST /runner/trades │   │ 5. PUT  /runner/trades/:id        │ geser SL/TP
 │    buka trade          │   │ 6. POST /runner/trades/:id/close  │ tutup + PnL
 └────────────────────────┘   │ 7. PUT  /runner/trades/:id        │ catatan setelah close
                              └──────────────────────────────────┘

  (opsional) 3. GET /runner/trades → semua trade (termasuk closed) untuk info / riwayat
```

Dalam satu siklus normal agent cukup **1 request baca** (`GET /runner/journal`) untuk tahu target, progres, dan posisi terbuka.

---

## 0. Persiapan (dilakukan pemilik jurnal di aplikasi)

Agent butuh **dua token**. Keduanya dibuat oleh pemilik jurnal (user yang login), bukan oleh agent:

| Langkah | Endpoint (token session user) | Hasil |
|---------|-------------------------------|-------|
| Buat runner | `POST /api/runners` | field `token` = **token runner** |
| Buat jurnal (atau pakai yang sudah ada) | `POST /api/journals` | field `token` = **token jurnal** |
| Daftarkan runner ke jurnal | `POST /api/journals/:id/runners` body `{ "runner_id": "..." }` | runner boleh menulis ke jurnal ini |

**Cara cepat:** pemilik bisa download **`AGENT.md`** lewat `GET /api/journals/:id/runners/:runnerId/agent.md` (lihat [journals.md](journals.md#get-apijournalsidrunnersrunneridagentmd)). File itu sudah berisi Base URL, Runner ID, kedua token, siklus kerja, endpoint, aturan wajib, dan penanganan error dalam bahasa Inggris, siap diberikan ke AI agent.

Simpan kedua token di konfigurasi agent (environment variable / secret), **jangan** di kode.

```
AUTOJURNAL_BASE_URL=http://localhost:1000/api
AUTOJURNAL_RUNNER_TOKEN=WB4eyPg9tSaMKvkil8AP3lErX4DS2ztlQNOTn9tU
AUTOJURNAL_JOURNAL_TOKEN=FeaLF1j0t9bY4fZnTVE3wUMtOfESAk1zbHHGDRI1
```

## Header di Setiap Request

```
Authorization: Bearer <token runner>
X-Journal-Token: <token jurnal>
Content-Type: application/json
```

`X-Journal-Token` boleh tidak dikirim hanya untuk `GET /api/runner/me`. Jurnal tujuan ditentukan dari token jurnal, jadi **tidak ada journal id di URL**.

Token session user (dari login aplikasi) tidak bisa dipakai di `/api/runner/*`, dan token runner tidak bisa dipakai di endpoint aplikasi.

Request ke endpoint jurnal & trade diterima kalau **ketiga syarat** terpenuhi:

1. Token runner valid (aktif, belum expired, runner & pemiliknya belum dihapus).
2. Token jurnal valid dan jurnal itu milik user yang sama dengan pemilik runner.
3. Runner sudah di-join ke jurnal tersebut oleh pemilik jurnal.

## Daftar Endpoint

| Method | Endpoint | Token jurnal | Keterangan | Langkah |
|--------|----------|--------------|------------|---------|
| GET | `/api/runner/me` | Tidak | Cek token runner dan identitas runner | [1](#1-cek-token-runner) |
| GET | `/api/runner/journal` | Ya | Detail jurnal, balance, target + progres periode berjalan, **semua trade open** | [2](#2-detail-jurnal-balance-dan-target) |
| GET | `/api/runner/trades` | Ya | Semua trade di jurnal (open & closed, semua sumber) dengan filter & pagination, untuk info / riwayat | [3](#3-semua-trade-info--riwayat) |
| GET | `/api/runner/trades/:tradeId` | Ya | Detail satu trade di jurnal (siapa pun pembuatnya) | [3](#3-semua-trade-info--riwayat) |
| POST | `/api/runner/trades` | Ya | Buka trade (selalu status open) | [4](#4-buka-trade-insert) |
| PUT | `/api/runner/trades/:tradeId` | Ya | Ubah trade (open: field open, closed: hanya `close_note`) | [5](#5-ubah-trade-yang-masih-open-geser-sltp-dll), [7](#7-tambah-catatan-setelah-close) |
| POST | `/api/runner/trades/:tradeId/close` | Ya | Tutup trade | [6](#6-tutup-trade-close) |

## Aturan Akses

- Runner bisa **melihat semua trade** di jurnal dari token jurnal: input manual pemilik, buatan runner ini, dan buatan runner lain.
- Runner hanya bisa **mengubah / menutup trade buatannya sendiri**. Mencoba mengubah trade milik pihak lain = `1502 trade not owned by this runner`.
- Trade di jurnal lain (bukan jurnal dari token) = `1500 trade not found`.
- Kalau runner dikeluarkan dari jurnal, request dengan token jurnal itu langsung ditolak (`1408`). Trade-nya tetap ada dan tetap bisa dikelola pemilik jurnal.
- Kalau token jurnal di-regenerate pemilik, agent harus memakai token baru (token lama `1407`).
- Runner **tidak bisa menghapus** trade. Hapus hanya oleh pemilik jurnal lewat aplikasi.
- Jurnal `archived`: runner tidak bisa buka trade baru (`1406`), tapi masih bisa menutup trade yang sudah ada.
- `runner_id` di trade otomatis diisi id runner.

## Aturan Penting Format Data

- Semua nominal dalam **USD**.
- Angka di **response** berupa **string** (`"1036.58333333"`) supaya presisi tidak hilang. Parse sebagai decimal, jangan float kalau dipakai untuk hitungan uang.
- Angka di **request** boleh number (`100.5`) atau string (`"100.5"`).
- Waktu memakai format RFC 3339, contoh `2026-10-19T08:00:00+07:00`. Kalau tidak dikirim, server memakai waktu sekarang.
- `pair` dinormalisasi: spasi di awal/akhir dibuang dan diubah jadi huruf besar (`btcusdt`, `BtcUsdt`, ` BTCusdt ` → `BTCUSDT`). Berlaku juga untuk filter `?pair=`. `exchange` otomatis huruf kecil, `position` tidak case-sensitive (`LONG` → `long`).

---

## 1. Cek Token Runner

```
GET /api/runner/me
Authorization: Bearer <token runner>
```

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "01a154d5-cbc1-7095-92bc-a53bc7a91ebe",
    "user_id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
    "name": "Agent Scalping",
    "expires_at": null
  }
}
```

Jalankan sekali saat agent start. Kalau gagal (`1301`, `1302`, `1303`), hentikan agent dan minta token baru ke pemilik.

---

## 2. Detail Jurnal, Balance, dan Target

```
GET /api/runner/journal
Authorization: Bearer <token runner>
X-Journal-Token: <token jurnal>
```

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "name": "Daily 2%",
    "description": null,
    "status": "active",
    "initial_balance": "1000",
    "balance": "1670",
    "period_type": "daily",
    "timezone": "Asia/Jakarta",
    "start_date": "2026-10-01",
    "current_target": {
      "amount": "31",
      "achieved": "20",
      "progress_percent": "64.52",
      "period_start": "2026-10-19",
      "period_end": "2026-10-19"
    },
    "instructions": {
      "journal": {
        "rules": "Maks risk 1% per trade, wajib pasang SL",
        "strategy": "Fokus BTCUSDT, hindari jam rilis news besar"
      },
      "runner": {
        "rules": "Leverage maks 5x",
        "strategy": "Mean reversion timeframe 15m, TP 1:1.5"
      }
    },
    "goal": {
      "goal_type": "profit",
      "goal_value": "500",
      "current_value": "70",
      "progress_percent": "14",
      "remaining": "430",
      "status": "in_progress",
      "achieved_at": null
    },
    "open_trades": [
      {
        "id": "01a154d5-d214-7726-937e-1f7f6e7d46c6",
        "journal_id": "01a154d5-c9bf-7fc0-902a-33a9f135b2b7",
        "runner_id": "01a154d5-cbc1-7095-92bc-a53bc7a91ebe",
        "exchange": "binance",
        "pair": "BTCUSDT",
        "position": "long",
        "margin": "100",
        "leverage": "10",
        "position_size": "1000",
        "entry_price": "60000",
        "exit_price": null,
        "stop_loss": "59000",
        "take_profit": "62000",
        "status": "open",
        "open_reason": "signal AI: breakout 1h",
        "close_reason": null,
        "close_note": null,
        "pnl": null,
        "fee": "0",
        "roi": null,
        "opened_at": "2026-10-19T22:44:21.524031+07:00",
        "closed_at": null,
        "created_at": "2026-10-19T22:44:21.523254+07:00",
        "updated_at": "2026-10-19T22:44:21.523254+07:00"
      }
    ],
    "created_at": "2026-10-19T22:55:12.463012+07:00",
    "updated_at": "2026-10-19T22:55:12.463012+07:00"
  }
}
```

### Field

| Field | Arti | Contoh pemakaian di agent |
|-------|------|---------------------------|
| `name`, `description` | Nama & deskripsi jurnal | Log / identifikasi |
| `status` | `active` / `archived` | Kalau `archived`, jangan buka trade baru (akan ditolak `1406`) |
| `initial_balance` | Modal awal jurnal (USD) | |
| `balance` | Total balance saat ini = initial + deposit − withdraw + PnL trade closed | Dasar menghitung ukuran posisi / risk per trade |
| `period_type` | `daily` / `weekly` (Senin-Minggu) / `monthly` (tgl 1 - akhir bulan) | |
| `timezone` | Batas hari dihitung di timezone ini | Menentukan kapan periode baru dimulai |
| `start_date` | Tanggal jurnal mulai | |
| `current_target` | Target profit yang harus dicapai di periode berjalan (lihat di bawah) | |
| `instructions` | Aturan & strategi dari pemilik jurnal (lihat di bawah) | **Baca sebelum mengambil keputusan trading** |
&
| `open_trades` | **Semua trade yang masih open** di jurnal (manual pemilik, runner ini, runner lain), terbaru dulu, tanpa pagination. `[]` kalau tidak ada. Format objek sama dengan langkah 3 | Sinkron dengan posisi di exchange, hitung total exposure, hindari entry dobel. Yang boleh diubah/ditutup hanya yang `runner_id`-nya milik runner ini |

### `instructions`

Instruksi teks dari pemilik jurnal yang **harus diikuti agent**. Ada dua lapis:

| Field | Berlaku untuk | Isi |
|-------|---------------|-----|
| `instructions.journal.rules` | Semua runner di jurnal ini | Aturan / batasan umum jurnal |
| `instructions.journal.strategy` | Semua runner di jurnal ini | Strategi umum jurnal |
| `instructions.runner.rules` | Hanya runner ini | Aturan tambahan khusus runner ini |
| `instructions.runner.strategy` | Hanya runner ini | Strategi khusus runner ini |

- Setiap field berupa teks bebas (bisa berisi Markdown), maks 5000 karakter, atau `null` kalau tidak diisi.
- Kalau ada yang bertentangan, anggap instruksi **runner** lebih spesifik daripada instruksi **journal**, kecuali pemilik menulis lain.
- Instruksi ini **panduan, tidak dipaksakan server**. Contoh: kalau `rules` berisi "leverage maks 5x", server tetap menerima trade 20x. Agent sendiri yang wajib patuh.
- Pemilik bisa mengubah instruksi kapan saja. Ambil ulang `GET /api/runner/journal` di tiap siklus supaya selalu memakai versi terbaru.

### `current_target`

| Field | Arti |
|-------|------|
| `amount` | Target profit periode ini dalam **USD**, sudah dihitung server |
| `achieved` | PnL yang sudah tercapai di periode ini (USD): total `pnl` trade yang **closed** dengan `closed_at` di dalam periode. Bisa negatif |
| `progress_percent` | `achieved / amount × 100`, 2 desimal. ≥ 100 berarti target periode ini tercapai. Bisa > 100 atau negatif |
| `period_start` | Awal periode (inklusif, `YYYY-MM-DD` di `timezone` jurnal) |
| `period_end` | Akhir periode (inklusif) |

Catatan `achieved`:

- Dihitung dari **semua trade di jurnal**: input manual pemilik, runner ini, dan runner lain. Target milik jurnal, bukan milik satu runner.
- Hanya trade **closed**. Trade yang masih open belum dihitung (floating PnL tidak diketahui server).
- Yang menentukan periode adalah `closed_at`: trade yang dibuka kemarin dan ditutup hari ini masuk ke `achieved` hari ini.
- Deposit / withdraw tidak dihitung (bukan profit).

Cara server menghitung `amount`:

- Target nominal: `amount` = nilai target yang diset pemilik jurnal.
- Target persen (compounding): `amount` = persen × **balance di awal periode** (sebelum `period_start`). Deposit/withdraw dan PnL yang terjadi **di dalam** periode berjalan tidak mengubah `amount` periode ini, baru berpengaruh ke periode berikutnya.
- Kalau pemilik mengganti target di tengah periode, perubahan baru berlaku periode berikutnya. `amount` periode berjalan tetap.

Contoh di atas: target 2% daily, balance di awal hari 1550 → `amount` = 31. Hari ini sudah ada profit +20 (`achieved` = 20, `progress_percent` = 64.52) dan deposit +100, sehingga `balance` = 1670, tapi target hari ini tetap 31.

Contoh pemakaian di agent:

```
if current_target.progress_percent >= 100:
    # target periode ini tercapai → misal stop buka posisi baru sampai period_end
sisa = current_target.amount - current_target.achieved
```

Endpoint ini adalah **endpoint utama tiap siklus**: satu request untuk target, progres, dan posisi terbuka. Panggil di awal siklus, sebelum membuka trade, dan setelah menutup trade untuk melihat progres terbaru.

---

## 3. Semua Trade (Info / Riwayat)

```
GET /api/runner/trades
Authorization: Bearer <token runner>
X-Journal-Token: <token jurnal>
```

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a154d5-d214-7726-937e-1f7f6e7d46c6",
      "journal_id": "01a154d5-c9bf-7fc0-902a-33a9f135b2b7",
      "runner_id": "01a154d5-cbc1-7095-92bc-a53bc7a91ebe",
      "exchange": "binance",
      "pair": "BTCUSDT",
      "position": "long",
      "margin": "100",
      "leverage": "10",
      "position_size": "1000",
      "entry_price": "60000",
      "exit_price": null,
      "stop_loss": "59000",
      "take_profit": "62000",
      "status": "open",
      "open_reason": "signal AI: breakout 1h",
      "close_reason": null,
      "close_note": null,
      "pnl": null,
      "fee": "0",
      "roi": null,
      "opened_at": "2026-10-19T22:44:21.524031+07:00",
      "closed_at": null,
      "created_at": "2026-10-19T22:44:21.523254+07:00",
      "updated_at": "2026-10-19T22:44:21.523254+07:00"
    },
    {
      "id": "01a154d5-c1f0-7b11-8e3a-5d2c9f0a1b77",
      "journal_id": "01a154d5-c9bf-7fc0-902a-33a9f135b2b7",
      "runner_id": null,
      "exchange": "bybit",
      "pair": "ETHUSDT",
      "position": "short",
      "margin": "50",
      "leverage": "5",
      "position_size": "250",
      "entry_price": "3000",
      "exit_price": null,
      "stop_loss": "3100",
      "take_profit": null,
      "status": "open",
      "open_reason": "entry manual pemilik",
      "close_reason": null,
      "close_note": null,
      "pnl": null,
      "fee": "0",
      "roi": null,
      "opened_at": "2026-10-19T20:10:00+07:00",
      "closed_at": null,
      "created_at": "2026-10-19T20:10:00.120000+07:00",
      "updated_at": "2026-10-19T20:10:00.120000+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 10, "total": 2, "total_page": 1 }
}
```

- Endpoint **info / riwayat**: semua trade di jurnal, open maupun closed, dari semua sumber (runner ini, input manual pemilik, runner lain). Untuk siklus normal tidak perlu dipanggil, karena trade yang masih open sudah ada di `open_trades` pada langkah 2.
- Contoh pemakaian: `?status=closed&from=2026-10-01` untuk evaluasi hasil, `?pair=BTCUSDT` untuk riwayat satu pair.
- **Kepemilikan** dilihat dari `runner_id`:

  | `runner_id` | Pembuat | Bisa diubah / ditutup runner ini? |
  |-------------|---------|-----------------------------------|
  | sama dengan `id` dari `GET /api/runner/me` | Runner ini | Ya |
  | `null` | Input manual pemilik jurnal | Tidak (`1502`) |
  | id lain | Runner lain | Tidak (`1502`) |

- Diurutkan dari `opened_at` terbaru. Pakai `limit` (maks 100) dan `page` kalau posisi banyak.

**Query parameter**

| Nama | Keterangan |
|------|------------|
| `status` | `open` / `closed` |
| `source` | `manual` (input pemilik) / `runner` (dari runner mana pun) |
| `pair` | Contoh `BTCUSDT` (tidak case-sensitive) |
| `position` | `long` / `short` |
| `from` / `to` | Tanggal `opened_at` (inklusif), format `YYYY-MM-DD`, timezone jurnal |
| `page`, `limit` | Pagination, `limit` maks 100 |



Detail satu trade (siapa pun pembuatnya): `GET /api/runner/trades/:tradeId`.

---

## 4. Buka Trade (Insert)

```
POST /api/runner/trades
Authorization: Bearer <token runner>
X-Journal-Token: <token jurnal>
Content-Type: application/json
```

```json
{
  "exchange": "binance",
  "pair": "BTCUSDT",
  "position": "long",
  "margin": 100,
  "leverage": 10,
  "entry_price": 60000,
  "stop_loss": 59000,
  "take_profit": 62000,
  "open_reason": "signal AI: breakout 1h"
}
```

| Field | Wajib | Aturan |
|-------|-------|--------|
| `pair` | Ya | 2-30 karakter: huruf, angka, `/ . _ -` |
| `position` | Ya | `long` / `short` |
| `margin` | Ya | > 0 (USD) |
| `leverage` | Ya | 1 - 500 |
| `entry_price` | Ya | > 0 |
| `stop_loss` | Tidak | > 0. Arah tidak dicek (boleh di area profit untuk breakeven) |
| `take_profit` | Tidak | > 0. Long: di atas `entry_price`. Short: di bawah `entry_price` |
| `exchange` | Tidak | Maks 30 karakter |
| `open_reason` | Tidak | Maks 1000 karakter. Isi alasan / sinyal entry |
| `opened_at` | Tidak | Default sekarang. Tidak boleh sebelum `start_date` jurnal dan tidak boleh di masa depan |

**Response 201**: objek trade lengkap (sama seperti di langkah 3) dengan `status: "open"`. **Simpan `data.id`**, dipakai untuk update dan close.

Trade dari runner **selalu dibuat dalam status open**, lalu ditutup lewat langkah 6. Field close (`exit_price`, `close_reason`, `pnl`, `fee`, `close_note`, `closed_at`) tidak boleh dikirim di sini:

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "exit_price": "not allowed when opening a trade as runner, open the trade first then close it"
  }
}
```

```json
{
  "code": 0,
  "message": "trade created",
  "data": { "id": "01a154d5-d214-7726-937e-1f7f6e7d46c6", "status": "open", "position_size": "1000", "...": "..." }
}
```

### Guard trade dobel (1 posisi open per pair + position)

Satu runner hanya boleh punya **1 trade open** untuk kombinasi **pair + position** yang sama di satu jurnal.

| Kondisi (runner sama, jurnal sama) | Hasil |
|------------------------------------|-------|
| BTCUSDT long sudah open, buka BTCUSDT long lagi | Ditolak `409` code `1503` |
| BTCUSDT long sudah open, buka BTCUSDT **short** | Boleh (hedging) |
| Runner **lain** buka BTCUSDT long | Boleh (guard per runner) |
| BTCUSDT long sudah **closed**, buka BTCUSDT long lagi | Boleh |
| Ubah trade short jadi long padahal long sudah open | Ditolak `1503` |

Pair dibandingkan setelah normalisasi (`btcusdt`, `BtcUsdt`, `BTCUSDT` dianggap sama). Guard dijaga di database, jadi aman walaupun beberapa request datang bersamaan.

Kalau ditolak, `data` berisi trade open yang sudah ada:

```json
{
  "code": 1503,
  "message": "runner already has an open trade for this pair and position",
  "data": {
    "id": "01a154f5-ecb7-7811-87ad-e48a1966f368",
    "pair": "BTCUSDT",
    "position": "long",
    "status": "open",
    "...": "..."
  }
}
```

**Retry setelah timeout:** kalau `POST /api/runner/trades` timeout atau koneksi putus, agent **boleh langsung kirim ulang request yang sama**. Kalau ternyata request pertama sudah tersimpan, response-nya `1503` dengan trade yang sudah ada di `data`. Pakai `data.id` itu dan lanjutkan seperti biasa (anggap sukses).

Kalau strategi agent memang butuh beberapa posisi di pair + arah yang sama (scale in), catat sebagai satu trade dan perbarui `margin` / `entry_price` (harga rata-rata) lewat langkah 5.

---

## 5. Ubah Trade yang Masih Open (geser SL/TP, dll)

Hanya untuk trade **milik runner ini** (`runner_id` = id runner). Trade milik pihak lain ditolak `1502`.

```
PUT /api/runner/trades/:tradeId
```

Kirim hanya field yang berubah.

```json
{ "stop_loss": 60000 }
```

| Field yang bisa diubah (trade open) | Catatan |
|-------------------------------------|---------|
| `stop_loss`, `take_profit` | Kirim `null` untuk menghapus |
| `entry_price`, `margin`, `leverage`, `position`, `pair`, `exchange` | Misal koreksi data atau tambah posisi (average) |
| `open_reason`, `opened_at` | |

`close_note` **tidak bisa** diisi saat trade masih open (`1002`), isi saat close.

**Response 200**: objek trade terbaru (`"message": "trade updated"`).

---

## 6. Tutup Trade (Close)

Hanya untuk trade **milik runner ini**. Trade milik pihak lain ditolak `1502`.

```
POST /api/runner/trades/:tradeId/close
```

```json
{
  "exit_price": 62000,
  "close_reason": "take_profit",
  "fee": 0.8,
  "close_note": "TP kena"
}
```

| Field | Wajib | Aturan |
|-------|-------|--------|
| `exit_price` | Ya | > 0 |
| `close_reason` | Ya | `take_profit` / `stop_loss` / `manual` / `liquidation` / `trailing_stop` |
| `pnl` | Tidak | PnL **bersih** (setelah fee & funding) dalam USD, boleh negatif. Kirim angka asli dari exchange kalau ada |
| `fee` | Tidak | ≥ 0, default 0 |
| `close_note` | Tidak | Maks 1000 karakter |
| `closed_at` | Tidak | Default sekarang. Tidak boleh sebelum `opened_at` dan tidak boleh di masa depan |

**Kalau `pnl` tidak dikirim**, server menghitung:

```
long : (exit_price − entry_price) / entry_price × margin × leverage − fee
short: (entry_price − exit_price) / entry_price × margin × leverage − fee
```

Contoh di atas: `(62000 − 60000) / 60000 × 1000 − 0.8 = 32.53333333`.

**Response 200**

```json
{
  "code": 0,
  "message": "trade closed",
  "data": {
    "id": "01a154d5-d214-7726-937e-1f7f6e7d46c6",
    "status": "closed",
    "exit_price": "62000",
    "close_reason": "take_profit",
    "close_note": "TP kena",
    "pnl": "32.53333333",
    "fee": "0.8",
    "roi": "32.53",
    "closed_at": "2026-10-19T22:44:22.457225+07:00",
    "...": "..."
  }
}
```

Setelah close, `pnl` langsung masuk ke `balance` jurnal dan dihitung ke target periode tanggal `closed_at`.

- Trade yang sudah closed **tidak bisa ditutup lagi atau dibuka kembali** (`1501`).
- Close dengan rugi yang membuat balance jurnal minus ditolak (`1404`); biasanya tanda data `pnl` / `margin` salah input.

---

## 7. Tambah Catatan Setelah Close

Setelah closed, **hanya `close_note`** yang bisa diubah. Field lain ditolak `1501`.

```
PUT /api/runner/trades/:tradeId
```

```json
{ "close_note": "TP kena, momentum kuat" }
```

---

## Penanganan Error

Semua error memakai format `{ "code": <angka>, "message": "...", "data": ... }`. Agent sebaiknya memutuskan tindakan berdasarkan `code`, bukan `message`.

| Code | HTTP | Arti | Tindakan agent |
|------|------|------|----------------|
| `1301` | 401 | Token runner salah / tidak ada | **Stop.** Minta token runner baru |
| `1302` | 401 | Token runner expired | **Stop.** Minta pemilik memperpanjang `expires_at` |
| `1303` | 403 | Runner dinonaktifkan | **Stop** sampai diaktifkan lagi |
| `1407` | 401 | Token jurnal salah / sudah di-regenerate | **Stop.** Minta token jurnal terbaru |
| `1408` | 403 | Runner belum / tidak lagi di-join ke jurnal | **Stop.** Minta pemilik menambahkan runner ke jurnal |
| `1406` | 422 | Jurnal diarsipkan | Jangan buka trade baru. Close trade yang ada tetap boleh |
| `1500` | 404 | Trade tidak ditemukan (bukan di jurnal ini / dihapus pemilik) | Hapus dari state lokal agent, sinkron ulang lewat `open_trades` di langkah 2 |
| `1502` | 403 | Mencoba mengubah / menutup trade milik pihak lain (manual pemilik atau runner lain) | Jangan retry. Hanya ubah trade dengan `runner_id` milik sendiri |
| `1503` | 409 | Runner sudah punya trade open untuk pair + position yang sama | Pakai trade di `data` (biasanya hasil retry), jangan buat baru |
| `1501` | 422 | Trade sudah closed | Anggap sudah selesai, jangan retry |
| `1404` | 422 | Balance jurnal jadi minus | Cek ulang angka `pnl` / `margin`, jangan retry dengan data sama |
| `1002` | 422 | Validasi gagal, detail per field di `data` | Perbaiki field sesuai pesan, jangan retry dengan data sama |
| `1001` | 400 | Body bukan JSON valid | Bug di agent |
| `1003` | 400 | `tradeId` bukan UUID valid | Bug di agent |
| `1006` | 429 | Kena rate limit | Tunggu sesuai header `Retry-After` (detik), lalu retry |
| `1000` | 500 | Error server | Retry dengan jeda (misal 5s, 15s, 60s). Hati-hati dobel untuk `POST /trades` |

Contoh error validasi:

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": { "take_profit": "must be above entry_price for long position" }
}
```

## Rate Limit

- Per runner: default **120 request per menit** (sama dengan batas per user).
- Per IP: default **300 request per menit**.
- Kalau kena, response `429` code `1006` dengan header `Retry-After`.

Untuk polling, cukup panggil `GET /api/runner/journal` sekali per beberapa detik (sudah berisi target, progres, dan trade open), jangan setiap tick harga.

---

## Contoh Satu Siklus Lengkap (curl)

```bash
BASE=http://localhost:1000/api
AUTH="Authorization: Bearer $AUTOJURNAL_RUNNER_TOKEN"
JRN="X-Journal-Token: $AUTOJURNAL_JOURNAL_TOKEN"

# 1. cek token runner (sekali saat start)
curl -s -H "$AUTH" $BASE/runner/me

# 2. detail jurnal: target + progres + semua trade open (tiap siklus)
curl -s -H "$AUTH" -H "$JRN" $BASE/runner/journal

# 4. buka trade, simpan id-nya
curl -s -H "$AUTH" -H "$JRN" -H "Content-Type: application/json" \
  -X POST $BASE/runner/trades \
  -d '{"exchange":"binance","pair":"BTCUSDT","position":"long","margin":100,"leverage":10,"entry_price":60000,"stop_loss":59000,"take_profit":62000,"open_reason":"signal AI: breakout 1h"}'

# 5. geser SL ke breakeven
curl -s -H "$AUTH" -H "$JRN" -H "Content-Type: application/json" \
  -X PUT $BASE/runner/trades/$TRADE_ID -d '{"stop_loss":60000}'

# 6. tutup
curl -s -H "$AUTH" -H "$JRN" -H "Content-Type: application/json" \
  -X POST $BASE/runner/trades/$TRADE_ID/close \
  -d '{"exit_price":62000,"close_reason":"take_profit","fee":0.8,"close_note":"TP kena"}'
```
