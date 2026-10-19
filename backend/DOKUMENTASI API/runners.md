# Runners

Runner adalah klien mesin (agent AI, bot, script, atau pihak lain) yang diberi **token** untuk mengakses API atas nama user pemiliknya, misalnya untuk insert jurnal.

Dokumen ini berisi **manajemen runner dari aplikasi** (`/api/runners`): membuat runner, melihat token, menonaktifkan, regenerate token, dan menghapus. Semua endpoint di sini memakai token session user yang login (`Authorization: Bearer <token session>`, lihat [auth.md](auth.md)).

Endpoint yang dipakai **agent** dengan token runner (`/api/runner/*`) didokumentasikan terpisah di [runner-guide.md](runner-guide.md).

Format response umum ada di [README.md](README.md), daftar error code di [error-codes.md](error-codes.md).

## Aturan

- Setiap runner dimiliki satu user (user yang membuatnya). User hanya bisa melihat dan mengelola runner miliknya sendiri. Runner milik user lain dianggap tidak ada (`1300`).
- Token berupa string acak 40 karakter alfanumerik (`a-z`, `A-Z`, `0-9`), dibuat dengan `crypto/rand`.
- Token **disimpan apa adanya** di database dan selalu ikut tampil di response manajemen runner. Perlakukan token seperti password: jangan dibagikan selain ke agent yang memakainya.
- Token runner tidak berlaku kalau: runner dinonaktifkan (`is_active: false`), lewat `expires_at`, token di-regenerate, runner dihapus, atau user pemiliknya dihapus.

## Objek Runner

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID v7) | ID runner |
| `user_id` | string (UUID) | Pemilik runner |
| `name` | string | Nama runner |
| `description` | string \| null | Keterangan |
| `token` | string | Token untuk agent |
| `is_active` | boolean | `false` = token ditolak (`1303`) |
| `expires_at` | string (RFC 3339) \| null | Batas berlaku token. `null` = berlaku selamanya |
| `last_used_at` | string (RFC 3339) \| null | Terakhir token dipakai. Diperbarui paling sering satu kali per menit |
| `created_at` | string (RFC 3339) | |
| `updated_at` | string (RFC 3339) | |

## Aturan Validasi

| Field | Aturan |
|-------|--------|
| `name` | Wajib, maksimal 100 karakter |
| `description` | Opsional, maksimal 500 karakter |
| `expires_at` | Opsional, format RFC 3339 (contoh `2030-01-01T00:00:00Z`), harus di masa depan |

---

# Manajemen Runner

Semua endpoint di bagian ini **wajib login** (token session). Tanpa token session yang valid: HTTP 401, code `1201`.

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/api/runners` | List runner milik user |
| GET | `/api/runners/:id` | Detail runner |
| POST | `/api/runners` | Buat runner + generate token |
| PUT | `/api/runners/:id` | Ubah runner (sebagian field) |
| POST | `/api/runners/:id/regenerate-token` | Ganti token |
| DELETE | `/api/runners/:id` | Hapus runner (soft delete) |

## GET /api/runners

List runner milik user yang login, diurutkan dari yang paling lama dibuat.

**Query parameter**

| Nama | Tipe | Default | Keterangan |
|------|------|---------|------------|
| `page` | integer | 1 | Halaman |
| `limit` | integer | 10 | Jumlah per halaman, maksimal 100 |
| `search` | string | - | Cari di `name` dan `description` |

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a15475-2a7d-797f-b5fb-66ebdebdaabf",
      "user_id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
      "name": "Agent Scalping",
      "description": "bot AI",
      "token": "WB4eyPg9tSaMKvkil8AP3lErX4DS2ztlQNOTn9tU",
      "is_active": true,
      "expires_at": null,
      "last_used_at": null,
      "created_at": "2026-10-19T20:58:47.165232+07:00",
      "updated_at": "2026-10-19T20:58:47.165232+07:00"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 1,
    "total_page": 1
  }
}
```

---

## GET /api/runners/:id

**Response 200**: `data` berisi satu objek runner (lihat contoh di atas).

**Response 404** (tidak ada, sudah dihapus, atau milik user lain)

```json
{ "code": 1300, "message": "runner not found", "data": null }
```

**Response 400** (id bukan UUID valid)

```json
{ "code": 1003, "message": "invalid id", "data": null }
```

---

## POST /api/runners

Token di-generate otomatis oleh server. `user_id` diisi otomatis dari user yang login.

**Request body**

| Field | Wajib |
|-------|-------|
| `name` | Ya |
| `description` | Tidak |
| `expires_at` | Tidak (kosong/`null` = berlaku selamanya) |

```json
{
  "name": "Agent Scalping",
  "description": "bot AI",
  "expires_at": "2030-01-01T00:00:00Z"
}
```

**Response 201**

```json
{
  "code": 0,
  "message": "runner created",
  "data": {
    "id": "01a15475-2a7d-797f-b5fb-66ebdebdaabf",
    "user_id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
    "name": "Agent Scalping",
    "description": "bot AI",
    "token": "WB4eyPg9tSaMKvkil8AP3lErX4DS2ztlQNOTn9tU",
    "is_active": true,
    "expires_at": "2030-01-01T07:00:00+07:00",
    "last_used_at": null,
    "created_at": "2026-10-19T20:58:47.165232+07:00",
    "updated_at": "2026-10-19T20:58:47.165232+07:00"
  }
}
```

**Response 422**

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "expires_at": "must be in the future",
    "name": "is required, max 100 characters"
  }
}
```

**Response 400** (body bukan JSON valid / format tanggal salah)

```json
{ "code": 1001, "message": "invalid request body", "data": null }
```

---

## PUT /api/runners/:id

Update sebagian. Hanya field yang dikirim yang diubah.

| Field | Keterangan |
|-------|------------|
| `name` | Ganti nama |
| `description` | Ganti keterangan. Kirim `""` untuk mengosongkan |
| `is_active` | `false` untuk menonaktifkan token tanpa menghapus runner, `true` untuk mengaktifkan lagi |
| `expires_at` | Ganti batas berlaku. Kirim `null` untuk membuat token berlaku selamanya |

**Contoh: nonaktifkan runner**

```json
{ "is_active": false }
```

**Contoh: hapus batas berlaku**

```json
{ "expires_at": null }
```

**Response 200**: `data` berisi objek runner terbaru.

Response error sama seperti POST (400, 422), ditambah 404 `runner not found` (code 1300).

---

## POST /api/runners/:id/regenerate-token

Membuat token baru. **Token lama langsung tidak berlaku**, jadi agent harus diperbarui dengan token baru. Tidak perlu body.

**Response 200**

```json
{
  "code": 0,
  "message": "token regenerated",
  "data": {
    "id": "01a15475-2a7d-797f-b5fb-66ebdebdaabf",
    "user_id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
    "name": "Agent Scalping",
    "description": "bot AI",
    "token": "hpA73AATXLeFMHF7o7d2BFxWOozRzApz8UQNZLEQ",
    "is_active": true,
    "expires_at": null,
    "last_used_at": "2026-10-19T20:58:49.879015+07:00",
    "created_at": "2026-10-19T20:58:47.165232+07:00",
    "updated_at": "2026-10-19T20:58:53.63984+07:00"
  }
}
```

**Response 404**

```json
{ "code": 1300, "message": "runner not found", "data": null }
```

---

## DELETE /api/runners/:id

Soft delete. Token runner langsung tidak berlaku.

**Response 200**

```json
{ "code": 0, "message": "runner deleted", "data": null }
```

**Response 404**

```json
{ "code": 1300, "message": "runner not found", "data": null }
```
