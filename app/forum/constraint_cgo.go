//go:build cgo
// +build cgo

package forum

import (
	"errors"
	"github.com/mattn/go-sqlite3"
)

func constraint(err error) bool {
	var value sqlite3.Error
	return errors.As(err, &value) && value.Code == sqlite3.ErrConstraint
}
