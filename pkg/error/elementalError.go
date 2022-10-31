package error

// ElementalError is our enhanced error that brings an error string + exit code
type ElementalError struct {
	s    string
	code int
}

func (e ElementalError) ExitCode() int {
	return e.code
}

func (e ElementalError) Error() string {
	return e.s
}

func New(err error, code int) *ElementalError {
	return &ElementalError{
		s:    err.Error(),
		code: code,
	}
}
