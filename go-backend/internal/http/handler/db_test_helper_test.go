package handler

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

func mustQueryInt(t *testing.T, r *repo.Repository, query string, args ...interface{}) int {
	t.Helper()
	var v int
	if err := r.DB().Raw(query, args...).Row().Scan(&v); err != nil {
		t.Fatalf("query int failed: %v (query=%q)", err, query)
	}
	return v
}

func mustQueryInt64Int64String(t *testing.T, r *repo.Repository, query string, args ...interface{}) (int64, int64, string) {
	t.Helper()
	var a int64
	var b int64
	var c string
	if err := r.DB().Raw(query, args...).Row().Scan(&a, &b, &c); err != nil {
		t.Fatalf("query int64+int64+string failed: %v (query=%q)", err, query)
	}
	return a, b, c
}

func mustQueryInt64Int64Int(t *testing.T, r *repo.Repository, query string, args ...interface{}) (int64, int64, int) {
	t.Helper()
	var a int64
	var b int64
	var c int
	if err := r.DB().Raw(query, args...).Row().Scan(&a, &b, &c); err != nil {
		t.Fatalf("query int64+int64+int failed: %v (query=%q)", err, query)
	}
	return a, b, c
}
