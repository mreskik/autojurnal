# Journals

Base path: `/api/journals`

Jurnal adalah "wadah" trading: menyimpan saldo awal, tipe periode (harian/mingguan/bulanan), target profit per periode, deposit/withdraw, dan runner yang join. Data trade ada di [trades.md](trades.md).

Format response umum ada di [README.md](README.md), daftar error code di [error-codes.md](error-codes.md).

**Semua endpoint di sini wajib login** (header `Authorization: Bearer <token session>`). User hanya bisa melihat dan mengelola jurnal miliknya sendiri. Jurnal milik user lain dianggap tidak ada (`1400`).

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/api/journals` | List jurnal |
| GET | `/api/journals/:id` | Detail jurnal |
| POST | `/api/journals` | Buat jurnal + target pertama |
| PUT | `/api/journals/:id` | Ubah jurnal (sebagian field) |
| DELETE | `/api/journals/:id` | Hapus jurnal (soft delete) |
| POST | `/api/journals/:id/regenerate-token` | Ganti journal token |
| GET | `/api/journals/:id/targets` | Riwayat target |
| POST | `/api/journals/:id/targets` | Ganti target (berlaku periode berikutnya) |
| GET | `/api/journals/:id/transactions` | List deposit / withdraw |
| POST | `/api/journals/:id/transactions` | Tambah deposit / withdraw |
| PUT | `/api/journals/:id/transactions/:txId` | Ubah deposit / withdraw |
| DELETE | `/api/journals/:id/transactions/:txId` | Hapus deposit / withdraw |
| GET | `/api/journals/:id/runners` | List runner yang join |
| GET | `/api/journals/:id/runners/options` | Dropdown runner yang bisa ditambahkan (belum join) |
| POST | `/api/journals/:id/runners` | Tambah runner ke jurnal |
| PUT | `/api/journals/:id/runners/:runnerId` | Ubah rules / strategy khusus runner di jurnal |
| GET | `/api/journals/:id/runners/:runnerId/agent.md` | Download AGENT.md (instruksi + token) untuk runner |
| DELETE | `/api/journals/:id/runners/:runnerId` | Keluarkan runner dari jurnal |

## Konsep

### Nominal uang

- Semua nominal dalam **USD**.
- Di response, angka uang dikirim sebagai **string** (contoh `"1000"`, `"25.5"`) supaya presisi tidak hilang. Di request boleh dikirim sebagai angka (`1000.5`) atau string (`"1000.5"`).
- Presisi maksimal 8 angka di belakang koma.

### Periode

| `period_type` | Satu periode |
|---------------|--------------|
| `daily` | Satu hari (termasuk Sabtu & Minggu) |
| `weekly` | Senin sampai Minggu |
| `monthly` | Tanggal 1 sampai akhir bulan |

- Periode pertama dimulai dari `start_date` dan bisa tidak penuh (misal weekly mulai hari Rabu, periode pertama Rabu-Minggu).
- Batas hari dihitung memakai `timezone` jurnal (default `Asia/Jakarta`).

### Target

| `target_type` | `target_value` | Target per periode |
|---------------|----------------|--------------------|
| `amount` | USD, contoh `20` | Selalu `20` USD |
| `percent` | Persen (maks 100), contoh `2` | `2%` × total balance di awal periode (compounding) |

Total balance = `initial_balance` + deposit − withdraw + PnL trade yang sudah **closed** (lihat [trades.md](trades.md)).

### Riwayat target

- Target disimpan sebagai riwayat. Setiap perubahan target menambah baris baru, baris lama tidak pernah diubah.
- Target baru **berlaku mulai periode berikutnya**, bukan langsung. Contoh: jurnal daily, target diubah hari Senin jam 14.00 → berlaku mulai Selasa. Satu periode selalu punya tepat satu target.
- Kalau jurnal belum mulai (`start_date` masih di masa depan), target baru langsung berlaku sejak `start_date`.
- Di response jurnal ada dua field target:
  - `current_target`: target yang berlaku di periode yang sedang berjalan.
  - `latest_target`: target terbaru. Bisa berbeda dengan `current_target` kalau ada perubahan yang baru berlaku periode berikutnya (lihat `effective_from`).

### Target jurnal (goal)

Selain target per periode, jurnal boleh punya **target akhir** (opsional), misal "bawa modal 1000 jadi 5000".

| `goal_type` | `goal_value` | `current_value` | Tercapai kalau |
|-------------|--------------|-----------------|----------------|
| `balance` | USD | Total balance saat ini | balance ≥ `goal_value` |
| `profit` | USD | Σ `pnl` trade closed (deposit tidak dihitung) | total PnL ≥ `goal_value` |
| `percent` | Persen | Σ `pnl` ÷ (`initial_balance` + Σ deposit) × 100 | persentase ≥ `goal_value` |

- Dicek otomatis setiap ada perubahan balance (trade closed, deposit/withdraw, hapus trade/transaksi) dan saat goal diubah.
- **Sekali tercapai tetap `achieved`**, walaupun nilainya turun lagi. `achieved_at` = waktu pertama kali tercapai.
- Goal boleh diubah kapan saja. Kalau `goal_type` / `goal_value` berubah, status di-reset dan dicek ulang (bisa langsung tercapai lagi).
- Tidak ada deadline.
- `balance` ikut naik karena deposit. Kalau ingin target yang murni dari hasil trading, pakai `profit` atau `percent`.

### Instruksi runner (rules & strategy)

Teks bebas berisi aturan dan strategi yang harus diikuti agent. Ada dua lapis, dan keduanya dikirim ke runner di `GET /api/runner/journal` (field `instructions`, lihat [runner-guide.md](runner-guide.md#instructions)).

| Lapis | Disimpan di | Diatur lewat | Berlaku untuk |
|-------|-------------|--------------|---------------|
| Jurnal | `rules`, `strategy` di objek Journal | `POST` / `PUT /api/journals/:id` | Semua runner di jurnal |
| Runner | `rules`, `strategy` di objek Joined Runner | `POST /api/journals/:id/runners`, `PUT /api/journals/:id/runners/:runnerId` | Satu runner di jurnal ini |

Contoh: jurnal punya rules "Maks risk 1% per trade, wajib SL", lalu Bot B diberi rules tambahan "Leverage maks 5x" dan strategy "Mean reversion 15m". Bot B menerima keduanya, runner lain hanya menerima instruksi jurnal.

- Maks 5000 karakter per field. Boleh berisi Markdown (server menyimpan apa adanya).
- Instruksi ini **panduan**, tidak dipaksakan server (misal "leverage maks 5x" tidak membuat server menolak trade 20x).
- Saran UI: textarea di form edit jurnal (instruksi umum) dan di daftar runner jurnal (instruksi per runner).

## Objek Journal

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID v7) | |
| `user_id` | string (UUID) | Pemilik |
| `name` | string | |
| `description` | string \| null | |
| `initial_balance` | string (decimal) | Saldo awal (USD) |
| `period_type` | string | `daily` / `weekly` / `monthly` |
| `timezone` | string | Nama timezone IANA, contoh `Asia/Jakarta` |
| `start_date` | string (`YYYY-MM-DD`) | Awal periode pertama |
| `status` | string | `active` / `archived` |
| `token` | string | Journal token untuk runner (header `X-Journal-Token`), 40 karakter acak, dibuat otomatis saat jurnal dibuat. Lihat [runner-guide.md](runner-guide.md) |
| `rules` | string \| null | Aturan umum untuk **semua runner** di jurnal ini (teks bebas, panduan untuk agent). Lihat [Instruksi runner](#instruksi-runner-rules--strategy) |
| `strategy` | string \| null | Strategi umum untuk semua runner di jurnal ini |
| `current_target` | objek Target \| null | Target periode berjalan |
| `latest_target` | objek Target \| null | Target terbaru |
| `balance` | string (decimal) | Total balance saat ini = `initial_balance` + deposit − withdraw + PnL trade closed |
| `goal` | objek Goal \| null | Target akhir jurnal. `null` kalau tidak diset |

## Objek Goal

| Field | Tipe | Keterangan |
|-------|------|------------|
| `goal_type` | string | `balance` / `profit` / `percent` |
| `goal_value` | string (decimal) | Nilai target |
| `current_value` | string (decimal) | Nilai saat ini (sesuai `goal_type`) |
| `progress_percent` | string (decimal) | `current_value / goal_value × 100`, 2 desimal. Bisa > 100 atau negatif |
| `remaining` | string (decimal) | Sisa menuju target, minimal `0` |
| `status` | string | `in_progress` / `achieved` |
| `achieved_at` | string (RFC 3339) \| null | Pertama kali tercapai |

```json
"goal": {
  "goal_type": "balance",
  "goal_value": "1100",
  "current_value": "1064.07377049",
  "progress_percent": "96.73",
  "remaining": "35.92622951",
  "status": "in_progress",
  "achieved_at": null
}
```
| `created_at`, `updated_at` | string (RFC 3339) | |

## Objek Target

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID v7) | |
| `target_type` | string | `amount` / `percent` |
| `target_value` | string (decimal) | USD kalau `amount`, persen kalau `percent` |
| `effective_from` | string (`YYYY-MM-DD`) | Mulai berlaku sejak awal periode tanggal ini |
| `note` | string \| null | Catatan |
| `created_at` | string (RFC 3339) | |

## Aturan Validasi

| Field | Aturan |
|-------|--------|
| `name` | Wajib, maksimal 100 karakter |
| `description` | Opsional, maksimal 1000 karakter |
| `initial_balance` | Wajib, lebih dari 0 |
| `period_type` | Wajib, `daily` / `weekly` / `monthly` |
| `timezone` | Opsional (default `Asia/Jakarta`), harus nama timezone IANA yang valid |
| `start_date` | Opsional (default hari ini di `timezone`), format `YYYY-MM-DD` |
| `status` | `active` / `archived` (hanya saat update) |
| `target_type` | Wajib, `amount` / `percent` |
| `target_value` | Wajib, lebih dari 0. Untuk `percent` maksimal 100 |
| `goal_type` | Opsional, `balance` / `profit` / `percent`. Wajib kalau `goal_value` diisi |
| `goal_value` | Opsional, lebih dari 0. Wajib kalau `goal_type` diisi |
| `rules`, `strategy` | Opsional, teks bebas maks 5000 karakter |
| `note` (target) | Opsional, maksimal 500 karakter |

---

## GET /api/journals

List jurnal milik user, diurutkan dari yang terbaru dibuat.

**Query parameter**

| Nama | Tipe | Default | Keterangan |
|------|------|---------|------------|
| `page` | integer | 1 | Halaman |
| `limit` | integer | 10 | Jumlah per halaman, maksimal 100 |
| `search` | string | - | Cari di `name` dan `description` |
| `status` | string | - | Filter `active` / `archived`. Kosong = semua |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a154a4-44e2-7bd7-b5c0-4185377efe08",
      "user_id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
      "name": "Scalping BTC",
      "description": "jurnal harian",
      "initial_balance": "1000",
      "period_type": "daily",
      "timezone": "Asia/Jakarta",
      "start_date": "2026-10-01",
      "status": "active",
      "current_target": {
        "id": "01a154a4-44e7-74d3-b5d6-290dbb576de8",
        "target_type": "percent",
        "target_value": "2",
        "effective_from": "2026-10-01",
        "note": null,
        "created_at": "2026-10-19T21:50:14.113668+07:00"
      },
      "latest_target": {
        "id": "01a154a4-4677-72e7-b85b-20face0d1b58",
        "target_type": "amount",
        "target_value": "25.5",
        "effective_from": "2026-10-20",
        "note": "ganti ke nominal",
        "created_at": "2026-10-19T21:50:14.51902+07:00"
      },
      "created_at": "2026-10-19T21:50:14.113668+07:00",
      "updated_at": "2026-10-19T21:50:14.113668+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 10, "total": 1, "total_page": 1 }
}
```

