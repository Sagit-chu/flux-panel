package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/store/repo"
)

func TestFederationShareCreateRejectsRemoteNode(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	if err := r.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx, is_remote, remote_url, remote_token, remote_config)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "remote-share-node", "remote-share-secret", "10.10.10.1", "10.10.10.1", "", "20000-20010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0, 1, "http://peer.example", "peer-token", `{"shareId":1}`).Error; err != nil {
		t.Fatalf("insert remote node: %v", err)
	}
	remoteNodeID := mustLastInsertID(t, r, "remote-share-node")

	body, err := json.Marshal(createPeerShareRequest{
		Name:           "remote-node-share",
		NodeID:         remoteNodeID,
		MaxBandwidth:   0,
		ExpiryTime:     0,
		PortRangeStart: 20000,
		PortRangeEnd:   20010,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	h.federationShareCreate(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != -1 {
		t.Fatalf("expected response code -1, got %d", payload.Code)
	}
	if payload.Msg != "Only local nodes can be shared" {
		t.Fatalf("expected rejection message %q, got %q", "Only local nodes can be shared", payload.Msg)
	}

	shareCount := mustQueryInt(t, r, `SELECT COUNT(1) FROM peer_share WHERE node_id = ?`, remoteNodeID)
	if shareCount != 0 {
		t.Fatalf("expected no share rows for remote node, got %d", shareCount)
	}
}

func TestFederationShareCreateRejectsInvalidAllowedIPs(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	if err := r.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx, is_remote, remote_url, remote_token, remote_config)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "local-share-node", "local-share-secret", "10.20.30.40", "10.20.30.40", "", "21000-21010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0, 0, "", "", "").Error; err != nil {
		t.Fatalf("insert local node: %v", err)
	}
	localNodeID := mustLastInsertID(t, r, "local-share-node")

	body, err := json.Marshal(createPeerShareRequest{
		Name:           "local-node-share",
		NodeID:         localNodeID,
		MaxBandwidth:   0,
		ExpiryTime:     0,
		PortRangeStart: 21000,
		PortRangeEnd:   21010,
		AllowedIPs:     "bad-ip-entry",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	h.federationShareCreate(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != -1 {
		t.Fatalf("expected response code -1, got %d", payload.Code)
	}
	if !strings.Contains(payload.Msg, "Invalid allowed IP or CIDR") {
		t.Fatalf("expected invalid IP message, got %q", payload.Msg)
	}

	shareCount := mustQueryInt(t, r, `SELECT COUNT(1) FROM peer_share WHERE node_id = ?`, localNodeID)
	if shareCount != 0 {
		t.Fatalf("expected no share rows for node, got %d", shareCount)
	}
}

func TestFederationShareListIncludesRemoteUsedPorts(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	if err := r.CreatePeerShare(&repo.PeerShare{
		Name:           "provider-share",
		NodeID:         9,
		Token:          "share-list-token",
		MaxBandwidth:   1024,
		CurrentFlow:    512,
		PortRangeStart: 22000,
		PortRangeEnd:   22010,
		IsActive:       1,
		CreatedTime:    now,
		UpdatedTime:    now,
	}); err != nil {
		t.Fatalf("create peer share: %v", err)
	}

	share, err := r.GetPeerShareByToken("share-list-token")
	if err != nil || share == nil {
		t.Fatalf("load peer share: %v", err)
	}

	if err := r.DB().Exec(`
		INSERT INTO peer_share_runtime(share_id, node_id, reservation_id, resource_key, binding_id, role, chain_name, service_name, protocol, strategy, port, target, applied, status, created_time, updated_time)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),
		      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),
		      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		share.ID, share.NodeID, "r-1", "rk-1", "b-1", "middle", "fed_chain_1", "fed_svc_1", "tls", "round", 22001, "", 1, 1, now, now,
		share.ID, share.NodeID, "r-2", "rk-2", "b-2", "exit", "", "fed_svc_2", "tls", "round", 22002, "", 1, 1, now, now,
		share.ID, share.NodeID, "r-3", "rk-3", "", "", "", "", "tls", "round", 22003, "", 0, 0, now, now,
	).Error; err != nil {
		t.Fatalf("insert peer_share_runtime rows: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/list", nil)
	res := httptest.NewRecorder()
	h.federationShareList(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 0 {
		t.Fatalf("expected response code 0, got %d (%s)", payload.Code, payload.Msg)
	}

	rows, ok := payload.Data.([]interface{})
	if !ok || len(rows) == 0 {
		t.Fatalf("expected non-empty share list, got %T", payload.Data)
	}

	first, ok := rows[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected share row object, got %T", rows[0])
	}

	if int(first["activeRuntimeNum"].(float64)) != 2 {
		t.Fatalf("expected activeRuntimeNum=2, got %v", first["activeRuntimeNum"])
	}

	usedPortsRaw, ok := first["usedPorts"].([]interface{})
	if !ok {
		t.Fatalf("expected usedPorts array, got %T", first["usedPorts"])
	}
	if len(usedPortsRaw) != 2 {
		t.Fatalf("expected 2 used ports, got %d", len(usedPortsRaw))
	}
	if int(usedPortsRaw[0].(float64)) != 22001 || int(usedPortsRaw[1].(float64)) != 22002 {
		t.Fatalf("unexpected used ports payload: %v", usedPortsRaw)
	}

	detailsRaw, ok := first["usedPortDetails"].([]interface{})
	if !ok {
		t.Fatalf("expected usedPortDetails array, got %T", first["usedPortDetails"])
	}
	if len(detailsRaw) != 2 {
		t.Fatalf("expected 2 usedPortDetails rows, got %d", len(detailsRaw))
	}
}

func TestFederationShareDeleteCleansUpRuntimes(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	if err := r.CreatePeerShare(&repo.PeerShare{
		Name:           "delete-cleanup-share",
		NodeID:         99,
		Token:          "delete-cleanup-token",
		MaxBandwidth:   4096,
		PortRangeStart: 40000,
		PortRangeEnd:   40010,
		IsActive:       1,
		CreatedTime:    now,
		UpdatedTime:    now,
	}); err != nil {
		t.Fatalf("create peer share: %v", err)
	}

	share, err := r.GetPeerShareByToken("delete-cleanup-token")
	if err != nil || share == nil {
		t.Fatalf("load peer share: %v", err)
	}

	if err := r.DB().Exec(`
		INSERT INTO peer_share_runtime(share_id, node_id, reservation_id, resource_key, binding_id, role, chain_name, service_name, protocol, strategy, port, target, applied, status, created_time, updated_time)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),
		      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		share.ID, 99, "dc-r1", "dc-rk1", "dc-b1", "exit", "", "fed_svc_dc1", "tls", "round", 40001, "", 1, 1, now, now,
		share.ID, 99, "dc-r2", "dc-rk2", "dc-b2", "middle", "fed_chain_dc2", "fed_svc_dc2", "tls", "round", 40002, "", 1, 1, now, now,
	).Error; err != nil {
		t.Fatalf("insert peer_share_runtime rows: %v", err)
	}

	runtimeCount := mustQueryInt(t, r, `SELECT COUNT(1) FROM peer_share_runtime WHERE share_id = ? AND status = 1`, share.ID)
	if runtimeCount != 2 {
		t.Fatalf("expected 2 active runtimes before delete, got %d", runtimeCount)
	}

	body, err := json.Marshal(deletePeerShareRequest{ID: share.ID})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/delete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	h.federationShareDelete(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 0 {
		t.Fatalf("expected response code 0, got %d (%s)", payload.Code, payload.Msg)
	}

	shareCount := mustQueryInt(t, r, `SELECT COUNT(1) FROM peer_share WHERE id = ?`, share.ID)
	if shareCount != 0 {
		t.Fatalf("expected peer_share deleted, got %d rows", shareCount)
	}

	runtimeCountAfter := mustQueryInt(t, r, `SELECT COUNT(1) FROM peer_share_runtime WHERE share_id = ?`, share.ID)
	if runtimeCountAfter != 0 {
		t.Fatalf("expected all peer_share_runtime rows deleted, got %d", runtimeCountAfter)
	}
}

func TestFederationRemoteUsageListSyncErrorFallback(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	if err := r.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx, is_remote, remote_url, remote_token, remote_config)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "sync-error-node", "sync-error-secret", "10.50.60.70", "10.50.60.70", "", "32000-32010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0, 1, "http://unreachable.invalid:9999", "bad-token", `{"shareId":42,"maxBandwidth":5368709120,"currentFlow":999999,"portRangeStart":32000,"portRangeEnd":32010}`).Error; err != nil {
		t.Fatalf("insert remote node: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/remote-usage/list", nil)
	res := httptest.NewRecorder()
	h.federationRemoteUsageList(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 0 {
		t.Fatalf("expected response code 0, got %d (%s)", payload.Code, payload.Msg)
	}

	rows, ok := payload.Data.([]interface{})
	if !ok || len(rows) == 0 {
		t.Fatalf("expected non-empty usage list, got %T", payload.Data)
	}

	first, ok := rows[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected row map, got %T", rows[0])
	}

	if int64(first["shareId"].(float64)) != 42 {
		t.Fatalf("expected stale shareId=42 on sync failure, got %v", first["shareId"])
	}
	if int64(first["currentFlow"].(float64)) != 999999 {
		t.Fatalf("expected stale currentFlow=999999 on sync failure, got %v", first["currentFlow"])
	}

	syncErr, _ := first["syncError"].(string)
	if syncErr == "" {
		t.Fatalf("expected non-empty syncError field on unreachable provider")
	}
}

func TestFederationShareResetFlow(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()
	if err := r.CreatePeerShare(&repo.PeerShare{
		Name:           "reset-flow-share",
		NodeID:         11,
		Token:          "reset-flow-token",
		MaxBandwidth:   4096,
		CurrentFlow:    2048,
		PortRangeStart: 23000,
		PortRangeEnd:   23010,
		IsActive:       1,
		CreatedTime:    now,
		UpdatedTime:    now,
	}); err != nil {
		t.Fatalf("create peer share: %v", err)
	}
	share, err := r.GetPeerShareByToken("reset-flow-token")
	if err != nil || share == nil {
		t.Fatalf("load peer share: %v", err)
	}

	body, err := json.Marshal(resetPeerShareFlowRequest{ID: share.ID})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/reset-flow", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	h.federationShareResetFlow(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 0 {
		t.Fatalf("expected response code 0, got %d (%s)", payload.Code, payload.Msg)
	}

	updated, err := r.GetPeerShare(share.ID)
	if err != nil || updated == nil {
		t.Fatalf("reload peer share: %v", err)
	}
	if updated.CurrentFlow != 0 {
		t.Fatalf("expected current flow reset to 0, got %d", updated.CurrentFlow)
	}
}

func TestFederationRemoteUsageList(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	if err := r.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx, is_remote, remote_url, remote_token, remote_config)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "remote-consumer-node", "remote-consumer-secret", "10.30.40.50", "10.30.40.50", "", "31000-31010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0, 1, "http://peer.example", "peer-token", `{"shareId":88,"maxBandwidth":2147483648,"currentFlow":1073741824,"portRangeStart":31000,"portRangeEnd":31010}`).Error; err != nil {
		t.Fatalf("insert remote node: %v", err)
	}
	nodeID := mustLastInsertID(t, r, "remote-consumer-node")

	if err := r.DB().Exec(`INSERT INTO tunnel(name, type, protocol, flow, created_time, updated_time, status, in_ip, inx) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, "consumer-tunnel-a", 2, "tls", 1, now, now, 1, "", 0).Error; err != nil {
		t.Fatalf("insert tunnel a: %v", err)
	}
	tunnelAID := mustLastInsertID(t, r, "consumer-tunnel-a")

	if err := r.DB().Exec(`INSERT INTO tunnel(name, type, protocol, flow, created_time, updated_time, status, in_ip, inx) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, "consumer-tunnel-b", 2, "tls", 1, now, now, 1, "", 0).Error; err != nil {
		t.Fatalf("insert tunnel b: %v", err)
	}
	tunnelBID := mustLastInsertID(t, r, "consumer-tunnel-b")

	if err := r.DB().Exec(`
		INSERT INTO federation_tunnel_binding(tunnel_id, node_id, chain_type, hop_inx, remote_url, resource_key, remote_binding_id, allocated_port, status, created_time, updated_time)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),
		      (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		tunnelAID, nodeID, 2, 1, "http://peer.example", "rk-a", "rb-a", 31001, 1, now, now,
		tunnelBID, nodeID, 3, 0, "http://peer.example", "rk-b", "rb-b", 31002, 1, now, now,
	).Error; err != nil {
		t.Fatalf("insert federation bindings: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/share/remote-usage/list", nil)
	res := httptest.NewRecorder()
	h.federationRemoteUsageList(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 0 {
		t.Fatalf("expected response code 0, got %d (%s)", payload.Code, payload.Msg)
	}

	rows, ok := payload.Data.([]interface{})
	if !ok || len(rows) == 0 {
		t.Fatalf("expected non-empty usage list, got %T", payload.Data)
	}

	first, ok := rows[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first usage row map, got %T", rows[0])
	}
	if int64(first["shareId"].(float64)) != 88 {
		t.Fatalf("expected shareId=88, got %v", first["shareId"])
	}

	usedPortsRaw, ok := first["usedPorts"].([]interface{})
	if !ok {
		t.Fatalf("expected usedPorts array, got %T", first["usedPorts"])
	}
	if len(usedPortsRaw) != 2 {
		t.Fatalf("expected 2 used ports, got %d", len(usedPortsRaw))
	}
	if int(usedPortsRaw[0].(float64)) != 31001 || int(usedPortsRaw[1].(float64)) != 31002 {
		t.Fatalf("unexpected used ports payload: %v", usedPortsRaw)
	}

	bindingsRaw, ok := first["bindings"].([]interface{})
	if !ok {
		t.Fatalf("expected bindings array, got %T", first["bindings"])
	}
	if len(bindingsRaw) != 2 {
		t.Fatalf("expected 2 binding rows, got %d", len(bindingsRaw))
	}
}

func TestAuthPeerAllowedIPs(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret")
	now := time.Now().UnixMilli()

	tests := []struct {
		name        string
		allowedIPs  string
		remoteAddr  string
		xff         string
		wantAllowed bool
	}{
		{
			name:        "exact ip allowed",
			allowedIPs:  "203.0.113.10",
			remoteAddr:  "203.0.113.10:23456",
			wantAllowed: true,
		},
		{
			name:        "cidr allowed",
			allowedIPs:  "203.0.113.0/24",
			remoteAddr:  "203.0.113.11:23456",
			wantAllowed: true,
		},
		{
			name:        "trusted proxy xff allowed",
			allowedIPs:  "198.51.100.20",
			remoteAddr:  "172.20.0.3:34567",
			xff:         "198.51.100.20, 172.20.0.3",
			wantAllowed: true,
		},
		{
			name:        "ipv4-mapped proxy xff allowed",
			allowedIPs:  "198.51.100.20",
			remoteAddr:  "[::ffff:172.20.0.3]:34567",
			xff:         "198.51.100.20, 172.20.0.3",
			wantAllowed: true,
		},
		{
			name:        "non whitelisted ip denied",
			allowedIPs:  "203.0.113.10",
			remoteAddr:  "203.0.113.99:23456",
			wantAllowed: false,
		},
	}

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := fmt.Sprintf("share-token-%d", idx)
			if err := r.CreatePeerShare(&repo.PeerShare{
				Name:           "share-" + tt.name,
				NodeID:         1,
				Token:          token,
				PortRangeStart: 10000,
				PortRangeEnd:   10010,
				IsActive:       1,
				CreatedTime:    now,
				UpdatedTime:    now,
				AllowedIPs:     tt.allowedIPs,
			}); err != nil {
				t.Fatalf("create peer share: %v", err)
			}

			nextCalled := false
			wrapped := h.authPeer(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				response.WriteJSON(w, response.OKEmpty())
			})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/connect", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			req.RemoteAddr = tt.remoteAddr

			res := httptest.NewRecorder()
			wrapped(res, req)

			var payload response.R
			if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if tt.wantAllowed {
				if !nextCalled {
					t.Fatalf("expected next handler to be called")
				}
				if payload.Code != 0 {
					t.Fatalf("expected code 0, got %d (%s)", payload.Code, payload.Msg)
				}
				return
			}

			if nextCalled {
				t.Fatalf("expected next handler not to be called")
			}
			if payload.Code != 403 {
				t.Fatalf("expected code 403, got %d (%s)", payload.Code, payload.Msg)
			}
			if payload.Msg != "IP not allowed" {
				t.Fatalf("expected IP rejection message, got %q", payload.Msg)
			}
		})
	}
}
