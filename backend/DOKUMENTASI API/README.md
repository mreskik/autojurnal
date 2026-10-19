# Dokumentasi API Autojurnal

Base URL: `http://localhost:{APP_PORT}/api` (default port `1000`)

Semua request dan response memakai JSON (`Content-Type: application/json`).

## Autentikasi

Semua endpoint wajib login, **kecuali** `GET /api/health` dan `POST /api/auth/login`.

Login dulu lewat `POST /api/auth/login`, lalu kirim token yang didapat di setiap request:

```
Authorization: Bearer <token>
```

Kalau token tidak ada / tidak valid / expired, response-nya HTTP 401 dengan code `1201`. Detail ada di [auth.md](auth.md).

**Pengecualian:** endpoint `/api/runner/*` (dipakai agent/bot) tidak memakai token session, tapi **token runner**, ditambah header `X-Journal-Token` (token jurnal) untuk endpoint jurnal & trade. Detail ada di [runner-guide.md](runner-guide.md).

> **Untuk frontend:** aplikasi (web / mobile) **tidak memanggil** `/api/runner/*`. Endpoint itu khusus agent. Semua kebutuhan aplikasi, termasuk mengelola runner, melihat token, mengatur rules / strategy, dan download `AGENT.md`, memakai endpoint biasa dengan token session (lihat [runners.md](runners.md) dan [journals.md](journals.md)).

### Dokumen per pembaca

| Pembaca | Dokumen |
|---------|---------|
| Frontend (aplikasi) | [auth.md](auth.md), [users.md](users.md), [runners.md](runners.md), [journals.md](journals.md), [trades.md](trades.md), [dashboard.md](dashboard.md), [frontend-dashboard-guide.md](frontend-dashboard-guide.md), [error-codes.md](error-codes.md) |
| Developer agent / bot | [runner-guide.md](runner-guide.md) (atau file `AGENT.md` hasil download), [error-codes.md](error-codes.md) |

## Rate Limit

Ada tiga lapis pembatasan. Kalau terkena, response-nya HTTP 429 dengan code `1006`, plus header `Retry-After` (sisa detik sampai boleh mencoba lagi).

| Lapis | Berlaku di | Dihitung per | Batas default | Env |
|-------|-----------|--------------|---------------|-----|
| Login | `POST /api/auth/login` | IP + username | 5 kali **gagal** per 15 menit (login sukses tidak dihitung) | `RATE_LIMIT_LOGIN_MAX`, `RATE_LIMIT_LOGIN_WINDOW_MINUTES` |
| IP | Semua endpoint `/api` | IP | 300 request per menit | `RATE_LIMIT_IP_MAX` |
| User | Endpoint yang wajib login | User yang login | 120 request per menit | `RATE_LIMIT_USER_MAX` |
| Runner | Endpoint `/api/runner/*` | Runner (dari token) | 120 request per menit (sama dengan User) | `RATE_LIMIT_USER_MAX` |

```json
{ "code": 1006, "message": "too many requests, try again later", "data": null }
```

Catatan:

- Memakai *sliding window*, jadi kuota pulih bertahap, bukan reset sekaligus di awal menit.
- Hitungan disimpan di memori server: reset saat server restart dan tidak sinkron kalau server dijalankan lebih dari satu instance.
- Bisa dimatikan dengan `RATE_LIMIT_ENABLED=false` (misal saat development/testing).
- Setiap response dari endpoint yang kena limiter juga membawa header `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`.

## CORS (akses dari browser)

Frontend yang berjalan di origin lain (misal Vite `http://localhost:5173`) harus didaftarkan:

```
CORS_ALLOWED_ORIGINS=http://localhost:5173,https://app.domain.com
```

Kalau kosong, default `http://localhost:5173` dan `http://127.0.0.1:5173` (development). Header yang boleh dikirim: `Authorization`, `Content-Type`, `X-Journal-Token`. Header response yang bisa dibaca JavaScript: `Content-Disposition` (download AGENT.md), `Retry-After`, `X-RateLimit-*`.

## URL Publik API

File `AGENT.md` (lihat [journals.md](journals.md#get-apijournalsidrunnersrunneridagentmd)) mencantumkan Base URL API untuk agent. Isi di `.env`:

```
PUBLIC_API_URL=https://api.domain.com/api
```

Kalau kosong, Base URL diambil dari alamat request yang masuk (cukup untuk development).

## Deploy di Belakang Nginx

Supaya rate limit dan data session (`ip_address`) memakai IP asli user, bukan IP Nginx:

1. Isi IP Nginx di `.env` (pisah koma, boleh CIDR):

   ```
   TRUSTED_PROXIES=127.0.0.1
   ```

2. Di konfigurasi Nginx, kirim IP asli lewat header `X-Real-IP`:

   ```nginx
   location /api/ {
       proxy_pass http://127.0.0.1:1000;
       proxy_set_header X-Real-IP $remote_addr;
       proxy_set_header Host $host;
   }
   ```

Header `X-Real-IP` hanya dipercaya kalau request datang dari IP yang terdaftar di `TRUSTED_PROXIES`. Kalau `TRUSTED_PROXIES` kosong, header ini diabaikan sehingga tidak bisa dipalsukan client.

Dipakai `X-Real-IP` (bukan `X-Forwarded-For`) karena nilai pertama di `X-Forwarded-For` bisa diisi sembarangan oleh client, sedangkan `X-Real-IP` di-set penuh oleh Nginx.

## Daftar Modul

| Modul | File |
|-------|------|
| Health check | [di bawah](#health-check) |
| Auth (login & session) | [auth.md](auth.md) |
| Master Users | [users.md](users.md) |
| Runners (manajemen runner & token di aplikasi) | [runners.md](runners.md) |
| Panduan integrasi agent (runner) | [runner-guide.md](runner-guide.md) |
| Journals (jurnal, target, deposit/withdraw, runner) | [journals.md](journals.md) |
| Trades | [trades.md](trades.md) |
| Dashboard & monitoring (API) | [dashboard.md](dashboard.md) |
| Panduan frontend dashboard | [frontend-dashboard-guide.md](frontend-dashboard-guide.md) |
| Error Codes | [error-codes.md](error-codes.md) |

## Format Response

Semua response punya field `code`, `message`, dan `data`.

- `code` = `0` artinya sukses.
- `code` >= `1000` artinya error. Daftarnya ada di [Kode Error](#kode-error).

### Sukses

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

Untuk endpoint list ada tambahan `meta`:

```json
{
  "code": 0,
  "message": "success",
  "data": [],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 25,
    "total_page": 3
  }
}
```

### Error

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "username": "must be 3-50 characters: letters, numbers, underscore or dot"
  }
}
```

Untuk error validasi (1002) dan data duplikat (11xx yang 409), `data` berisi detail per field dengan key nama field. Untuk error lain, `data` bernilai `null`.

## Kode Error

Daftar lengkap error code (HTTP status, kapan terjadi, isi `data`, dan cara menangani di client) ada di [error-codes.md](error-codes.md).

## Health Check

`GET /api/health`

Mengecek server dan koneksi database.

**Response 200**

```json
{ "code": 0, "message": "ok", "data": null }
```

**Response 503**

```json
{ "code": 1005, "message": "database unavailable", "data": null }
```