Pada contoh di atas, hari ini (19 Oktober) target masih 2% (`current_target`), dan mulai besok (20 Oktober) target berubah jadi 25.5 USD (`latest_target`).

**Response 422** (`status` tidak valid)

```json
{ "code": 1002, "message": "validation failed", "data": { "status": "must be active or archived" } }
```

---

## GET /api/journals/:id

**Response 200**: `data` berisi satu objek Journal (lihat contoh di atas).

**Response 404**

```json
{ "code": 1400, "message": "journal not found", "data": null }
```

**Response 400** (id bukan UUID valid)

```json
{ "code": 1003, "message": "invalid id", "data": null }
```

---

## POST /api/journals

Membuat jurnal sekaligus target pertamanya. Target pertama berlaku sejak `start_date`.

**Request body**

| Field | Wajib |
|-------|-------|
| `name` | Ya |
| `description` | Tidak |
| `initial_balance` | Ya |
| `period_type` | Ya |
| `timezone` | Tidak (default `Asia/Jakarta`) |
| `start_date` | Tidak (default hari ini) |
| `target_type` | Ya |
| `target_value` | Ya |
| `goal_type` | Tidak (target akhir jurnal) |
| `goal_value` | Tidak (wajib kalau `goal_type` diisi) |
| `rules` | Tidak (aturan umum untuk semua runner) |
| `strategy` | Tidak (strategi umum untuk semua runner) |

