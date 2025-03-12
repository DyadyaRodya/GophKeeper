package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/pkg/files"
)

type DataStorageServer struct {
	dir string
}

func NewDataStorageServer(dir string) *DataStorageServer {
	return &DataStorageServer{dir: dir}
}

func (s *DataStorageServer) SaveData(ctx context.Context, meta *domainmodels.DataInfo, data []byte) error {
	userDir := filepath.Join(s.dir, meta.OwnerUUID)
	if !files.DirectoryExists(userDir) {
		err := os.MkdirAll(userDir, 0700)
		if err != nil {
			return fmt.Errorf("DataStorageServer.SaveData os.MkdirAll: %w", err)
		}
	}
	dataPath := filepath.Join(userDir, meta.UUID)
	dataFile, err := os.Create(dataPath)
	if err != nil {
		return fmt.Errorf("DataStorageServer.SaveData os.Create: %w", err)
	}
	defer dataFile.Close()

	done := make(chan error, 1)
	go func() {
		_, err := dataFile.Write(data)
		if err != nil {
			err = fmt.Errorf("DataStorageServer.SaveData Write data: %w", err)
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

func (s *DataStorageServer) ReadData(ctx context.Context, userUUID, dataUUID string) ([]byte, error) {
	userDir := filepath.Join(s.dir, userUUID)
	dataPath := filepath.Join(userDir, dataUUID)
	if !files.FileExists(dataPath) {
		return nil, domainmodels.ErrDataInfoNotFound
	}

	done := make(chan error, 1)
	var data []byte
	go func() {
		var err error
		data, err = os.ReadFile(dataPath)
		if err != nil {
			err = fmt.Errorf("DataStorageServer.ReadData: os.ReadFile: %w", err)
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
		return data, nil
	}
}

func (s *DataStorageServer) DeleteData(ctx context.Context, meta *domainmodels.DataInfo) error {
	dataPath := filepath.Join(s.dir, meta.OwnerUUID, meta.UUID)
	if !files.FileExists(dataPath) {
		return nil
	}

	done := make(chan error, 1)

	go func() {
		err := os.Remove(dataPath)
		if err != nil {
			err = fmt.Errorf("DataStorageServer.DeleteData: os.Remove: %w", err)
		}
		done <- err
	}()

	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return ctx.Err()
	case err := <-done:
		// Read operation completed, return its result
		return err
	}
}
