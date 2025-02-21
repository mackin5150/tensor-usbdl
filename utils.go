package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func isDir(paths ...string) error {
	path := pathJoin(paths...)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("Directory '%s' does not exist", path)
		}
		return fmt.Errorf("Error opening directory '%s': %v", path, err)
	}
	return nil
}

func isFile(paths ...string) error {
	path := pathJoin(paths...)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("File '%s' does not exist", path)
		}
		return fmt.Errorf("Error opening file '%s': %v", path, err)
	}
	return nil
}

func pathJoin(paths ...string) string {
	path := strings.Join(paths, "/")
	if runtime.GOOS == "windows" {
		strings.ReplaceAll(path, "/", "\\")
	} else {
		strings.ReplaceAll(path, "\\", "/")
	}
	return path
}
