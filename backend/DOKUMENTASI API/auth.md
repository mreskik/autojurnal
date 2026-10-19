# Auth (Login & Session)

Base path: `/api/auth`

Format response umum ada di [README.md](README.md), daftar error code di [error-codes.md](error-codes.md).

## Cara Kerja

- Login memakai **session yang disimpan di database** (bukan JWT), sehingga session bisa dimatikan dari server kapan saja.
- Saat login berhasil, server mengembalikan `token`. Simpan token ini di client.
- Setiap request ke endpoint yang wajib login, kirim header:

  ```
  Authorization: Bearer <token>
  ```

- Token berlaku selama `SESSION_TTL_HOURS` (default 168 jam = 7 hari) sejak login. Setelah lewat, request akan ditolak dengan code `1201` dan user harus login ulang.
- Di database hanya disimpan hash SHA-256 dari token, token aslinya hanya dipegang client.
- Session otomatis tidak berlaku kalau: logout, di-revoke, expired, atau user-nya dihapus.

| Method | Endpoint | Wajib Login | Keterangan |
|--------|----------|-------------|------------|
| POST | `/api/auth/login` | Tidak | Login, dapat token |
| GET | `/api/auth/me` | Ya | Data user yang sedang login |
| POST | `/api/auth/logout` | Ya | Logout session saat ini |
| POST | `/api/auth/logout-all` | Ya | Logout dari semua perangkat (termasuk session saat ini) |
| GET | `/api/auth/sessions` | Ya | List session aktif milik user |
| DELETE | `/api/auth/sessions/:id` | Ya | Matikan satu session tertentu (misal perangkat lain) |

Semua endpoint yang wajib login akan mengembalikan error ini kalau token tidak dikirim, salah, expired, atau sudah di-logout:

**Response 401**

```json
{ "code": 1201, "message": "unauthorized", "data": null }
```

---

## POST /api/auth/login

Username tidak membedakan huruf besar/kecil.

**Request body**

| Field | Wajib |
|-------|-------|
| `username` | Ya |
| `password` | Ya |

```json
{
  "username": "adminres",
  "password": "Surabaya1!"
}
```

**Response 200**

```json
{
  "code": 0,
  "message": "login success",
  "data": {
    "token": "1bOdgP3svX0pOHG86tj7quH_VZIDGTOE_Vj24lt0lr0",
    "expires_at": "2026-10-26T20:42:22.112086+07:00",
    "user": {
      "id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
      "username": "adminres",
      "name": "Administrator",
      "email": null,
      "created_at": "2026-10-19T17:11:32.300351+07:00",
      "updated_at": "2026-10-19T17:11:32.300351+07:00"
    }
  }
}
```

**Response 401** (username tidak ada atau password salah, pesannya sengaja sama)

```json
{ "code": 1200, "message": "invalid username or password", "data": null }
```

**Response 422** (field kosong)

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "password": "is required",
    "username": "is required"
  }
}
```

**Response 400** (body bukan JSON valid)

```json
{ "code": 1001, "message": "invalid request body", "data": null }
```

**Response 429** (terlalu banyak login gagal)

Default: 5 kali gagal per 15 menit, dihitung per IP + username. Login yang berhasil tidak dihitung. Selama terkena limit, login dengan password yang benar pun tetap ditolak sampai waktunya habis. Header `Retry-After` berisi sisa detik.

```json
{ "code": 1006, "message": "too many requests, try again later", "data": null }
```

---

## GET /api/auth/me

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
    "username": "adminres",
    "name": "Administrator",
    "email": null,
    "created_at": "2026-10-19T17:11:32.300351+07:00",
    "updated_at": "2026-10-19T17:11:32.300351+07:00"
  }
}
```

---

## POST /api/auth/logout

Mematikan session yang sedang dipakai. Token tidak bisa dipakai lagi.

**Response 200**

```json
{ "code": 0, "message": "logout success", "data": null }
```

---

## POST /api/auth/logout-all

Mematikan semua session milik user, termasuk session yang sedang dipakai. Cocok dipakai kalau akun dicurigai dipakai orang lain.

**Response 200**

```json
{ "code": 0, "message": "all sessions logged out", "data": null }
```

---

## GET /api/auth/sessions

List session aktif (belum expired) milik user yang sedang login, diurutkan dari yang terakhir dipakai. `current: true` menandai session yang sedang dipakai untuk request ini.

`last_used_at` diperbarui paling sering satu kali per menit.

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a15466-25a0-7dab-a179-962e7d41a4a4",
      "ip_address": "127.0.0.1",
      "user_agent": "Mozilla/5.0 (iPhone ...)",
      "current": false,
      "created_at": "2026-10-19T20:42:22.880673+07:00",
      "last_used_at": "2026-10-19T20:42:22.880673+07:00",
      "expires_at": "2026-10-26T20:42:22.874491+07:00"
    },
    {
      "id": "01a15466-241b-70ee-a6e9-8664ee1c4576",
      "ip_address": "127.0.0.1",
      "user_agent": "Mozilla/5.0 (Windows NT 10.0 ...)",
      "current": true,
      "created_at": "2026-10-19T20:42:22.490864+07:00",
      "last_used_at": "2026-10-19T20:42:22.490864+07:00",
      "expires_at": "2026-10-26T20:42:22.484796+07:00"
    }
  ]
}
```

---

## DELETE /api/auth/sessions/:id

Mematikan satu session milik user yang sedang login. `:id` diambil dari `GET /api/auth/sessions`. Session milik user lain tidak bisa dimatikan (dianggap tidak ditemukan).

**Response 200**

```json
{ "code": 0, "message": "session revoked", "data": null }
```

**Response 404**

```json
{ "code": 1202, "message": "session not found", "data": null }
```

**Response 400** (id bukan UUID valid)

```json
{ "code": 1003, "message": "invalid id", "data": null }
```
