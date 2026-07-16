package filetxn

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
)

var appendMu sync.Mutex

// Append writes data to a regular file without following symlinks or writing
// through a hard link. Single-link files retain O_APPEND behavior; hard-linked
// files are copied and atomically replaced at path.
func Append(path string, data []byte, mode os.FileMode) error {
	appendMu.Lock()
	defer appendMu.Unlock()
	if err := validateParent(path); err != nil {
		return err
	}
	for {
		info, err := os.Lstat(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if os.IsExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			return writeAndClose(file, data)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("append target is not a regular file: %s", path)
		}
		if links, known := linkCount(info); known && links == 1 {
			file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
			if err != nil {
				return err
			}
			openedInfo, err := file.Stat()
			if err != nil {
				_ = file.Close()
				return err
			}
			openedLinks, openedLinksKnown := linkCount(openedInfo)
			if !os.SameFile(info, openedInfo) {
				_ = file.Close()
				return fmt.Errorf("append target changed while opening: %s", path)
			}
			if openedLinksKnown && openedLinks == 1 {
				return writeAndClose(file, data)
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
		return replaceAppended(path, info, data)
	}
}

func linkCount(info os.FileInfo) (uint64, bool) {
	value := reflect.ValueOf(info.Sys())
	if !value.IsValid() {
		return 0, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return 0, false
	}
	field := value.FieldByName("Nlink")
	if !field.IsValid() {
		return 0, false
	}
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		links := field.Int()
		if links >= 0 {
			return uint64(links), true
		}
	}
	return 0, false
}

func replaceAppended(path string, expected os.FileInfo, data []byte) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	if !os.SameFile(expected, openedInfo) {
		_ = file.Close()
		return fmt.Errorf("append target changed while opening: %s", path)
	}
	existing, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	return Replace(path, append(existing, data...), expected.Mode())
}

func writeAndClose(file io.WriteCloser, data []byte) error {
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
