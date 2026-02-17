package contract

import (
	"testing"

	"go-backend/internal/store/repo"
)

func mustLastInsertID(t *testing.T, r *repo.Repository, label string) int64 {
	t.Helper()
	var id int64
	if err := r.DB().Raw("SELECT last_insert_rowid()").Row().Scan(&id); err != nil {
		t.Fatalf("read last_insert_rowid for %s: %v", label, err)
	}
	if id <= 0 {
		t.Fatalf("invalid last_insert_rowid for %s: %d", label, id)
	}
	return id
}
