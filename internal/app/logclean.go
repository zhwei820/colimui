package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const logCleanDefault = 24 * time.Hour
const logCleanEnv = "COLIMUI_LOG_CLEAN"

// parseLogClean reads COLIMUI_LOG_CLEAN with the same rules as
// COLIMUI_AUTO_PRUNE: empty keeps the daily default, "off"/"0"/"false"
// disables it, otherwise a duration of at least a minute.
func parseLogClean(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "":
		return logCleanDefault, nil
	case "off", "0", "false":
		return 0, nil
	}
	every, err := time.ParseDuration(value)
	if err != nil || every < time.Minute {
		return 0, fmt.Errorf("invalid %s %q: use a duration of 1m or more (e.g. 24h) or \"off\"", logCleanEnv, value)
	}
	return every, nil
}

type logCleanTickMsg struct{}

const logCleanBatch = 100

// logCleanMsg carries a sweep between batches: pending is what is still to be
// deleted, so the TUI gets control back every logCleanBatch files.
type logCleanMsg struct {
	pending []string
	removed int
	failed  int
}

func logCleanTick(every time.Duration) tea.Cmd {
	if every <= 0 {
		return nil
	}
	return tea.Tick(every, func(time.Time) tea.Msg { return logCleanTickMsg{} })
}

// findLogs lists every regular *.log file under root, hidden directories
// included. Unreadable directories (macOS privacy-protected ones under
// ~/Library) are skipped rather than failing the whole sweep; symlinks are not
// followed.
func findLogs(root string) []string {
	var paths []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() && filepath.Ext(path) == ".log" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths
}

func (m *model) cleanLogs() tea.Cmd {
	if m.logCleanRunning || m.logCleanRoot == "" {
		return nil
	}
	m.logCleanRunning = true
	root := m.logCleanRoot
	return func() tea.Msg { return logCleanMsg{pending: findLogs(root)} }
}

func removeLogBatch(msg logCleanMsg) tea.Cmd {
	return func() tea.Msg {
		n := min(logCleanBatch, len(msg.pending))
		for _, path := range msg.pending[:n] {
			if os.Remove(path) != nil {
				msg.failed++
			} else {
				msg.removed++
			}
		}
		msg.pending = msg.pending[n:]
		return msg
	}
}
