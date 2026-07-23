package renderer

import "statusphere-client/internal/presence"

type Renderer interface {
	Run() error
	UpdateDevices(devices []presence.Snapshot)
}
