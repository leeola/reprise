package reprise

import "errors"

var (
	// ErrNoRequest is returned when Reprise() was called but no
	// reprise.Request exists for the current Step to create an
	// http request from.
	ErrNoRequest = errors.New("no request found")

	// ErrNoResponse is returned when RepriseDiff() was called but
	// no reprise.Response exists to compare the http response to.
	ErrNoResponse = errors.New("no response found")
)

func IsErrNoRequest(err error) bool {
	return err == ErrNoRequest
}

func IsErrNoResponse(err error) bool {
	return err == ErrNoResponse
}