```json
{
  "name": "Scalping BTC",
  "description": "jurnal harian",
  "initial_balance": 1000,
  "period_type": "daily",
  "start_date": "2026-10-01",
  "target_type": "percent",
  "target_value": 2,
  "goal_type": "balance",
  "goal_value": 5000
}
```

**Response 201**: `data` berisi objek Journal, `current_target` dan `latest_target` sama-sama berisi target pertama.

**Response 422**

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "initial_balance": "must be greater than 0",
    "name": "is required, max 100 characters",
    "period_type": "must be daily, weekly or monthly",
    "start_date": "must be a date in YYYY-MM-DD format",
    "target_value": "must be greater than 0 and at most 100",
    "timezone": "invalid timezone, example: Asia/Jakarta"
  }
}
```

**Response 400** (body bukan JSON valid, atau angka tidak valid seperti `"initial_balance": "abc"`)

```json
{ "code": 1001, "message": "invalid request body", "data": null }
```

---

## PUT /api/journals/:id

Update sebagian. Hanya field yang dikirim yang diubah. Target **tidak** diubah lewat endpoint ini, pakai `POST /api/journals/:id/targets`.

| Field | Keterangan |
|-------|------------|
| `name` | Ganti nama |
| `description` | Ganti deskripsi. Kirim `""` untuk mengosongkan |
| `status` | `active` / `archived` |
| `initial_balance` | Ganti saldo awal |
| `period_type` | Ganti tipe periode |
| `timezone` | Ganti timezone |
| `start_date` | Ganti tanggal mulai |
| `goal_type` | Ganti tipe target jurnal. Kirim `null` untuk menghapus target jurnal |
| `goal_value` | Ganti nilai target jurnal |
| `rules` | Ganti aturan umum jurnal (untuk semua runner). `""` untuk mengosongkan |
| `strategy` | Ganti strategi umum jurnal (untuk semua runner). `""` untuk mengosongkan |

**Penting:** kalau `period_type`, `timezone`, atau `start_date` berubah, batas periode ikut berubah. Riwayat target akan **diringkas jadi satu**: target terbaru, berlaku sejak `start_date`.

**Terkunci:** `initial_balance`, `period_type`, `timezone`, dan `start_date` tidak bisa diubah lagi begitu jurnal punya deposit/withdraw atau trade pertama. Kalau tetap dikirim dengan nilai berbeda, response-nya HTTP 422 code `1405`. Field lain (`name`, `description`, `status`) tetap bisa diubah.

**Contoh: arsipkan jurnal**

```json
{ "status": "archived" }
```

**Response 200**: `data` berisi objek Journal terbaru.

Response error sama seperti POST (400, 422), ditambah 404 `journal not found` (code 1400).

---

## DELETE /api/journals/:id

Soft delete.

**Response 200**

```json
{ "code": 0, "message": "journal deleted", "data": null }
```

**Response 404**

```json
{ "code": 1400, "message": "journal not found", "data": null }
```

---

## POST /api/journals/:id/regenerate-token

Membuat journal token baru. **Token lama langsung tidak berlaku**, jadi semua agent yang memakai jurnal ini harus diperbarui dengan token baru. Tidak perlu body.

**Response 200**: `data` berisi objek Journal dengan `token` baru.

```json
{ "code": 0, "message": "token regenerated", "data": { "id": "...", "token": "5t0bS4nqpV9cg08sNBc3rE7TvEVnIqr7iiPFdqrw", "...": "..." } }
```

**Response 404**

```json
{ "code": 1400, "message": "journal not found", "data": null }
```

---

## GET /api/journals/:id/targets

Riwayat target, diurutkan dari yang paling lama berlaku. Baris terakhir = target terbaru.

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a154a4-44e7-74d3-b5d6-290dbb576de8",
      "target_type": "percent",
      "target_value": "2",
      "effective_from": "2026-10-01",
      "note": null,
      "created_at": "2026-10-19T21:50:14.113668+07:00"
    },
    {
      "id": "01a154a4-4677-72e7-b85b-20face0d1b58",
      "target_type": "amount",
      "target_value": "25.5",
      "effective_from": "2026-10-20",
      "note": "ganti ke nominal",
      "created_at": "2026-10-19T21:50:14.51902+07:00"
    }
  ]
}
```

