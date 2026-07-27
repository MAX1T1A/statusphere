package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"statusphere-client/internal/selfupdate"
	"statusphere-client/internal/version"
)

func stubVersion(t *testing.T, v string) {
	t.Helper()
	old := version.Version
	version.Version = v
	t.Cleanup(func() { version.Version = old })
}

func TestTitleBarShowsVersionInRightCorner(t *testing.T) {
	stubVersion(t, "v0.3.0")

	bar := titleBar(60, "")
	if !strings.Contains(bar, "statusphere") || !strings.Contains(bar, "v0.3.0") {
		t.Fatalf("title bar missing parts: %q", bar)
	}
	if w := ansi.StringWidth(bar); w != 60 {
		t.Fatalf("title bar width = %d, want 60", w)
	}
	if !strings.HasSuffix(ansi.Strip(bar), "v0.3.0") {
		t.Fatalf("version should sit in the right corner: %q", ansi.Strip(bar))
	}
}

func TestTitleBarAnnouncesUpdate(t *testing.T) {
	stubVersion(t, "v0.3.0")

	bar := ansi.Strip(titleBar(70, "v0.4.0"))
	if !strings.Contains(bar, "v0.4.0 available") || !strings.Contains(bar, "v0.3.0") {
		t.Fatalf("expected both the new and current version: %q", bar)
	}
}

func TestTitleBarDegradesOnNarrowTerminal(t *testing.T) {
	stubVersion(t, "v0.3.0")
	bar := titleBar(8, "")
	if strings.Contains(ansi.Strip(bar), "v0.3.0") {
		t.Fatal("version should be dropped when there is no room for it")
	}
}

func TestSettingsUpdateEntryReflectsState(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()

	if got := m.settingsRows()[0].label; got != "Check for updates" {
		t.Fatalf("idle label = %q", got)
	}

	m.updateStage = updateAvailable
	m.updateRel = &selfupdate.Release{Version: "v0.4.0"}
	if got := m.settingsRows()[0].label; got != "Update to v0.4.0" {
		t.Fatalf("available label = %q", got)
	}

	m.updateStage = updateCurrent
	if got := m.settingsRows()[0].desc; !strings.Contains(got, "up to date") {
		t.Fatalf("up-to-date desc = %q", got)
	}

	m.updateStage = updateDone
	if got := m.settingsRows()[0].label; got != "Update installed" {
		t.Fatalf("done label = %q", got)
	}
}

func TestUpdateFlowStages(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()

	m.mode = modeSettings
	m.menuIndex = 0
	next, cmd := m.runMenu()
	m = next.(model)
	if m.mode != modeUpdate || m.updateStage != updateChecking || cmd == nil {
		t.Fatalf("expected a check to start, mode=%v stage=%v cmd=%v", m.mode, m.updateStage, cmd != nil)
	}
	if !strings.Contains(ansi.Strip(m.View()), "checking for updates") {
		t.Fatal("modal should show the checking state")
	}

	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}})
	if m.updateStage != updateAvailable {
		t.Fatalf("stage = %v, want available", m.updateStage)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "v0.4.0 is available") || !strings.Contains(view, "enter") {
		t.Fatalf("modal should offer the install:\n%s", view)
	}
	if m.updateBadge() != "v0.4.0" {
		t.Fatalf("badge = %q", m.updateBadge())
	}

	next, cmd = m.updateModalKeys(key("enter"))
	m = next.(model)
	if m.updateStage != updateInstalling || cmd == nil {
		t.Fatalf("enter should start installing, stage=%v", m.updateStage)
	}

	m = send(m, updateAppliedMsg{})
	if m.updateStage != updateDone {
		t.Fatalf("stage = %v, want done", m.updateStage)
	}
	if !strings.Contains(ansi.Strip(m.View()), "restart") {
		t.Fatal("done state should tell the user to restart")
	}
}

