package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-backend/internal/auth"
	httpserver "go-backend/internal/http"
	"go-backend/internal/http/handler"
	"go-backend/internal/http/response"
	"go-backend/internal/store/sqlite"
)

func TestDiagnosisChainCoverageContracts(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupDiagnosisContractRouter(t, secret)
	now := time.Now().UnixMilli()

	if _, err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(2, 'normal_user', '3c85cdebade1c51cf64ca9f3c09d182d', 1, 2727251700000, 99999, 0, 0, 1, 99999, ?, ?, 1)
	`, now, now); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	tunnelRes, err := repo.DB().Exec(`
		INSERT INTO tunnel(name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "diagnose-chain-tunnel", 1.0, 2, "tls", 99999, now, now, 1, nil, 0)
	if err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}
	tunnelID, err := tunnelRes.LastInsertId()
	if err != nil {
		t.Fatalf("get tunnel id: %v", err)
	}

	insertNode := func(name, ip string) int64 {
		res, err := repo.DB().Exec(`
			INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, name, name+"-secret", ip, ip, "", "30000-30010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0)
		if err != nil {
			t.Fatalf("insert node %s: %v", name, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("get node id %s: %v", name, err)
		}
		return id
	}

	entryNodeID := insertNode("entry-node", "10.0.1.10")
	chainNodeID := insertNode("chain-node", "10.0.1.20")
	exitNodeID := insertNode("exit-node", "10.0.1.30")

	if _, err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 1, ?, 30001, 'round', 1, 'tls')
	`, tunnelID, entryNodeID); err != nil {
		t.Fatalf("insert entry chain: %v", err)
	}
	if _, err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 2, ?, 30002, 'round', 1, 'tls')
	`, tunnelID, chainNodeID); err != nil {
		t.Fatalf("insert middle chain: %v", err)
	}
	if _, err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(?, 3, ?, 30003, 'round', 1, 'tls')
	`, tunnelID, exitNodeID); err != nil {
		t.Fatalf("insert exit chain: %v", err)
	}

	forwardRes, err := repo.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(?, ?, ?, ?, ?, ?, 0, 0, ?, ?, 1, ?)
	`, 2, "normal_user", "chain-forward", tunnelID, "8.8.8.8:53", "fifo", now, now, 0)
	if err != nil {
		t.Fatalf("insert forward: %v", err)
	}
	forwardID, err := forwardRes.LastInsertId()
	if err != nil {
		t.Fatalf("get forward id: %v", err)
	}

	userToken, err := auth.GenerateToken(2, "normal_user", 1, secret)
	if err != nil {
		t.Fatalf("generate user token: %v", err)
	}
	adminToken, err := auth.GenerateToken(1, "admin_user", 0, secret)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}

	t.Run("forward diagnose includes entry chain exit paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/diagnose", bytes.NewBufferString(`{"forwardId":`+strconv.FormatInt(forwardID, 10)+`}`))
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

		hasEntryToChain := false
		hasChainToExit := false
		hasExitToTarget := false
		for _, raw := range results {
			item, ok := raw.(map[string]interface{})
			if !ok {
				t.Fatalf("expected result object, got %T", raw)
			}
			if strings.TrimSpace(valueAsString(item["message"])) == "" {
				t.Fatalf("expected non-empty message field")
			}
			from := valueAsInt(item["fromChainType"])
			to := valueAsInt(item["toChainType"])
			if from == 1 && to == 2 {
				hasEntryToChain = true
			}
			if from == 2 && to == 3 {
				hasChainToExit = true
			}
			if from == 3 {
				hasExitToTarget = true
			}
		}

		if !hasEntryToChain || !hasChainToExit || !hasExitToTarget {
			t.Fatalf("expected entry->chain, chain->exit, exit->target coverage; got entry=%v chain=%v exit=%v", hasEntryToChain, hasChainToExit, hasExitToTarget)
		}
	})

	t.Run("tunnel diagnose includes entry chain exit groups", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tunnel/diagnose", bytes.NewBufferString(`{"tunnelId":`+strconv.FormatInt(tunnelID, 10)+`}`))
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

		hasEntry := false
		hasChain := false
		hasExit := false
		for _, raw := range results {
			item, ok := raw.(map[string]interface{})
			if !ok {
				t.Fatalf("expected result object, got %T", raw)
			}
			if strings.TrimSpace(valueAsString(item["message"])) == "" {
				t.Fatalf("expected non-empty message field")
			}
			switch valueAsInt(item["fromChainType"]) {
			case 1:
				hasEntry = true
			case 2:
				hasChain = true
			case 3:
				hasExit = true
			}
		}

		if !hasEntry || !hasChain || !hasExit {
			t.Fatalf("expected entry/chain/exit groups, got entry=%v chain=%v exit=%v", hasEntry, hasChain, hasExit)
		}
	})
}

func valueAsInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func valueAsString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func setupDiagnosisContractRouter(t *testing.T, jwtSecret string) (http.Handler, *sqlite.Repository) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "diagnosis-contract.db")
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = repo.Close()
	})

	h := handler.New(repo, jwtSecret)
	return httpserver.NewRouter(h, jwtSecret), repo
}
