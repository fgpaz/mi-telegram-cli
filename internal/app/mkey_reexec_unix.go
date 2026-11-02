//go:build unix

package app

import (
	"os"
	"syscall"
)

func Reexec(argv []string) error {
	return syscall.Exec(argv[0], argv, os.Environ())
}
