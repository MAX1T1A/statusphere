package renderer

type Renderer interface {
	Run() error
	UpdateDevices(devices []map[string]any)
}
