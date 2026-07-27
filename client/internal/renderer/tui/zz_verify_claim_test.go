package tui

import (
	"testing"

	"statusphere-client/internal/selfupdate"
)

// Replays the reviewer's exact 5-step repro against the real code.
func TestVerifyClaimSettingsRechecksDuringInstall(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()

	// 1) an install is running, user presses esc
	m.mode = modeUpdate
	m.updateStage = updateInstalling
	m.updateRel = &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}
	next, _ := m.updateModalKeys(key("esc"))
	m = next.(model)
	t.Logf("step1 mode=%v stage=%v", m.mode, m.updateStage)

	// 2) open Settings, inspect the row
	m.mode = modeSettings
	m.menuIndex = actionIndex(m.settingsMenu(), "update")
	row := m.settingsMenu()[0]
	t.Logf("step2 label=%q desc=%q", row.label, row.desc)
	if row.label == "Check for updates" {
		t.Errorf("CLAIM HOLDS: row hides the in-flight install: %q", row.label)
	}

	// 3) enter -- does it reset to checking and fire a command?
	next, cmd := m.runMenu()
	m = next.(model)
	t.Logf("step3 stage=%v cmd_issued=%v", m.updateStage, cmd != nil)
	if cmd != nil {
		t.Errorf("CLAIM HOLDS: redundant check issued while installing")
	}
	if m.updateStage != updateInstalling {
		t.Errorf("CLAIM HOLDS: installing state destroyed, stage=%v", m.updateStage)
	}

	// 4) install completes
	m = send(m, updateAppliedMsg{})
	t.Logf("step4 stage=%v label=%q", m.updateStage, m.settingsMenu()[0].label)

	// 5) the (hypothetical) check returns -- must not resurrect "available"
	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}})
	t.Logf("step5 stage=%v badge=%q label=%q", m.updateStage, m.updateBadge(), m.settingsMenu()[0].label)
	if m.updateStage == updateAvailable {
		t.Errorf("CLAIM HOLDS: stale check re-offered an already-installed release")
	}
}

// Second half of the claim: enter while a check is in flight fires a duplicate.
func TestVerifyClaimEscDuringChecking(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateChecking

	next, _ := m.updateModalKeys(key("esc"))
	m = next.(model)

	m.mode = modeSettings
	m.menuIndex = actionIndex(m.settingsMenu(), "update")
	t.Logf("row while checking: label=%q", m.settingsMenu()[0].label)

	_, cmd := m.runMenu()
	t.Logf("anotherCheck=%v", cmd != nil)
	if cmd != nil {
		t.Errorf("CLAIM HOLDS: duplicate check fired while one is in flight")
	}
}
