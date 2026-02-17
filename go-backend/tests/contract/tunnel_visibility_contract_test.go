package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-backend/internal/auth"
	"go-backend/internal/http/response"
)

func TestUserTunnelVisibleListContracts(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupDiagnosisContractRouter(t, secret)
	now := time.Now().UnixMilli()

	if err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'normal_user', '3c85cdebade1c51cf64ca9f3c09d182d', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	insertTunnel := func(name string, status int, inx int64) int64 {
		if err := repo.DB().Exec(`
			INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, name, 1.0, 1, "tls", 99999, now, now, status, nil, inx).Error; err != nil {
			t.Fatalf("insert tunnel %s: %v", name, err)
		}
		return mustLastInsertID(t, repo, name)
	}

	enabledA := insertTunnel("enabled-A", 1, 1)
	enabledB := insertTunnel("enabled-B", 1, 2)
	disabledC := insertTunnel("disabled-C", 0, 3)

	if err := repo.DB().Exec(`
		INSERT INTO user_tunnel(user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(?, ?, NULL, ?, ?, 0, 0, ?, ?, ?)
	`, 2, enabledA, 100, 1000, 1, 2727251700000, 0).Error; err != nil {
		t.Fatalf("insert user_tunnel enabledA: %v", err)
	}
	if err := repo.DB().Exec(`
		INSERT INTO user_tunnel(user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(?, ?, NULL, ?, ?, 0, 0, ?, ?, ?)
	`, 2, enabledB, 100, 1000, 1, 2727251700000, 1).Error; err != nil {
		t.Fatalf("insert user_tunnel enabledB: %v", err)
	}
	if err := repo.DB().Exec(`
		INSERT INTO user_tunnel(user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(?, ?, NULL, ?, ?, 0, 0, ?, ?, ?)
	`, 2, disabledC, 100, 1000, 1, 2727251700000, 1).Error; err != nil {
		t.Fatalf("insert user_tunnel disabledC: %v", err)
	}

	adminToken, err := auth.GenerateToken(1, "admin_user", 0, secret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	userToken, err := auth.GenerateToken(2, "normal_user", 1, secret)
	if err != nil {
		t.Fatalf("generate user token: %v", err)
	}

	t.Run("admin sees all enabled tunnels without user_tunnel rows", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/user/tunnel", nil)
		req.Header.Set("Authorization", adminToken)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		var out response.R
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Code != 0 {
			t.Fatalf("expected code 0, got %d (%s)", out.Code, out.Msg)
		}

		ids := collectTunnelIDs(t, out.Data)
		if !ids[enabledA] || !ids[enabledB] {
			t.Fatalf("expected enabled tunnels for admin, got %v", ids)
		}
		if ids[disabledC] {
			t.Fatalf("did not expect disabled tunnel for admin")
		}
	})

	t.Run("normal user sees enabled assigned tunnels regardless of user_tunnel status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/user/tunnel", nil)
		req.Header.Set("Authorization", userToken)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		var out response.R
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if out.Code != 0 {
			t.Fatalf("expected code 0, got %d (%s)", out.Code, out.Msg)
		}

		ids := collectTunnelIDs(t, out.Data)
		if !ids[enabledA] || !ids[enabledB] {
			t.Fatalf("expected enabled assigned tunnels for user, got %v", ids)
		}
		if ids[disabledC] {
			t.Fatalf("did not expect disabled tunnel for user")
		}
	})
}

func collectTunnelIDs(t *testing.T, data interface{}) map[int64]bool {
	t.Helper()
	arr, ok := data.([]interface{})
	if !ok {
		t.Fatalf("expected array data, got %T", data)
	}
	ids := make(map[int64]bool, len(arr))
	for _, item := range arr {
		obj, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected object item, got %T", item)
		}
		id := int64(obj["id"].(float64))
		ids[id] = true
	}
	return ids
}