**Response 404**

```json
{ "code": 1400, "message": "journal not found", "data": null }
```

---

## POST /api/journals/:id/targets

Mengganti target. `effective_from` diisi otomatis oleh server:

| Kondisi | `effective_from` |
|---------|------------------|
| Jurnal sudah berjalan, `daily` | Besok |
| Jurnal sudah berjalan, `weekly` | Senin depan |
| Jurnal sudah berjalan, `monthly` | Tanggal 1 bulan depan |
| Jurnal belum mulai | `start_date` |

Kalau target diganti lebih dari sekali dalam periode yang sama, yang dipakai adalah yang terakhir dibuat.

**Request body**

| Field | Wajib |
|-------|-------|
| `target_type` | Ya |
| `target_value` | Ya |
| `note` | Tidak |

```json
{
  "target_type": "amount",
  "target_value": 25.5,
  "note": "ganti ke nominal"
}
```

**Response 201**

```json
{
  "code": 0,
  "message": "target created",
  "data": {
    "id": "01a154a4-4677-72e7-b85b-20face0d1b58",
    "target_type": "amount",
    "target_value": "25.5",
    "effective_from": "2026-10-20",
    "note": "ganti ke nominal",
    "created_at": "2026-10-19T21:50:14.51902+07:00"
  }
}
```

