package middleware

import (
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

// faultErrorRate is read once at startup (env is set per-overlay/per-Rollout
// step, and a pod restart is how a new value takes effect anyway) rather
// than per-request -- keeps this a pure config value, not a live toggle.
var faultErrorRate = parseFaultErrorRate(os.Getenv("FAULT_ERROR_RATE"))

func parseFaultErrorRate(v string) float64 {
	if v == "" {
		return 0
	}
	rate, err := strconv.ParseFloat(v, 64)
	if err != nil || rate < 0 || rate > 1 {
		return 0
	}
	return rate
}

// Fault wraps a handler, injecting a synthetic 500 response at the rate set
// by FAULT_ERROR_RATE (0.0-1.0, default 0). Milestone 6: deliberately an
// HTTP-response-level failure on real traffic, not a pod-level crash --
// /health and /ready are never wrapped with this, so a plain rolling
// update's own health checks don't catch it and the bad version ships
// straight through. That gap is the whole reason canary analysis and
// blue-green manual promotion gates exist; it's what makes comparing the
// three deployment strategies against the same injected fault meaningful.
func Fault(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if faultErrorRate > 0 && rand.Float64() < faultErrorRate {
			http.Error(w, "internal server error (fault injected)", http.StatusInternalServerError)
			return
		}
		next(w, r)
	}
}
