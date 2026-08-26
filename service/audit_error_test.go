package service

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// TestAuditLogReturnsErrorOnStoreFailure verifies audit failures are
// propagated to callers instead of being silently dropped.
func TestAuditLogReturnsErrorOnStoreFailure(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := store.NewStore(filepath.Join(blocker, "data.json"))
	svc := NewAuditService(st)
	err := svc.Log(domain.AuditSampleIngest, "eng", "sample", "s1", "detail", "rid")
	if err == nil {
		t.Fatal("audit log must surface persistence errors")
	}
}
