// Package httpapi implements the REST API layer. Handlers translate HTTP
// requests into service calls and render the unified response envelope.
package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/middleware"
)

// Response is the unified response envelope used by every API endpoint.
//
//	{"code":0,"message":"ok","data":...}
//	{"code":404,"message":"...","error":"not_found","request_id":"..."}
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	// Total carries the total number of matching records for paginated
	// list endpoints. The data field itself remains an array so existing
	// clients and tests stay compatible.
	Total int `json:"total,omitempty"`
}

// OK writes a successful unified response.
func OK(w http.ResponseWriter, r *http.Request, data interface{}) {
	writeJSON(w, r, http.StatusOK, Response{Code: 0, Message: "ok", Data: data})
}

// OKStatus writes a successful response with an explicit HTTP status.
func OKStatus(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	writeJSON(w, r, status, Response{Code: 0, Message: "ok", Data: data})
}

// OKPaged writes a successful list response, keeping `data` as a plain array
// for backward compatibility while exposing pagination metadata through both
// the top-level `total` field and the X-Total-Count/X-Limit/X-Offset headers.
func OKPaged(w http.ResponseWriter, r *http.Request, data interface{}, total, limit, offset int) {
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Offset", strconv.Itoa(offset))
	writeJSON(w, r, http.StatusOK, Response{Code: 0, Message: "ok", Data: data, Total: total})
}

// Fail writes an error response derived from a domain error.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	de := domain.AsDomainError(err)
	status := http.StatusInternalServerError
	code := "internal"
	message := "internal server error"
	if de != nil {
		code = string(de.Code)
		message = de.Message
		switch de.Code {
		case domain.CodeNotFound:
			status = http.StatusNotFound
		case domain.CodeInvalidInput:
			status = http.StatusBadRequest
		case domain.CodeConflict:
			status = http.StatusConflict
		case domain.CodeUnauthorized:
			status = http.StatusUnauthorized
		}
	}
	if status == http.StatusInternalServerError {
		slog.Error("httpapi: internal error", "error", err)
	}
	writeJSON(w, r, status, Response{
		Code:      status,
		Message:   message,
		Error:     code,
		RequestID: requestIDFrom(r),
	})
}

// FailPlain writes an error response with an explicit HTTP status and
// message (used for routing errors and malformed JSON).
func FailPlain(w http.ResponseWriter, r *http.Request, status int, message string) {
	writeJSON(w, r, status, Response{
		Code:      status,
		Message:   message,
		Error:     "request_error",
		RequestID: requestIDFrom(r),
	})
}

// requestIDFrom extracts the trace id set by the request-id middleware.
func requestIDFrom(r *http.Request) string {
	return middleware.RequestID(r.Context())
}

// writeJSON renders the envelope and enforces a trailing newline.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		slog.Error("httpapi: write response", "error", err)
	}
}

// decodeJSON parses a JSON request body, returning a domain invalid-input
// error for malformed payloads, unknown fields or trailing data.
func decodeJSON(r *http.Request, dst interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.InvalidInput("malformed JSON body: " + err.Error())
	}
	// Reject requests carrying more than one JSON value.
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return domain.InvalidInput("malformed JSON body: trailing data after JSON value")
	}
	return nil
}
