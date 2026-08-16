package domain

import "errors"

var (
	// ErrDrift indicates target paths contain unowned or modified files.
	ErrDrift = errors.New("Karl target paths contain unowned client files")

	// ErrOutOfSync indicates that the client projection does not match Karl configuration.
	ErrOutOfSync = errors.New("client projection is out of sync")

	// ErrNotFound indicates that the projection is not installed.
	ErrNotFound = errors.New("client projection is not installed")

	// ErrClientNotFound indicates that the client executable is not available on PATH.
	ErrClientNotFound = errors.New("client executable was not found")

	// ErrCancelled indicates that an interactive workflow was cancelled.
	ErrCancelled = errors.New("operation cancelled")

	// ErrInputExhausted indicates an interactive input stream ended prematurely.
	ErrInputExhausted = errors.New("input stream exhausted")
)
