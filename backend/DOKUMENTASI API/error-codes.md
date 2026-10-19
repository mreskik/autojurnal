# Error Codes

Semua response API memakai format:

```json
{
  "code": 0,
  "message": "...",
  "data": null
}
```

- `code` = `0` artinya sukses.
- `code` >= `1000` artinya error.
- Untuk logika di client, pakai `code`, bukan `message`. Teks `message` bisa berubah, sedangkan `code` tetap.

Definisi konstanta di kode: [internal/response/codes.go](../internal/response/codes.go)

## Pembagian Range

| Range | Modul | Keterangan |
|-------|-------|------------|
| 0 | - | Sukses |
| 1000 - 1099 | Umum | Error yang bisa muncul di semua endpoint |
| 1100 - 1199 | Master Users | Error khusus endpoint `/api/users` |
| 1200 - 1299 | Auth | Login dan session (`/api/auth`, plus semua endpoint yang wajib login) |
| 1300 - 1399 | Runners | Manajemen runner (`/api/runners`) dan token runner (`/api/runner`) |
| 1400 - 1499 | Journals | Jurnal dan target (`/api/journals`) |
| 1500 - 1599 | Trades | Trade di dalam jurnal (`/api/journals/:id/trades`) |

## Daftar Lengkap

### Sukses

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Isi `data` |
|------|-----------|-------------|---------|---------------|------------|
| 0 | `CodeOK` | 200 OK / 201 Created | bervariasi (`success`, `user created`, `user updated`, `user deleted`, `ok`) | Request berhasil. 201 dipakai saat data baru dibuat (POST) | Objek / array hasil request, atau `null` (misal DELETE) |

