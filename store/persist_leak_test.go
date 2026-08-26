package store

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// TestSaveFailureLeaksNoTempFiles verifies a failed save never leaves
// temporary snapshot files behind.
func TestSaveFailureLeaksNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data.json")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	st := NewStore(target)
	z := domain.NewCleanZone("z1", "Zone", "PA-A", domain.Iso5, domain.ProcessLithography)
	if _, err := st.CleanZones().Create(z); err == nil {
		t.Fatal("expected a save failure when the target is a directory")
	}
	leftovers, err := filepath.Glob(filepath.Join(dir, "data.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("failed save leaked temp files: %v", leftovers)
	}
}

// TestCorruptSnapshotQuarantinesOriginal verifies a corrupt snapshot file
// is moved out of the way (quarantined to .bak) so a restart does not keep
// re-reading the same corrupt file.
func TestCorruptSnapshotQuarantinesOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := NewStore(path)
	if err := st.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt original must be quarantined, still present: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("corrupt snapshot must be backed up to .bak: %v", err)
	}
}

// TestCorruptSnapshotLoadReturnsWarning verifies a corrupt snapshot is
// quarantined to a .bak file and reported through LoadWarning.
func TestCorruptSnapshotLoadReturnsWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st := NewStore(path)
	if err := st.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	if st.LoadWarning() == nil {
		t.Fatal("corrupt snapshot must be reported through LoadWarning")
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("corrupt snapshot must be backed up to .bak: %v", err)
	}
}