Response error: 400 (`1001`, `1003`), 404 (`1400`), 422 (`1002`).

---

# Deposit / Withdraw

Mencatat penambahan atau penarikan modal di tengah jalan.

- Mempengaruhi **total balance**, sehingga target `percent` periode **berikutnya** ikut berubah. Target periode yang sedang berjalan tidak berubah.
- **Tidak** dihitung sebagai pencapaian target (bukan profit).
- Balance tidak boleh minus. Setiap tambah / ubah / hapus transaksi dicek: kalau total balance setelah perubahan jadi minus, perubahan ditolak (`1404`) dan tidak ada yang tersimpan.
- Begitu jurnal punya transaksi, `initial_balance`, `period_type`, `timezone`, dan `start_date` terkunci.

**Contoh (target percent 2% daily, initial 1000):**

| Hari | Kejadian | Balance awal hari | Target hari itu |
|------|----------|-------------------|-----------------|
| Senin | Deposit 500 jam 10.00 | 1000 | 20 |
| Selasa | | 1500 | 30 |
| Rabu | Withdraw 200 jam 08.00 | 1500 | 30 |
| Kamis | | 1300 | 26 |

> Catatan: pengecekan balance minus memakai total balance **saat ini** (semua transaksi), bukan balance pada tanggal `occurred_at`. Jadi withdraw yang di-backdate ke tanggal sebelum deposit tetap diterima selama total akhirnya tidak minus.

## Objek Transaction

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID v7) | |
| `journal_id` | string (UUID) | |
| `type` | string | `deposit` / `withdraw` |
| `amount` | string (decimal) | Selalu positif (USD) |
| `note` | string \| null | |
| `occurred_at` | string (RFC 3339) | Waktu transaksi terjadi |
| `created_at`, `updated_at` | string (RFC 3339) | |

## Aturan Validasi

| Field | Aturan |
|-------|--------|
| `type` | Wajib, `deposit` / `withdraw` |
| `amount` | Wajib, lebih dari 0 |
| `note` | Opsional, maksimal 500 karakter |
| `occurred_at` | Opsional (default sekarang), format RFC 3339. Tidak boleh sebelum `start_date` jurnal (di timezone jurnal) dan tidak boleh di masa depan |

