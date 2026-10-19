package journals

import (
	"context"

	"autojurnal/internal/module/runners"

	"github.com/google/uuid"
)

// Scope: siapa yang mengakses jurnal/trade.
//   - user  : semua jurnal miliknya, lihat & ubah semua trade di dalamnya.
//   - runner: hanya jurnal milik pemilik runner yang dia join; boleh lihat semua trade di jurnal itu,
//     tapi hanya boleh ubah / tutup trade buatannya sendiri.
//
// Jurnal di luar scope dianggap tidak ada (404).
type Scope struct {
	UserID   uuid.UUID  // pemilik jurnal
	RunnerID *uuid.UUID // diisi kalau yang akses adalah runner
}

func UserScope(userID uuid.UUID) Scope {
	return Scope{UserID: userID}
}

func RunnerScope(r *runners.Runner) Scope {
	return Scope{UserID: r.UserID, RunnerID: &r.ID}
}

func (sc Scope) journal(ctx context.Context, repo *Repository, id uuid.UUID, forUpdate bool) (*Journal, error) {
	if sc.RunnerID != nil {
		return repo.FindForRunner(ctx, sc.UserID, *sc.RunnerID, id, forUpdate)
	}
	if forUpdate {
		return repo.FindByUserForUpdate(ctx, sc.UserID, id)
	}
	return repo.FindByUser(ctx, sc.UserID, id)
}

// editableTrade: trade yang boleh diubah / ditutup oleh scope ini.
// Runner yang mencoba mengubah trade milik pihak lain -> ErrTradeNotOwned.
func (sc Scope) editableTrade(ctx context.Context, repo *Repository, journalID, id uuid.UUID) (*Trade, error) {
	t, err := repo.FindTrade(ctx, journalID, id)
	if err != nil {
		return nil, err
	}
	if sc.RunnerID != nil && (t.RunnerID == nil || *t.RunnerID != *sc.RunnerID) {
		return nil, ErrTradeNotOwned
	}
	return t, nil
}
