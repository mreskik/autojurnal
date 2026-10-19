# Trades

Base path: `/api/journals/:id/trades`

Data tiap trade di dalam jurnal. Endpoint di halaman ini dipakai **user pemilik jurnal** (token session). Endpoint trade untuk runner (agent) ada di [runner-guide.md](runner-guide.md), dengan format yang sama.

Format response umum ada di [README.md](README.md), daftar error code di [error-codes.md](error-codes.md), konsep jurnal/periode/target di [journals.md](journals.md).

**Semua endpoint wajib login.** Jurnal milik user lain dianggap tidak ada (`1400`).

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/api/journals/:id/trades` | List trade |
| GET | `/api/journals/:id/trades/:tradeId` | Detail trade |
| POST | `/api/journals/:id/trades` | Buka trade (atau catat trade yang sudah selesai) |
| PUT | `/api/journals/:id/trades/:tradeId` | Ubah trade |
| POST | `/api/journals/:id/trades/:tradeId/close` | Tutup trade |
| DELETE | `/api/journals/:id/trades/:tradeId` | Hapus trade (soft delete) |

## Siklus Trade

```
open ──(POST .../close)──> closed
```

- Trade **closed tidak bisa dibuka lagi** dan tidak bisa ditutup ulang (`1501`).
- Setelah closed, **hanya `close_note` yang bisa diubah**. Field lain ditolak (`1501`).
- Saat masih open, semua field open boleh diubah. `close_note` hanya bisa diisi saat close.
- Hanya trade **closed** yang dihitung ke balance dan pencapaian target. PnL masuk ke periode tanggal `closed_at`.

## PnL

- `pnl` = **PnL bersih** (yang benar-benar masuk balance). `fee` hanya informasi.
- Saat close, kalau `pnl` dikirim (misal angka asli dari exchange), angka itu yang dipakai.
- Kalau `pnl` tidak dikirim, server menghitung:

  | Posisi | Rumus |
  |--------|-------|
  | long | `(exit_price − entry_price) / entry_price × margin × leverage − fee` |
  | short | `(entry_price − exit_price) / entry_price × margin × leverage − fee` |

  Dibulatkan 8 angka di belakang koma.

**Contoh:** long, margin 100, leverage 20 (position size 2000), entry 61000, exit 63000, fee 1.5
→ `(63000 − 61000) / 61000 × 2000 − 1.5 = 64.07377049`

## Balance

- Total balance jurnal = `initial_balance` + deposit − withdraw + Σ `pnl` trade closed.
- Balance tidak boleh minus. Close trade (rugi), buat trade langsung closed, atau hapus trade (profit) yang membuat balance minus akan ditolak (`1404`).
- Begitu jurnal punya trade (open maupun closed), `initial_balance`, `period_type`, `timezone`, `start_date` terkunci (`1405`).
- Jurnal `archived` tidak bisa ditambah trade baru (`1406`). Trade yang sudah ada tetap bisa ditutup / diedit / dihapus.
- Trade milik **runner**: satu runner hanya boleh punya 1 trade open per pair + position di satu jurnal. Mengubah `pair` / `position` trade runner yang membuatnya bentrok ditolak `1503` (juga kalau diubah dari sisi user). Input manual user tidak dibatasi.

## Objek Trade

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID v7) | |
| `journal_id` | string (UUID) | |
| `runner_id` | string (UUID) \| null | Runner pembuat trade. `null` = input manual user |
| `exchange` | string \| null | Huruf kecil, contoh `binance` |
| `pair` | string | Huruf besar, contoh `BTCUSDT` |
| `position` | string | `long` / `short` |
| `margin` | string (decimal) | Modal yang dipakai (USD) |
| `leverage` | string (decimal) | |
| `position_size` | string (decimal) | `margin × leverage` (dihitung) |
| `entry_price` | string (decimal) | |
| `exit_price` | string (decimal) \| null | Terisi saat closed |
| `stop_loss` | string (decimal) \| null | |
| `take_profit` | string (decimal) \| null | |
| `status` | string | `open` / `closed` |
| `open_reason` | string \| null | Alasan entry |
| `close_reason` | string \| null | `take_profit` / `stop_loss` / `manual` / `liquidation` / `trailing_stop` |
| `close_note` | string \| null | Catatan saat / setelah close |
| `pnl` | string (decimal) \| null | PnL bersih (USD), terisi saat closed |
| `fee` | string (decimal) | Default `0` |
| `roi` | string (decimal) \| null | `pnl / margin × 100` (persen, 2 desimal), hanya closed |
| `opened_at` | string (RFC 3339) | |
| `closed_at` | string (RFC 3339) \| null | |
| `created_at`, `updated_at` | string (RFC 3339) | |

## Aturan Validasi

| Field | Aturan |
|-------|--------|
| `pair` | Wajib, 2-30 karakter: huruf, angka, `/ . _ -`. Otomatis huruf besar |
| `position` | Wajib, `long` / `short` (tidak case-sensitive) |
| `margin` | Wajib, > 0 |
| `leverage` | Wajib, 1 - 500 |
| `entry_price` | Wajib, > 0 |
| `stop_loss` | Opsional, > 0. Arahnya **tidak** dicek (SL boleh digeser ke area profit untuk breakeven / trailing) |
| `take_profit` | Opsional, > 0. Long: harus di atas `entry_price`. Short: harus di bawah `entry_price` |
| `exchange` | Opsional, maksimal 30 karakter. Otomatis huruf kecil |
| `open_reason`, `close_note` | Opsional, maksimal 1000 karakter |
| `opened_at` | Opsional (default sekarang). Tidak boleh sebelum `start_date` jurnal dan tidak boleh di masa depan |
| `exit_price` | Wajib saat close, > 0 |
| `close_reason` | Wajib saat close: `take_profit` / `stop_loss` / `manual` / `liquidation` / `trailing_stop` |
| `pnl` | Opsional saat close (boleh negatif). Kosong = dihitung server |
| `fee` | Opsional saat close, ≥ 0 (default 0) |
| `closed_at` | Opsional saat close (default sekarang). Tidak boleh sebelum `opened_at` dan tidak boleh di masa depan |

---

## GET /api/journals/:id/trades

Diurutkan dari `opened_at` terbaru.

**Query parameter**

| Nama | Tipe | Default | Keterangan |
|------|------|---------|------------|
| `page` | integer | 1 | |
| `limit` | integer | 10 | Maksimal 100 |
| `status` | string | - | `open` / `closed` |
| `position` | string | - | `long` / `short` |
| `pair` | string | - | Contoh `BTCUSDT` (tidak case-sensitive) |
| `source` | string | - | `manual` (input user) / `runner` (dari agent) |
| `from` | string (`YYYY-MM-DD`) | - | `opened_at` dari tanggal ini (inklusif, timezone jurnal) |
| `to` | string (`YYYY-MM-DD`) | - | `opened_at` sampai tanggal ini (inklusif, timezone jurnal) |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a154b8-94ec-728f-8efc-c5ef9826a57e",
      "journal_id": "01a154b8-93d6-74b7-a96f-04b95ead00c5",
      "runner_id": null,
      "exchange": "binance",
      "pair": "BTCUSDT",
      "position": "long",
      "margin": "100",
      "leverage": "20",
      "position_size": "2000",
      "entry_price": "61000",
      "exit_price": "63000",
      "stop_loss": null,
      "take_profit": "66000",
      "status": "closed",
      "open_reason": "breakout",
      "close_reason": "take_profit",
      "close_note": "TP1 kena",
      "pnl": "64.07377049",
      "fee": "1.5",
      "roi": "64.07",
      "opened_at": "2026-10-19T22:12:25.322671+07:00",
      "closed_at": "2026-10-19T22:12:27.104512+07:00",
      "created_at": "2026-10-19T22:12:25.321865+07:00",
      "updated_at": "2026-10-19T22:12:27.104512+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 10, "total": 1, "total_page": 1 }
}
```

