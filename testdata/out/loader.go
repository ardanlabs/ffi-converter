package calculator

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jupiterrider/ffi"
)

var lib ffi.Lib

func Load(path string) error {
	var err error
	lib, err = ffi.Load(getLibraryPath(path))
	if err != nil {
		return fmt.Errorf("failed to load library: %w", err)
	}

	if err := loadFuncs(); err != nil {
		return err
	}

	return nil
}

func getLibraryPath(basePath string) string {
	var filename string
	switch runtime.GOOS {
	case "linux", "freebsd":
		filename = "libcalculator.so"
	case "darwin":
		filename = "libcalculator.dylib"
	case "windows":
		filename = "calculator.dll"
	default:
		filename = "libcalculator.so"
	}
	return filepath.Join(basePath, filename)
}
