//go:build windows

package app

import "errors"

func Reexec(argv []string) error {
	return errors.New("mkey run-scoped reexec is only implemented on unix")
}
