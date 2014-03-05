package breaker

import (
	"net/http"
	"time"
)

// Handler is an experimental implementation of a circuit-breaking
// http.Handler that returns 503 with an empty body after 5% failure rate over
// a sliding window of 5 seconds with a 1 second cooldown period before
// retrying with a single request. Failure is defined as any response code
// greater than or equal to 500.
func Handler(next http.Handler) http.Handler {
	return &handler{
		circuit: NewCircuit(0.05),
		next:    next,
	}
}

type handler struct {
	circuit Circuit
	next    http.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.circuit.Allow() {
		h.serveClosed(w, r)
	} else {
		h.serveOpened(w, r)
	}
}

func (h *handler) serveClosed(w http.ResponseWriter, r *http.Request) {
	cw := &codeWriter{w, 200}
	begin := time.Now()

	h.next.ServeHTTP(cw, r)

	duration := time.Since(begin)
	if cw.code < 500 {
		h.circuit.Success(duration)
	} else {
		h.circuit.Failure(duration)
	}
}

func (h *handler) serveOpened(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusServiceUnavailable)
}

type codeWriter struct {
	http.ResponseWriter
	code int
}

func (w *codeWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
