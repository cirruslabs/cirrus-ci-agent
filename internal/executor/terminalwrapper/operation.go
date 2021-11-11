package terminalwrapper

type Operation interface {
	isOperation()
}

type LogOperation struct {
	Message string
}

func (*LogOperation) isOperation() {}

type ExpiringOperation struct {
	// empty for now
}

func (*ExpiringOperation) isOperation() {}

type ExitOperation struct {
	Success bool
}

func (*ExitOperation) isOperation() {}
