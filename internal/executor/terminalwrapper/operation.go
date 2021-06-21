package terminalwrapper

type Operation interface {
	isEntry()
}

type LogOperation struct {
	Message string
}

func (*LogOperation) isEntry() {}

type ExitOperation struct {
	Success bool
}

func (*ExitOperation) isEntry() {}
