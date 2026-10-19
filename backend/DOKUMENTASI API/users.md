# Master Users

Base path: `/api/users`

Format response umum dan arti status code ada di [README.md](README.md).

**Semua endpoint di sini wajib login** (header `Authorization: Bearer <token>`, lihat [auth.md](auth.md)). Tanpa token yang valid, response-nya HTTP 401 dengan code `1201`.

| Method | Endpoint | Keterangan |
|--------|----------|------------|
| GET | `/api/users` | List user (pagination + search) |
| GET | `/api/users/:id` | Detail user |
| POST | `/api/users` | Tambah user |
| PUT | `/api/users/:id` | Ubah user (sebagian field) |
| DELETE | `/api/users/:id` | Hapus user (soft delete) |

## Objek User

| Field | Tipe | Keterangan |
|-------|------|------------|
| `id` | string (UUID v7) | ID user, contoh `01a153a5-1d7e-71c0-9b39-daa201608efd` |
| `username` | string | Unik, tidak membedakan huruf besar/kecil |
| `name` | string | Nama lengkap |
| `email` | string \| null | Unik kalau diisi |
| `created_at` | string (RFC 3339) | Waktu dibuat |
| `updated_at` | string (RFC 3339) | Waktu terakhir diubah |

Password tidak pernah dikembalikan di response. Password disimpan sebagai hash bcrypt.

## Aturan Validasi

| Field | Aturan |
|-------|--------|
| `username` | Wajib, 3-50 karakter, hanya huruf, angka, `_` dan `.` |
| `name` | Wajib, maksimal 100 karakter |
| `email` | Opsional, format email valid, maksimal 255 karakter |
| `password` | Wajib, 8-72 karakter |

Spasi di awal/akhir `username`, `name`, dan `email` otomatis dihapus.

---

## GET /api/users

List user yang belum dihapus, diurutkan dari yang paling lama dibuat (UUID v7 urut berdasarkan waktu).

**Query parameter**

| Nama | Tipe | Default | Keterangan |
|------|------|---------|------------|
| `page` | integer | 1 | Halaman |
| `limit` | integer | 10 | Jumlah per halaman, maksimal 100 |
| `search` | string | - | Cari di `username`, `name`, `email` (tidak membedakan huruf besar/kecil) |

**Contoh request**

```
GET /api/users?page=1&limit=10&search=admin
```

**Response 200**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "01a153a5-1d7e-71c0-9b39-daa201608efd",
      "username": "adminres",
      "name": "Administrator",
      "email": null,
      "created_at": "2026-10-19T17:04:29.801742+07:00",
      "updated_at": "2026-10-19T17:04:29.801742+07:00"
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

## GET /api/users/:id

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
    "created_at": "2026-10-19T17:04:29.801742+07:00",
    "updated_at": "2026-10-19T17:04:29.801742+07:00"
  }
}
```

**Response 404**

```json
{ "code": 1100, "message": "user not found", "data": null }
```

**Response 400** (id bukan UUID valid)

```json
{ "code": 1003, "message": "invalid id", "data": null }
```

---

## POST /api/users

**Request body**

| Field | Wajib |
|-------|-------|
| `username` | Ya |
| `name` | Ya |
| `email` | Tidak |
| `password` | Ya |

```json
{
  "username": "budi_trader",
  "name": "Budi",
  "email": "budi@mail.com",
  "password": "rahasia123"
}
```

**Response 201**

```json
{
  "code": 0,
  "message": "user created",
  "data": {
    "id": "01a153a5-1ff7-7fdb-9bf9-e2d5e8940da8",
    "username": "budi_trader",
    "name": "Budi",
    "email": "budi@mail.com",
    "created_at": "2026-10-19T17:07:31.026495+07:00",
    "updated_at": "2026-10-19T17:07:31.026495+07:00"
  }
}
```

**Response 422**

```json
{
  "code": 1002,
  "message": "validation failed",
  "data": {
    "email": "invalid email format",
    "name": "is required, max 100 characters",
    "password": "must be 8-72 characters",
    "username": "must be 3-50 characters: letters, numbers, underscore or dot"
  }
}
```

**Response 409**

```json
{
  "code": 1101,
  "message": "username already used",
  "data": { "username": "username already used" }
}
```

Untuk email duplikat: `"code": 1102`, `"message": "email already used"`, `"data": { "email": "email already used" }`.

**Response 400** (body bukan JSON valid)

```json
{ "code": 1001, "message": "invalid request body", "data": null }
```

---

## PUT /api/users/:id

Update sebagian. Hanya field yang dikirim yang diubah, field yang tidak dikirim tetap.

| Field | Keterangan |
|-------|------------|
| `username` | Ganti username |
| `name` | Ganti nama |
| `email` | Ganti email. Kirim `""` untuk mengosongkan email |
| `password` | Ganti password (di-hash ulang dengan bcrypt) |

**Contoh: ganti nama dan kosongkan email**

```json
{
  "name": "Budi Santoso",
  "email": ""
}
```

**Response 200**

```json
{
  "code": 0,
  "message": "user updated",
  "data": {
    "id": "01a153a5-1ff7-7fdb-9bf9-e2d5e8940da8",
    "username": "budi_trader",
    "name": "Budi Santoso",
    "email": null,
    "created_at": "2026-10-19T17:07:31.026495+07:00",
    "updated_at": "2026-10-19T17:07:33.398979+07:00"
  }
}
```

Response error sama seperti POST (400, 409, 422), ditambah 404 `user not found` (code 1100).

---

## DELETE /api/users/:id

Soft delete: data tidak benar-benar dihapus dari database (kolom `deleted_at` diisi), tapi tidak muncul lagi di list/detail. Username dan email dari user yang sudah dihapus bisa dipakai lagi untuk user baru.

**Response 200**

```json
{ "code": 0, "message": "user deleted", "data": null }
```

**Response 404** (tidak ada / sudah dihapus)

```json
{ "code": 1100, "message": "user not found", "data": null }
```
