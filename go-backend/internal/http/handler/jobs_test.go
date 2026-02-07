package handler

import (
	"path/filepath"
	"testing"
	"time"

	"go-backend/internal/store/sqlite"
)

func TestRunStatisticsFlowJobTracksIncrementAndPrunes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "jobs-stats.db")
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	h := New(repo, "secret")
	now := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)
	nowMs := now.UnixMilli()

	if _, err := repo.DB().Exec(`UPDATE user SET in_flow = 100, out_flow = 200 WHERE id = 1`); err != nil {
		t.Fatalf("seed user flow: %v", err)
	}

	if _, err := repo.DB().Exec(`INSERT INTO statistics_flow(user_id, flow, total_flow, time, created_time) VALUES(1, 250, 250, '11:00', ?)`, now.Add(-time.Hour).UnixMilli()); err != nil {
		t.Fatalf("seed recent statistics row: %v", err)
	}
	if _, err := repo.DB().Exec(`INSERT INTO statistics_flow(user_id, flow, total_flow, time, created_time) VALUES(1, 10, 10, '00:00', ?)`, now.Add(-49*time.Hour).UnixMilli()); err != nil {
		t.Fatalf("seed stale statistics row: %v", err)
	}

	h.runStatisticsFlowJob(now)

	var staleCount int
	if err := repo.DB().QueryRow(`SELECT COUNT(1) FROM statistics_flow WHERE created_time < ?`, nowMs-int64((48*time.Hour)/time.Millisecond)).Scan(&staleCount); err != nil {
		t.Fatalf("query stale statistics rows: %v", err)
	}
	if staleCount != 0 {
		t.Fatalf("expected stale statistics rows to be pruned, got %d", staleCount)
	}

	var flow int64
	var total int64
	var hour string
	if err := repo.DB().QueryRow(`SELECT flow, total_flow, time FROM statistics_flow WHERE user_id = 1 ORDER BY id DESC LIMIT 1`).Scan(&flow, &total, &hour); err != nil {
		t.Fatalf("query latest statistics row: %v", err)
	}
	if flow != 50 {
		t.Fatalf("expected increment flow 50, got %d", flow)
	}
	if total != 300 {
		t.Fatalf("expected total flow 300, got %d", total)
	}
	if hour != "12:00" {
		t.Fatalf("expected hour mark 12:00, got %s", hour)
	}
}

func TestRunResetAndExpiryJobResetsFlowAndDisablesExpiredRecords(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "jobs-reset.db")
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	h := New(repo, "secret")
	now := time.Date(2026, 3, 15, 0, 0, 5, 0, time.UTC)
	nowMs := now.UnixMilli()

	if _, err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'expired_user', 'x', 1, ?, 100, 1000, 2000, 15, 1, ?, ?, 1)
	`, nowMs-1000, nowMs, nowMs); err != nil {
		t.Fatalf("insert expired user: %v", err)
	}

	if _, err := repo.DB().Exec(`
		INSERT INTO tunnel(id, name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES(1, 't1', 1.0, 1, 'tls', 1, ?, ?, 1, NULL, 0)
	`, nowMs, nowMs); err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}

	if _, err := repo.DB().Exec(`
		INSERT INTO user_tunnel(id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(10, 2, 1, NULL, 1, 1, 300, 400, 15, ?, 1)
	`, nowMs-1000); err != nil {
		t.Fatalf("insert expired user_tunnel: %v", err)
	}

	if _, err := repo.DB().Exec(`
		INSERT INTO forward(id, user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(20, 2, 'expired_user', 'f1', 1, '1.1.1.1:443', 'fifo', 0, 0, ?, ?, 1, 0)
	`, nowMs, nowMs); err != nil {
		t.Fatalf("insert forward: %v", err)
	}

	h.runResetAndExpiryJob(now)

	var userIn, userOut int64
	var userStatus int
	if err := repo.DB().QueryRow(`SELECT in_flow, out_flow, status FROM user WHERE id = 2`).Scan(&userIn, &userOut, &userStatus); err != nil {
		t.Fatalf("query user after maintenance: %v", err)
	}
	if userIn != 0 || userOut != 0 || userStatus != 0 {
		t.Fatalf("expected user reset+disabled, got in=%d out=%d status=%d", userIn, userOut, userStatus)
	}

	var utIn, utOut int64
	var utStatus int
	if err := repo.DB().QueryRow(`SELECT in_flow, out_flow, status FROM user_tunnel WHERE id = 10`).Scan(&utIn, &utOut, &utStatus); err != nil {
		t.Fatalf("query user_tunnel after maintenance: %v", err)
	}
	if utIn != 0 || utOut != 0 || utStatus != 0 {
		t.Fatalf("expected user_tunnel reset+disabled, got in=%d out=%d status=%d", utIn, utOut, utStatus)
	}

	var forwardStatus int
	if err := repo.DB().QueryRow(`SELECT status FROM forward WHERE id = 20`).Scan(&forwardStatus); err != nil {
		t.Fatalf("query forward after maintenance: %v", err)
	}
	if forwardStatus != 0 {
		t.Fatalf("expected forward status=0 after expiry handling, got %d", forwardStatus)
	}
}
