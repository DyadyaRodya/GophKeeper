package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/pkg/files"
)

const sessionKeyStorage = "sessionkey"

func (s *KeyStorageLocal) SaveKeys(ctx context.Context, keysInfo *domainmodels.ShortKeyInfo) error {
	if !files.DirectoryExists(s.dir) {
		err := os.MkdirAll(s.dir, 0700)
		if err != nil {
			return fmt.Errorf("KeyStorageLocal.SaveKeys os.MkdirAll: %w", err)
		}
	}

	filePath := filepath.Join(s.dir, sessionKeyStorage)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("KeyStorageLocal.SaveKeys: os.Create: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	done := make(chan error, 1)
	go func() {
		err := encoder.Encode(keysInfo)
		if err != nil {
			err = fmt.Errorf("KeyStorageLocal.SaveKeys: encoder.Encode: %w", err)
		}
		done <- err
	}()
	// Wait for either write to complete or the context to be canceled
	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return ctx.Err()
	case err := <-done:
		// Write operation completed, return its result
		return err
	}
}

func (s *KeyStorageLocal) ReadKeys(ctx context.Context) (*domainmodels.ShortKeyInfo, error) {
	filePath := filepath.Join(s.dir, sessionKeyStorage)
	if !files.FileExists(filePath) {
		return nil, domainmodels.ErrUserKeysNotFound
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("KeyStorageLocal.ReadKeys: os.Open: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	done := make(chan error, 1)
	keysInfo := &domainmodels.ShortKeyInfo{}
	go func() {
		err := decoder.Decode(keysInfo)
		if err != nil {
			err = fmt.Errorf("KeyStorageLocal.SaveKeys: decoder.Decode: %w", err)
		}
		done <- err
	}()

	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return nil, ctx.Err()
	case err := <-done:
		// Read operation completed, return its result
		if err != nil {
			return nil, err
		}
		return keysInfo, nil
	}
}
