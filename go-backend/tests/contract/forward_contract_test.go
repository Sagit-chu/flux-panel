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

	if err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'normal_user', '3c85cdebade1c51cf64ca9f3c09d182d', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "contract-tunnel", 1.0, 1, "tls", 99999, now, now, 1, nil, 0).Error; err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}
	tunnelID := mustLastInsertID(t, repo, "contract-tunnel")

	if err := repo.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "entry-node", "entry-secret", "10.0.0.10", "10.0.0.10", "", "20000-20010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0).Error; err != nil {
		t.Fatalf("insert node: %v", err)
	}
	entryNodeID := mustLastInsertID(t, repo, "entry-node")

	if err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 1, ?, 20001, 'round', 1, 'tls')
	`, tunnelID, entryNodeID).Error; err != nil {
		t.Fatalf("insert chain_tunnel: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(?, ?, ?, ?, ?, ?, 0, 0, ?, ?, 1, ?)
	`, 1, "admin_user", "admin-forward", tunnelID, "1.1.1.1:443", "fifo", now, now, 0).Error; err != nil {
		t.Fatalf("insert admin forward: %v", err)
	}
	adminForwardID := mustLastInsertID(t, repo, "admin-forward")

	if err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(?, ?, ?, ?, ?, ?, 0, 0, ?, ?, 1, ?)
	`, 2, "normal_user", "user-forward", tunnelID, "8.8.8.8:53", "fifo", now, now, 1).Error; err != nil {
		t.Fatalf("insert user forward: %v", err)
	}
	userForwardID := mustLastInsertID(t, repo, "user-forward")

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

func TestForwardSwitchTunnelRollbackOnSyncFailure(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	now := time.Now().UnixMilli()

	adminToken, err := auth.GenerateToken(1, "admin_user", 0, secret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'switch_user', '3c85cdebade1c51cf64ca9f3c09d182d', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	insertTunnel := func(name string, inx int) int64 {
		if err := repo.DB().Exec(`
			INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, name, 1.0, 1, "tls", 99999, now, now, 1, nil, inx).Error; err != nil {
			t.Fatalf("insert tunnel %s: %v", name, err)
		}
		return mustLastInsertID(t, repo, name)
	}

	insertNode := func(name, ip, portRange string, inx int) int64 {
		if err := repo.DB().Exec(`
			INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, name, name+"-secret", ip, ip, "", portRange, "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", inx).Error; err != nil {
			t.Fatalf("insert node %s: %v", name, err)
		}
		return mustLastInsertID(t, repo, name)
	}

	tunnelA := insertTunnel("switch-tunnel-a", 0)
	tunnelB := insertTunnel("switch-tunnel-b", 1)
	nodeA := insertNode("switch-node-a", "10.10.0.1", "21000-21010", 0)
	nodeB := insertNode("switch-node-b", "10.10.0.2", "22000-22010", 1)

	if err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 1, ?, 21001, 'round', 1, 'tls')
	`, tunnelA, nodeA).Error; err != nil {
		t.Fatalf("insert chain_tunnel tunnelA: %v", err)
	}
	if err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 1, ?, 22001, 'round', 1, 'tls')
	`, tunnelB, nodeB).Error; err != nil {
		t.Fatalf("insert chain_tunnel tunnelB: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO user_tunnel(id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(10, 2, ?, NULL, 999, 99999, 0, 0, 1, 2727251700000, 1)
	`, tunnelA).Error; err != nil {
		t.Fatalf("insert user_tunnel A: %v", err)
	}
	if err := repo.DB().Exec(`
		INSERT INTO user_tunnel(id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(11, 2, ?, NULL, 999, 99999, 0, 0, 1, 2727251700000, 1)
	`, tunnelB).Error; err != nil {
		t.Fatalf("insert user_tunnel B: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(2, 'switch_user', 'switch-forward', ?, '8.8.8.8:53', 'fifo', 0, 0, ?, ?, 1, 0)
	`, tunnelA, now, now).Error; err != nil {
		t.Fatalf("insert forward: %v", err)
	}
	forwardID := mustLastInsertID(t, repo, "switch-forward")

	if err := repo.DB().Exec(`INSERT INTO forward_port(forward_id, node_id, port) VALUES(?, ?, ?)`, forwardID, nodeA, 21001).Error; err != nil {
		t.Fatalf("insert forward_port: %v", err)
	}

	payload := `{"id":` + jsonNumber(forwardID) + `,"tunnelId":` + jsonNumber(tunnelB) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/update", bytes.NewBufferString(payload))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	var out response.R
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Code == 0 {
		t.Fatalf("expected update failure when node is offline")
	}

	tunnelAfter := mustQueryInt64(t, repo, `SELECT tunnel_id FROM forward WHERE id = ?`, forwardID)
	if tunnelAfter != tunnelA {
		t.Fatalf("expected tunnel rollback to %d, got %d", tunnelA, tunnelAfter)
	}

	nodeAfter, portAfter := mustQueryInt64Int(t, repo, `SELECT node_id, port FROM forward_port WHERE forward_id = ? LIMIT 1`, forwardID)
	if nodeAfter != nodeA || portAfter != 21001 {
		t.Fatalf("expected forward_port rollback to node=%d port=21001, got node=%d port=%d", nodeA, nodeAfter, portAfter)
	}
}

func TestForwardBatchChangeTunnelRollbackOnSyncFailure(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	now := time.Now().UnixMilli()

	adminToken, err := auth.GenerateToken(1, "admin_user", 0, secret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'batch_switch_user', '3c85cdebade1c51cf64ca9f3c09d182d', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES('batch-switch-tunnel-a', 1.0, 1, 'tls', 99999, ?, ?, 1, NULL, 0)
	`, now, now).Error; err != nil {
		t.Fatalf("insert tunnel A: %v", err)
	}
	tunnelA := mustLastInsertID(t, repo, "batch-switch-tunnel-a")

	if err := repo.DB().Exec(`
		INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES('batch-switch-tunnel-b', 1.0, 1, 'tls', 99999, ?, ?, 1, NULL, 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert tunnel B: %v", err)
	}
	tunnelB := mustLastInsertID(t, repo, "batch-switch-tunnel-b")

	if err := repo.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
		VALUES('batch-switch-node-a', 'batch-switch-node-a-secret', '10.11.0.1', '10.11.0.1', '', '23000-23010', '', 'v1', 1, 1, 1, ?, ?, 1, '[::]', '[::]', 0)
	`, now, now).Error; err != nil {
		t.Fatalf("insert node A: %v", err)
	}
	nodeA := mustLastInsertID(t, repo, "batch-switch-node-a")

	if err := repo.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
		VALUES('batch-switch-node-b', 'batch-switch-node-b-secret', '10.11.0.2', '10.11.0.2', '', '24000-24010', '', 'v1', 1, 1, 1, ?, ?, 1, '[::]', '[::]', 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert node B: %v", err)
	}
	nodeB := mustLastInsertID(t, repo, "batch-switch-node-b")

	if err := repo.DB().Exec(`INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol) VALUES(?, 1, ?, 23001, 'round', 1, 'tls')`, tunnelA, nodeA).Error; err != nil {
		t.Fatalf("insert chain_tunnel A: %v", err)
	}
	if err := repo.DB().Exec(`INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol) VALUES(?, 1, ?, 24001, 'round', 1, 'tls')`, tunnelB, nodeB).Error; err != nil {
		t.Fatalf("insert chain_tunnel B: %v", err)
	}

	if err := repo.DB().Exec(`INSERT INTO user_tunnel(id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status) VALUES(20, 2, ?, NULL, 999, 99999, 0, 0, 1, 2727251700000, 1)`, tunnelA).Error; err != nil {
		t.Fatalf("insert user_tunnel A: %v", err)
	}
	if err := repo.DB().Exec(`INSERT INTO user_tunnel(id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status) VALUES(21, 2, ?, NULL, 999, 99999, 0, 0, 1, 2727251700000, 1)`, tunnelB).Error; err != nil {
		t.Fatalf("insert user_tunnel B: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(2, 'batch_switch_user', 'batch-switch-forward', ?, '1.1.1.1:443', 'fifo', 0, 0, ?, ?, 1, 0)
	`, tunnelA, now, now).Error; err != nil {
		t.Fatalf("insert forward: %v", err)
	}
	forwardID := mustLastInsertID(t, repo, "batch-switch-forward")

	if err := repo.DB().Exec(`INSERT INTO forward_port(forward_id, node_id, port) VALUES(?, ?, ?)`, forwardID, nodeA, 23001).Error; err != nil {
		t.Fatalf("insert forward_port: %v", err)
	}

	payload := `{"forwardIds":[` + jsonNumber(forwardID) + `],"targetTunnelId":` + jsonNumber(tunnelB) + `}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/batch-change-tunnel", bytes.NewBufferString(payload))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	var out response.R
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Code != 0 {
		t.Fatalf("expected API success envelope, got code=%d msg=%q", out.Code, out.Msg)
	}

	result, ok := out.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", out.Data)
	}
	if int(result["failCount"].(float64)) != 1 {
		t.Fatalf("expected failCount=1, got %v", result["failCount"])
	}

	tunnelAfter := mustQueryInt64(t, repo, `SELECT tunnel_id FROM forward WHERE id = ?`, forwardID)
	if tunnelAfter != tunnelA {
		t.Fatalf("expected tunnel rollback to %d, got %d", tunnelA, tunnelAfter)
	}

	nodeAfter, portAfter := mustQueryInt64Int(t, repo, `SELECT node_id, port FROM forward_port WHERE forward_id = ? LIMIT 1`, forwardID)
	if nodeAfter != nodeA || portAfter != 23001 {
		t.Fatalf("expected forward_port rollback to node=%d port=23001, got node=%d port=%d", nodeA, nodeAfter, portAfter)
	}
}

func TestUserTunnelReassignmentKeepsStableID(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	now := time.Now().UnixMilli()

	adminToken, err := auth.GenerateToken(1, "admin_user", 0, secret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(100, 'stable_user', 'pwd', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES('stable-tunnel', 1.0, 1, 'tls', 99999, ?, ?, 1, NULL, 0)
	`, now, now).Error; err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}
	tunnelID := mustLastInsertID(t, repo, "stable-tunnel")

	// 1. Assign permission (creates new user_tunnel)
	// userTunnelBatchAssign expects structure: {userId: 123, tunnels: [{tunnelId: 456, ...}]}
	assignPayload := `{"userId":100,"tunnels":[{"tunnelId":` + jsonNumber(tunnelID) + `}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/user/batch-assign", bytes.NewBufferString(assignPayload))
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	var out response.R
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Code != 0 {
		t.Fatalf("expected code 0, got %d msg=%q", out.Code, out.Msg)
	}

	initialID := mustQueryInt64(t, repo, `SELECT id FROM user_tunnel WHERE user_id = 100 AND tunnel_id = ?`, tunnelID)

	// 2. Re-assign permission (should UPDATE, not INSERT)
	reassignPayload := `{"userId":100,"tunnels":[{"tunnelId":` + jsonNumber(tunnelID) + `}]}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/user/batch-assign", bytes.NewBufferString(reassignPayload))
	req2.Header.Set("Authorization", adminToken)
	req2.Header.Set("Content-Type", "application/json")
	res2 := httptest.NewRecorder()
	router.ServeHTTP(res2, req2)

	var out2 response.R
	if err := json.NewDecoder(res2.Body).Decode(&out2); err != nil {
		t.Fatalf("decode response 2: %v", err)
	}
	if out2.Code != 0 {
		t.Fatalf("expected code 0, got %d msg=%q", out2.Code, out2.Msg)
	}

	// 3. Verify stable ID and no duplicates
	count := mustQueryInt(t, repo, `SELECT COUNT(1) FROM user_tunnel WHERE user_id = 100 AND tunnel_id = ?`, tunnelID)
	if count != 1 {
		t.Fatalf("expected exactly 1 user_tunnel record, got %d", count)
	}

	currentID := mustQueryInt64(t, repo, `SELECT id FROM user_tunnel WHERE user_id = 100 AND tunnel_id = ?`, tunnelID)

	if currentID != initialID {
		t.Fatalf("user_tunnel ID changed from %d to %d (unstable ID!)", initialID, currentID)
	}
}

func jsonNumber(v int64) string {
	return strconv.FormatInt(v, 10)
}
