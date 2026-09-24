package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseFaultErrorRate(t *testing.T) {
	cases := map[string]float64{
		"":       0,
		"0":      0,
		"0.5":    0.5,
		"1":      1,
		"-0.1":   0, // out of range, falls back to 0
		"1.1":    0, // out of range, falls back to 0
		"banana": 0, // unparseable, falls back to 0
	}
	for in, want := range cases {
		if got := parseFaultErrorRate(in); got != want {
			t.Errorf("parseFaultErrorRate(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestFault_RateZero_AlwaysCallsThrough(t *testing.T) {
	faultErrorRate = 0
	t.Cleanup(func() { faultErrorRate = 0 })

	called := false
	handler := Fault(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if !called {
		t.Fatal("expected the wrapped handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestFault_RateOne_AlwaysInjectsFailure(t *testing.T) {
	faultErrorRate = 1
	t.Cleanup(func() { faultErrorRate = 0 })

	called := false
	handler := Fault(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if called {
		t.Fatal("expected the wrapped handler NOT to be called")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}
