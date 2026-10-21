//go:build !cgo
// +build !cgo

package forum

// The driver stub fails at Open before any database operation can occur.
func constraint(err error) bool { return false }
