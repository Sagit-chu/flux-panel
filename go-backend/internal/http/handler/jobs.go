package handler

import (
	"context"
	"database/sql"
	"time"
)

func (h *Handler) StartBackgroundJobs() {
	if h == nil || h.repo == nil || h.repo.DB() == nil {
		return
	}

	h.jobsMu.Lock()
	if h.jobsStarted {
		h.jobsMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.jobsCancel = cancel
	h.jobsStarted = true
	h.jobsWG.Add(2)
	h.jobsMu.Unlock()

	go h.runHourlyStatsLoop(ctx)
	go h.runDailyMaintenanceLoop(ctx)
}

func (h *Handler) StopBackgroundJobs() {
	if h == nil {
		return
	}

	h.jobsMu.Lock()
	if !h.jobsStarted {
		h.jobsMu.Unlock()
		return
	}
	cancel := h.jobsCancel
	h.jobsCancel = nil
	h.jobsStarted = false
	h.jobsMu.Unlock()

	if cancel != nil {
		cancel()
	}
	h.jobsWG.Wait()
}

func (h *Handler) runHourlyStatsLoop(ctx context.Context) {
	defer h.jobsWG.Done()

	for {
		wait := durationUntilNextHour(time.Now())
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			h.runStatisticsFlowJob(time.Now())
		}
	}
}

func (h *Handler) runDailyMaintenanceLoop(ctx context.Context) {
	defer h.jobsWG.Done()

	for {
		wait := durationUntilNextDailyMaintenance(time.Now())
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			h.runResetAndExpiryJob(time.Now())
		}
	}
}

func durationUntilNextHour(now time.Time) time.Duration {
	next := now.Truncate(time.Hour).Add(time.Hour)
	return next.Sub(now)
}

func durationUntilNextDailyMaintenance(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 5, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next.Sub(now)
}

func (h *Handler) runStatisticsFlowJob(now time.Time) {
	if h == nil || h.repo == nil || h.repo.DB() == nil {
		return
	}

	db := h.repo.DB()
	nowMs := now.UnixMilli()
	cutoffMs := nowMs - int64((48*time.Hour)/time.Millisecond)
	_, _ = db.Exec(`DELETE FROM statistics_flow WHERE created_time < ?`, cutoffMs)

	hourMark := now.Truncate(time.Hour)
	hourText := hourMark.Format("15:04")
	createdTime := hourMark.UnixMilli()

	rows, err := db.Query(`SELECT id, in_flow, out_flow FROM user ORDER BY id ASC`)
	if err != nil {
		return
	}
	type userFlowSnapshot struct {
		userID  int64
		inFlow  int64
		outFlow int64
	}
	users := make([]userFlowSnapshot, 0)

	for rows.Next() {
		var userID int64
		var inFlow int64
		var outFlow int64
		if err := rows.Scan(&userID, &inFlow, &outFlow); err != nil {
			continue
		}
		users = append(users, userFlowSnapshot{userID: userID, inFlow: inFlow, outFlow: outFlow})
	}
	_ = rows.Close()

	for _, user := range users {
		currentTotal := user.inFlow + user.outFlow
		increment := currentTotal

		var lastTotal sql.NullInt64
		err := db.QueryRow(`SELECT total_flow FROM statistics_flow WHERE user_id = ? ORDER BY id DESC LIMIT 1`, user.userID).Scan(&lastTotal)
		if err == nil && lastTotal.Valid {
			increment = currentTotal - lastTotal.Int64
			if increment < 0 {
				increment = currentTotal
			}
		}

		_, _ = db.Exec(`
			INSERT INTO statistics_flow(user_id, flow, total_flow, time, created_time)
			VALUES(?, ?, ?, ?, ?)
		`, user.userID, increment, currentTotal, hourText, createdTime)
	}
}

func (h *Handler) runResetAndExpiryJob(now time.Time) {
	if h == nil || h.repo == nil || h.repo.DB() == nil {
		return
	}

	h.resetMonthlyFlow(now)
	h.disableExpiredUsers(now.UnixMilli())
	h.disableExpiredUserTunnels(now.UnixMilli())
}

func (h *Handler) resetMonthlyFlow(now time.Time) {
	db := h.repo.DB()
	currentDay := now.Day()
	lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()

	if currentDay == lastDay {
		_, _ = db.Exec(`
			UPDATE user
			SET in_flow = 0, out_flow = 0
			WHERE flow_reset_time != 0
			  AND (flow_reset_time = ? OR flow_reset_time > ?)
		`, currentDay, lastDay)
		_, _ = db.Exec(`
			UPDATE user_tunnel
			SET in_flow = 0, out_flow = 0
			WHERE flow_reset_time != 0
			  AND (flow_reset_time = ? OR flow_reset_time > ?)
		`, currentDay, lastDay)
		return
	}

	_, _ = db.Exec(`
		UPDATE user
		SET in_flow = 0, out_flow = 0
		WHERE flow_reset_time != 0
		  AND flow_reset_time = ?
	`, currentDay)
	_, _ = db.Exec(`
		UPDATE user_tunnel
		SET in_flow = 0, out_flow = 0
		WHERE flow_reset_time != 0
		  AND flow_reset_time = ?
	`, currentDay)
}

func (h *Handler) disableExpiredUsers(nowMs int64) {
	db := h.repo.DB()
	rows, err := db.Query(`
		SELECT id
		FROM user
		WHERE role_id != 0
		  AND status = 1
		  AND exp_time IS NOT NULL
		  AND exp_time < ?
	`, nowMs)
	if err != nil {
		return
	}
	userIDs := make([]int64, 0)

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			continue
		}
		userIDs = append(userIDs, userID)
	}
	_ = rows.Close()

	for _, userID := range userIDs {
		forwards, err := h.listActiveForwardsByUser(userID)
		if err == nil {
			h.pauseForwardRecords(forwards, nowMs)
		}
		_, _ = db.Exec(`UPDATE user SET status = 0 WHERE id = ?`, userID)
	}
}

func (h *Handler) disableExpiredUserTunnels(nowMs int64) {
	db := h.repo.DB()
	rows, err := db.Query(`
		SELECT id, user_id, tunnel_id
		FROM user_tunnel
		WHERE status = 1
		  AND exp_time IS NOT NULL
		  AND exp_time < ?
	`, nowMs)
	if err != nil {
		return
	}
	type expiredUserTunnel struct {
		userTunnelID int64
		userID       int64
		tunnelID     int64
	}
	items := make([]expiredUserTunnel, 0)

	for rows.Next() {
		var userTunnelID int64
		var userID int64
		var tunnelID int64
		if err := rows.Scan(&userTunnelID, &userID, &tunnelID); err != nil {
			continue
		}
		items = append(items, expiredUserTunnel{userTunnelID: userTunnelID, userID: userID, tunnelID: tunnelID})
	}
	_ = rows.Close()

	for _, item := range items {
		forwards, err := h.listActiveForwardsByUserTunnel(item.userID, item.tunnelID)
		if err == nil {
			h.pauseForwardRecords(forwards, nowMs)
		}
		_, _ = db.Exec(`UPDATE user_tunnel SET status = 0 WHERE id = ?`, item.userTunnelID)
	}
}
