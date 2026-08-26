package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestAsDomainErrorWalksDeepChain verifies a domain error buried under
// multiple wrapping layers is still recovered.
func TestAsDomainErrorWalksDeepChain(t *testing.T) {
	inner := fmt.Errorf("inner: %w", NotFound("interlock log", "log_1"))
	outer := fmt.Errorf("outer: %w", inner)
	de := AsDomainError(outer)
	if de == nil {
		t.Fatal("AsDomainError must find a domain error through multiple wraps")
	}
	if de.Code != CodeNotFound {
		t.Fatalf("want not_found, got %s", de.Code)
	}
}

// TestWrappedErrorTextKeepsCause verifies the rendered error text keeps the
// underlying cause so logs stay debuggable.
func TestWrappedErrorTextKeepsCause(t *testing.T) {
	root := errors.New("root cause detail")
	de := WrapError(CodeInternal, "interlock failed", root)
	if !strings.Contains(de.Error(), "root cause detail") {
		t.Fatalf("error text must include the underlying cause, got %q", de.Error())
	}
}