### Umum (1000 - 1099)

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Isi `data` | Saran Penanganan di Client |
|------|-----------|-------------|---------|---------------|------------|----------------------------|
| 1000 | `CodeInternalError` | 500 Internal Server Error | `internal server error` | Error tak terduga di server, misal query database gagal. Detailnya hanya dicatat di log server, tidak dikirim ke client | `null` | Tampilkan pesan umum ("Terjadi kesalahan, coba lagi"). Laporkan ke tim backend kalau terus terjadi |
| 1001 | `CodeInvalidRequestBody` | 400 Bad Request | `invalid request body` | Body bukan JSON yang valid, atau tipe field salah (misal `"name": 123`) | `null` | Bug di client: cek format JSON dan header `Content-Type: application/json` |
| 1002 | `CodeValidationFailed` | 422 Unprocessable Entity | `validation failed` | JSON valid, tapi isi field tidak lolos aturan validasi (wajib diisi, panjang, format) | Objek `{ "<nama_field>": "<pesan>" }`, satu entri untuk setiap field yang salah | Tampilkan pesan di bawah input form sesuai key field |
| 1003 | `CodeInvalidParam` | 400 Bad Request | `invalid id` | Parameter `:id` di URL bukan UUID valid, atau UUID nol (`00000000-0000-0000-0000-000000000000`) | `null` | Bug di client: pastikan id yang dikirim adalah UUID dari response API |
| 1004 | `CodeNotFound` | 404 Not Found | `route not found` | URL di bawah `/api` tidak terdaftar | `null` | Bug di client: cek path dan method endpoint |
| 1005 | `CodeDatabaseUnavailable` | 503 Service Unavailable | `database unavailable` | Database tidak bisa dihubungi (saat ini hanya dari `GET /api/health`) | `null` | Tampilkan status layanan sedang gangguan, coba lagi beberapa saat |
| 1006 | `CodeTooManyRequests` | 429 Too Many Requests | `too many requests, try again later` | Kena rate limit: terlalu banyak login gagal (per IP + username), terlalu banyak request dari satu IP, atau dari satu user. Detail batas di [README.md](README.md#rate-limit) | `null`. Header `Retry-After` berisi sisa detik sampai boleh mencoba lagi | Tampilkan "Terlalu banyak percobaan, coba lagi dalam X menit" pakai nilai `Retry-After`. Jangan retry otomatis sebelum waktunya |

### Master Users (1100 - 1199)

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Endpoint | Isi `data` | Saran Penanganan di Client |
|------|-----------|-------------|---------|---------------|----------|------------|----------------------------|
| 1100 | `CodeUserNotFound` | 404 Not Found | `user not found` | User dengan id tersebut tidak ada atau sudah dihapus (soft delete) | `GET`, `PUT`, `DELETE` `/api/users/:id` | `null` | Tampilkan "User tidak ditemukan", refresh list |
| 1101 | `CodeUsernameTaken` | 409 Conflict | `username already used` | Username sudah dipakai user lain yang aktif. Pengecekan tidak membedakan huruf besar/kecil (`Admin` = `admin`) | `POST /api/users`, `PUT /api/users/:id` | `{ "username": "username already used" }` | Tampilkan pesan di input username |
| 1102 | `CodeEmailTaken` | 409 Conflict | `email already used` | Email sudah dipakai user lain yang aktif. Pengecekan tidak membedakan huruf besar/kecil | `POST /api/users`, `PUT /api/users/:id` | `{ "email": "email already used" }` | Tampilkan pesan di input email |

### Auth (1200 - 1299)

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Endpoint | Isi `data` | Saran Penanganan di Client |
|------|-----------|-------------|---------|---------------|----------|------------|----------------------------|
| 1200 | `CodeInvalidCredentials` | 401 Unauthorized | `invalid username or password` | Username tidak ada atau password salah. Pesannya sengaja sama untuk dua kasus ini supaya tidak bisa dipakai menebak username yang terdaftar | `POST /api/auth/login` | `null` | Tampilkan "Username atau password salah" di form login |
| 1201 | `CodeUnauthorized` | 401 Unauthorized | `unauthorized` | Header `Authorization: Bearer <token>` tidak dikirim, token salah, session expired, sudah logout/di-revoke, atau user-nya sudah dihapus | Semua endpoint yang wajib login (`/api/users/*`, `/api/auth/*` kecuali login) | `null` | Hapus token di client, arahkan ke halaman login |
| 1202 | `CodeSessionNotFound` | 404 Not Found | `session not found` | Session dengan id tersebut tidak ada, sudah mati, atau milik user lain | `DELETE /api/auth/sessions/:id` | `null` | Refresh list session |

### Runners (1300 - 1399)

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Endpoint | Isi `data` | Saran Penanganan di Client |
|------|-----------|-------------|---------|---------------|----------|------------|----------------------------|
| 1300 | `CodeRunnerNotFound` | 404 Not Found | `runner not found` | Runner tidak ada, sudah dihapus, atau milik user lain | `GET`, `PUT`, `DELETE` `/api/runners/:id`, `POST /api/runners/:id/regenerate-token` | `null` | Tampilkan "Runner tidak ditemukan", refresh list |
| 1301 | `CodeInvalidRunnerToken` | 401 Unauthorized | `missing or invalid runner token` | Header `Authorization: Bearer <token runner>` tidak dikirim, token salah, token sudah di-regenerate, runner dihapus, atau user pemiliknya dihapus | Semua `/api/runner/*` | `null` | (Agent) Hentikan request, minta token baru ke pemilik runner |
| 1302 | `CodeRunnerTokenExpired` | 401 Unauthorized | `runner token expired` | Waktu sekarang sudah melewati `expires_at` runner | Semua `/api/runner/*` | `null` | (Agent) Minta pemilik memperpanjang `expires_at` atau membuat runner baru |
| 1303 | `CodeRunnerInactive` | 403 Forbidden | `runner inactive` | Runner dinonaktifkan pemiliknya (`is_active: false`) | Semua `/api/runner/*` | `null` | (Agent) Hentikan request sampai runner diaktifkan lagi |

### Journals (1400 - 1499)

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Endpoint | Isi `data` | Saran Penanganan di Client |
|------|-----------|-------------|---------|---------------|----------|------------|----------------------------|
| 1400 | `CodeJournalNotFound` | 404 Not Found | `journal not found` | Jurnal tidak ada, sudah dihapus, atau milik user lain | Semua `/api/journals/:id` dan turunannya (`/targets`, `/transactions`, `/runners`, `/trades`, `/regenerate-token`) | `null` | Tampilkan "Jurnal tidak ditemukan", refresh list |
| 1401 | `CodeTransactionNotFound` | 404 Not Found | `transaction not found` | Deposit/withdraw tidak ada, sudah dihapus, atau bukan milik jurnal ini | `PUT`, `DELETE` `/api/journals/:id/transactions/:txId` | `null` | Refresh list transaksi |
| 1402 | `CodeRunnerAlreadyJoined` | 409 Conflict | `runner already joined this journal` | Runner sudah terdaftar di jurnal ini | `POST /api/journals/:id/runners` | `null` | Tampilkan "Runner sudah ada di jurnal ini" |
| 1403 | `CodeRunnerNotInJournal` | 404 Not Found | `runner not in this journal` | Runner yang mau dikeluarkan tidak terdaftar di jurnal ini | `DELETE /api/journals/:id/runners/:runnerId` | `null` | Refresh list runner jurnal |
| 1404 | `CodeInsufficientBalance` | 422 Unprocessable Entity | `insufficient balance` | Tambah/ubah/hapus deposit/withdraw membuat total balance jurnal jadi minus. Perubahan tidak disimpan | `POST`, `PUT`, `DELETE` `/api/journals/:id/transactions` | `{ "balance_after": "-500" }` (balance kalau perubahan disimpan) | Tampilkan "Saldo tidak cukup" beserta saldo yang tersedia |
| 1405 | `CodeJournalLocked` | 422 Unprocessable Entity | `journal settings are locked` | Mengubah `initial_balance`, `period_type`, `timezone`, atau `start_date` padahal jurnal sudah punya deposit/withdraw atau trade | `PUT /api/journals/:id` | Objek `{ "<field>": "cannot be changed after journal has transactions or trades" }` untuk setiap field yang ditolak | Nonaktifkan input field tersebut di form edit kalau jurnal sudah punya aktivitas |
| 1406 | `CodeJournalArchived` | 422 Unprocessable Entity | `journal is archived` | Membuat trade baru di jurnal berstatus `archived` | `POST /api/journals/:id/trades`, `POST /api/runner/trades` | `null` | Tawarkan aktifkan jurnal lagi (`status: active`) |
| 1407 | `CodeInvalidJournalToken` | 401 Unauthorized | `missing or invalid journal token` | Header `X-Journal-Token` tidak dikirim, token salah, sudah di-regenerate, jurnal dihapus, atau jurnal milik user lain (bukan pemilik runner) | Endpoint jurnal & trade runner (`/api/runner/journal`, `/api/runner/trades/*`) | `null` | (Agent) Hentikan request, minta token jurnal terbaru ke pemilik jurnal |
| 1408 | `CodeRunnerNotJoined` | 403 Forbidden | `runner has not joined this journal` | Token runner & token jurnal valid, tapi runner belum di-join ke jurnal (atau sudah dikeluarkan) | Endpoint jurnal & trade runner | `null` | (Agent) Minta pemilik jurnal menambahkan runner lewat `POST /api/journals/:id/runners` |

### Trades (1500 - 1599)

| Code | Konstanta | HTTP Status | Message | Kapan Terjadi | Endpoint | Isi `data` | Saran Penanganan di Client |
|------|-----------|-------------|---------|---------------|----------|------------|----------------------------|
| 1500 | `CodeTradeNotFound` | 404 Not Found | `trade not found` | Trade tidak ada, sudah dihapus, atau bukan milik jurnal ini. Untuk runner: trade tidak ada di jurnal dari token jurnal | `GET`, `PUT`, `DELETE` `/api/journals/:id/trades/:tradeId`, `POST .../close`, dan `/api/runner/trades/:tradeId/*` | `null` | Refresh list trade |
| 1501 | `CodeTradeAlreadyClosed` | 422 Unprocessable Entity | `trade already closed` | Menutup trade yang sudah closed, atau mengubah field selain `close_note` pada trade closed | `POST .../close`, `PUT /api/journals/:id/trades/:tradeId` | Saat close: `null`. Saat update: objek `{ "<field>": "cannot be changed after trade is closed, only close_note can be changed" }` | Kunci form trade closed, sisakan input catatan saja |
| 1502 | `CodeTradeNotOwned` | 403 Forbidden | `trade not owned by this runner` | Runner mencoba mengubah / menutup trade yang bukan buatannya (input manual pemilik atau runner lain). Runner tetap boleh melihat trade tersebut | `PUT /api/runner/trades/:tradeId`, `POST /api/runner/trades/:tradeId/close` | `null` | (Agent) Jangan retry, hanya ubah trade dengan `runner_id` milik sendiri |
| 1503 | `CodeOpenTradeExists` | 409 Conflict | `runner already has an open trade for this pair and position` | Runner membuka trade (atau mengubah pair / position trade open) padahal runner itu sudah punya trade open dengan pair + position yang sama di jurnal ini. Input manual user tidak dibatasi | `POST /api/runner/trades`, `PUT /api/runner/trades/:tradeId`, `PUT /api/journals/:id/trades/:tradeId` (trade milik runner) | Objek trade open yang sudah ada (bisa `null` kalau tertahan constraint database saat request bersamaan) | (Agent) Anggap sukses dan pakai `data.id`, biasanya ini hasil retry |

## Detail Pesan Validasi (1002)

Isi `data` untuk code `1002`, per field:

### Master Users

| Field | Pesan | Aturan |
|-------|-------|--------|
| `username` | `must be 3-50 characters: letters, numbers, underscore or dot` | Wajib, 3-50 karakter, hanya `a-z`, `A-Z`, `0-9`, `_`, `.` |
| `name` | `is required, max 100 characters` | Wajib, maksimal 100 karakter |
| `email` | `invalid email format` | Kalau diisi, harus format email valid |
| `email` | `max 255 characters` | Kalau diisi, maksimal 255 karakter |
| `password` | `must be 8-72 characters` | Wajib saat create, 8-72 karakter (batas 72 karena bcrypt) |

### Journals

| Field | Pesan | Aturan |
|-------|-------|--------|
| `name` | `is required, max 100 characters` | Wajib, maksimal 100 karakter |
| `description` | `max 1000 characters` | Opsional |
| `initial_balance` | `must be greater than 0` | Wajib, > 0 |
| `period_type` | `must be daily, weekly or monthly` | Wajib |
| `timezone` | `invalid timezone, example: Asia/Jakarta` | Nama timezone IANA |
| `start_date` | `must be a date in YYYY-MM-DD format` | Format tanggal |
| `status` | `must be active or archived` | Update dan filter list |
| `target_type` | `must be amount or percent` | Wajib saat buat jurnal / target |
| `target_value` | `must be greater than 0` | Untuk `amount` |
| `target_value` | `must be greater than 0 and at most 100` | Untuk `percent` |
| `goal_type` | `must be balance, profit or percent` | Target jurnal |
| `goal_type` | `is required when goal_value is set` | Target jurnal |
| `goal_value` | `is required when goal_type is set` | Target jurnal |
| `goal_value` | `must be greater than 0` | Target jurnal |
| `note` | `max 500 characters` | Catatan target / transaksi |
| `type` | `must be deposit or withdraw` | Transaksi (body dan filter list) |
| `amount` | `must be greater than 0` | Transaksi |
| `occurred_at` | `must not be before journal start_date (YYYY-MM-DD)` | Transaksi |
| `occurred_at` | `must not be in the future` | Transaksi |
| `from` / `to` | `must be a date in YYYY-MM-DD format` | Filter list transaksi |
| `runner_id` | `is required` | Tambah runner ke jurnal |
| `rules`, `strategy` | `max 5000 characters` | Instruksi jurnal (`POST`/`PUT /api/journals/:id`) dan instruksi runner (`POST /api/journals/:id/runners`, `PUT /api/journals/:id/runners/:runnerId`) |

### Trades

| Field | Pesan | Aturan |
|-------|-------|--------|
| `pair` | `is required, 2-30 characters: letters, numbers, / . _ -  (example: BTCUSDT)` | Wajib |
| `position` | `must be long or short` | Wajib (juga filter list) |
| `margin`, `entry_price`, `exit_price`, `stop_loss`, `take_profit` | `must be greater than 0` | |
| `leverage` | `must be between 1 and 500` | Wajib |
| `take_profit` | `must be above entry_price for long position` | Long |
| `take_profit` | `must be below entry_price for short position` | Short |
| `exchange` | `max 30 characters` | |
| `open_reason`, `close_note` | `max 1000 characters` | |
| `opened_at` | `must not be before journal start_date (YYYY-MM-DD)` / `must not be in the future` | |
| `exit_price` | `is required` | Saat close |
| `close_reason` | `must be take_profit, stop_loss, manual, liquidation or trailing_stop` | Saat close |
| `fee` | `must not be negative` | Saat close |
| `closed_at` | `must not be before opened_at` / `must not be in the future` | Saat close |
| `close_note` | `can only be set when closing the trade` | Update trade yang masih open |
| `exit_price`, `close_reason`, `close_note`, `pnl`, `fee`, `closed_at` | `not allowed when opening a trade as runner, open the trade first then close it` | `POST /api/runner/trades`: runner harus buka trade dulu lalu close (tidak bisa langsung closed) |
| `status` | `must be open or closed` | Filter list |
| `source` | `must be manual or runner` | Filter list |

### Auth (login)

| Field | Pesan | Aturan |
|-------|-------|--------|
| `username` | `is required` | Wajib diisi |
| `password` | `is required` | Wajib diisi |

## Contoh Response

**1001: body tidak valid**

```json
{
  "code": 1001,
  "message": "invalid request body",
  "data": null
}
```

**1002: validasi gagal (beberapa field sekaligus)**

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "name": "is required, max 100 characters",
    "password": "must be 8-72 characters",
    "username": "must be 3-50 characters: letters, numbers, underscore or dot"
  }
}
```

**1003: id tidak valid**

```json
{
  "code": 1003,
  "message": "invalid id",
  "data": null
}
```

**1100: user tidak ditemukan**

```json
{
  "code": 1100,
  "message": "user not found",
  "data": null
}
```

**1101: username sudah dipakai**

```json
{
  "code": 1101,
  "message": "username already used",
  "data": {
    "username": "username already used"
  }
}
```

## Urutan Pengecekan

Kalau satu request punya beberapa masalah sekaligus, hanya error pertama yang dikembalikan, dengan urutan:

1. `1006` rate limit per IP (semua endpoint)
2. `1201` belum login / token tidak valid (untuk endpoint yang wajib login), atau `1301` / `1302` / `1303` untuk endpoint runner (`/api/runner/*`), lalu `1407` / `1408` untuk endpoint jurnal & trade runner
3. `1006` rate limit per user / per runner, atau rate limit login gagal (`POST /api/auth/login`)
4. `1003` id di URL tidak valid
5. `1001` body bukan JSON valid
6. `1002` validasi field (semua field yang salah dikembalikan sekaligus)
7. `1100` user / `1300` runner / `1400` jurnal tidak ditemukan (khusus `PUT` dan `POST /api/journals/:id/targets`)
8. `1101` username duplikat
9. `1102` email duplikat

## Menambah Error Code Baru

1. Tambahkan konstanta di [internal/response/codes.go](../internal/response/codes.go) sesuai range modulnya. Modul baru mengambil range ratusan berikutnya yang belum dipakai/dipesan (saat ini 1600).

2. Petakan error-nya di `handleError` pada handler modul tersebut.
3. Tambahkan barisnya ke dokumen ini.
4. Code yang sudah dipakai jangan diubah atau dipakai ulang untuk arti lain, karena client mungkin sudah bergantung padanya.
