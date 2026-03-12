package renderer

type Renderer interface {
	Run() error
	Update(data map[string]any)
}
