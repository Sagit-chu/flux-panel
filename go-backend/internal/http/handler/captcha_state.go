package handler

import "time"

const captchaTokenTTL = 5 * time.Minute

func (h *Handler) storeCaptchaToken(token string) {
	if h == nil {
		return
	}
	token = normalizeCaptchaToken(token)
	if token == "" {
		return
	}

	h.captchaMu.Lock()
	defer h.captchaMu.Unlock()

	now := time.Now().UnixMilli()
	h.pruneExpiredCaptchaTokensLocked(now)
	h.captchaTokens[token] = now + int64(captchaTokenTTL/time.Millisecond)
}

func (h *Handler) consumeCaptchaToken(token string) bool {
	if h == nil {
		return false
	}
	token = normalizeCaptchaToken(token)
	if token == "" {
		return false
	}

	h.captchaMu.Lock()
	defer h.captchaMu.Unlock()

	now := time.Now().UnixMilli()
	h.pruneExpiredCaptchaTokensLocked(now)
	expiresAt, ok := h.captchaTokens[token]
	if !ok || expiresAt <= now {
		delete(h.captchaTokens, token)
		return false
	}
	delete(h.captchaTokens, token)
	return true
}

func (h *Handler) pruneExpiredCaptchaTokensLocked(now int64) {
	for token, expiresAt := range h.captchaTokens {
		if expiresAt <= now {
			delete(h.captchaTokens, token)
		}
	}
}

func normalizeCaptchaToken(token string) string {
	return trimToken(token)
}

func trimToken(token string) string {
	for len(token) > 0 && (token[0] == ' ' || token[0] == '\t' || token[0] == '\n' || token[0] == '\r') {
		token = token[1:]
	}
	for len(token) > 0 && (token[len(token)-1] == ' ' || token[len(token)-1] == '\t' || token[len(token)-1] == '\n' || token[len(token)-1] == '\r') {
		token = token[:len(token)-1]
	}
	return token
}