## GET /api/journals/:id/transactions

Diurutkan dari `occurred_at` terbaru.

**Query parameter**

| Nama | Tipe | Default | Keterangan |
|------|------|---------|------------|
| `page` | integer | 1 | |
| `limit` | integer | 10 | Maksimal 100 |
| `type` | string | - | `deposit` / `withdraw` |
| `from` | string (`YYYY-MM-DD`) | - | Dari tanggal ini (inklusif, timezone jurnal) |
| `to` | string (`YYYY-MM-DD`) | - | Sampai tanggal ini (inklusif, timezone jurnal) |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a154ae-85b6-7164-a781-1e9f83267847",
      "journal_id": "01a154ae-8472-7f81-8c63-b90e32f2650e",
      "type": "deposit",
      "amount": "500",
      "note": "top up",
      "occurred_at": "2026-10-19T22:01:26.069348+07:00",
      "created_at": "2026-10-19T22:01:26.068893+07:00",
      "updated_at": "2026-10-19T22:01:26.068893+07:00"
    }
  ],
  "meta": { "page": 1, "limit": 10, "total": 1, "total_page": 1 }
}
```

**Response 422** (filter tidak valid)

```json
{ "code": 1002, "message": "validation failed", "data": { "from": "must be a date in YYYY-MM-DD format" } }
```

## POST /api/journals/:id/transactions

```json
{
  "type": "withdraw",
  "amount": 1500,
  "note": "tarik profit",
  "occurred_at": "2026-10-10T09:00:00+07:00"
}
```

**Response 201**: `data` berisi objek Transaction.

**Response 422** (balance jadi minus). `balance_after` = total balance kalau perubahan ini disimpan.

```json
{ "code": 1404, "message": "insufficient balance", "data": { "balance_after": "-500" } }
```

**Response 422** (validasi)

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": { "occurred_at": "must not be before journal start_date (2026-10-01)" }
}
```

## PUT /api/journals/:id/transactions/:txId

Update sebagian: `type`, `amount`, `note` (`""` untuk mengosongkan), `occurred_at`. Balance dicek ulang setelah perubahan (`1404`).

**Response 200**: `data` berisi objek Transaction terbaru.

**Response 404**

```json
{ "code": 1401, "message": "transaction not found", "data": null }
```

## DELETE /api/journals/:id/transactions/:txId

Soft delete. Bisa ditolak `1404` kalau menghapus deposit membuat balance minus (misal deposit itu sudah dipakai untuk withdraw).

**Response 200**

```json
{ "code": 0, "message": "transaction deleted", "data": null }
```

---

# Runner di Jurnal

Menentukan runner mana yang boleh mengirim trade ke jurnal ini (endpoint trade untuk runner ada di [runner-guide.md](runner-guide.md)).

- Hanya runner **milik user yang sama** dengan pemilik jurnal yang bisa ditambahkan. Runner milik user lain dianggap tidak ada (`1300`).
- Satu runner boleh join banyak jurnal, satu jurnal boleh punya banyak runner.
- Runner yang sudah dihapus otomatis tidak muncul di list. Runner yang dinonaktifkan tetap muncul (`is_active: false`).
- Runner yang dikeluarkan tidak bisa lagi menambah/mengubah trade di jurnal ini, tapi trade yang pernah dibuatnya tetap ada.

### Alur di frontend (form "Tambah runner ke jurnal")

```
1. GET    /api/journals/:id/runners/options   → isi dropdown (runner yang belum join)
2. POST   /api/journals/:id/runners           → body { "runner_id": "<id dari dropdown>" }
3. GET    /api/journals/:id/runners           → refresh daftar runner yang sudah join
                                                 (response POST juga sudah berisi daftar terbaru)
4. DELETE /api/journals/:id/runners/:runnerId → tombol "keluarkan" di daftar
5. GET    /api/journals/:id/runners/:runnerId/agent.md → tombol "Download AGENT.md" di tiap runner
```

Kalau user belum punya runner sama sekali (dropdown kosong), arahkan ke halaman buat runner (`POST /api/runners`, lihat [runners.md](runners.md)).

## GET /api/journals/:id/runners/options

Pilihan runner untuk **dropdown** tambah runner ke jurnal.

