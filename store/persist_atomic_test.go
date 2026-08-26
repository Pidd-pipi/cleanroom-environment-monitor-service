package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCorruptFileBacksUpAndStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte("{definitely not valid json"), 0o644); err != nil {
		t.Fatalf("write corrupt file: %v", err)
	}

	s := NewStore(path)
	if err := s.Load(); err != nil {
		t.Fatalf("corrupt file must degrade instead of returning an error: %v", err)
	}
	if s.LoadWarning() == nil {
		t.Fatal("expected a load warning after degraded start")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt file should have been moved away, stat returned %v", err)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("backup file must exist: %v", err)
	}
	zones, err := s.CleanZones().List()
	if err != nil {
		t.Fatalf("list after degraded load: %v", err)
	}
	if len(zones) != 0 {
		t.Fatalf("degraded load must start empty, got %d zones", len(zones))
	}
}

func TestSaveIsAtomicAndReloadable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	s := NewStore(path)
	if err := s.Save(); err != nil {
		t.Fatalf("save empty state: %v", err)
	}

	s2 := NewStore(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	if s2.LoadWarning() != nil {
		t.Fatalf("unexpected load warning: %v", s2.LoadWarning())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected only the persisted file, got %d entries", len(entries))
	}
}
