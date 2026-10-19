//go:build !windows

package jobs

import (
	"errors"
	"os"
	"syscall"
)

func lockLease(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) {
		return ErrBusy
	}
	return err
}
