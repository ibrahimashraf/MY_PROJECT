package certificatepublichttp

import (
	"encoding/json"
	"integin/internal/certificatepg"
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

func replyHTML(w http.ResponseWriter, s int, view certificatepg.PublicProjection) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(s)
	html := `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Certificate ` + htmlEscape(view.CertificateNumber) + ` | INTEGIN Verified</title>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;background:#f8fafc;color:#0f172a;margin:0;padding:24px 16px;display:flex;justify-content:center}
.card{background:#ffffff;border:1px solid #e2e8f0;border-radius:12px;box-shadow:0 4px 6px -1px rgba(0,0,0,0.1);max-width:540px;width:100%;padding:24px;box-sizing:border-box}
.badge{display:inline-block;padding:4px 12px;border-radius:9999px;font-size:12px;font-weight:700;letter-spacing:0.05em;background:#ecfdf5;color:#047857}
h1{font-size:20px;margin:16px 0 4px}
.cert-no{color:#64748b;font-size:14px;margin-bottom:20px}
.grid{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin:16px 0;font-size:14px}
.label{color:#64748b;font-size:12px;text-transform:uppercase}
.val{font-weight:600;margin-top:2px}
.footer{margin-top:24px;padding-top:16px;border-top:1px solid #f1f5f9;font-size:12px;color:#94a3b8;text-align:center}
</style>
</head>
<body>
<div class="card">
<span class="badge">INTEGIN VERIFIED &#10003;</span>
<h1>Equipment Certificate</h1>
<div class="cert-no">` + htmlEscape(view.CertificateNumber) + `</div>
<div class="grid">
<div><div class="label">Status</div><div class="val">` + htmlEscape(view.Status) + `</div></div>
<div><div class="label">Asset ID</div><div class="val">` + htmlEscape(view.AssetID) + `</div></div>
<div><div class="label">Issued Date</div><div class="val">` + htmlEscape(view.IssuedAt.Format("2006-01-02")) + `</div></div>
<div><div class="label">Expiry Date</div><div class="val">` + htmlEscape(view.ExpiresAt.Format("2006-01-02")) + `</div></div>
</div>
<div class="footer">Sovereign Global Trust Platform &bull; Server Authoritative Verification</div>
</div>
</body>
</html>`
	_, _ = w.Write([]byte(html))
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return strings.ReplaceAll(s, "'", "&#39;")
}
