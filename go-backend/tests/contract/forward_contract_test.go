package contract_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"go-backend/internal/auth"
	"go-backend/internal/http/response"
)

func TestForwardOwnershipAndScopeContracts(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	now := time.Now().UnixMilli()

	if _, err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'normal_user', '3c85cdebade1c51cf64ca9f3c09d182d', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	res, err := repo.DB().Exec(`
		INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "contract-tunnel", 1.0, 1, "tls", 99999, now, now, 1, nil, 0)
	if err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}
	tunnelID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("get tunnel id: %v", err)
	}

	nodeRes, err := repo.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "entry-node", "entry-secret", "10.0.0.10", "10.0.0.10", "", "20000-20010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0)
	if err != nil {
		t.Fatalf("insert node: %v", err)
	}
	entryNodeID, err := nodeRes.LastInsertId()
	if err != nil {
		t.Fatalf("get node id: %v", err)
	}

	if _, err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 1, ?, 20001, 'round', 1, 'tls')
	`, tunnelID, entryNodeID); err != nil {
		t.Fatalf("insert chain_tunnel: %v", err)
	}

	resAdmin, err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(?, ?, ?, ?, ?, ?, 0, 0, ?, ?, 1, ?)
	`, 1, "admin_user", "admin-forward", tunnelID, "1.1.1.1:443", "fifo", now, now, 0)
	if err != nil {
		t.Fatalf("insert admin forward: %v", err)
	}
	adminForwardID, err := resAdmin.LastInsertId()
	if err != nil {
		t.Fatalf("get admin forward id: %v", err)
	}

	resUser, err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(?, ?, ?, ?, ?, ?, 0, 0, ?, ?, 1, ?)
	`, 2, "normal_user", "user-forward", tunnelID, "8.8.8.8:53", "fifo", now, now, 1)
	if err != nil {
		t.Fatalf("insert user forward: %v", err)
	}
	userForwardID, err := resUser.LastInsertId()
	if err != nil {
		t.Fatalf("get user forward id: %v", err)
	}

	userToken, err := auth.GenerateToken(2, "normal_user", 1, secret)
	if err != nil {
		t.Fatalf("generate user token: %v", err)
	}
	adminToken, err := auth.GenerateToken(1, "admin_user", 0, secret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	t.Run("non-owner cannot delete another user's forward", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/delete", bytes.NewBufferString(`{"id":`+jsonNumber(adminForwardID)+`}`))
		req.Header.Set("Authorization", userToken)
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		assertCodeMsg(t, res, -1, "转发不存在")
	})

	t.Run("non-admin forward list is scoped to owner", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/list", bytes.NewBufferString(`{}`))
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
		arr, ok := out.Data.([]interface{})
		if !ok {
			t.Fatalf("expected array data, got %T", out.Data)
		}
		if len(arr) != 1 {
			t.Fatalf("expected 1 forward, got %d", len(arr))
		}
		item, ok := arr[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected object item, got %T", arr[0])
		}
		if got := int64(item["id"].(float64)); got != userForwardID {
			t.Fatalf("expected forward id %d, got %d", userForwardID, got)
		}
	})

	t.Run("forward diagnose returns structured payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/diagnose", bytes.NewBufferString(`{"forwardId":`+jsonNumber(userForwardID)+`}`))
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

		payload, ok := out.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected object payload, got %T", out.Data)
		}
		results, ok := payload["results"].([]interface{})
		if !ok || len(results) == 0 {
			t.Fatalf("expected non-empty results, got %v", payload["results"])
		}
		first, ok := results[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected result object, got %T", results[0])
		}
		if _, ok := first["message"]; !ok {
			t.Fatalf("expected message field in diagnosis result")
		}
		if got := int(first["fromChainType"].(float64)); got != 1 {
			t.Fatalf("expected fromChainType=1, got %d", got)
		}
	})

	t.Run("tunnel diagnose returns structured payload", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/diagnose", bytes.NewBufferString(`{"tunnelId":`+jsonNumber(tunnelID)+`}`))
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

		payload, ok := out.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("expected object payload, got %T", out.Data)
		}
		results, ok := payload["results"].([]interface{})
		if !ok || len(results) == 0 {
			t.Fatalf("expected non-empty results, got %v", payload["results"])
		}
		first, ok := results[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected result object, got %T", results[0])
		}
		if _, ok := first["message"]; !ok {
			t.Fatalf("expected message field in tunnel diagnosis result")
		}
	})
}

func jsonNumber(v int64) string {
	return strconv.FormatInt(v, 10)
}
