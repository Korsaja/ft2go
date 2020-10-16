package main

import "errors"

var (
	ErrFtioInvalid = errors.New("Error in initialization ftio structure.")
	ErrInvalidIP = errors.New("Error  bad IP address.")
	ErrInvalidExAddr = errors.New("Error  Invalid exAddress or not entered.")
	ErrNoFilters = errors.New("Error not entered filters.")
)



type timeout interface {
	Timeout() bool
}

type Error struct {
	Err error
	filename string
}

func(e *Error) Unwrap() error { return e.Err }

func(e *Error)Error()string {
	return e.Err.Error() + " " + e.filename
}

func (e *Error) Timeout() bool {
	t, ok := e.Err.(timeout)
	return ok && t.Timeout()
}