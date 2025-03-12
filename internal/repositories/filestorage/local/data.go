package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	domainmodels "github.com/DyadyaRodya/GophKeeper/internal/domain/models"
	"github.com/DyadyaRodya/GophKeeper/pkg/files"
)

const (
	dataPostfix = ".bin"
	metaPostfix = ".meta"
)

func (s *DataStorageLocal) SaveData(ctx context.Context, meta *domainmodels.DataInfo, data []byte) error {
	userDir := filepath.Join(s.dir, meta.OwnerUUID)
	if !files.DirectoryExists(userDir) {
		err := os.MkdirAll(userDir, 0700)
		if err != nil {
			return fmt.Errorf("DataStorageLocal.SaveData os.MkdirAll: %w", err)
		}
	}

	metaPath := filepath.Join(userDir, fmt.Sprintf("%s.%s", meta.UUID, metaPostfix))
	dataPath := filepath.Join(userDir, fmt.Sprintf("%s.%s", meta.UUID, dataPostfix))

	metaFile, err := os.Create(metaPath)
	if err != nil {
		return fmt.Errorf("DataStorageLocal.SaveData os.Create: %w", err)
	}
	defer metaFile.Close()
	encoder := json.NewEncoder(metaFile)

	dataFile, err := os.Create(dataPath)
	if err != nil {
		return fmt.Errorf("DataStorageLocal.SaveData os.Create: %w", err)
	}
	defer dataFile.Close()

	done := make(chan error, 1)
	go func() {
		if len(data) > 0 {
			_, err := dataFile.Write(data)
			if err != nil {
				done <- fmt.Errorf("DataStorageLocal.SaveData Write data: %w", err)
				return
			}
		}

		err := encoder.Encode(meta)
		if err != nil {
			err = fmt.Errorf("DataStorageLocal.SaveData: encoder.Encode: %w", err)
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

func (s *DataStorageLocal) ReadDataWithMeta(
	ctx context.Context,
	userUUID string,
	dataUUID string,
) (*domainmodels.DataInfo, []byte, error) {
	userDir := filepath.Join(s.dir, userUUID)
	metaPath := filepath.Join(userDir, fmt.Sprintf("%s.%s", dataUUID, metaPostfix))
	dataPath := filepath.Join(userDir, fmt.Sprintf("%s.%s", dataUUID, dataPostfix))
	if !files.FileExists(metaPath) || !files.FileExists(dataPath) {
		return nil, nil, domainmodels.ErrDataInfoNotFound
	}

	metaFile, err := os.Open(metaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("DataStorageLocal.ReadDataWithMeta: os.Open: %w", err)
	}
	defer metaFile.Close()

	decoder := json.NewDecoder(metaFile)
	done := make(chan error, 1)
	meta := &domainmodels.DataInfo{}
	var data []byte
	go func() {
		err := decoder.Decode(meta)
		if err != nil {
			done <- fmt.Errorf("DataStorageLocal.ReadDataWithMeta: decoder.Decode: %w", err)
			return
		}

		if !meta.IsDeleted {
			data, err = os.ReadFile(dataPath)
			if err != nil {
				err = fmt.Errorf("DataStorageLocal.ReadDataWithMeta: os.ReadFile: %w", err)
			}
		}
		done <- err
	}()

	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return nil, nil, ctx.Err()
	case err := <-done:
		// Read operation completed, return its result
		if err != nil {
			return nil, nil, err
		}
		return meta, data, nil
	}
}

func (s *DataStorageLocal) ListData(ctx context.Context, userUUID string) ([]*domainmodels.DataInfo, error) {
	userDir := filepath.Join(s.dir, userUUID)
	if !files.DirectoryExists(userDir) {
		return []*domainmodels.DataInfo{}, nil
	}

	dirEntries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, fmt.Errorf("DataStorageLocal.ListData os.ReadDir: %w", err)
	}

	res := make([]*domainmodels.DataInfo, 0, len(dirEntries)/2) // half is data files
	// Channel to collect file contents
	contentChan := make(chan *domainmodels.DataInfo)

	// errgroup to wait for all goroutines to finish
	eg, _ := errgroup.WithContext(ctx)
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), metaPostfix) {
			filePath := filepath.Join(userDir, entry.Name())
			eg.Go(func() error {
				var metaFile *os.File
				metaFile, err = os.Open(filePath)
				if err != nil {
					return fmt.Errorf("DataStorageLocal.ListData: os.Open: %w", err)
				}
				defer metaFile.Close()

				decoder := json.NewDecoder(metaFile)
				meta := &domainmodels.DataInfo{}
				err = decoder.Decode(meta)
				if err != nil {
					return fmt.Errorf("DataStorageLocal.ListData: decoder.Decode: %w", err)
				}
				contentChan <- meta
				return nil
			})
		}
	}

	// Wait for all goroutines to finish
	go func() {
		defer close(contentChan) // close chanel when awaited to unlock main loop
		err = eg.Wait()
	}()

	// collect data from chan
	for meta := range contentChan {
		res = append(res, meta)
	}
	if err != nil {
		return nil, fmt.Errorf("DataStorageLocal.ListData eg.Wait: %w", err)
	}
	return res, nil
}

func (s *DataStorageLocal) DeleteData(ctx context.Context, meta *domainmodels.DataInfo) error {
	userDir := filepath.Join(s.dir, meta.OwnerUUID)

	done := make(chan error, 1)

	go func() {
		metaPath := filepath.Join(userDir, fmt.Sprintf("%s.%s", meta.UUID, metaPostfix))
		if files.FileExists(metaPath) {
			err := os.Remove(metaPath)
			if err != nil {
				done <- fmt.Errorf("DataStorageLocal.DeleteData: os.Remove: %w", err)
				return
			}
		}

		dataPath := filepath.Join(userDir, fmt.Sprintf("%s.%s", meta.UUID, dataPostfix))
		if files.FileExists(dataPath) {
			err := os.Remove(dataPath)
			if err != nil {
				done <- fmt.Errorf("DataStorageLocal.DeleteData: os.Remove: %w", err)
				return
			}
		}
		done <- nil
	}()

	select {
	case <-ctx.Done():
		// Context was canceled, return the context error
		return ctx.Err()
	case err := <-done:
		// Read operation completed, return its result
		if err != nil {
			return err
		}
		return nil
	}
}
