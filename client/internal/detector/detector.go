package detector

type Context struct {
	OSFamily string
	Distro   string
	DEWM     string
	Terminal string
}

func Detect() Context {
	return Context{
		OSFamily: detectOS(),
		Distro:   detectDistro(),
		DEWM:     detectDEWM(),
		Terminal: detectTerminal(),
	}
}
