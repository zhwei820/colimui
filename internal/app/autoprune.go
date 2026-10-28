package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const autoPruneDefault = 4 * time.Hour
const autoPruneEnv = "COLIMUI_AUTO_PRUNE"

type pruneBackend interface {
	Prune(string) error
}

// Prune is the unattended counterpart of Cleanup: without --all it keeps
// tagged images and named volumes, removing only dangling images, stopped
// containers, unused networks, build cache and anonymous volumes.
func (execBackend) Prune(profile string) error {
	for _, args := range [][]string{
		{"volume", "prune", "--force"},
		{"system", "prune", "--force"},
	} {
		if _, err := commandOutput(profile, 2*time.Minute, "docker", args...); err != nil {
			return fmt.Errorf("docker %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

// parseAutoPrune reads COLIMUI_AUTO_PRUNE: empty keeps the default interval,
// "off"/"0"/"false" disables it (returns 0), otherwise a duration of at least
// a minute.
func parseAutoPrune(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "":
		return autoPruneDefault, nil
	case "off", "0", "false":
		return 0, nil
	}
	every, err := time.ParseDuration(value)
	if err != nil || every < time.Minute {
		return 0, fmt.Errorf("invalid %s %q: use a duration of 1m or more (e.g. 4h) or \"off\"", autoPruneEnv, value)
	}
	return every, nil
}

type autoPruneTickMsg struct{}

type autoPruneMsg struct {
	profiles []string
	err      error
}

func autoPruneTick(every time.Duration) tea.Cmd {
	if every <= 0 {
		return nil
	}
	return tea.Tick(every, func(time.Time) tea.Msg { return autoPruneTickMsg{} })
}

// autoPrune prunes every running profile; it shares cleanupRunning with the
// manual cleanup so the two never prune the same daemon concurrently. A
// skipped round simply waits for the next tick.
func (m *model) autoPrune() tea.Cmd {
	if m.cleanupRunning {
		return nil
	}
	backend, ok := m.currentBackend().(pruneBackend)
	if !ok {
		return nil
	}
	var names []string
	for _, p := range m.profiles {
		if isRunning(p.Status) {
			names = append(names, p.Name)
		}
	}
	if len(names) == 0 {
		return nil
	}
	m.cleanupRunning = true
	return func() tea.Msg {
		var errs []error
		for _, name := range names {
			if err := backend.Prune(name); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
		return autoPruneMsg{profiles: names, err: errors.Join(errs...)}
	}
}
