package tui

import (
	"strings"

	"statusphere-client/internal/config"
)

const panelFile = "panel"

func loadPanelView() panelView {
	if data, err := config.Read(panelFile); err == nil {
		if strings.TrimSpace(string(data)) == "board" {
			return panelBoard
		}
	}
	return panelChat
}

func savePanelView(p panelView) {
	v := "chat"
	if p == panelBoard {
		v = "board"
	}
	_ = config.Write(panelFile, []byte(v), 0o600)
}
