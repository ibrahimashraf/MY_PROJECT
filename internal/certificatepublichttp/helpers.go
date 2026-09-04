package certificatepublichttp

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"
	"unicode"
)

func (h *Handler) allow(k string, n time.Time) bool {
	l := h.Limit
	if l <= 0 {
		l = 60
	}
	w := h.Window
	if w <= 0 {
		w = time.Minute
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rates == nil {
		h.rates = map[string]rateWindow{}
	}
	// Bounded cleanup: prevent slowloris/botnet memory creep
	if len(h.rates) > 2048 {
		for key, win := range h.rates {
			if n.Sub(win.started) > 2*w {
				delete(h.rates, key)
			}
		}
	}
	s := h.rates[k]
	if s.started.IsZero() || n.Sub(s.started) >= w {
		s = rateWindow{started: n}
	}
	if s.count >= l {
		h.rates[k] = s
		return false
	}
	s.count++
	h.rates[k] = s
	return true
}

func extractToken(p string) (string, bool) {
	a := strings.Split(strings.Trim(p, "/"), "/")
	if len(a) != 3 || a[0] != "verify" || a[1] != "certificates" || !validToken(a[2]) {
		return "", false
	}
	return a[2], true
}
func validToken(v string) bool {
	if len(v) < 32 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == 45 || r == 95) {
			return false
		}
	}
	return true
}
func (h *Handler) remoteKey(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil || remoteHost == "" {
		remoteHost = strings.TrimSpace(r.RemoteAddr)
	}

	if len(h.TrustedProxies) > 0 {
		isTrusted := false
		for _, trusted := range h.TrustedProxies {
			if remoteHost == strings.TrimSpace(trusted) {
				isTrusted = true
				break
			}
		}
		if isTrusted {
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				clientIP := strings.TrimSpace(parts[0])
				if clientIP != "" {
					return clientIP
				}
			}
			if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
				return xri
			}
		}
	}

	return remoteHost
}

func reply(w http.ResponseWriter, s int, p any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	if p != nil {
		_ = json.NewEncoder(w).Encode(p)
	}
}
