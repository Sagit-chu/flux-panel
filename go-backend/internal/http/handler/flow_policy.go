package handler

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const bytesPerGB int64 = 1024 * 1024 * 1024

type userTunnelPolicy struct {
	ID       int64
	UserID   int64
	TunnelID int64
	Flow     int64
	InFlow   int64
	OutFlow  int64
	ExpTime  int64
	Status   int
}

type gostConfigSnapshot struct {
	Services []namedConfigItem `json:"services"`
	Chains   []namedConfigItem `json:"chains"`
	Limiters []namedConfigItem `json:"limiters"`
}

type namedConfigItem struct {
	Name string `json:"name"`
}

func (h *Handler) processFlowItem(item flowItem) {
	serviceName := strings.TrimSpace(item.N)
	if serviceName == "" || serviceName == "web_api" {
		return
	}

	forwardID, userID, userTunnelID, ok := parseFlowServiceIDs(serviceName)
	if !ok {
		return
	}

	inFlow, outFlow := h.scaleFlowByTunnel(forwardID, item.D, item.U)
	_ = h.repo.AddFlow(forwardID, userID, userTunnelID, inFlow, outFlow)

	if userTunnelID > 0 {
		h.enforceFlowPolicies(userID, userTunnelID)
	}
}

func parseFlowServiceIDs(serviceName string) (int64, int64, int64, bool) {
	parts := strings.Split(serviceName, "_")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}

	forwardID, err1 := strconv.ParseInt(parts[0], 10, 64)
	userID, err2 := strconv.ParseInt(parts[1], 10, 64)
	userTunnelID, err3 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil || forwardID <= 0 || userID <= 0 {
		return 0, 0, 0, false
	}

	return forwardID, userID, userTunnelID, true
}

func (h *Handler) scaleFlowByTunnel(forwardID int64, inFlow int64, outFlow int64) (int64, int64) {
	forward, err := h.getForwardRecord(forwardID)
	if err != nil || forward == nil {
		return inFlow, outFlow
	}

	tunnel, err := h.getTunnelRecord(forward.TunnelID)
	if err != nil || tunnel == nil {
		return inFlow, outFlow
	}

	scaledIn := int64(float64(inFlow)*tunnel.TrafficRatio) * tunnel.Flow
	scaledOut := int64(float64(outFlow)*tunnel.TrafficRatio) * tunnel.Flow
	return scaledIn, scaledOut
}

func (h *Handler) enforceFlowPolicies(userID int64, userTunnelID int64) {
	now := time.Now().UnixMilli()

	if h.shouldPauseUser(userID, now) {
		h.pauseUserForwards(userID, now)
	}

	policy, err := h.getUserTunnelPolicy(userTunnelID)
	if err != nil || policy == nil {
		return
	}

	if shouldPauseUserTunnel(policy, now) {
		h.pauseUserTunnelForwards(policy.UserID, policy.TunnelID, now)
	}
}

func (h *Handler) shouldPauseUser(userID int64, now int64) bool {
	user, err := h.repo.GetUserByID(userID)
	if err != nil || user == nil {
		return false
	}

	flowLimit := user.Flow * bytesPerGB
	current := user.InFlow + user.OutFlow
	if flowLimit < current {
		return true
	}
	if user.ExpTime > 0 && user.ExpTime <= now {
		return true
	}
	return user.Status != 1
}

func shouldPauseUserTunnel(policy *userTunnelPolicy, now int64) bool {
	if policy == nil {
		return false
	}

	flowLimit := policy.Flow * bytesPerGB
	current := policy.InFlow + policy.OutFlow
	if current >= flowLimit {
		return true
	}
	if policy.ExpTime > 0 && policy.ExpTime <= now {
		return true
	}
	return policy.Status != 1
}

