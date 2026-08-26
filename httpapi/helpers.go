package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

const (
	defaultListLimit = 100
	maxListLimit     = 1000
)

// parseTime parses an RFC3339 timestamp; empty means "now".
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, domain.InvalidInput("invalid timestamp, expected RFC3339: " + s)
	}
	return t.UTC(), nil
}

// parseIntParam parses an optional integer query parameter with a default.
func parseIntParam(r *http.Request, name string, def int) (int, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, domain.InvalidInput("invalid integer for " + name)
	}
	return n, nil
}

// parsePagination parses `limit` and `offset` query parameters. The limit
// defaults to defaultListLimit, is capped at maxListLimit, and a value of 0
// is treated as "no limit" for backward compatibility. Negative values are
// rejected as invalid input.
func parsePagination(r *http.Request) (limit, offset int, err error) {
	limit, err = parseIntParam(r, "limit", defaultListLimit)
	if err != nil {
		return 0, 0, err
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	offset, err = parseIntParam(r, "offset", 0)
	if err != nil {
		return 0, 0, err
	}
	return limit, offset, nil
}

// paginate slices items according to limit/offset and returns the page plus
// the total count. limit == 0 means no upper bound (return all remaining).
func paginate(itemsLen, limit, offset int) (start, end int) {
	if offset > itemsLen {
		offset = itemsLen
	}
	start = offset
	end = itemsLen
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	return start, end
}

// validateFinite checks that the supplied float fields are real numbers and
// rejects NaN/Inf, which cannot represent valid sensor readings.
func validateFinite(name string, v float64) error {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return domain.InvalidInput(name + " must be a finite number")
	}
	return nil
}
