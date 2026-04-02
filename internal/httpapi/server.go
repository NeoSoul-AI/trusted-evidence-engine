package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/evoevo/trusted-evidence-engine/internal/engine"
	"github.com/evoevo/trusted-evidence-engine/internal/metrics"
	"github.com/evoevo/trusted-evidence-engine/internal/schema"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewHandler(e *engine.Engine) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/health", observeHTTP("/health", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})))
	mux.Handle("/v1/evidence/resolve", observeHTTP("/v1/evidence/resolve", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var input schema.ResolveInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		pack, err := e.Resolve(r.Context(), input)
		if err != nil {
			status := http.StatusInternalServerError
			if errors.Is(err, engine.ErrEmptyInput) {
				status = http.StatusBadRequest
			}
			metrics.ObserveResolve("error", time.Since(startedAt), 0)
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		metrics.ObserveResolve("success", time.Since(startedAt), len(pack.Items))
		writeJSON(w, http.StatusOK, pack)
	})))
	return mux
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func observeHTTP(path string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		metrics.ObserveHTTP(path, r.Method, recorder.status, time.Since(startedAt))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