- Hanya runner milik user yang login, belum dihapus, dan **belum join** ke jurnal ini (yang sudah join otomatis tidak muncul, jadi tidak akan kena `1402`).
- Diurutkan berdasarkan nama (tidak case-sensitive), tanpa pagination.
- Runner nonaktif tetap muncul dengan `is_active: false`. Saran UI: tampilkan abu-abu / beri label "nonaktif". Runner nonaktif tetap boleh ditambahkan, tapi tokennya ditolak sampai diaktifkan lagi.
- Token runner **tidak** ikut dikirim.

**Query parameter**

| Nama | Tipe | Keterangan |
|------|------|------------|
| `search` | string | Opsional, cari di nama runner (tidak case-sensitive). Cocok untuk dropdown yang bisa diketik |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    { "id": "01a15501-811a-7b5c-bede-36e8a5aa3c1b", "name": "Bot Swing", "is_active": false },
    { "id": "01a15501-7ebf-74c9-96e8-b1bde9ffca28", "name": "bot Zeta", "is_active": true }
  ]
}
```

`data` berisi `[]` kalau tidak ada runner yang bisa ditambahkan.

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID) | Dipakai sebagai `runner_id` di `POST /api/journals/:id/runners` |
| `name` | string | Label dropdown |
| `is_active` | boolean | `false` = runner sedang dinonaktifkan |

**Response 404**

```json
{ "code": 1400, "message": "journal not found", "data": null }
```

## Objek Joined Runner

| Field | Tipe | Keterangan |
|-------|------|------------|
| `runner_id` | string (UUID) | |
| `name` | string | Nama runner |
| `is_active` | boolean | |
| `expires_at` | string (RFC 3339) \| null | |
| `last_used_at` | string (RFC 3339) \| null | |
| `rules` | string \| null | Aturan khusus runner ini di jurnal ini |
| `strategy` | string \| null | Strategi khusus runner ini di jurnal ini |
| `joined_at` | string (RFC 3339) | Kapan runner ditambahkan ke jurnal |

Token runner tidak ikut ditampilkan di sini (lihat di [runners.md](runners.md)).

## GET /api/journals/:id/runners

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "runner_id": "01a154ae-9a8c-7075-b20f-9ffbd510a267",
      "name": "Bot A",
      "is_active": true,
      "expires_at": null,
      "last_used_at": null,
      "joined_at": "2026-10-19T22:01:32.77787+07:00"
    }
  ]
}
```

## POST /api/journals/:id/runners

