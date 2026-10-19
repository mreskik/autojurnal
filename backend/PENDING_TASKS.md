# Pending Tasks

## Modul Jurnal

Desain sudah disepakati. Dikerjakan bertahap:

- [x] **Tahap 1: journals + journal_targets** (lihat `DOKUMENTASI API/journals.md`)
- [x] **Tahap 2: journal_transactions + journal_runners** (lihat `DOKUMENTASI API/journals.md`)
- [x] **Tahap 3: trades (sisi user)** (lihat `DOKUMENTASI API/trades.md`), menunggu review
- [x] **Target jurnal (goal)**: `goal_type` balance/profit/percent, tanpa deadline, sekali tercapai tetap achieved (lihat `DOKUMENTASI API/journals.md`)
- [x] **Tahap 4: trades (sisi runner)** (lihat `DOKUMENTASI API/runners.md`)
- [x] **Tahap 5: rekap / dashboard** (lihat `DOKUMENTASI API/dashboard.md` dan `frontend-dashboard-guide.md`)
  - Endpoint: `/api/dashboard/overview`, `/api/dashboard/runners`, `/api/journals/:id/dashboard/{summary,equity,periods,calendar,breakdown}`.
  - Keputusan: status periode dari hasil akhir periode, `no_trade` terpisah dari `missed` dan tidak memutus streak, drawdown dari PnL kumulatif.
  - Data dihitung di memori per request (muat semua trade jurnal). Kalau nanti satu jurnal punya puluhan ribu trade, pertimbangkan cache / tabel rekap harian.

## Catatan lain

- Token runner masih disimpan plain text (keputusan sementara). Kalau nanti mau di-hash, cukup tambah migration (kolom hash + backfill) dan ubah `FindByToken`.
- Ganti password user lewat `PUT /api/users/:id` belum otomatis mematikan session lain milik user tersebut.
- Cek balance minus saat ini hanya terhadap total balance akhir (opsi A). Evaluasi lagi setelah trade ada: opsi B = balance tidak boleh minus di titik waktu mana pun (urut `occurred_at` / `closed_at`).
- Ide nanti: *lock period* (periode yang sudah lewat dibekukan, tidak berubah walau trade lama diedit).