**Response 422** (filter tidak valid)

```json
{ "code": 1002, "message": "validation failed", "data": { "status": "must be open or closed" } }
```

---

## GET /api/journals/:id/trades/:tradeId

**Response 200**: `data` berisi satu objek Trade.

**Response 404**

```json
{ "code": 1500, "message": "trade not found", "data": null }
```

---

## POST /api/journals/:id/trades

Membuka trade baru. `runner_id` otomatis `null` (input manual).

**Buka trade (status open)**

```json
{
  "exchange": "binance",
  "pair": "BTCUSDT",
  "position": "long",
  "margin": 100,
  "leverage": 10,
  "entry_price": 60000,
  "stop_loss": 58000,
  "take_profit": 66000,
  "open_reason": "breakout"
}
```

**Catat trade yang sudah selesai (langsung closed)**: sertakan field close (`exit_price`, `close_reason`, dan opsional `pnl`, `fee`, `close_note`, `closed_at`). Aturannya sama seperti endpoint close. Fitur ini hanya untuk user di aplikasi; runner selalu buka trade dulu lalu close.

```json
{
  "pair": "ETHUSDT",
  "position": "short",
  "margin": 50,
  "leverage": 5,
  "entry_price": 3000,
  "opened_at": "2026-10-05T10:00:00+07:00",
  "exit_price": 3100,
  "close_reason": "stop_loss",
  "closed_at": "2026-10-05T12:00:00+07:00"
}
```