`runner_id` diambil dari dropdown [`GET /api/journals/:id/runners/options`](#get-apijournalsidrunnersoptions).

```json
{
  "runner_id": "01a154ae-9a8c-7075-b20f-9ffbd510a267",
  "rules": "Leverage maks 5x",
  "strategy": "Mean reversion timeframe 15m, TP 1:1.5"
}
```

| Field | Wajib | Keterangan |
|-------|-------|------------|
| `runner_id` | Ya | Dari dropdown `GET /api/journals/:id/runners/options` |
| `rules` | Tidak | Aturan khusus runner ini di jurnal ini, maks 5000 karakter |
| `strategy` | Tidak | Strategi khusus runner ini di jurnal ini, maks 5000 karakter |

`rules` & `strategy` bisa diubah nanti lewat `PUT /api/journals/:id/runners/:runnerId`.

**Response 201**: `data` berisi list runner yang join (terbaru).

**Response 409** (sudah join)

```json
{ "code": 1402, "message": "runner already joined this journal", "data": null }
```

**Response 404** (runner tidak ada / milik user lain)

```json
{ "code": 1300, "message": "runner not found", "data": null }
```

**Response 422**

```json
{ "code": 1002, "message": "validation failed", "data": { "runner_id": "is required" } }
```

## PUT /api/journals/:id/runners/:runnerId

Mengubah instruksi **khusus runner ini** di jurnal ini. Hanya field yang dikirim yang diubah. Kirim `""` untuk mengosongkan.

| Field | Keterangan |
|-------|------------|
| `rules` | Aturan khusus runner ini, maks 5000 karakter |
| `strategy` | Strategi khusus runner ini, maks 5000 karakter |

```json
{
  "rules": "Leverage maks 5x",
  "strategy": "Mean reversion timeframe 15m, TP 1:1.5"
}
```

**Response 200**: `data` berisi objek Joined Runner terbaru.

```json
{
  "code": 0,
  "message": "runner updated",
  "data": {
    "runner_id": "01a154ae-9a8c-7075-b20f-9ffbd510a267",
    "name": "Bot B",
    "is_active": true,
    "expires_at": null,
    "last_used_at": null,
    "rules": "Leverage maks 5x",
    "strategy": "Mean reversion timeframe 15m, TP 1:1.5",
    "joined_at": "2026-10-19T22:01:32.77787+07:00"
  }
}
```

**Response 404** (runner tidak join ke jurnal ini)

```json
{ "code": 1403, "message": "runner not in this journal", "data": null }
```

**Response 422**

```json
{ "code": 1002, "message": "validation failed", "data": { "rules": "max 5000 characters" } }
```

---

## GET /api/journals/:id/runners/:runnerId/agent.md

Download file **`AGENT.md`** untuk runner ini di jurnal ini. File ini siap diberikan ke agent (misal AI agent / bot) supaya agent langsung tahu cara memakai API.

- Wajib login (token session pemilik jurnal), seperti endpoint lain. Runner harus sudah join ke jurnal.
- Isi file (bahasa Inggris):
  - **Kredensial**: Base URL API, Runner ID, **token runner**, dan **token jurnal**, plus peringatan bahwa file berisi token rahasia.
  - **Instruksi trading**: tidak disalin ke file, tapi diarahkan untuk selalu diambil dari `GET /api/runner/journal` (field `instructions`), jadi perubahan rules / strategy langsung terbaca agent tanpa download ulang.
  - Siklus kerja, daftar endpoint runner, contoh body buka / ubah / tutup trade, aturan wajib (guard 1 trade open per pair + position, tidak boleh langsung closed, dll), dan tabel penanganan error.
- Base URL diambil dari env `PUBLIC_API_URL` (contoh `https://api.domain.com/api`). Kalau kosong, memakai alamat request yang masuk.
- Kalau token runner atau token jurnal di-regenerate, file lama tidak berlaku lagi: download ulang.

**Response 200**

```
Content-Type: text/markdown; charset=utf-8
Content-Disposition: attachment; filename="AGENT.md"
Cache-Control: no-store
```

Body berupa teks Markdown, contoh awalnya:

```markdown
# AGENT.md — Scalping BTC × Bot Swing

You are a trading agent that records trades into the Autojurnal journal **Scalping BTC** as the runner **Bot Swing**. Follow this file exactly.

## Credentials

> **SECRET.** This file contains access tokens. Never print, log, commit, or share them. ...

| Key | Value |
|-----|-------|
| Base URL | `https://api.domain.com/api` |
| Runner ID | `01a15514-9b8f-76b9-b688-21cab4611660` |
| Runner token | `TyOsiQgyXChLMRwh8ixdXNFQEjI2bA3D4lzoUKzD` |
| Journal token | `b04s2AepIrsEN5bcCKNiOA678YjcntrY6E9PMfOH` |
...
```

Template lengkapnya ada di `internal/module/journals/templates/agent.md.tmpl`.

**Contoh di frontend**: karena butuh header `Authorization`, download lewat `fetch` lalu simpan sebagai file:

```js
const res = await fetch(`${API}/journals/${journalId}/runners/${runnerId}/agent.md`, {
  headers: { Authorization: `Bearer ${sessionToken}` },
});
if (!res.ok) throw await res.json();          // error tetap format JSON biasa
const blob = await res.blob();
const url = URL.createObjectURL(blob);
Object.assign(document.createElement("a"), { href: url, download: "AGENT.md" }).click();
URL.revokeObjectURL(url);
```

**Response error** (format JSON biasa)

| HTTP | Code | Kapan |
|------|------|-------|
| 401 | `1201` | Belum login |
| 404 | `1400` | Jurnal tidak ada / milik user lain |
| 404 | `1403` | Runner belum join ke jurnal ini |
| 400 | `1003` | `id` / `runnerId` bukan UUID valid |

---

## DELETE /api/journals/:id/runners/:runnerId

**Response 200**

```json
{ "code": 0, "message": "runner removed", "data": null }
```

**Response 404**

```json
{ "code": 1403, "message": "runner not in this journal", "data": null }
```
