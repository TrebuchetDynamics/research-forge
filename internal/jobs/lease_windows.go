package jobs

import (
	"os"
	"syscall"
	"unsafe"
)

var lockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

func lockLease(f *os.File) error {
	var overlapped syscall.Overlapped
	// LOCKFILE_FAIL_IMMEDIATELY | LOCKFILE_EXCLUSIVE_LOCK; closing the
	// handle releases the lock, including when the host process is killed.
	ok, _, err := lockFileEx.Call(f.Fd(), 3, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if ok != 0 {
		return nil
	}
	if err == syscall.Errno(33) {
		return ErrBusy
	} // ERROR_LOCK_VIOLATION
	return err
}
