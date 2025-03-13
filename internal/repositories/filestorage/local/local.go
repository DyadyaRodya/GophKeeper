package local

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	workDir = ".gophkeeper"
)

type KeyStorageLocal struct {
	dir string
}

func NewKeyStorageLocal() (*KeyStorageLocal, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("NewKeyStorageLocal: could not get home directory: %w", err)
	}
	dir := filepath.Join(homeDir, workDir)
	return &KeyStorageLocal{dir: dir}, nil
}

type DataStorageLocal struct {
	dir string
}

func NewDataStorageLocal() (*DataStorageLocal, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("NewDataStorageLocal: could not get home directory: %w", err)
	}
	dir := filepath.Join(homeDir, workDir)
	return &DataStorageLocal{dir: dir}, nil
}