**Response 201**: `data` berisi objek Trade.

**Response 422** (validasi)

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": { "take_profit": "must be above entry_price for long position" }
}
```

**Response 422** (jurnal diarsipkan)

```json
{ "code": 1406, "message": "journal is archived", "data": null }
```

**Response 422** (trade langsung closed dengan rugi yang membuat balance minus)

```json
{ "code": 1404, "message": "insufficient balance", "data": { "balance_after": "-94.25956284" } }
```

---

## PUT /api/journals/:id/trades/:tradeId

Update sebagian, hanya field yang dikirim yang diubah.

**Trade masih open**: boleh ubah `exchange`, `pair`, `position`, `margin`, `leverage`, `entry_price`, `stop_loss`, `take_profit`, `open_reason`, `opened_at`. Kirim `null` untuk mengosongkan `stop_loss` / `take_profit`, `""` untuk mengosongkan `exchange` / `open_reason`.

```json
{ "stop_loss": 61000, "take_profit": null }
```

**Trade sudah closed**: hanya `close_note`.

```json
{ "close_note": "catatan tambahan setelah close" }
```

**Response 200**: `data` berisi objek Trade terbaru.

**Response 422** (ubah field selain `close_note` pada trade closed)

```json
{
  "code": 1501,
  "message": "trade already closed",
  "data": {
    "entry_price": "cannot be changed after trade is closed, only close_note can be changed"
  }
}
```

**Response 422** (`close_note` pada trade yang masih open)

```json
{ "code": 1002, "message": "validation failed", "data": { "close_note": "can only be set when closing the trade" } }
```

---

## POST /api/journals/:id/trades/:tradeId/close

**Request body**

| Field | Wajib |
|-------|-------|
| `exit_price` | Ya |
| `close_reason` | Ya |
| `pnl` | Tidak (kosong = dihitung server) |
| `fee` | Tidak (default 0) |
| `close_note` | Tidak |
| `closed_at` | Tidak (default sekarang) |

```json
{
  "exit_price": 63000,
  "close_reason": "take_profit",
  "fee": 1.5,
  "close_note": "TP1 kena"
}
```

**Response 200**: `data` berisi objek Trade dengan `status: "closed"`, `pnl`, dan `roi`.

**Response 422** (sudah closed)

```json
{ "code": 1501, "message": "trade already closed", "data": null }
```

**Response 422** (rugi membuat balance minus)

```json
{ "code": 1404, "message": "insufficient balance", "data": { "balance_after": "-12.5" } }
```

---

## DELETE /api/journals/:id/trades/:tradeId

Soft delete. Bisa ditolak `1404` kalau menghapus trade profit membuat balance minus (misal profitnya sudah ditarik lewat withdraw).

**Response 200**

```json
{ "code": 0, "message": "trade deleted", "data": null }
```

**Response 404**

```json
{ "code": 1500, "message": "trade not found", "data": null }
```
