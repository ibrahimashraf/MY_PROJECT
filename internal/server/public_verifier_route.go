package server

import (
	"net/http"

	"integin/pkg/httputil"
	"integin/pkg/verification"
)

var publicVerifierHTML = []byte(verification.PublicVerifierHTML)

func publicVerifierHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/verify" && r.URL.Path != "/verify/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			httputil.WriteProblem(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "Allowed methods: GET, HEAD")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(publicVerifierHTML)
		}
	})
}