func TestUpToDateAndFailurePaths(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()
	m.mode = modeUpdate

	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.3.0"}})
	if m.updateStage != updateCurrent || m.updateBadge() != "" {
		t.Fatalf("same version should read as up to date, stage=%v badge=%q", m.updateStage, m.updateBadge())
	}
	if !strings.Contains(ansi.Strip(m.View()), "up to date") {
		t.Fatal("modal should say up to date")
	}

	m = send(m, updateCheckedMsg{err: errBoom{}})
	if m.updateStage != updateFailed {
		t.Fatalf("stage = %v, want failed", m.updateStage)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "update failed") || !strings.Contains(view, "boom") {
		t.Fatalf("failure should surface the reason:\n%s", view)
	}

	next, cmd := m.updateModalKeys(key("enter"))
	if next.(model).updateStage != updateChecking || cmd == nil {
		t.Fatal("enter should retry after a failure")
	}
}

func TestUpdateModalEscapes(t *testing.T) {
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateAvailable
	m.updateRel = &selfupdate.Release{Version: "v0.4.0"}

	next, _ := m.updateModalKeys(key("esc"))
	if next.(model).mode != modeNone {
		t.Fatal("esc should close the update modal")
	}
	if next.(model).updateStage != updateAvailable {
		t.Fatal("closing the modal must not discard the pending update")
	}
}

func TestNoSecondInstallWhileOneIsRunning(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateAvailable
	m.updateRel = &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}

	next, cmd := m.updateModalKeys(key("enter"))
	m = next.(model)
	if m.updateStage != updateInstalling || cmd == nil {
		t.Fatal("first install should start")
	}

	next, _ = m.updateModalKeys(key("esc"))
	m = next.(model)
	m.mode = modeSettings
	m.menuIndex = actionIndex(m.settingsRows(), "update")
	next, cmd = m.runMenu()
	m = next.(model)
	if cmd != nil {
		t.Fatal("no new command may be issued while a download is in flight")
	}
	if m.updateStage != updateInstalling {
		t.Fatalf("stage should stay installing, got %v", m.updateStage)
	}
	if got := m.settingsRows()[0].label; got != "Downloading update…" {
		t.Fatalf("settings should show the in-flight state, got %q", got)
	}
}

func TestStaleCheckDoesNotUndoInstall(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateInstalling
	m.updateRel = &selfupdate.Release{Version: "v0.4.0"}

	m = send(m, updateAppliedMsg{})
	if m.updateStage != updateDone {
		t.Fatal("install should complete")
	}

	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}})
	if m.updateStage != updateDone {
		t.Fatalf("stale check overwrote the installed state: %v", m.updateStage)
	}
	if m.updateBadge() != "" {
		t.Fatalf("badge should be gone after installing, got %q", m.updateBadge())
	}
}

func TestInstallWithoutReleaseFailsInsteadOfPanicking(t *testing.T) {
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateAvailable
	m.updateRel = nil

	next, cmd := m.updateModalKeys(key("enter"))
	m = next.(model)
	if cmd != nil || m.updateStage != updateFailed {
		t.Fatalf("a nil release must fail cleanly, stage=%v cmd=%v", m.updateStage, cmd != nil)
	}
	_ = m.View()
}

func TestInstallingRendersWithoutReleaseInfo(t *testing.T) {
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateInstalling
	m.updateRel = nil
	if !strings.Contains(ansi.Strip(m.View()), "downloading") {
		t.Fatal("installing view should render even without release details")
	}
}

func TestNilReleaseFromCheckIsAFailure(t *testing.T) {
	m := chatModel()
	m.mode = modeUpdate
	m = send(m, updateCheckedMsg{})
	if m.updateStage != updateFailed {
		t.Fatalf("a nil release must not be treated as up to date, stage=%v", m.updateStage)
	}
}

type errBoom struct{}

func (errBoom) Error() string { return "boom" }