func (h *Handler) getUserTunnelPolicy(userTunnelID int64) (*userTunnelPolicy, error) {
	if userTunnelID <= 0 {
		return nil, nil
	}

	row := h.repo.DB().QueryRow(`
		SELECT id, user_id, tunnel_id, flow, in_flow, out_flow, exp_time, status
		FROM user_tunnel
		WHERE id = ?
		LIMIT 1
	`, userTunnelID)

	var policy userTunnelPolicy
	if err := row.Scan(&policy.ID, &policy.UserID, &policy.TunnelID, &policy.Flow, &policy.InFlow, &policy.OutFlow, &policy.ExpTime, &policy.Status); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (h *Handler) pauseUserForwards(userID int64, now int64) {
	forwards, err := h.listActiveForwardsByUser(userID)
	if err != nil {
		return
	}
	h.pauseForwardRecords(forwards, now)
}

func (h *Handler) pauseUserTunnelForwards(userID int64, tunnelID int64, now int64) {
	forwards, err := h.listActiveForwardsByUserTunnel(userID, tunnelID)
	if err != nil {
		return
	}
	h.pauseForwardRecords(forwards, now)
}

func (h *Handler) pauseForwardRecords(forwards []forwardRecord, now int64) {
	for i := range forwards {
		forward := forwards[i]
		_ = h.controlForwardServices(&forward, "PauseService", false)
		_, _ = h.repo.DB().Exec(`UPDATE forward SET status = 0, updated_time = ? WHERE id = ?`, now, forward.ID)
	}
}

func (h *Handler) listActiveForwardsByUser(userID int64) ([]forwardRecord, error) {
	rows, err := h.repo.DB().Query(`
		SELECT id, user_id, user_name, name, tunnel_id, remote_addr, strategy, status
		FROM forward
		WHERE user_id = ? AND status = 1
		ORDER BY id ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanForwardRecords(rows)
}

func (h *Handler) listActiveForwardsByUserTunnel(userID int64, tunnelID int64) ([]forwardRecord, error) {
	rows, err := h.repo.DB().Query(`
		SELECT id, user_id, user_name, name, tunnel_id, remote_addr, strategy, status
		FROM forward
		WHERE user_id = ? AND tunnel_id = ? AND status = 1
		ORDER BY id ASC
	`, userID, tunnelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanForwardRecords(rows)
}

func scanForwardRecords(rows *sql.Rows) ([]forwardRecord, error) {
	out := make([]forwardRecord, 0)
	for rows.Next() {
		var record forwardRecord
		if err := rows.Scan(&record.ID, &record.UserID, &record.UserName, &record.Name, &record.TunnelID, &record.RemoteAddr, &record.Strategy, &record.Status); err != nil {
			return nil, err
		}
		if strings.TrimSpace(record.Strategy) == "" {
			record.Strategy = "fifo"
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (h *Handler) cleanNodeConfigs(nodeID int64, rawConfig string) {
	if h == nil || h.repo == nil || h.repo.DB() == nil || nodeID <= 0 {
		return
	}
	if strings.TrimSpace(rawConfig) == "" {
		return
	}

	var snapshot gostConfigSnapshot
	if err := json.Unmarshal([]byte(rawConfig), &snapshot); err != nil {
		return
	}

	h.cleanOrphanedServices(nodeID, snapshot.Services)
	h.cleanOrphanedChains(nodeID, snapshot.Chains)
	h.cleanOrphanedLimiters(nodeID, snapshot.Limiters)
}

func (h *Handler) cleanOrphanedServices(nodeID int64, services []namedConfigItem) {
	for _, item := range services {
		name := strings.TrimSpace(item.Name)
		if name == "" || name == "web_api" {
			continue
		}

		parts := strings.Split(name, "_")
		if len(parts) >= 3 {
			forwardID, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil && forwardID > 0 && !h.forwardExists(forwardID) {
				_, _ = h.sendNodeCommand(nodeID, "DeleteService", map[string]interface{}{"services": []string{name, parts[0] + "_" + parts[1] + "_" + parts[2], parts[0] + "_" + parts[1] + "_" + parts[2] + "_tcp", parts[0] + "_" + parts[1] + "_" + parts[2] + "_udp"}}, false, true)
				continue
			}
		}
		suffix := parts[len(parts)-1]

		switch suffix {
		case "tls":
			tunnelID, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil || tunnelID <= 0 || h.tunnelExists(tunnelID) {
				continue
			}
			_, _ = h.sendNodeCommand(nodeID, "DeleteService", map[string]interface{}{"services": []string{name}}, false, true)
		case "tcp":
			if len(parts) < 4 {
				continue
			}
			forwardID, err := strconv.ParseInt(parts[0], 10, 64)
			if err != nil || forwardID <= 0 || h.forwardExists(forwardID) {
				continue
			}
			base := strings.TrimSuffix(name, "_tcp")
			_, _ = h.sendNodeCommand(nodeID, "DeleteService", map[string]interface{}{"services": []string{base + "_tcp", base + "_udp"}}, false, true)
		}
	}
}

func (h *Handler) cleanOrphanedChains(nodeID int64, chains []namedConfigItem) {
	for _, item := range chains {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}

		idx := strings.LastIndex(name, "_")
		if idx <= 0 || idx >= len(name)-1 {
			continue
		}
		tunnelID, err := strconv.ParseInt(name[idx+1:], 10, 64)
		if err != nil || tunnelID <= 0 || h.tunnelExists(tunnelID) {
			continue
		}
		_, _ = h.sendNodeCommand(nodeID, "DeleteChains", map[string]interface{}{"chain": name}, false, true)
	}
}

func (h *Handler) cleanOrphanedLimiters(nodeID int64, limiters []namedConfigItem) {
	for _, item := range limiters {
		name := strings.TrimSpace(item.Name)
		if name == "" || h.speedLimiterExists(name) {
			continue
		}
		_, _ = h.sendNodeCommand(nodeID, "DeleteLimiters", map[string]interface{}{"limiter": name}, false, true)
	}
}

func (h *Handler) tunnelExists(tunnelID int64) bool {
	var count int
	err := h.repo.DB().QueryRow(`SELECT COUNT(1) FROM tunnel WHERE id = ?`, tunnelID).Scan(&count)
	return err == nil && count > 0
}

func (h *Handler) forwardExists(forwardID int64) bool {
	var count int
	err := h.repo.DB().QueryRow(`SELECT COUNT(1) FROM forward WHERE id = ?`, forwardID).Scan(&count)
	return err == nil && count > 0
}

func (h *Handler) speedLimiterExists(name string) bool {
	if name == "" {
		return false
	}
	id, err := strconv.ParseInt(name, 10, 64)
	if err != nil || id <= 0 {
		return false
	}

	var count int
	err = h.repo.DB().QueryRow(`SELECT COUNT(1) FROM speed_limit WHERE id = ?`, id).Scan(&count)
	return err == nil && count > 0
}
