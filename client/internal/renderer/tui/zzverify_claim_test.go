package tui

import (
	"testing"

	"statusphere-client/internal/selfupdate"
)

// Reviewer's exact 6-step repro, run against the real working-tree code.
func TestVerifyStaleCheckClobbersInstalling(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()

	// 1+2) Init's check #1 is in flight; user opens Settings and presses enter -> check #2
	m.mode = modeSettings
	m.menuIndex = actionIndex(m.settingsMenu(), "update")
	next, cmd := m.runMenu()
	m = next.(model)
	if m.mode != modeUpdate || m.updateStage != updateChecking || cmd == nil {
		t.Fatalf("step2: mode=%v stage=%v cmd=%v", m.mode, m.updateStage, cmd != nil)
	}

	// 3) check #2 returns v0.4.0
	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}})
	if m.updateStage != updateAvailable {
		t.Fatalf("step3: stage=%v", m.updateStage)
	}

	// 4) enter -> installing, 3-minute download starts
	next, cmd = m.updateModalKeys(key("enter"))
	m = next.(model)
	if m.updateStage != updateInstalling || cmd == nil {
		t.Fatalf("step4: stage=%v cmd=%v", m.updateStage, cmd != nil)
	}

	// 5) check #1's stale msg finally lands
	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}})
	t.Logf("step5: stage after stale check = %d (installing=%d available=%d)",
		m.updateStage, updateInstalling, updateAvailable)
	if m.updateStage != updateInstalling {
		t.Fatalf("CLAIM CONFIRMED: stale check moved installing -> %d", m.updateStage)
	}

	// 6) enter -> second concurrent applyUpdateCmd?
	_, cmd2 := m.updateModalKeys(key("enter"))
	t.Logf("step6: secondApplyCmd=%v", cmd2 != nil)
	if cmd2 != nil {
		t.Fatal("CLAIM CONFIRMED: a second concurrent apply was issued")
	}

	// and via the settings row too
	m.mode = modeSettings
	m.menuIndex = actionIndex(m.settingsMenu(), "update")
	_, cmd3 := m.runMenu()
	if cmd3 != nil {
		t.Fatal("CLAIM CONFIRMED: settings row issued a second apply")
	}
}

// Second half: stale check after updateDone resurrects the badge / re-offers install.
func TestVerifyStaleCheckAfterDone(t *testing.T) {
	stubVersion(t, "v0.3.0")
	m := chatModel()
	m.mode = modeUpdate
	m.updateStage = updateInstalling
	m.updateRel = &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}

	m = send(m, updateAppliedMsg{})
	if m.updateStage != updateDone {
		t.Fatalf("want done, got %d", m.updateStage)
	}

	// process still reports v0.3.0, so IsNewer("v0.4.0","v0.3.0") is true
	if !selfupdate.IsNewer("v0.4.0", "v0.3.0") {
		t.Fatal("precondition: IsNewer should be true here")
	}

	m = send(m, updateCheckedMsg{rel: &selfupdate.Release{Version: "v0.4.0", AssetURL: "http://x"}})
	t.Logf("after done+stale check: stage=%d badge=%q row=%q",
		m.updateStage, m.updateBadge(), m.settingsMenu()[0].label)
	if m.updateStage != updateDone {
		t.Fatalf("CLAIM CONFIRMED: stage flipped to %d after done", m.updateStage)
	}
	if m.updateBadge() != "" {
		t.Fatalf("CLAIM CONFIRMED: badge resurrected: %q", m.updateBadge())
	}

	// enter on done closes the modal, does not re-download
	next, cmd := m.updateModalKeys(key("enter"))
	if cmd != nil {
		t.Fatal("CLAIM CONFIRMED: re-download issued after done")
	}
	if next.(model).mode != modeNone {
		t.Fatalf("enter on done should close the modal, mode=%v", next.(model).mode)
	}
}
