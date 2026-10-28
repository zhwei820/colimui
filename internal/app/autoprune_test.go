package app

import (
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fakePruneBackend struct {
	fakeBackend
	pruned []string
	err    error
}

func (b *fakePruneBackend) Prune(profile string) error {
	b.pruned = append(b.pruned, profile)
	return b.err
}

func TestParseAutoPrune(t *testing.T) {
	tests := []struct {
		env     string
		every   time.Duration
		wantErr bool
	}{
		{env: "", every: autoPruneDefault},
		{env: "off", every: 0},
		{env: "0", every: 0},
		{env: "90m", every: 90 * time.Minute},
		{env: "30s", wantErr: true},
		{env: "soon", wantErr: true},
	}
	for _, tt := range tests {
		every, err := parseAutoPrune(tt.env)
		if (err != nil) != tt.wantErr || (err == nil && every != tt.every) {
			t.Errorf("parseAutoPrune(%q) = %v, %v", tt.env, every, err)
		}
	}
	if autoPruneTick(0) != nil {
		t.Error("disabled auto-prune scheduled a tick")
	}
}

func TestAutoPrunePrunesRunningProfiles(t *testing.T) {
	backend := &fakePruneBackend{}
	m := newModel(backend, func() tea.Cmd { return nil })
	m.profiles = []profile{{Name: "default", Status: "Running"}, {Name: "idle", Status: "Stopped"}, {Name: "work", Status: "Running"}}

	cmd := m.autoPrune()
	if cmd == nil || !m.cleanupRunning {
		t.Fatal("auto-prune did not start")
	}
	if m.autoPrune() != nil {
		t.Fatal("auto-prune started twice concurrently")
	}
	updated, _ := m.Update(cmd())
	m = updated.(model)
	if len(backend.pruned) != 2 || backend.pruned[0] != "default" || backend.pruned[1] != "work" {
		t.Fatalf("pruned %v", backend.pruned)
	}
	if m.cleanupRunning || m.status != "auto-pruned default, work" {
		t.Fatalf("after prune: running=%t status=%q", m.cleanupRunning, m.status)
	}
}

func TestAutoPruneFailure(t *testing.T) {
	backend := &fakePruneBackend{err: errors.New("daemon failed")}
	m := newModel(backend, func() tea.Cmd { return nil })
	m.profiles = []profile{{Name: "default", Status: "Running"}}
	cmd := m.autoPrune()
	updated, _ := m.Update(cmd())
	m = updated.(model)
	if m.cleanupRunning || m.err == nil || m.status != "auto-prune failed" {
		t.Fatalf("failure: running=%t err=%v status=%q", m.cleanupRunning, m.err, m.status)
	}
}
