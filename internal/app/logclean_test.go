package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestParseLogClean(t *testing.T) {
	tests := []struct {
		env     string
		every   time.Duration
		wantErr bool
	}{
		{env: "", every: logCleanDefault},
		{env: "off", every: 0},
		{env: "12h", every: 12 * time.Hour},
		{env: "30s", wantErr: true},
		{env: "daily", wantErr: true},
	}
	for _, tt := range tests {
		every, err := parseLogClean(tt.env)
		if (err != nil) != tt.wantErr || (err == nil && every != tt.every) {
			t.Errorf("parseLogClean(%q) = %v, %v", tt.env, every, err)
		}
	}
	if logCleanTick(0) != nil {
		t.Error("disabled log cleanup scheduled a tick")
	}
}

func TestCleanLogsRemovesLogFilesRecursively(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) string {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	logs := []string{write("a.log"), write("deep/nested/b.log"), write(".colima/_lima/c.log")}
	kept := []string{write("notes.txt"), write("deep/b.log.gz"), write("deep/catalog")}
	outside := write("../outside.log")
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}

	m := newModel(&fakeBackend{}, func() tea.Cmd { return nil })
	m.logCleanRoot = root
	cmd := m.cleanLogs()
	if cmd == nil || m.cleanLogs() != nil {
		t.Fatal("log cleanup did not start exactly once")
	}
	for cmd != nil {
		updated, next := m.Update(cmd())
		m = updated.(model)
		cmd = next
	}

	for _, p := range logs {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists", p)
		}
	}
	for _, p := range append(kept, outside) {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s was removed: %v", p, err)
		}
	}
	if m.logCleanRunning || m.status != "removed 3 log files" {
		t.Fatalf("after cleanup: running=%t status=%q", m.logCleanRunning, m.status)
	}
}

func TestCleanLogsDeletesInBatches(t *testing.T) {
	root := t.TempDir()
	for i := range 250 {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f%d.log", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel(&fakeBackend{}, func() tea.Cmd { return nil })
	m.logCleanRoot = root
	cmd := m.cleanLogs()
	var statuses []string
	for cmd != nil {
		updated, next := m.Update(cmd())
		m = updated.(model)
		statuses = append(statuses, m.status)
		cmd = next
	}
	want := []string{
		"removing log files: 0 done, 250 left",
		"removing log files: 100 done, 150 left",
		"removing log files: 200 done, 50 left",
		"removed 250 log files",
	}
	if !slices.Equal(statuses, want) || m.logCleanRunning {
		t.Fatalf("statuses %q, running=%t", statuses, m.logCleanRunning)
	}
}
