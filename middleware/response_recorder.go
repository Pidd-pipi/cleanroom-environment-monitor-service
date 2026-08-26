package middleware

import "net/http"

// responseRecorder captures the response status and body size so the audit
// and request-log middlewares can report accurate metrics without buffering
// the whole body. It also forwards Flush for streaming handlers such as
// http.FileServer.
type responseRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

// WriteHeader records the first status code and forwards it.
func (r *responseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.status = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

// Write records the body size and lazily writes a 200 status if none was
// written yet (matching net/http semantics).
func (r *responseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Flush forwards to the underlying writer when it supports streaming.
func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
